package models

import (
	"os"
	"path/filepath"
	"strings"
)

func (a *ApplicationCommand) IsUserApplication() bool {

	if strings.HasPrefix(a.Path, "/Applications") {
		return true
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		userAppPath := filepath.Join(homeDir, "Applications")
		if strings.HasPrefix(a.Path, userAppPath) {
			return true
		}
	}

	return false
}
