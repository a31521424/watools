package models

import (
	"os"
	"path/filepath"
	"strings"
)

// isUserApplication returns true for user-installed applications on Windows.
// On Windows, user applications are those installed in:
// - C:\Program Files
// - C:\Program Files (x86)
// - %LOCALAPPDATA%\Programs
//
// System tools and utilities are those in:
// - C:\Windows
// - C:\Windows\System32
func isUserApplication(path string) bool {
	lowerPath := strings.ToLower(path)

	// System directories (not user applications)
	systemPrefixes := []string{
		`c:\windows\`,
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(lowerPath, prefix) {
			return false
		}
	}

	// User application directories
	userPrefixes := []string{
		`c:\program files\`,
		`c:\program files (x86)\`,
	}

	for _, prefix := range userPrefixes {
		if strings.HasPrefix(lowerPath, prefix) {
			return true
		}
	}

	// Check user's local programs directory
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		userProgramsPath := strings.ToLower(filepath.Join(localAppData, "Programs"))
		if strings.HasPrefix(lowerPath, userProgramsPath) {
			return true
		}
	}

	// Default to false for unknown paths
	return false
}
