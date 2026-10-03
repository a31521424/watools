package handler

import (
	"net/http"
	"strings"
)

type WaHandler struct {
	http.Handler
	isEnabled func(string) bool
}

func NewWaHandler(selectors ...func(string) bool) *WaHandler {
	var enabled func(string) bool
	if len(selectors) > 0 {
		enabled = selectors[0]
	}
	return &WaHandler{isEnabled: enabled}
}

func (w *WaHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, "/api") {
		return
	}

	url := req.URL.Path

	if strings.HasPrefix(url, "/api/application-icon") {
		applicationIconRoute(res, req)
	} else if strings.HasPrefix(url, "/api/plugin") {
		pluginRouteWithState(res, req, w.isEnabled)
	} else {
		http.NotFound(res, req)
	}
}

// WithAssets keeps the revision-aware API routes alongside the embedded shell.
// Wails' own runtime routes are handled before this application handler.
func WithAssets(assets http.Handler, selectors ...func(string) bool) http.Handler {
	api := NewWaHandler(selectors...)
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/api" || strings.HasPrefix(req.URL.Path, "/api/") {
			api.ServeHTTP(res, req)
			return
		}
		// v3 does not provide v2's index fallback for client-side routes.
		// Keep missing static/plugin assets as errors rather than serving HTML.
		switch req.URL.Path {
		case "/plugin", "/plugin-management", "/log-management", "/update-management":
			req = req.Clone(req.Context())
			req.URL.Path = "/"
			req.URL.RawPath = ""
		}
		assets.ServeHTTP(res, req)
	})
}
