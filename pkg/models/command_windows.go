package models

import (
	"os"
	"path/filepath"
	"strings"
)

func (a *ApplicationCommand) IsUserApplication() bool {
	path := strings.ToLower(filepath.Clean(a.Path))

	if homeDir, err := os.UserHomeDir(); err == nil {
		homeDir = strings.ToLower(filepath.Clean(homeDir))
		if strings.HasPrefix(path, homeDir) {
			return true
		}
	}

	userDirs := []string{
		os.Getenv("LOCALAPPDATA"),
		os.Getenv("APPDATA"),
	}
	for _, dir := range userDirs {
		if dir == "" {
			continue
		}
		dir = strings.ToLower(filepath.Clean(dir))
		if strings.HasPrefix(path, dir) {
			return true
		}
	}

	return false
}
