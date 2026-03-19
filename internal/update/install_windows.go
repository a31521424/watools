//go:build windows

package update

import (
	"fmt"
	"os/exec"
)

func installUpdateAsset(downloadedPath string) (InstallResult, error) {
	cmd := exec.Command("cmd", "/c", "start", "", downloadedPath)
	if err := cmd.Start(); err != nil {
		return InstallResult{}, fmt.Errorf("failed to launch installer: %w", err)
	}

	return InstallResult{
		Status:     "installing",
		Message:    "安装器已启动，应用即将退出",
		OpenPath:   downloadedPath,
		ShouldQuit: true,
	}, nil
}
