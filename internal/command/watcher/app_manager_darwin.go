package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"
	"watools/internal/command/application"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"
)

// appWatchManager app watch manager darwin implementation
type appWatchManager struct {
	fsWatcher     *FSWatcher
	eventHandler  AppEventHandler
	errorHandler  ErrorHandler
	config        *WatcherConfig
	metrics       *WatcherMetrics
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	running       bool
	processedApps map[string]time.Time // prevent duplicate processing
}

// NewAppWatchManager create app watch manager
func NewAppWatchManager(handler AppEventHandler) (AppWatchManager, error) {
	return NewAppWatchManagerWithConfig(handler, DefaultWatcherConfig())
}

// NewAppWatchManagerWithConfig create app watch manager with config
func NewAppWatchManagerWithConfig(handler AppEventHandler, config *WatcherConfig) (AppWatchManager, error) {
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

	ctx, cancel := context.WithCancel(context.Background())

	errorHandler := NewDefaultErrorHandler(&config.RetryConfig)
	metrics := NewWatcherMetrics()

	awm := &appWatchManager{
		fsWatcher:     fsWatcher,
		eventHandler:  handler,
		errorHandler:  errorHandler,
		config:        config,
		metrics:       metrics,
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		processedApps: make(map[string]time.Time),
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
	go awm.handleAppEvents()

	// start cleanup goroutine
	go awm.cleanupProcessedApps()

	logger.Info("AppWatchManager started")
	return nil
}

// Stop stop watch manager
func (awm *appWatchManager) Stop() error {
	awm.mu.Lock()
	defer awm.mu.Unlock()

	if !awm.running {
		return nil
	}

	awm.cancel()
	awm.running = false

	if err := awm.fsWatcher.Stop(); err != nil {
		logger.Error(err, "Failed to stop fs watcher")
	}

	logger.Info("AppWatchManager stopped")
	return nil
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

			// check if recently processed
			if awm.isRecentlyProcessed(event.Path) {
				continue
			}

			awm.processAppEvent(event)
		}
	}
}

// isRecentlyProcessed check if recently processed
func (awm *appWatchManager) isRecentlyProcessed(path string) bool {
	awm.mu.RLock()
	lastProcessed, exists := awm.processedApps[path]
	awm.mu.RUnlock()

	if !exists {
		return false
	}

	// skip if processed within 5 seconds
	return time.Since(lastProcessed) < 5*time.Second
}

// markAsProcessed mark as processed
func (awm *appWatchManager) markAsProcessed(path string) {
	awm.mu.Lock()
	awm.processedApps[path] = time.Now()
	awm.mu.Unlock()
}

// processAppEvent process single app event
func (awm *appWatchManager) processAppEvent(event AppChangeEvent) {
	startTime := time.Now()
	defer func() {
		awm.metrics.AddProcessingTime(time.Since(startTime))
	}()

	awm.markAsProcessed(event.Path)
	awm.metrics.IncrementEventByType(event.Type)

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

	if err := retryableOp.Execute(); err != nil {
		awm.errorHandler.HandleEventError(err, event)
		awm.metrics.IncrementErrorsCount()
	} else {
		awm.metrics.IncrementEventsProcessed()
	}
}

// handleAppAddedWithRetry handle app added event with retry
func (awm *appWatchManager) handleAppAddedWithRetry(path string) error {
	// delay to ensure app is fully installed
	time.Sleep(awm.config.GetAppAddedDelay())

	command, err := application.Scanner.ParseApplication(path)
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
	time.Sleep(awm.config.GetAppModifiedDelay())

	command, err := application.Scanner.ParseApplication(path)
	if err != nil {
		// if parsing fails, app might be removed
		return awm.handleAppRemovedWithRetry(path)
	}

	if err := awm.eventHandler.OnAppModified(command); err != nil {
		return fmt.Errorf("failed to handle modified application: %w", err)
	}

	logger.Info(fmt.Sprintf("Successfully handled modified application: %s", command.Name))
	return nil
}

// cleanupProcessedApps cleanup processed apps records
func (awm *appWatchManager) cleanupProcessedApps() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-awm.ctx.Done():
			return

		case <-ticker.C:
			awm.mu.Lock()
			now := time.Now()
			for path, processedTime := range awm.processedApps {
				// cleanup records older than 5 minutes
				if now.Sub(processedTime) > 5*time.Minute {
					delete(awm.processedApps, path)
				}
			}
			awm.mu.Unlock()
		}
	}
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
	if awm.metrics == nil {
		return NewWatcherMetrics()
	}
	return awm.metrics
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
			// if exists, update instead of insert
			return h.OnAppModified(command)
		}
	}

	// insert new app
	return h.db.BatchInsertCommands(h.ctx, []*models.ApplicationCommand{command})
}

// OnAppRemoved handle app removed
func (h *defaultAppEventHandler) OnAppRemoved(path string) error {
	// find and delete command by path
	commands := h.db.GetCommands(h.ctx)
	for _, command := range commands {
		if command.Path == path {
			return h.db.DeleteCommand(h.ctx, command.ID)
		}
	}
	return nil
}

// OnAppModified handle app modified
func (h *defaultAppEventHandler) OnAppModified(command *models.ApplicationCommand) error {
	// find existing command and update
	commands := h.db.GetCommands(h.ctx)
	for _, existing := range commands {
		if existing.Path == command.Path {
			command.ID = existing.ID
			return h.db.BatchUpdateCommands(h.ctx, []*models.ApplicationCommand{command})
		}
	}

	// if not found, treat as new app
	return h.OnAppAdded(command)
}
