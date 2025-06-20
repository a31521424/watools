package launch

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"watools/config"
)

func GetIconCachePath(iconPath string) string {
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
