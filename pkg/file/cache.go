package file

import (
	"os"
	"path/filepath"
)

func GetAppCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(userCacheDir, "watools")
	return cacheDir
}
