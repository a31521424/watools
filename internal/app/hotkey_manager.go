package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"watools/config"
	"watools/pkg/logger"
)

type HotkeyManager struct {
	listeners map[string]*HotkeyListener
	configs   map[string]HotkeyConfig
	configDir string
	mu        sync.RWMutex
}

var (
	hotkeyManagerInstance *HotkeyManager
	hotkeyManagerOnce     sync.Once
	defaultConfigs        = map[string]HotkeyConfig{
		"show-hide-window": {
			ID:     "show-hide-window",
			Name:   "Show/Hide Window",
			Hotkey: "cmd+Space",
		},
	}
)

func GetHotkeyManager() *HotkeyManager {
	hotkeyManagerOnce.Do(func() {
		hotkeyManagerInstance = &HotkeyManager{
			listeners: make(map[string]*HotkeyListener),
			configs:   make(map[string]HotkeyConfig),
			configDir: filepath.Join(config.ProjectCacheDir(), "hotkeys"),
		}
	})
	return hotkeyManagerInstance
}

func (hm *HotkeyManager) LoadConfigs() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Create config directory
	if err := os.MkdirAll(hm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create hotkey config directory: %w", err)
	}

	// Load config file
	configFile := filepath.Join(hm.configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// If config file doesn't exist, use default config and save
			hm.configs = defaultConfigs
			return hm.saveConfigsUnsafe()
		}
		return fmt.Errorf("failed to read hotkey config file: %w", err)
	}

	// Parse config
	var configs []HotkeyConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse hotkey config file: %w", err)
	}

	// Convert to map
	hm.configs = make(map[string]HotkeyConfig)
	for _, cfg := range configs {
		hm.configs[cfg.ID] = cfg
	}

	// Ensure all default configs exist
	for id, defaultCfg := range defaultConfigs {
		if _, exists := hm.configs[id]; !exists {
			hm.configs[id] = defaultCfg
		}
	}

	return nil
}

func (hm *HotkeyManager) SaveConfigs() error {
	hm.mu.RLock()
	configs := make([]HotkeyConfig, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		configs = append(configs, cfg)
	}
	hm.mu.RUnlock()

	return hm.saveConfigsUnsafe(configs)
}

// saveConfigsUnsafe saves configs without acquiring locks
// Should only be called when lock is already held
func (hm *HotkeyManager) saveConfigsUnsafe(configs ...[]HotkeyConfig) error {
	var configsToSave []HotkeyConfig
	if len(configs) > 0 {
		configsToSave = configs[0]
	} else {
		configsToSave = make([]HotkeyConfig, 0, len(hm.configs))
		for _, cfg := range hm.configs {
			configsToSave = append(configsToSave, cfg)
		}
	}

	// Serialize
	data, err := json.MarshalIndent(configsToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hotkey configs: %w", err)
	}

	// Save to file
	configFile := filepath.Join(hm.configDir, "config.json")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write hotkey config file: %w", err)
	}

	return nil
}

func (hm *HotkeyManager) GetConfig(id string) (HotkeyConfig, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	cfg, exists := hm.configs[id]
	return cfg, exists
}

func (hm *HotkeyManager) SetConfig(cfg HotkeyConfig) error {
	// Validate hotkey format
	if _, _, err := cfg.ParseHotkey(); err != nil {
		return fmt.Errorf("invalid hotkey format: %w", err)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.configs[cfg.ID] = cfg
	return nil
}

func (hm *HotkeyManager) GetAllConfigs() []HotkeyConfig {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	configs := make([]HotkeyConfig, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		configs = append(configs, cfg)
	}
	return configs
}

func (hm *HotkeyManager) RegisterAll() error {
	// Get WaApp instance first
	// Note: We don't directly call GetHotkeyManager in GetWaApp to avoid circular dependencies

	// Clear existing listeners
	hm.unregisterAll()

	// Create listener for each config
	hm.mu.RLock()
	// For safety, copy configs values
	configs := make([]HotkeyConfig, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		configs = append(configs, cfg)
	}
	// Release read lock before performing registration
	hm.mu.RUnlock()

	for _, cfg := range configs {
		modifiers, key, err := cfg.ParseHotkey()
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to parse hotkey, id: %s, error: %v", cfg.ID, err))
			continue
		}

		listener := &HotkeyListener{
			ID:        cfg.ID,
			Modifiers: modifiers,
			Key:       key,
		}

		switch cfg.ID {
		case "show-hide-window":
			listener.OnTrigger = func() {
				// Delay getting WaApp instance to avoid init order issues
				app := GetWaApp()
				if app != nil {
					app.HideOrShowApp()
				}
			}
		default:
			// Skip registration for unknown hotkey IDs
			logger.Info(fmt.Sprintf("Unknown hotkey ID, skipping registration, id: %s", cfg.ID))
			continue
		}

		// Register hotkey
		if err := listener.Register(); err != nil {
			logger.Info(fmt.Sprintf("Failed to register hotkey, id: %s, error: %v", cfg.ID, err))
			continue
		}

		// Re-lock to update listeners
		hm.mu.Lock()
		hm.listeners[cfg.ID] = listener
		hm.mu.Unlock()

		logger.Info(fmt.Sprintf("Hotkey registered, id: %s, hotkey: %s", cfg.ID, cfg.Hotkey))
	}

	return nil
}

func (hm *HotkeyManager) UnregisterAll() {
	hm.unregisterAll()
}

func (hm *HotkeyManager) unregisterAll() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for id, listener := range hm.listeners {
		if err := listener.Unregister(); err != nil {
			logger.Error(err, "Failed to unregister hotkey", "id", id)
		}
		delete(hm.listeners, id)
	}
}

func (hm *HotkeyManager) IsRegistered(id string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	listener, exists := hm.listeners[id]
	if !exists {
		return false
	}
	return listener.IsRegistered()
}
