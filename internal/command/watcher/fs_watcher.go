//go:build darwin || windows

package watcher

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"sync"
	"time"
	"watools/pkg/logger"
)

// The event loop owns its debounce state and output channel. Stop waits for it;
// no delayed callback can send to a channel after shutdown.
type FSWatcher struct {
	watcher   *fsnotify.Watcher
	ctx       context.Context
	cancel    context.CancelFunc
	eventCh   chan AppChangeEvent
	debounce  time.Duration
	watchDirs []string
	mu        sync.RWMutex
	running   bool
	closed    bool
	done      chan struct{}
}

func NewFSWatcher() (*FSWatcher, error) {
	return NewFSWatcherWithConfig(DefaultWatcherConfig())
}

func NewFSWatcherWithConfig(config *WatcherConfig) (*FSWatcher, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid fs watcher config: %w", err)
	}
	native, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &FSWatcher{watcher: native, ctx: ctx, cancel: cancel, eventCh: make(chan AppChangeEvent, config.EventBufferSize), debounce: config.GetDebounceInterval(), watchDirs: getDefaultWatchDirs(), done: make(chan struct{})}, nil
}
func (fw *FSWatcher) Start() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.closed || fw.running {
		return fmt.Errorf("watcher is closed or already running")
	}
	for _, dir := range fw.watchDirs {
		if err := fw.watchTree(dir); err != nil {
			logger.Error(err, "Failed to watch directory", "path", dir)
		}
	}
	fw.running = true
	go fw.handleEvents(fw.watcher.Events, fw.watcher.Errors)
	return nil
}
func (fw *FSWatcher) Stop() error {
	fw.mu.Lock()
	if fw.closed {
		done := fw.done
		fw.mu.Unlock()
		<-done
		return nil
	}
	fw.closed = true
	fw.cancel()
	started := fw.running
	fw.running = false
	err := fw.watcher.Close()
	if !started {
		close(fw.eventCh)
		close(fw.done)
	}
	fw.mu.Unlock()
	<-fw.done
	return err
}
func (fw *FSWatcher) EventChannel() <-chan AppChangeEvent { return fw.eventCh }
func (fw *FSWatcher) handleEvents(events <-chan fsnotify.Event, watcherErrors <-chan error) {
	defer close(fw.done)
	defer close(fw.eventCh)
	type pendingEvent struct {
		event fsnotify.Event
		due   time.Time
	}
	pending := make(map[string]pendingEvent)
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var due <-chan time.Time
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		due = nil
		var next time.Time
		for _, item := range pending {
			if next.IsZero() || item.due.Before(next) {
				next = item.due
			}
		}
		if !next.IsZero() {
			timer.Reset(time.Until(next))
			due = timer.C
		}
	}
	for {
		select {
		case <-fw.ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if fw.ctx.Err() != nil {
				return
			}
			// Newly created Start Menu folders must acquire their own recursive watch.
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := fw.watchTree(event.Name); err != nil {
						logger.Error(err, "Failed to watch new directory")
					}
				}
			}
			if fw.isAppEvent(event.Name) {
				path := normalizeAppPath(event.Name)
				event.Name = path
				// Preserve all observed operations while the path is settling.
				event.Op |= pending[path].event.Op
				pending[path] = pendingEvent{event, time.Now().Add(fw.debounce)}
				resetTimer()
			}
		case <-due:
			for path, item := range pending {
				if !time.Now().Before(item.due) {
					if !fw.processEvent(item.event) {
						return
					}
					delete(pending, path)
				}
			}
			resetTimer()
		case err, ok := <-watcherErrors:
			if !ok {
				return
			}
			logger.Error(err, "FSWatcher error; periodic scan will reconcile applications")
		}
	}
}
func (fw *FSWatcher) processEvent(event fsnotify.Event) bool {
	if fw.ctx.Err() != nil {
		return false
	}
	path := normalizeAppPath(event.Name)
	change := AppModified
	// Reconcile the final disk state instead of trusting a stale Create/Remove flag.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		change = AppRemoved
	} else if err != nil {
		logger.Error(err, "Failed to inspect changed application")
		return true
	} else if event.Has(fsnotify.Create) {
		change = AppAdded
	}
	if fw.ctx.Err() != nil {
		return false
	}
	select {
	case fw.eventCh <- AppChangeEvent{Type: change, Path: path}:
		return true
	case <-fw.ctx.Done():
		return false
	}
}
func (fw *FSWatcher) IsRunning() bool { fw.mu.RLock(); defer fw.mu.RUnlock(); return fw.running }
func (fw *FSWatcher) AddWatchDir(dir string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.closed {
		return fmt.Errorf("watcher is closed")
	}
	if fw.running {
		if err := fw.watchTree(dir); err != nil {
			return err
		}
	}
	fw.watchDirs = append(fw.watchDirs, dir)
	return nil
}
func (fw *FSWatcher) GetWatchDirs() []string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return append([]string(nil), fw.watchDirs...)
}
