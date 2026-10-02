//go:build darwin || windows

package watcher

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
	"watools/internal/command/application"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// appWatchManager app watch manager darwin implementation
type appWatchManager struct {
	fsWatcher    *FSWatcher
	eventHandler AppEventHandler
	errorHandler ErrorHandler
	config       *WatcherConfig
	metrics      *WatcherMetrics
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	running      bool
	workers      sync.WaitGroup
}

// NewAppWatchManager create app watch manager
func NewAppWatchManager(handler AppEventHandler, ctx context.Context) (AppWatchManager, error) {
	return NewAppWatchManagerWithConfig(handler, ctx, DefaultWatcherConfig())
}

// NewAppWatchManagerWithConfig create app watch manager with config
func NewAppWatchManagerWithConfig(handler AppEventHandler, ctx context.Context, config *WatcherConfig) (AppWatchManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if !config.Enabled {
		logger.Info("App watcher is disabled by configuration")
		return nil, nil
	}

	fsWatcher, err := NewFSWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fs watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	errorHandler := NewDefaultErrorHandler(&config.RetryConfig)
	metrics := NewWatcherMetrics()

	awm := &appWatchManager{
		fsWatcher:    fsWatcher,
		eventHandler: handler,
		errorHandler: errorHandler,
		config:       config,
		metrics:      metrics,
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
	}

	// add custom watch directories
	for _, dir := range config.CustomWatchDirs {
		if err := fsWatcher.AddWatchDir(dir); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to add custom watch dir: %s", dir))
		}
	}

	return awm, nil
}

// Start start watch manager
func (awm *appWatchManager) Start() error {
	awm.mu.Lock()
	defer awm.mu.Unlock()

	if awm.running {
		return fmt.Errorf("app watch manager is already running")
	}

	// start file system watcher
	if err := awm.fsWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start fs watcher: %w", err)
	}

	awm.running = true

	// start event handling goroutine
	awm.workers.Add(1)
	go func() { defer awm.workers.Done(); awm.handleAppEvents() }()

	logger.Info("AppWatchManager started")
	return nil
}

// Stop stop watch manager
func (awm *appWatchManager) Stop() error {
	awm.mu.Lock()
	awm.running = false
	awm.cancel()
	awm.mu.Unlock()
	err := awm.fsWatcher.Stop()
	awm.workers.Wait()
	return err
}

// handleAppEvents handle app events
func (awm *appWatchManager) handleAppEvents() {
	eventCh := awm.fsWatcher.EventChannel()

	for {
		select {
		case <-awm.ctx.Done():
			return

		case event, ok := <-eventCh:
			if !ok {
				return
			}

			awm.processAppEvent(event)
		}
	}
}

// processAppEvent process single app event
func (awm *appWatchManager) processAppEvent(event AppChangeEvent) {
	startTime := time.Now()
	defer func() {
		awm.mu.Lock()
		defer awm.mu.Unlock()
		awm.metrics.AddProcessingTime(time.Since(startTime))
	}()

	awm.mu.Lock()
	awm.metrics.IncrementEventByType(event.Type)
	awm.mu.Unlock()

	// use retryable operation to process event
	operation := func() error {
		switch event.Type {
		case AppAdded:
			return awm.handleAppAddedWithRetry(event.Path)
		case AppRemoved:
			return awm.handleAppRemovedWithRetry(event.Path)
		case AppModified:
			return awm.handleAppModifiedWithRetry(event.Path)
		default:
			return fmt.Errorf("unknown event type: %v", event.Type)
		}
	}

	retryableOp := NewRetryableOperation(operation, awm.errorHandler, &awm.config.RetryConfig)

	if err := retryableOp.ExecuteContext(awm.ctx); err != nil {
		awm.errorHandler.HandleEventError(err, event)
		awm.mu.Lock()
		awm.metrics.IncrementErrorsCount()
		awm.mu.Unlock()
	} else {
		awm.mu.Lock()
		awm.metrics.IncrementEventsProcessed()
		awm.mu.Unlock()
	}
}

// handleAppAddedWithRetry handle app added event with retry
func (awm *appWatchManager) handleAppAddedWithRetry(path string) error {
	// delay to ensure app is fully installed
	if err := awm.wait(awm.config.GetAppAddedDelay()); err != nil {
		return err
	}

	command, err := application.ParseApplication(path)
	if err != nil {
		return fmt.Errorf("failed to parse added application: %w", err)
	}

	if err := awm.eventHandler.OnAppAdded(command); err != nil {
		return fmt.Errorf("failed to handle added application: %w", err)
	}

	logger.Info(fmt.Sprintf("Successfully handled added application: %s", command.Name))
	return nil
}

