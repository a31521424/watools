package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"watools/config"
	"watools/pkg/logger"
)

func pluginRoute(res http.ResponseWriter, req *http.Request) {
	relativePath := strings.TrimPrefix(req.URL.Path, "/api/plugin")
	pluginPath := path.Join(config.ProjectCacheDir(), "plugins", relativePath)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		logger.Error(err, fmt.Sprintf("Plugin file not found: %s", pluginPath))
		http.NotFound(res, req)
		return
	}
	http.ServeFile(res, req, pluginPath)
}
