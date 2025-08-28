package watcher

import (
	"time"
	"watools/pkg/models"
)

// AppChangeType app change event type
type AppChangeType int

const (
	AppAdded AppChangeType = iota
	AppRemoved
	AppModified
)

// AppChangeEvent app change event
type AppChangeEvent struct {
	Type AppChangeType
	Path string
}

// AppEventHandler app event handler interface
type AppEventHandler interface {
	OnAppAdded(command *models.ApplicationCommand) error
	OnAppRemoved(path string) error
	OnAppModified(command *models.ApplicationCommand) error
}

// AppWatchManager interface for app watch manager
type AppWatchManager interface {
	Start() error
	Stop() error
	IsRunning() bool
	GetWatchDirs() []string
	AddWatchDir(dir string) error
	GetMetrics() *WatcherMetrics
	GetConfig() *WatcherConfig
}

// ProcessDelayConfig process delay configuration
type ProcessDelayConfig struct {
	// app added process delay (milliseconds)
	AppAddedDelay int `yaml:"appAddedDelay" json:"appAddedDelay"`

	// app modified process delay (milliseconds)
	AppModifiedDelay int `yaml:"appModifiedDelay" json:"appModifiedDelay"`
}

// RetryConfig retry configuration
type RetryConfig struct {
	// max retry attempts
	MaxRetries int `yaml:"maxRetries" json:"maxRetries"`

	// retry interval (milliseconds)
	RetryInterval int `yaml:"retryInterval" json:"retryInterval"`

	// enable exponential backoff
	ExponentialBackoff bool `yaml:"exponentialBackoff" json:"exponentialBackoff"`
}

// WatcherConfig watcher configuration
type WatcherConfig struct {
	// enable file system watching
	Enabled bool `yaml:"enabled" json:"enabled"`

	// custom watch directories
	CustomWatchDirs []string `yaml:"customWatchDirs" json:"customWatchDirs"`

	// event debounce interval (milliseconds)
	DebounceInterval int `yaml:"debounceInterval" json:"debounceInterval"`

	// event buffer size
	EventBufferSize int `yaml:"eventBufferSize" json:"eventBufferSize"`

	// process delay config
	ProcessDelay ProcessDelayConfig `yaml:"processDelay" json:"processDelay"`

	// error retry config
	RetryConfig RetryConfig `yaml:"retryConfig" json:"retryConfig"`
}

// WatcherMetrics watcher metrics
type WatcherMetrics struct {
	EventsProcessed    int64            `json:"eventsProcessed"`
	EventsDropped      int64            `json:"eventsDropped"`
	ErrorsCount        int64            `json:"errorsCount"`
	EventsByType       map[string]int64 `json:"eventsByType"`
	LastEventTime      time.Time        `json:"lastEventTime" ts_type:"string"`
	WatcherStartTime   time.Time        `json:"watcherStartTime" ts_type:"string"`
	ProcessingDuration time.Duration    `json:"processingDuration"`
}

// NewWatcherMetrics create watcher metrics
func NewWatcherMetrics() *WatcherMetrics {
	return &WatcherMetrics{
		EventsByType:     make(map[string]int64),
		WatcherStartTime: time.Now(),
	}
}

// DefaultAppEventHandler default app event handler interface
type DefaultAppEventHandler interface {
	AppEventHandler
}
