package watcher

// DefaultWatcherConfig default config
func DefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		Enabled:          true,
		CustomWatchDirs:  []string{},
		DebounceInterval: 500, // 500ms
		EventBufferSize:  100,
		ProcessDelay: ProcessDelayConfig{
			AppAddedDelay:    5000, // 5s
			AppModifiedDelay: 5000, // 5s
		},
		RetryConfig: RetryConfig{
			MaxRetries:         3,
			RetryInterval:      1000, // 1s
			ExponentialBackoff: true,
		},
	}
}
