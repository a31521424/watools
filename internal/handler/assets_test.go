package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestAPIAndShellAssetRouting(t *testing.T) {
	shell := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("shell asset")) })
	handler := WithAssets(shell)
	for _, path := range []string{"/", "/assets/index.js", "/apiculture"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Body.String() != "shell asset" {
			t.Fatalf("%s did not reach shell handler", path)
		}
	}
	for _, path := range []string{"/api", "/api/unknown"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d", path, res.Code)
		}
	}
}

func TestShellRoutesReloadWithoutSwallowingMissingAssets(t *testing.T) {
	assets := fstest.MapFS{
		"frontend/dist/index.html":      {Data: []byte("<!DOCTYPE html><html>launcher</html>")},
		"frontend/dist/assets/index.js": {Data: []byte("export const ready = true;")},
	}
	handler := WithAssets(application.AssetFileServerFS(assets))
	for _, route := range []string{"/", "/plugin?packageId=probe&file=index.html", "/plugin-management", "/log-management", "/update-management"} {
		req := httptest.NewRequest("GET", route, nil)
		original := req.URL.String()
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK || res.Body.String() != "<!DOCTYPE html><html>launcher</html>" {
			t.Fatalf("%s cannot reload shell: %d %s", route, res.Code, res.Body.String())
		}
		if req.URL.String() != original {
			t.Fatalf("request URL was mutated: %s", route)
		}
	}
	for _, route := range []string{"/assets/missing.js", "/api/plugin/missing/index.html", "/unknown-route"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", route, nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("missing %s returned %d", route, res.Code)
		}
	}
}
