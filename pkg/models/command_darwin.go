package models

import (
	"os"
	"path/filepath"
	"strings"
)

// isUserApplication returns true for user-installed applications on macOS.
// On macOS, user applications are those installed in:
// - /Applications (common user applications)
// - ~/Applications (user-specific applications)
//
// System tools and utilities are those in:
// - /System/Applications
// - /System/Applications/Utilities
// - /System/Library/CoreServices
func isUserApplication(path string) bool {
	// System directories (not user applications)
	systemPrefixes := []string{
		"/System/Applications",
		"/System/Library/CoreServices",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}

	// User application directories
	if strings.HasPrefix(path, "/Applications") {
		return true
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		userAppPath := filepath.Join(homeDir, "Applications")
		if strings.HasPrefix(path, userAppPath) {
			return true
		}
	}

	// Default to false for unknown paths
	return false
}
