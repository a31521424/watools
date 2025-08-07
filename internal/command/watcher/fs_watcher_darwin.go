package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"watools/pkg/logger"

	"github.com/fsnotify/fsnotify"
)

// FSWatcher file system event watcher
type FSWatcher struct {
	watcher   *fsnotify.Watcher
	ctx       context.Context
	cancel    context.CancelFunc
	eventCh   chan AppChangeEvent
	watchDirs []string
	mu        sync.RWMutex
	running   bool
}

// NewFSWatcher create new file system watcher
func NewFSWatcher() (*FSWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &FSWatcher{
		watcher:   watcher,
		ctx:       ctx,
		cancel:    cancel,
		eventCh:   make(chan AppChangeEvent, 100), // buffer events
		watchDirs: getDefaultWatchDirs(),
		running:   false,
	}, nil
}

// getDefaultWatchDirs get default watch directories
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

// Start start watcher
func (fw *FSWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.running {
		return fmt.Errorf("watcher is already running")
	}

	// add watch directories
	for _, dir := range fw.watchDirs {
		if err := fw.addWatchDir(dir); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to watch directory: %s", dir))
			// continue adding other directories
		}
	}

	fw.running = true

	// start event handling goroutine
	go fw.handleEvents()

	logger.Info(fmt.Sprintf("FSWatcher started, watching %d directories", len(fw.watchDirs)))
	return nil
}

// Stop stop watcher
func (fw *FSWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		return nil
	}

	fw.cancel()
	fw.running = false

	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close fsnotify watcher: %w", err)
	}

	close(fw.eventCh)
	logger.Info("FSWatcher stopped")
	return nil
}

// EventChannel get event channel
func (fw *FSWatcher) EventChannel() <-chan AppChangeEvent {
	return fw.eventCh
}

// addWatchDir add watch directory
func (fw *FSWatcher) addWatchDir(dir string) error {
	// check directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	if err := fw.watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to add watch for %s: %w", dir, err)
	}

	logger.Info(fmt.Sprintf("Added watch for directory: %s", dir))
	return nil
}

// handleEvents handle file system events
func (fw *FSWatcher) handleEvents() {
	debounceMap := make(map[string]*time.Timer)
	debounceMu := sync.Mutex{}

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			if fw.isAppEvent(event.Name) {
				// debounce to avoid frequent triggers
				debounceMu.Lock()
				if timer, exists := debounceMap[event.Name]; exists {
					timer.Stop()
				}

				debounceMap[event.Name] = time.AfterFunc(500*time.Millisecond, func() {
					fw.processEvent(event)
					debounceMu.Lock()
					delete(debounceMap, event.Name)
					debounceMu.Unlock()
				})
				debounceMu.Unlock()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			logger.Error(err, "FSWatcher error")
		}
	}
}

// isAppEvent check if event is app related
func (fw *FSWatcher) isAppEvent(path string) bool {
	// check if .app file/directory
	return strings.HasSuffix(path, ".app") ||
		(filepath.Ext(path) == ".app") ||
		strings.Contains(path, ".app/")
}

// processEvent process specific event
func (fw *FSWatcher) processEvent(event fsnotify.Event) {
	path := event.Name

	// if .app internal file change, get .app root directory
	if strings.Contains(path, ".app/") && !strings.HasSuffix(path, ".app") {
		parts := strings.Split(path, ".app/")
		if len(parts) > 0 {
			path = parts[0] + ".app"
		}
	}

	var changeType AppChangeType

	switch {
	case event.Has(fsnotify.Create):
		changeType = AppAdded

	case event.Has(fsnotify.Remove):
		changeType = AppRemoved

	case event.Has(fsnotify.Write) || event.Has(fsnotify.Chmod):
		// check if file still exists, if not then it's removed
		if _, err := os.Stat(path); os.IsNotExist(err) {
			changeType = AppRemoved
		} else {
			changeType = AppModified
		}

	case event.Has(fsnotify.Rename):
		// rename might be remove or move, check if target exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			changeType = AppRemoved
		} else {
			changeType = AppModified
		}

	default:
		return // ignore other events
	}

	// send event
	select {
	case fw.eventCh <- AppChangeEvent{Type: changeType, Path: path}:
		logger.Info(fmt.Sprintf("App %s detected: %s",
			map[AppChangeType]string{
				AppAdded:    "added",
				AppRemoved:  "removed",
				AppModified: "modified",
			}[changeType], path))

	case <-fw.ctx.Done():
		return

	default:
		// if channel is full, log warning but don't block
		logger.Error(nil, "FSWatcher event channel is full, dropping event")
	}
}

// IsRunning check if running
func (fw *FSWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.running
}

// AddWatchDir dynamically add watch directory
func (fw *FSWatcher) AddWatchDir(dir string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.running {
		fw.watchDirs = append(fw.watchDirs, dir)
		return nil
	}

	return fw.addWatchDir(dir)
}

// GetWatchDirs get watch directories list
func (fw *FSWatcher) GetWatchDirs() []string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	dirs := make([]string, len(fw.watchDirs))
	copy(dirs, fw.watchDirs)
	return dirs
}
