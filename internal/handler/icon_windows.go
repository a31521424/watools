package handler

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func icon2Png(iconPath string, pngPath string) error {
	iconPath = strings.TrimSpace(iconPath)
	if iconPath == "" {
		return fmt.Errorf("icon path is empty")
	}

	if strings.EqualFold(filepath.Ext(iconPath), ".png") {
		return copyFile(iconPath, pngPath)
	}

	script := "Add-Type -AssemblyName System.Drawing; " +
		"$source = '" + escapePowerShellSingleQuoted(iconPath) + "'; " +
		"$dest = '" + escapePowerShellSingleQuoted(pngPath) + "'; " +
		"if (-not (Test-Path -LiteralPath $source)) { exit 1 }; " +
		"$icon = $null; " +
		"try { if ($source.ToLower().EndsWith('.ico')) { $icon = New-Object System.Drawing.Icon($source) } else { $icon = [System.Drawing.Icon]::ExtractAssociatedIcon($source) } } catch {}; " +
		"if ($icon -eq $null) { exit 2 }; " +
		"$bmp = $icon.ToBitmap(); " +
		"$bmp.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png)"

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to convert icon to png: %w\n%s", err, output)
	}
	return nil
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
