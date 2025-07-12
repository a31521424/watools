package handler

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"watools/config"
)

func getPngIconCachePath(iconPath string) string {
	cacheDir := filepath.Join(config.ProjectCacheDir(), "icons")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			fmt.Println(err)
			return ""
		}
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
			fmt.Println(err)
			return
		}
	}
	http.ServeFile(res, req, pngIconPath)
}
