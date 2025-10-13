package handler

import (
	"net/http"
	"strings"
)

type WaHandler struct {
	http.Handler
}

func NewWaHandler() *WaHandler {
	return &WaHandler{}
}

func (w *WaHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, "/api") {
		return
	}

	url := req.URL.Path

	if strings.HasPrefix(url, "/api/application-icon") {
		applicationIconRoute(res, req)
	} else if strings.HasPrefix(url, "/api/plugin") {
		pluginRoute(res, req)
	} else {
		http.NotFound(res, req)
	}
}
