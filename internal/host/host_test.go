package host

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"github.com/wailsapp/wails/v3/pkg/application"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"watools/internal/api"
	"watools/internal/command"
	"watools/internal/plugin"
	"watools/pkg/db"
)

func errorCode(t *testing.T, err error, want string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != want {
		t.Fatalf("error = %v, want %s", err, want)
	}
	var wire map[string]string
	if err := json.Unmarshal(MarshalError(err), &wire); err != nil || wire["code"] != want {
		t.Fatalf("invalid error wire: %+v, %v", wire, err)
	}
}
func TestTypedRequestsRejectInvalidTypesAndMissingUsage(t *testing.T) {
	for _, raw := range []string{`{"url":"https://example.test","timeout":1.5}`, `{"url":"https://example.test","headers":{"Accept":4}}`, `{"body":42}`} {
		var request api.HttpProxyRequest
		if err := json.Unmarshal([]byte(raw), &request); err == nil {
			t.Fatalf("accepted bad HTTP argument: %s", raw)
		} else {
			errorCode(t, &Error{Code: "INVALID_ARGUMENT", Message: "调用参数格式不正确", cause: err}, "INVALID_ARGUMENT")
		}
	}
	s := NewCommandService(command.NewWaLaunch(), nil)
	for _, raw := range []string{`[{"id":"test","lastUsedAt":"2026-10-04T00:00:00Z"}]`, `[{"id":"test","lastUsedAt":"2026-10-04T00:00:00Z","usedCount":-1}]`, `[{"id":"","lastUsedAt":"2026-10-04T00:00:00Z","usedCount":0}]`, `[{"id":"test","usedCount":0}]`} {
		var request []ApplicationUsage
		if e := json.Unmarshal([]byte(raw), &request); e != nil {
			t.Fatal(e)
		}
		errorCode(t, s.UpdateApplicationUsageApi(request), "INVALID_ARGUMENT")
	}
	var request []ApplicationUsage
	if err := json.Unmarshal([]byte(`[{"id":"test","lastUsedAt":"2026-10-04T00:00:00Z","usedCount":1.5}]`), &request); err == nil {
		t.Fatal("fractional count silently truncated")
	}
	errorCode(t, s.TriggerCommandApi("id", "unknown"), "INVALID_ARGUMENT")
}
func TestPluginStorageAndUsageWithoutWailsEventLoop(t *testing.T) {
	t.Setenv("WATOOLS_DATA_DIR", t.TempDir())
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	database := db.GetWaDB()
	t.Cleanup(func() { database.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	core := plugin.NewWaPlugin()
	service := NewPluginService(core, nil)
	service.ServiceStartup(ctx, application.ServiceOptions{})
	defer service.ServiceShutdown()
	packagePath := filepath.Join(t.TempDir(), "test.wt")
	file, e := os.Create(packagePath)
	if e != nil {
		t.Fatal(e)
	}
	archive := zip.NewWriter(file)
	for name, body := range map[string]string{"manifest.json": `{"packageId":"watools.plugin.contract","name":"Contract","version":"1.0.0","entry":"app.js"}`, "app.js": "export default [];"} {
		w, e := archive.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write([]byte(body)); e != nil {
			t.Fatal(e)
		}
	}
	if e := archive.Close(); e != nil {
		t.Fatal(e)
	}
	file.Close()
	if e := service.InstallPluginApi(packagePath); e != nil {
		t.Fatal(e)
	}
	storage := NewStorageService(core)
	key := StorageKeyRequest{PackageID: "watools.plugin.contract", Key: "probe"}
	if e := storage.SetPluginStorageKeyApi(StorageSetRequest{PackageID: key.PackageID, Key: key.Key, Value: json.RawMessage(`{"__proto__":"data","value":[1,null]}`)}); e != nil {
		t.Fatal(e)
	}
	value, e := storage.GetPluginStorageKeyApi(key)
	if e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(value)
	if !strings.Contains(string(data), `"__proto__":"data"`) {
		t.Fatalf("lost data: %s", data)
	}
	errorCode(t, storage.SetPluginStorageKeyApi(StorageSetRequest{PackageID: key.PackageID, Key: key.Key}), "INVALID_ARGUMENT")
	if e := storage.SetPluginStorageKeyApi(StorageSetRequest{PackageID: key.PackageID, Key: key.Key, Value: json.RawMessage(`null`)}); e != nil {
		t.Fatal(e)
	}
	if v, e := storage.GetPluginStorageKeyApi(key); e != nil || string(v) != "null" {
		t.Fatalf("explicit null: %v, %v", v, e)
	}
	keys, e := storage.ListPluginStorageKeysApi(PackageRequest{PackageID: key.PackageID})
	if e != nil || len(keys) != 1 {
		t.Fatalf("null key disappeared: %v %v", keys, e)
	}
	now := time.Now().UTC()
	count := int64(3)
	bad := int64(-1)
	errorCode(t, service.UpdatePluginUsageApi([]PluginUsage{{PackageID: key.PackageID, LastUsedAt: &now, UsedCount: &count}, {PackageID: key.PackageID, LastUsedAt: &now, UsedCount: &bad}}), "INVALID_ARGUMENT")
	plugins, e := service.GetPluginsApi()
	if e != nil || len(plugins) != 1 || plugins[0].UsedCount != 0 {
		t.Fatalf("invalid batch partially committed: %v %v", plugins, e)
	}
	if e := service.UpdatePluginUsageApi([]PluginUsage{{PackageID: key.PackageID, LastUsedAt: &now, UsedCount: &count}}); e != nil {
		t.Fatal(e)
	}
	plugins, e = service.GetPluginsApi()
	if e != nil || plugins[0].UsedCount != count {
		t.Fatalf("usage was not durable: %v %v", plugins, e)
	}
	synthetic := "synthetic-test-value"
	if e := storage.SetPluginSecretApi(key.PackageID, "synthetic", &synthetic); e != nil {
		t.Fatal(e)
	}
	if v, e := storage.GetPluginSecretApi(key.PackageID, "synthetic"); e != nil || v != "synthetic-test-value" {
		t.Fatalf("secret roundtrip failed: %v", e)
	}
	errorCode(t, service.TogglePluginApi(key.PackageID, nil), "INVALID_ARGUMENT")
	errorCode(t, storage.SetPluginSecretApi(key.PackageID, "synthetic", nil), "INVALID_ARGUMENT")
	disabled := false
	if e := service.TogglePluginApi(key.PackageID, &disabled); e != nil {
		t.Fatal(e)
	}
	_, e = storage.GetPluginSecretApi(key.PackageID, "synthetic")
	errorCode(t, e, "PLUGIN_DISABLED")
	_, e = storage.GetPluginStorageKeyApi(key)
	errorCode(t, e, "PLUGIN_DISABLED")
	errorCode(t, service.InstallPluginByFileDialogApi(), "UNAVAILABLE")
}
func TestHTTPServiceCancellationAndShutdownWithoutDesktop(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { started <- struct{}{}; <-r.Context().Done() }))
	defer server.Close()
	s := NewSystemService(api.NewWaApi())
	s.ServiceStartup(context.Background(), application.ServiceOptions{})
	for _, shutdown := range []bool{false, true} {
		result := make(chan error, 1)
		go func() {
			_, e := s.HttpProxyApi(api.HttpProxyRequest{URL: server.URL, RequestID: "request", Timeout: 5000})
			result <- e
		}()
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("HTTP request never started")
		}
		if shutdown {
			s.ServiceShutdown()
		} else {
			s.CancelHttpProxyApi("request")
		}
		select {
		case e := <-result:
			errorCode(t, e, "CANCELLED")
			if !errors.Is(e, context.Canceled) {
				t.Fatal("underlying cause lost")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("request was not cancelled")
		}
	}
	errorCode(t, func() error { _, e := s.HttpProxyApi(api.HttpProxyRequest{URL: "file:///tmp/file"}); return e }(), "INVALID_ARGUMENT")
}
func TestErrorWireDoesNotExposeSensitiveUnderlyingDetails(t *testing.T) {
	source := errors.New("synthetic credential in request URL")
	err := failure("system.http", source)
	if strings.Contains(err.Error(), "credential") {
		t.Fatal("Wails outer message exposes underlying details")
	}
	if !errors.Is(err, source) {
		t.Fatal("underlying context lost")
	}
	wire := string(MarshalError(err))
	if strings.Contains(wire, "credential") || !strings.Contains(wire, "OPERATION_FAILED") {
		t.Fatal(wire)
	}
	var payload map[string]string
	if e := json.Unmarshal(MarshalError(errors.New("bad argument")), &payload); e != nil || payload["code"] != "INVALID_ARGUMENT" {
		t.Fatal("transport decoding error not classified")
	}
}

type testWindow struct{ w, h int }

func (w *testWindow) Size() (int, int) { return w.w, w.h }
func (w *testWindow) SetSize(width, height int) application.Window {
	w.w, w.h = width, height
	return nil
}
func TestLauncherGeometryWithoutDesktop(t *testing.T) {
	window := &testWindow{920, 64}
	service := NewAppService(nil, nil, func() WindowGeometry { return window })
	if e := service.SetLauncherSizeApi(920, 760); e != nil {
		t.Fatal(e)
	}
	size, e := service.GetLauncherSizeApi()
	if e != nil || size.Width != 920 || size.Height != 760 {
		t.Fatalf("size: %v %v", size, e)
	}
	errorCode(t, service.SetLauncherSizeApi(0, 760), "INVALID_ARGUMENT")
	size, _ = service.GetLauncherSizeApi()
	if size.Height != 760 {
		t.Fatal("invalid request changed geometry")
	}
}
