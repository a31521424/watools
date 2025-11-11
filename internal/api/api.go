package api

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"sync"
	"time"
)

var (
	waApiInstance *WaApi
	waApiOnce     sync.Once
)

type WaApi struct {
}

func GetWaApi() *WaApi {
	waApiOnce.Do(func() {
		waApiInstance = &WaApi{}
	})
	return waApiInstance
}

func (a *WaApi) SaveBase64Image(base64Data string) string {
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return ""
	}
	downloadFolder := []string{"Downloads", "downloads", "Download", "download"}
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	for _, ddn := range downloadFolder {
		downloadPath := path.Join(userHomeDir, ddn)
		if _, err := os.Stat(downloadPath); err == nil {
			filePath := path.Join(downloadPath, fmt.Sprint("wa-image", time.Now().Unix(), ".png"))
			err = os.WriteFile(filePath, imgBytes, 0644)
			if err != nil {
				continue
			}
			return filePath
		}
	}
	return ""
}
