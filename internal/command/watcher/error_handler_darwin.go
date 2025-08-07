package watcher

import (
	"fmt"
	"time"
	"watools/pkg/logger"
)

// ErrorHandler error handler interface
type ErrorHandler interface {
	HandleWatcherError(err error, context string)
	HandleEventError(err error, event AppChangeEvent)
	ShouldRetry(err error, attempt int) bool
}

// DefaultErrorHandler default error handler
type DefaultErrorHandler struct {
	config *RetryConfig
}

// NewDefaultErrorHandler create default error handler
func NewDefaultErrorHandler(config *RetryConfig) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		config: config,
	}
}

// HandleWatcherError handle watcher error
func (h *DefaultErrorHandler) HandleWatcherError(err error, context string) {
	logger.Error(err, fmt.Sprintf("Watcher error in %s", context))
}

// HandleEventError handle event error
func (h *DefaultErrorHandler) HandleEventError(err error, event AppChangeEvent) {
	logger.Error(err, fmt.Sprintf("Failed to process %s event for %s",
		h.getEventTypeName(event.Type), event.Path))
}

// ShouldRetry check if should retry
func (h *DefaultErrorHandler) ShouldRetry(err error, attempt int) bool {
	if h.config == nil {
		return false
	}

	return attempt < h.config.MaxRetries
}

// getEventTypeName get event type name
func (h *DefaultErrorHandler) getEventTypeName(eventType AppChangeType) string {
	switch eventType {
	case AppAdded:
		return "add"
	case AppRemoved:
		return "remove"
	case AppModified:
		return "modify"
	default:
		return "unknown"
	}
}

// RetryableOperation retryable operation
type RetryableOperation struct {
	operation func() error
	handler   ErrorHandler
	config    *RetryConfig
}

// NewRetryableOperation create retryable operation
func NewRetryableOperation(operation func() error, handler ErrorHandler, config *RetryConfig) *RetryableOperation {
	return &RetryableOperation{
		operation: operation,
		handler:   handler,
		config:    config,
	}
}

// Execute execute retryable operation
func (ro *RetryableOperation) Execute() error {
	var lastErr error

	for attempt := 0; attempt <= ro.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// wait retry interval
			delay := ro.calculateRetryDelay(attempt)
			time.Sleep(delay)
		}

		if err := ro.operation(); err != nil {
			lastErr = err

			if !ro.handler.ShouldRetry(err, attempt) {
				break
			}

			logger.Error(err, fmt.Sprintf("Operation failed, attempt %d/%d",
				attempt+1, ro.config.MaxRetries+1))
			continue
		}

		// operation succeeded
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts: %w",
		ro.config.MaxRetries+1, lastErr)
}

// calculateRetryDelay calculate retry delay
func (ro *RetryableOperation) calculateRetryDelay(attempt int) time.Duration {
	baseDelay := time.Duration(ro.config.RetryInterval) * time.Millisecond

	if !ro.config.ExponentialBackoff {
		return baseDelay
	}

	// exponential backoff strategy
	multiplier := 1
	for i := 1; i < attempt; i++ {
		multiplier *= 2
	}

	return baseDelay * time.Duration(multiplier)
}

// Metrics methods for WatcherMetrics

// IncrementEventsProcessed increment processed events count
func (m *WatcherMetrics) IncrementEventsProcessed() {
	m.EventsProcessed++
	m.LastEventTime = time.Now()
}

// IncrementEventsDropped increment dropped events count
func (m *WatcherMetrics) IncrementEventsDropped() {
	m.EventsDropped++
}

// IncrementErrorsCount increment errors count
func (m *WatcherMetrics) IncrementErrorsCount() {
	m.ErrorsCount++
}

// IncrementEventByType increment specific event type count
func (m *WatcherMetrics) IncrementEventByType(eventType AppChangeType) {
	typeName := m.getEventTypeName(eventType)
	m.EventsByType[typeName]++
}

// AddProcessingTime add processing time
func (m *WatcherMetrics) AddProcessingTime(duration time.Duration) {
	m.ProcessingDuration += duration
}

// GetUptime get uptime
func (m *WatcherMetrics) GetUptime() time.Duration {
	return time.Since(m.WatcherStartTime)
}

// getEventTypeName get event type name
func (m *WatcherMetrics) getEventTypeName(eventType AppChangeType) string {
	switch eventType {
	case AppAdded:
		return "added"
	case AppRemoved:
		return "removed"
	case AppModified:
		return "modified"
	default:
		return "unknown"
	}
}

// Reset reset metrics
func (m *WatcherMetrics) Reset() {
	m.EventsProcessed = 0
	m.EventsDropped = 0
	m.ErrorsCount = 0
	m.EventsByType = make(map[string]int64)
	m.LastEventTime = time.Time{}
	m.WatcherStartTime = time.Now()
	m.ProcessingDuration = 0
}
