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
	url := strings.TrimPrefix(req.URL.Path, "/api")
	url = strings.TrimPrefix(url, "/")
	switch url {
	case "application-icon":
		HandleApplicationIcon(res, req)
	}
}
