package handler

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/utils"
)

func pluginRoute(res http.ResponseWriter, req *http.Request) {
	pluginBasePath := config.ProjectCacheDir() + string(os.PathSeparator) + "plugins"
	relativePath := strings.TrimPrefix(req.URL.Path, "/api/plugin/")
	if relativePath == req.URL.Path {
		http.NotFound(res, req)
		return
	}
	packageID := strings.SplitN(relativePath, "/", 2)[0]
	if err := utils.ValidatePluginPackageID(packageID); err != nil {
		logger.Error(err, fmt.Sprintf("Invalid plugin packageId in asset path: %s", req.URL.Path))
		http.NotFound(res, req)
		return
	}

	pluginPath, err := utils.ResolvePathWithinBase(pluginBasePath, relativePath)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Invalid plugin asset path: %s", req.URL.Path))
		http.NotFound(res, req)
		return
	}

	fileStat, err := os.Stat(pluginPath)
	if err != nil {
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

	contentType := mime.TypeByExtension(filepath.Ext(pluginPath))
	if contentType != "" {
		res.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(res, req, file.Name(), fileStat.ModTime(), file)
}
