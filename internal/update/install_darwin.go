//go:build darwin

package update

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func installUpdateAsset(downloadedPath string) (InstallResult, error) {
	currentBundlePath, err := currentAppBundlePath()
	if err != nil {
		return InstallResult{
			Status:                "manual",
			Message:               "当前不是从 .app 安装包运行，已下载更新包，请手动替换应用",
			OpenPath:              downloadedPath,
			RequiresManualInstall: true,
		}, nil
	}

	targetDir := filepath.Dir(currentBundlePath)
	if err := ensureDirWritable(targetDir); err != nil {
		return InstallResult{
			Status:                "manual",
			Message:               "当前应用目录不可写，已下载更新包，请手动替换应用",
			OpenPath:              downloadedPath,
			RequiresManualInstall: true,
		}, nil
	}

	stagingDir, err := os.MkdirTemp("", "watools-update-*")
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to create update staging directory: %w", err)
	}

	extractDir := filepath.Join(stagingDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	if output, err := exec.Command("ditto", "-x", "-k", downloadedPath, extractDir).CombinedOutput(); err != nil {
		return InstallResult{}, fmt.Errorf("failed to extract update archive: %w: %s", err, strings.TrimSpace(string(output)))
	}

	nextBundlePath, err := findExtractedAppBundle(extractDir)
	if err != nil {
		return InstallResult{}, err
	}

	scriptPath := filepath.Join(stagingDir, "apply-update.sh")
	script := fmt.Sprintf(`#!/bin/sh
sleep 2
rm -rf %q
ditto %q %q
xattr -dr com.apple.quarantine %q >/dev/null 2>&1 || true
open %q
`, currentBundlePath, nextBundlePath, currentBundlePath, currentBundlePath, currentBundlePath)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to write update script: %w", err)
	}

	if err := exec.Command("sh", scriptPath).Start(); err != nil {
		return InstallResult{}, fmt.Errorf("failed to start update script: %w", err)
	}

	return InstallResult{
		Status:     "installing",
		Message:    "更新已准备完成，应用即将退出并替换自身",
		OpenPath:   currentBundlePath,
		ShouldQuit: true,
	}, nil
}

func currentAppBundlePath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	cleanPath := filepath.Clean(executablePath)
	marker := string(filepath.Separator) + "Contents" + string(filepath.Separator) + "MacOS" + string(filepath.Separator)
	index := strings.Index(cleanPath, marker)
	if index <= 0 {
		return "", fmt.Errorf("executable is not inside an .app bundle")
	}

	bundlePath := cleanPath[:index]
	if !strings.HasSuffix(strings.ToLower(bundlePath), ".app") {
		return "", fmt.Errorf("executable is not inside an .app bundle")
	}
	return bundlePath, nil
}

func ensureDirWritable(dir string) error {
	probeFile, err := os.CreateTemp(dir, ".watools-update-permission-*")
	if err != nil {
		return err
	}
	probePath := probeFile.Name()
	if err := probeFile.Close(); err != nil {
		_ = os.Remove(probePath)
		return err
	}
	return os.Remove(probePath)
}

func findExtractedAppBundle(root string) (string, error) {
	var bundlePath string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
			bundlePath = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to inspect extracted update bundle: %w", err)
	}
	if bundlePath == "" {
		return "", fmt.Errorf("update archive does not contain an .app bundle")
	}
	return bundlePath, nil
}
