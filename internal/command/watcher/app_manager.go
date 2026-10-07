//go:build darwin || windows

package watcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
	"watools/internal/command/application"
	"watools/pkg/logger"
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
	config = config.clone()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if !config.Enabled {
		logger.Info("App watcher is disabled by configuration")
		return nil, nil
	}

	fsWatcher, err := NewFSWatcherWithConfig(config)
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
	if err := awm.ctx.Err(); err != nil {
		return fmt.Errorf("start app watch manager: %w", err)
	}

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
	if awm.ctx.Err() != nil {
		return
	}
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
		return awm.handlePathChange(event)
	}

	retryableOp := NewRetryableOperation(operation, awm.errorHandler, &awm.config.RetryConfig)

	if err := retryableOp.ExecuteContext(awm.ctx); err != nil {
		if awm.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}
		awm.errorHandler.HandleEventError(err, event)
		awm.mu.Lock()
		awm.metrics.IncrementErrorsCount()
		awm.mu.Unlock()
	} else {
		awm.mu.Lock()
		if awm.ctx.Err() == nil {
			awm.metrics.IncrementEventsProcessed()
		}
		awm.mu.Unlock()
	}
}

// handlePathChange reads the final state after the configured install delay.
func (awm *appWatchManager) handlePathChange(event AppChangeEvent) error {
	var delay time.Duration
	switch event.Type {
	case AppAdded:
		delay = awm.config.GetAppAddedDelay()
	case AppModified:
		delay = awm.config.GetAppModifiedDelay()
	case AppRemoved:
	default:
		return fmt.Errorf("unknown event type: %v", event.Type)
	}
	if err := awm.wait(delay); err != nil {
		return err
	}
	if err := awm.ctx.Err(); err != nil {
		return err
	}
	var err error
	if handler, ok := awm.eventHandler.(PathEventHandler); ok {
		err = handler.ReconcileApplicationPath(awm.ctx, event.Path)
	} else {
		err = awm.handleLegacyPathChange(event)
	}
	if cancelErr := awm.ctx.Err(); cancelErr != nil {
		return cancelErr
	}
	if err != nil {
		return fmt.Errorf("reconcile changed application %q: %w", event.Path, err)
	}
	return nil
}

// Legacy handlers keep their callbacks, but only a missing root authorizes removal.
func (awm *appWatchManager) handleLegacyPathChange(event AppChangeEvent) error {
	if _, err := os.Stat(event.Path); err != nil {
		if os.IsNotExist(err) {
			return awm.callRemoved(event.Path)
		}
		return fmt.Errorf("inspect application root: %w", err)
	}
	if err := awm.ctx.Err(); err != nil {
		return err
	}
	command, err := application.ParseApplication(event.Path)
	if cancelErr := awm.ctx.Err(); cancelErr != nil {
		return cancelErr
	}
	if err != nil {
		// The root may disappear while metadata is being read. Metadata failures
		// on an existing root preserve the previous valid command.
		if _, statErr := os.Stat(event.Path); os.IsNotExist(statErr) {
			return awm.callRemoved(event.Path)
		}
		return fmt.Errorf("parse changed application: %w", err)
	}
	if event.Type == AppAdded {
		return awm.eventHandler.OnAppAdded(command)
	}
	return awm.eventHandler.OnAppModified(command)
}

func (awm *appWatchManager) callRemoved(path string) error {
	if err := awm.ctx.Err(); err != nil {
		return err
	}
	return awm.eventHandler.OnAppRemoved(path)
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
	return awm.config.clone()
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
