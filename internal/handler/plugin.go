package handler

import (
	"fmt"
	"mime"
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
	fileStat, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		logger.Error(err, fmt.Sprintf("Plugin file not found: %s", pluginPath))
		http.NotFound(res, req)
		return
	}

	file, err := os.Open(pluginPath)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to open plugin file: %s", pluginPath))
		http.NotFound(res, req)
		return
	}
	defer file.Close()

	contentType := mime.TypeByExtension(path.Ext(pluginPath))
	if contentType != "" {
		res.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(res, req, file.Name(), fileStat.ModTime(), file)
}
