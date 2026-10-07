package watcher

import (
	"fmt"
	"time"
)

// Validate validate config
func (c *WatcherConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("watcher config is required")
	}

	if c.DebounceInterval < 0 {
		return fmt.Errorf("debounceInterval must be non-negative")
	}

	if c.EventBufferSize <= 0 {
		return fmt.Errorf("eventBufferSize must be positive")
	}

	if c.ProcessDelay.AppAddedDelay < 0 {
		return fmt.Errorf("appAddedDelay must be non-negative")
	}

	if c.ProcessDelay.AppModifiedDelay < 0 {
		return fmt.Errorf("appModifiedDelay must be non-negative")
	}

	if c.RetryConfig.MaxRetries < 0 {
		return fmt.Errorf("maxRetries must be non-negative")
	}

	if c.RetryConfig.RetryInterval < 0 {
		return fmt.Errorf("retryInterval must be non-negative")
	}

	return nil
}

// GetDebounceInterval get debounce interval
func (c *WatcherConfig) GetDebounceInterval() time.Duration {
	return time.Duration(c.DebounceInterval) * time.Millisecond
}

// GetAppAddedDelay get app added delay
func (c *WatcherConfig) GetAppAddedDelay() time.Duration {
	return time.Duration(c.ProcessDelay.AppAddedDelay) * time.Millisecond
}

// GetAppModifiedDelay get app modified delay
func (c *WatcherConfig) GetAppModifiedDelay() time.Duration {
	return time.Duration(c.ProcessDelay.AppModifiedDelay) * time.Millisecond
}

// GetRetryInterval get retry interval
func (c *WatcherConfig) GetRetryInterval() time.Duration {
	return time.Duration(c.RetryConfig.RetryInterval) * time.Millisecond
}

// clone owns the mutable directory list as well as the value configuration.
func (c *WatcherConfig) clone() *WatcherConfig {
	if c == nil {
		return nil
	}
	copy := *c
	copy.CustomWatchDirs = append([]string(nil), c.CustomWatchDirs...)
	return &copy
}
