package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"watools/config"
	"watools/pkg/logger"

	"golang.design/x/hotkey"
)

type hotkeyConfigFile interface {
	io.WriteCloser
	Name() string
}

type HotkeyManager struct {
	listeners        map[string]*HotkeyListener
	configs          map[string]HotkeyConfig
	configDir        string
	mu               sync.RWMutex
	registrationMu   sync.Mutex
	newHotkey        func([]hotkey.Modifier, hotkey.Key) registeredHotkey
	createConfigTemp func(string, string) (hotkeyConfigFile, error)
}

var (
	hotkeyManagerInstance *HotkeyManager
	hotkeyManagerOnce     sync.Once
)

func GetHotkeyManager() *HotkeyManager {
	hotkeyManagerOnce.Do(func() {
		hotkeyManagerInstance = &HotkeyManager{
			listeners: make(map[string]*HotkeyListener),
			configs:   make(map[string]HotkeyConfig),
			configDir: filepath.Join(config.ProjectDataDir(), "hotkeys"),
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
			hm.configs = defaultHotkeyConfigs()
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
	for id, defaultCfg := range defaultHotkeyConfigs() {
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

// saveConfigsUnsafe saves an explicit snapshot without acquiring locks. When
// no snapshot is provided, the caller must hold hm.mu.
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
	create := hm.createConfigTemp
	if create == nil {
		create = func(dir, pattern string) (hotkeyConfigFile, error) { return os.CreateTemp(dir, pattern) }
	}
	file, err := create(hm.configDir, ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary hotkey configuration: %w", err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Error(err, "Failed to remove temporary hotkey configuration")
		}
	}()
	n, writeErr := file.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	closeErr := file.Close()
	if writeErr != nil {
		writeErr = fmt.Errorf("write hotkey configuration: %w", writeErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close hotkey configuration: %w", closeErr)
	}
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), configFile); err != nil {
		return fmt.Errorf("replace hotkey configuration: %w", err)
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
	if _, exists := hm.configs[cfg.ID]; !exists {
		return fmt.Errorf("unknown hotkey ID: %s", cfg.ID)
	}
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
	return runHotkeyLifecycle(hm.registerAll)
}

func (hm *HotkeyManager) registerAll() error {
	hm.registrationMu.Lock()
	defer hm.registrationMu.Unlock()
	return hm.registerConfigsLocked(hm.GetAllConfigs())
}

// registerConfigs restores a captured registration set without consulting
// current configuration, which may itself have failed to recover.
func (hm *HotkeyManager) registerConfigs(configs []HotkeyConfig) error {
	return runHotkeyLifecycle(func() error { return hm.registerConfigsOnMain(configs) })
}

func (hm *HotkeyManager) registerConfigsOnMain(configs []HotkeyConfig) error {
	hm.registrationMu.Lock()
	defer hm.registrationMu.Unlock()
	return hm.registerConfigsLocked(configs)
}

func (hm *HotkeyManager) registerConfigsLocked(configs []HotkeyConfig) error {

	// Clear existing listeners
	if err := hm.unregisterAll(); err != nil {
		return err
	}

	var failures []error
	for _, cfg := range configs {
		modifiers, key, err := cfg.ParseHotkey()
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", cfg.ID, err))
			continue
		}

		listener := &HotkeyListener{
			ID:        cfg.ID,
			Modifiers: modifiers,
			Key:       key,
			hotkey:    cfg.Hotkey,
			newHotkey: hm.newHotkey,
		}

		switch cfg.ID {
		case "show-hide-window":
			listener.OnTrigger = func() {
				// Delay getting WaApp instance to avoid init order issues
				app := GetWaApp()
				if app != nil {
					if app.isHidden.Load() {
						app.ShowApp()
					} else {
						app.HideApp()
					}
				}
			}
		default:
			// Skip registration for unknown hotkey IDs
			failures = append(failures, fmt.Errorf("unknown hotkey ID: %s", cfg.ID))
			continue
		}

		// Register hotkey
		if err := listener.Register(); err != nil {
			if listener.hasResources() {
				hm.mu.Lock()
				hm.listeners[cfg.ID] = listener
				hm.mu.Unlock()
			}
			failures = append(failures, fmt.Errorf("%s: %w", cfg.ID, err))
			continue
		}

		// Re-lock to update listeners
		hm.mu.Lock()
		hm.listeners[cfg.ID] = listener
		hm.mu.Unlock()

		logger.Info(fmt.Sprintf("Hotkey registered, id: %s, hotkey: %s", cfg.ID, cfg.Hotkey))
	}

	return errors.Join(failures...)
}

func (hm *HotkeyManager) UnregisterAll() {
	if err := runHotkeyLifecycle(hm.unregisterAllLocked); err != nil {
		logger.Error(err, "Failed to unregister hotkeys")
	}
}

func (hm *HotkeyManager) unregisterAllLocked() error {
	hm.registrationMu.Lock()
	defer hm.registrationMu.Unlock()
	return hm.unregisterAll()
}

func (hm *HotkeyManager) unregisterAll() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	var failures []error
	for id, listener := range hm.listeners {
		if err := listener.Unregister(); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", id, err))
			continue
		}
		delete(hm.listeners, id)
	}
	return errors.Join(failures...)
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

func (hm *HotkeyManager) GetHotkeyStates() []HotkeyConfigAPI {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	states := make([]HotkeyConfigAPI, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		state := HotkeyConfigAPI{ID: cfg.ID, Name: cfg.Name, Hotkey: cfg.Hotkey}
		if listener, ok := hm.listeners[cfg.ID]; ok && listener.IsRegistered() {
			state.Registered = true
			state.RegisteredHotkey = listener.hotkey
		}
		states = append(states, state)
	}
	return states
}