// handleAppRemovedWithRetry handle app removed event with retry
func (awm *appWatchManager) handleAppRemovedWithRetry(path string) error {
	if err := awm.eventHandler.OnAppRemoved(path); err != nil {
		return fmt.Errorf("failed to handle removed application: %w", err)
	}

	logger.Info(fmt.Sprintf("Successfully handled removed application: %s", path))
	return nil
}

// handleAppModifiedWithRetry handle app modified event with retry
func (awm *appWatchManager) handleAppModifiedWithRetry(path string) error {
	// delay to ensure modification is complete
	if err := awm.wait(awm.config.GetAppModifiedDelay()); err != nil {
		return err
	}

	command, err := application.ParseApplication(path)
	if err != nil {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return awm.handleAppRemovedWithRetry(path)
		}
		return err
	}

	if err := awm.eventHandler.OnAppModified(command); err != nil {
		return fmt.Errorf("failed to handle modified application: %w", err)
	}

	logger.Info(fmt.Sprintf("Successfully handled modified application: %s", command.Name))
	return nil
}

// IsRunning check if running
func (awm *appWatchManager) IsRunning() bool {
	awm.mu.RLock()
	defer awm.mu.RUnlock()
	return awm.running
}

// GetWatchDirs get watch directories
func (awm *appWatchManager) GetWatchDirs() []string {
	return awm.fsWatcher.GetWatchDirs()
}

// AddWatchDir add watch directory
func (awm *appWatchManager) AddWatchDir(dir string) error {
	return awm.fsWatcher.AddWatchDir(dir)
}

// GetMetrics get watcher metrics
func (awm *appWatchManager) GetMetrics() *WatcherMetrics {
	awm.mu.RLock()
	defer awm.mu.RUnlock()
	if awm.metrics == nil {
		return NewWatcherMetrics()
	}
	snapshot := *awm.metrics
	snapshot.EventsByType = make(map[string]int64, len(awm.metrics.EventsByType))
	for key, value := range awm.metrics.EventsByType {
		snapshot.EventsByType[key] = value
	}
	return &snapshot
}

// GetConfig get config
func (awm *appWatchManager) GetConfig() *WatcherConfig {
	return awm.config
}

// defaultAppEventHandler default app event handler darwin implementation
type defaultAppEventHandler struct {
	db  *db.WaDB
	ctx context.Context
}

// NewDefaultAppEventHandler create default event handler
func NewDefaultAppEventHandler(ctx context.Context) DefaultAppEventHandler {
	return &defaultAppEventHandler{
		db:  db.GetWaDB(),
		ctx: ctx,
	}
}

// OnAppAdded handle app added
func (h *defaultAppEventHandler) OnAppAdded(command *models.ApplicationCommand) error {
	// check if already exists
	existingCommands := h.db.GetCommands(h.ctx)
	for _, existing := range existingCommands {
		if existing.Path == command.Path {
			// if exists, update instead of insert (no event emission here to avoid duplicate)
			return h.onAppModifiedInternal(command)
		}
	}

	// insert new app
	err := h.db.BatchInsertCommands(h.ctx, []*models.ApplicationCommand{command})
	if err == nil {
		h.emitApplicationChanged()
	}
	return err
}

// OnAppRemoved handle app removed
func (h *defaultAppEventHandler) OnAppRemoved(path string) error {
	// find and delete command by path
	commands := h.db.GetCommands(h.ctx)
	for _, command := range commands {
		if command.Path == path {
			err := h.db.DeleteCommands(h.ctx, []string{command.ID})
			if err == nil {
				h.emitApplicationChanged()
			}
			return err
		}
	}
	return nil
}

// OnAppModified handle app modified
func (h *defaultAppEventHandler) OnAppModified(command *models.ApplicationCommand) error {
	err := h.onAppModifiedInternal(command)
	if err == nil {
		h.emitApplicationChanged()
	}
	return err
}

// onAppModifiedInternal internal method for handling app modified without emitting events
func (h *defaultAppEventHandler) onAppModifiedInternal(command *models.ApplicationCommand) error {
	// find existing command and update
	commands := h.db.GetCommands(h.ctx)
	for _, existing := range commands {
		if existing.Path == command.Path {
			command.ID = existing.ID
			command.UsedCount = existing.UsedCount
			command.LastUsedAt = existing.LastUsedAt
			return h.db.BatchUpdateCommands(h.ctx, []*models.ApplicationCommand{command})
		}
	}

	// if not found, treat as new app (no event emission here to avoid duplicate)
	return h.db.BatchInsertCommands(h.ctx, []*models.ApplicationCommand{command})
}

// emitApplicationChanged emit application changed event to frontend
func (h *defaultAppEventHandler) emitApplicationChanged() {
	runtime.EventsEmit(h.ctx, "watools.applicationChanged")
}

func (awm *appWatchManager) wait(delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-awm.ctx.Done():
		return awm.ctx.Err()
	case <-timer.C:
		return nil
	}
}
