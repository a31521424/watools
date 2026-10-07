package watcher

// DefaultWatcherConfig default config
func DefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		Enabled:          true,
		CustomWatchDirs:  []string{},
		DebounceInterval: 500,
		EventBufferSize:  100,
		ProcessDelay: ProcessDelayConfig{
			AppAddedDelay:    1000,
			AppModifiedDelay: 500,
		},
		RetryConfig: RetryConfig{
			MaxRetries:         3,
			RetryInterval:      1000,
			ExponentialBackoff: true,
		},
	}
}
