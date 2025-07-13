package handler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func icon2Png(iconPath string, pngPath string) error {
	return icns2Png(strings.TrimSpace(iconPath), pngPath)
}

func icns2Png(icnsPath string, pngPath string) error {
	if _, err := os.Stat(icnsPath); os.IsNotExist(err) {
		return fmt.Errorf("failed to find icns file: %w", err)
	}
	cmd := exec.Command("sips", "-s", "format", "png", icnsPath, "--out", pngPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to convert icns to png: %w\n%s", err, output)
	}
	return nil
}
