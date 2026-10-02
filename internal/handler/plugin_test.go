package handler

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"watools/config"
	"watools/internal/plugin"
	"watools/pkg/db"
	"watools/pkg/models"
)

func TestPluginAssetsConfinedAndDisabledPackagesDenied(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", filepath.Join(root, "state"))
	t.Setenv("WATOOLS_CACHE_DIR", filepath.Join(root, "cache"))
	config.ParseProject([]byte(`{"name":"watools"}`))
	archive := filepath.Join(root, "test.wt")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range map[string]string{"manifest.json": `{"packageId":"watools.plugin.test","name":"Test","version":"1.0.0","entry":"app.js"}`, "app.js": "export default []"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte(data))
	}
	writer.Close()
	file.Close()
	backend := plugin.GetWaPlugin()
	backend.OnStartup(context.Background())
	defer db.GetWaDB().Close()
	if err := backend.InstallPlugin(archive); err != nil {
		t.Fatal(err)
	}
	packageRoot, revision, err := models.ResolveInstalledPlugin(filepath.Join(config.ProjectDataDir(), "plugins"), "watools.plugin.test")
	if err != nil {
		t.Fatal(err)
	}
	prefix := "/api/plugin/watools.plugin.test/_v/" + revision + "/"
	request := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		pluginRoute(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	response := request(prefix + "app.js")
	if response.Code != 200 || response.Body.String() != "export default []" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("asset not served: %d %s", response.Code, response.Body.String())
	}
	if response := request(prefix + "../manifest.json"); response.Code != 404 {
		t.Fatal("traversal served")
	}
	outside := filepath.Join(root, "private")
	os.WriteFile(outside, []byte("private"), 0600)
	if err := os.Symlink(outside, filepath.Join(packageRoot, "escape.js")); err == nil {
		if response := request(prefix + "escape.js"); response.Code != 404 {
			t.Fatal("escaping symlink served")
		}
	} else {
		t.Logf("symlink creation unavailable: %v", err)
	}
	if err := backend.TogglePlugin("watools.plugin.test", false); err != nil {
		t.Fatal(err)
	}
	if response := request(prefix + "app.js"); response.Code != 404 {
		t.Fatal("disabled package still served")
	}
	if _, err := backend.GetStorage("watools.plugin.test", "key"); err == nil {
		t.Fatal("disabled storage request accepted")
	}
}
