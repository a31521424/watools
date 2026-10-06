package host

import (
	"context"
	"encoding/json"
	"github.com/wailsapp/wails/v3/pkg/application"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"watools/internal/api"
)

type fakePluginWindow struct {
	shown, closed atomic.Int32
	onClose       func()
}

func TestWindowOwnedHTTPRequestsDoNotCollideOrCancelAnotherWindow(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { started <- struct{}{}; <-r.Context().Done() }))
	defer server.Close()
	core := api.NewWaApi()
	service := NewSystemService(core)
	service.ServiceStartup(context.Background(), application.ServiceOptions{})
	defer service.ServiceShutdown()
	ctxA := context.WithValue(context.Background(), application.WindowKey, namedTestWindow("plugin-a"))
	ctxB := context.WithValue(context.Background(), application.WindowKey, namedTestWindow("plugin-b"))
	a, b := make(chan error, 1), make(chan error, 1)
	for _, task := range []struct {
		ctx    context.Context
		result chan error
	}{{ctxA, a}, {ctxB, b}} {
		go func(ctx context.Context, result chan error) {
			_, err := service.HttpProxyApi(ctx, api.HttpProxyRequest{URL: server.URL, RequestID: "same", Timeout: 5000})
			result <- err
		}(task.ctx, task.result)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("same public request ID collided across windows")
		}
	}
	core.CancelHttpProxyPrefix("plugin-a:")
	select {
	case err := <-a:
		errorCode(t, err, "CANCELLED")
	case <-time.After(time.Second):
		t.Fatal("closing window did not cancel request")
	}
	select {
	case <-b:
		t.Fatal("another window was cancelled")
	default:
	}
	service.CancelHttpProxyApi(ctxB, "same")
	select {
	case err := <-b:
		errorCode(t, err, "CANCELLED")
	case <-time.After(time.Second):
		t.Fatal("owner could not cancel request")
	}
}

func (w *fakePluginWindow) Show() { w.shown.Add(1) }
func (w *fakePluginWindow) Close() {
	w.closed.Add(1)
	if w.onClose != nil {
		w.onClose()
	}
}

type namedTestWindow string

func (w namedTestWindow) Name() string { return string(w) }

func TestPluginWindowHandshakeAndMultipleInstances(t *testing.T) {
	var windows []*fakePluginWindow
	manager := pluginWindows{factory: func(session PluginWindowSession, closed func()) (PluginWindow, error) {
		w := &fakePluginWindow{onClose: closed}
		windows = append(windows, w)
		return w, nil
	}}
	defer manager.shutdown()
	request := DetachPluginRequest{PackageID: "watools.plugin.json", Context: PluginLaunchContext{Input: PluginInput{Value: "  \x00😀 original", ValueType: "clipboard"}}, State: json.RawMessage(`{"kind":"hooks","value":{"text":"edited"}}`)}
	id, err := manager.create(request, "JSON")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.create(request, "JSON"); err == nil {
		t.Fatal("duplicate pending detach allowed")
	}
	if windows[0].shown.Load() != 0 {
		t.Fatal("window shown before state restored")
	}
	if _, err := manager.get(id, "launcher"); err == nil {
		t.Fatal("launcher may not claim child context")
	}
	if err := manager.ready(id, "other-window"); err == nil {
		t.Fatal("another window acknowledged session")
	}
	session, err := manager.get(id, "plugin-"+id)
	if err != nil || session.Context.Input.Value != request.Context.Input.Value {
		t.Fatal("origin corrupted")
	}
	if err := manager.ready(id, session.WindowName); err != nil {
		t.Fatal(err)
	}
	manager.expire(id)
	if windows[0].closed.Load() != 0 {
		t.Fatal("a late timeout closed an acknowledged window")
	}
	if err := manager.ready(id, session.WindowName); err != nil || windows[0].shown.Load() != 1 {
		t.Fatal("ready is not idempotent")
	}
	second, err := manager.create(request, "JSON")
	if err != nil || second == id {
		t.Fatal("same plugin did not create independent instance")
	}
	manager.expire(second)
	if windows[0].closed.Load() != 0 || windows[1].closed.Load() != 1 {
		t.Fatal("closing one instance closed another")
	}
	if _, err := manager.get(second, "plugin-"+second); err == nil {
		t.Fatal("closed session returned context")
	}
	manager.closePackage(request.PackageID)
	if windows[0].closed.Load() != 1 || len(manager.items) != 0 {
		t.Fatal("disable/uninstall did not release instances")
	}
}

func TestPluginWindowCreationAndShutdownDoNotHoldLifecycleLock(t *testing.T) {
	manager := pluginWindows{}
	window := &fakePluginWindow{}
	manager.factory = func(_ PluginWindowSession, _ func()) (PluginWindow, error) { manager.shutdown(); return window, nil }
	if _, err := manager.create(DetachPluginRequest{}, "test"); err == nil || window.closed.Load() != 1 {
		t.Fatal("creation after shutdown leaked native window")
	}
	if len(manager.items) != 0 {
		t.Fatal("shutdown resurrected context")
	}
}

func TestPluginWindowSourceAndUnavailableMode(t *testing.T) {
	ctx := context.WithValue(context.Background(), application.WindowKey, namedTestWindow("plugin-test"))
	if sourceWindow(ctx) != "plugin-test" {
		t.Fatal("binding window identity missing")
	}
	s := &PluginService{}
	if _, err := s.DetachPluginApi(ctx, DetachPluginRequest{}); err == nil {
		t.Fatal("child detached another plugin")
	}
	if _, err := s.windows.create(DetachPluginRequest{}, "test"); err == nil {
		t.Fatal("mock/server created native window")
	}
}
