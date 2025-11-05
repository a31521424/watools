package api

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"watools/pkg/logger"
)

func (a *WaApi) OpenFolderWithPath(path string) {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	logger.Info(fmt.Sprintf("Opening folder %s, isDir %s", path, stat.IsDir()))
	if !stat.IsDir() {
		_ = exec.Command("open", "-R", path).Start()
	} else {
		_ = exec.Command("open", path).Start()
	}
}
