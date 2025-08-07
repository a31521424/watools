//go:build !darwin

package watcher

import (
	"context"
	"fmt"
	"watools/pkg/models"
)

// appWatchManagerStub stub implementation for non-darwin platforms
type appWatchManagerStub struct{}

// NewAppWatchManager create app watch manager (stub for non-darwin)
func NewAppWatchManager(handler AppEventHandler) (AppWatchManager, error) {
	return &appWatchManagerStub{}, nil
}

// Start start watch manager (stub for non-darwin)
func (awm *appWatchManagerStub) Start() error {
	return fmt.Errorf("app watcher is only supported on macOS")
}

// Stop stop watch manager (stub for non-darwin)
func (awm *appWatchManagerStub) Stop() error {
	return nil
}

// IsRunning check if running (stub for non-darwin)
func (awm *appWatchManagerStub) IsRunning() bool {
	return false
}

// GetWatchDirs get watch directories (stub for non-darwin)
func (awm *appWatchManagerStub) GetWatchDirs() []string {
	return []string{}
}

// AddWatchDir add watch directory (stub for non-darwin)
func (awm *appWatchManagerStub) AddWatchDir(dir string) error {
	return fmt.Errorf("app watcher is only supported on macOS")
}

// GetMetrics get watcher metrics (stub for non-darwin)
func (awm *appWatchManagerStub) GetMetrics() *WatcherMetrics {
	return NewWatcherMetrics()
}

// GetConfig get config (stub for non-darwin)
func (awm *appWatchManagerStub) GetConfig() *WatcherConfig {
	return DefaultWatcherConfig()
}

// DefaultWatcherConfig default config (stub for non-darwin)
func DefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		Enabled: false,
	}
}

// defaultAppEventHandlerStub default app event handler (stub for non-darwin)
type defaultAppEventHandlerStub struct{}

// NewDefaultAppEventHandler create default event handler (stub for non-darwin)
func NewDefaultAppEventHandler(ctx context.Context) DefaultAppEventHandler {
	return &defaultAppEventHandlerStub{}
}

// OnAppAdded handle app added (stub for non-darwin)
func (h *defaultAppEventHandlerStub) OnAppAdded(command *models.ApplicationCommand) error {
	return fmt.Errorf("app watcher is only supported on macOS")
}

// OnAppRemoved handle app removed (stub for non-darwin)
func (h *defaultAppEventHandlerStub) OnAppRemoved(path string) error {
	return fmt.Errorf("app watcher is only supported on macOS")
}

// OnAppModified handle app modified (stub for non-darwin)
func (h *defaultAppEventHandlerStub) OnAppModified(command *models.ApplicationCommand) error {
	return fmt.Errorf("app watcher is only supported on macOS")
}
