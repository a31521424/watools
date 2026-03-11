package api

import (
	"errors"
	"fmt"
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

func (a *WaApi) copyImageBytesToClipboard(imgBytes []byte) error {
	if len(imgBytes) == 0 {
		return fmt.Errorf("image data is empty")
	}

	tempFile, err := os.CreateTemp("", "watools-clipboard-*.png")
	if err != nil {
		return fmt.Errorf("failed to create temp image: %w", err)
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(imgBytes); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to write temp image: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to close temp image: %w", err)
	}
	defer os.Remove(tempPath)

	script := fmt.Sprintf(
		"Add-Type -AssemblyName System.Windows.Forms; "+
			"Add-Type -AssemblyName System.Drawing; "+
			"$path = '%s'; "+
			"$img = [System.Drawing.Image]::FromFile($path); "+
			"try { [System.Windows.Forms.Clipboard]::SetImage($img) } finally { $img.Dispose() }",
		strings.ReplaceAll(tempPath, "'", "''"),
	)

	cmd := exec.Command("powershell", "-STA", "-NoProfile", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to write image to clipboard: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}
