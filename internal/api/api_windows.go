package api

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a *WaApi) OpenFolderWithPath(path string) {
	if strings.HasPrefix(path, "~/") {
		path = strings.TrimPrefix(path, "~/")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		path = filepath.Join(homeDir, path)
	}
	path = filepath.Clean(path)
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if !stat.IsDir() {
		_ = exec.Command("explorer", "/select,", path).Start()
	} else {
		_ = exec.Command("explorer", path).Start()
	}
}
