package handler

import (
	"net/http"
	"os"
	"path"
	"strings"
	"watools/config"
)

func pluginRoute(res http.ResponseWriter, req *http.Request) {
	relativePath := strings.Replace(req.URL.Path, "/api/plugin", "", 1)
	pluginPath := path.Join(config.ProjectCacheDir(), "plugin", relativePath)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		http.NotFound(res, req)
		return
	}
	http.ServeFile(res, req, pluginPath)
}
