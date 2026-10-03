package handler

import (
	"fmt"
	"github.com/google/uuid"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"watools/config"
	"watools/internal/plugin"
	"watools/pkg/logger"
	"watools/pkg/models"
	"watools/pkg/utils"
)

func pluginRoute(res http.ResponseWriter, req *http.Request) {
	pluginRouteWithState(res, req, plugin.GetWaPlugin().IsEnabled)
}

func pluginRouteWithState(res http.ResponseWriter, req *http.Request, isEnabled func(string) bool) {
	pluginBasePath := filepath.Join(config.ProjectDataDir(), "plugins")
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

	if isEnabled == nil || !isEnabled(packageID) {
		http.NotFound(res, req)
		return
	}
	parts := strings.SplitN(relativePath, "/", 2)
	if len(parts) != 2 {
		http.NotFound(res, req)
		return
	}
	assetPath := parts[1]
	root, _, err := resolvePluginAssetRoot(pluginBasePath, packageID, &assetPath)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	// A directory handle prevents asset symlinks escaping the installed package.
	packageRoot, err := os.OpenRoot(root)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	defer packageRoot.Close()
	if _, err := utils.ResolvePathWithinBase(root, assetPath); err != nil {
		http.NotFound(res, req)
		return
	}
	file, err := packageRoot.Open(assetPath)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil || !fileStat.Mode().IsRegular() {
		http.NotFound(res, req)
		return
	}
	pluginPath := assetPath
	res.Header().Set("X-Content-Type-Options", "nosniff")
	contentType := mime.TypeByExtension(filepath.Ext(pluginPath))
	if contentType != "" {
		res.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(res, req, file.Name(), fileStat.ModTime(), file)
}

func resolvePluginAssetRoot(base, packageID string, assetPath *string) (string, string, error) {
	if strings.HasPrefix(*assetPath, "_v/") {
		parts := strings.SplitN(*assetPath, "/", 3)
		if len(parts) != 3 {
			return "", "", fmt.Errorf("invalid revision path")
		}
		if _, err := uuid.Parse(parts[1]); err != nil {
			return "", "", err
		}
		*assetPath = parts[2]
		return filepath.Join(base, packageID, "versions", parts[1]), parts[1], nil
	}
	return models.ResolveInstalledPlugin(base, packageID)
}
