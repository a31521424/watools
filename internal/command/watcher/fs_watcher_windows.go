package watcher

import (
	"os"
	"path/filepath"
	"strings"
)

func getDefaultWatchDirs() []string {
	var dirs []string

	if appData := os.Getenv("APPDATA"); appData != "" {
		dirs = append(dirs, filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		dirs = append(dirs, filepath.Join(programData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}

	return dirs
}

func (fw *FSWatcher) isAppEvent(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".lnk" || ext == ".exe"
}

func (fw *FSWatcher) watchTree(dir string) error {
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return fw.watcher.Add(path)
		}
		return nil
	})
}
func normalizeAppPath(path string) string { return path }
