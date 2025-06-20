package handler

import (
	"encoding/hex"
	"fmt"
	"github.com/jackmordaunt/icns/v3"
	"hash/fnv"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"watools/config"
)

func getPngIconCachePath(iconPath string) string {
	projectName := config.ProjectName()
	if projectName == "" {
		return ""
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(userCacheDir, config.ProjectName(), "icons")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return ""
		}
	}
	hasher := fnv.New64a()
	hasher.Write([]byte(iconPath))
	iconFileName := fmt.Sprintf("%x.png", hex.EncodeToString(hasher.Sum(nil)))
	iconCachePath := filepath.Join(cacheDir, iconFileName)
	return iconCachePath
}

func icns2Png(icnsPath string, pngPath string) error {
	icnsFile, err := os.Open(icnsPath)
	if err != nil {
		return fmt.Errorf("failed to open icns file '%s': %w", icnsPath, err)
	}
	defer icnsFile.Close()
	// TODO: optimize icon size
	pngImage, err := icns.Decode(icnsFile)
	if err != nil {
		return fmt.Errorf("failed to decode icns file '%s': %w", icnsPath, err)
	}

	pngFile, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("failed to create png file '%s': %w", pngPath, err)
	}
	defer pngFile.Close()

	err = png.Encode(pngFile, pngImage)
	if err != nil {
		return fmt.Errorf("failed to encode png data to '%s': %w", pngPath, err)
	}
	return nil
}

func HandleApplicationIcon(res http.ResponseWriter, req *http.Request) {
	IconPath := req.URL.Query().Get("path")
	pngIconPath := getPngIconCachePath(IconPath)
	if _, err := os.Stat(pngIconPath); os.IsNotExist(err) {
		err := icns2Png(IconPath, pngIconPath)
		if err != nil {
			fmt.Println(err)
		}
	}
	http.ServeFile(res, req, pngIconPath)
}
