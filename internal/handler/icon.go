package handler

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"watools/config"
	"watools/pkg/logger"
)

func getPngIconCachePath(iconPath string) string {
	cacheDir := filepath.Join(config.ProjectCacheDir(), "icons")

	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		logger.Error(err, "Failed to create icon cache dir")
		return ""
	}

	hasher := fnv.New64a()
	hasher.Write([]byte(iconPath))
	iconFileName := fmt.Sprintf("%x.png", hex.EncodeToString(hasher.Sum(nil)))
	iconCachePath := filepath.Join(cacheDir, iconFileName)
	return iconCachePath
}

func HandleApplicationIcon(res http.ResponseWriter, req *http.Request) {
	IconPath := req.URL.Query().Get("path")
	pngIconPath := getPngIconCachePath(IconPath)
	if _, err := os.Stat(pngIconPath); os.IsNotExist(err) {
		err := icon2Png(IconPath, pngIconPath)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to convert icon to png %v", IconPath))
			return
		}
	}
	http.ServeFile(res, req, pngIconPath)
}
