package watcher

import (
	"os"
	"path/filepath"
	"strings"
)

func getDefaultWatchDirs() []string {
	dirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		"/System/Library/CoreServices",
	}

	// add user applications directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		userAppsDir := filepath.Join(homeDir, "Applications")
		if _, err := os.Stat(userAppsDir); err == nil {
			dirs = append(dirs, userAppsDir)
		}
	}

	return dirs
}

func (fw *FSWatcher) isAppEvent(path string) bool {
	// check if .app file/directory
	return strings.HasSuffix(path, ".app") ||
		(filepath.Ext(path) == ".app") ||
		strings.Contains(path, ".app/")
}

func (fw *FSWatcher) watchTree(dir string) error { return fw.watcher.Add(dir) }
func normalizeAppPath(path string) string {
	if i := strings.Index(path, ".app/"); i >= 0 {
		return path[:i] + ".app"
	}
	return path
}
