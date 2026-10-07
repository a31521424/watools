//go:build darwin || windows

package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"
	"watools/pkg/logger"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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
	if config == nil {
		return &DefaultErrorHandler{}
	}
	copy := *config
	return &DefaultErrorHandler{config: &copy}
}

// HandleWatcherError handle watcher error
func (h *DefaultErrorHandler) HandleWatcherError(err error, context string) {
	logger.Error(err, fmt.Sprintf("Watcher error in %s", context))
}

// HandleEventError handle event error
func (h *DefaultErrorHandler) HandleEventError(err error, event AppChangeEvent) {
	if errors.Is(err, context.Canceled) {
		return
	}
	logger.Error(err, fmt.Sprintf("Failed to process %s event for %s",
		h.getEventTypeName(event.Type), event.Path))
}

// ShouldRetry check if should retry
func (h *DefaultErrorHandler) ShouldRetry(err error, attempt int) bool {
	if h.config == nil || err == nil || attempt >= h.config.MaxRetries ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// The watcher retries the entire DB operation after its transaction has ended.
	// Unknown parse/I/O errors are not evidence of a transient failure. SQLITE_LOCKED
	// may require fixing a cursor on the same connection rather than waiting.
	return retryableDatabaseError(err)
}

// A busy primary error does not make a failed rollback safe to retry. Every
// leaf must be a known contention error, including errors joined during cleanup.
func retryableDatabaseError(err error) bool {
	switch cause := err.(type) {
	case interface{ Unwrap() []error }:
		children := cause.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !retryableDatabaseError(child) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return retryableDatabaseError(cause.Unwrap())
	case *sqlite.Error:
		return cause.Code()&0xff == sqlite3.SQLITE_BUSY
	default:
		return false
	}
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
	config    RetryConfig
}

// NewRetryableOperation create retryable operation
func NewRetryableOperation(operation func() error, handler ErrorHandler, config *RetryConfig) *RetryableOperation {
	var retry RetryConfig
	if config != nil {
		retry = *config
	}
	if handler == nil {
		handler = NewDefaultErrorHandler(&retry)
	}
	return &RetryableOperation{
		operation: operation,
		handler:   handler,
		config:    retry,
	}
}

// Execute execute retryable operation
func (ro *RetryableOperation) Execute() error { return ro.ExecuteContext(context.Background()) }

func (ro *RetryableOperation) ExecuteContext(ctx context.Context) error {
	if ro.config.MaxRetries < 0 || ro.config.RetryInterval < 0 {
		return fmt.Errorf("retry count and interval must be non-negative")
	}
	var lastErr error
	attempts := 0

	for attempt := 0; attempt <= ro.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// wait retry interval
			delay := ro.calculateRetryDelay(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if err := ro.operation(); err != nil {
			lastErr = err
			attempts++
			if ctxErr := ctx.Err(); ctxErr != nil {
				lastErr = errors.Join(err, ctxErr)
				break
			}

			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
				attempt == ro.config.MaxRetries || !ro.handler.ShouldRetry(err, attempt) {
				break
			}

			logger.Debug(fmt.Sprintf("Retrying operation after attempt %d/%d: %v",
				attempt+1, ro.config.MaxRetries+1, err))
			continue
		}

		// operation succeeded
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts: %w",
		attempts, lastErr)
}

// calculateRetryDelay calculate retry delay
func (ro *RetryableOperation) calculateRetryDelay(attempt int) time.Duration {
	const maxDelay = time.Duration(1<<63 - 1)
	if uint64(ro.config.RetryInterval) > uint64(maxDelay/time.Millisecond) {
		return maxDelay
	}
	delay := time.Duration(ro.config.RetryInterval) * time.Millisecond

	if !ro.config.ExponentialBackoff {
		return delay
	}

	// exponential backoff strategy
	for i := 1; i < attempt; i++ {
		if delay > maxDelay/2 {
			return maxDelay
		}
		delay *= 2
	}

	return delay
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
