package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/lo"
)

var (
	waPluginInstance *WaPlugin
	waPluginOnce     sync.Once
)

type WaPlugin struct {
	ctx          context.Context
	pluginStates []*models.PluginState
	installer    *PluginInstaller
	stateMu      sync.RWMutex // protect storage operations
}

func GetWaPlugin() *WaPlugin {
	waPluginOnce.Do(func() {
		waPluginInstance = &WaPlugin{}
	})
	return waPluginInstance
}

func (p *WaPlugin) OnStartup(ctx context.Context) {
	p.ctx = ctx
	p.installer = NewPluginInstaller(ctx)
	p.loadPlugins()
}

func (p *WaPlugin) OnShutdown(ctx context.Context) {

}

func (p *WaPlugin) loadPlugins() {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if err := p.loadPluginsLocked(); err != nil {
		logger.Error(err, "Failed to load plugins")
	}
}

func (p *WaPlugin) loadPluginsLocked() error {
	states, err := db.GetWaDB().GetPlugins(p.ctx)
	if err != nil {
		return err
	}
	p.pluginStates = states
	return nil
}
func (p *WaPlugin) GetPlugins() ([]map[string]interface{}, error) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if err := p.loadPluginsLocked(); err != nil {
		return nil, err
	}
	return lo.Map(p.pluginStates, func(item *models.PluginState, index int) map[string]interface{} { return item.GetFullInfo() }), nil
}

func (p *WaPlugin) GetJsEntryUrl(packageID string) string {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	plugin, find := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !find {
		return ""
	}
	if !plugin.Enabled {
		return ""
	}
	return plugin.GetJsEntryUrl()
}

func (p *WaPlugin) UpdatePluginUsage(updates []models.PluginUsageUpdate) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	dbInstance := db.GetWaDB()
	if err := dbInstance.BatchUpdatePluginUsage(p.ctx, updates); err != nil {
		return err
	}
	return p.loadPluginsLocked()
}

// InstallPlugin installs a plugin from a .wt file
func (p *WaPlugin) InstallPlugin(wtFilePath string) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if err := p.installer.InstallFromWtFile(wtFilePath); err != nil {
		return err
	}
	// Reload after committing the new revision.
	return p.loadPluginsLocked()
}

// InstallPluginByFileDialog InstallPlugin install a plugin from a.wt file by file dialog
func (p *WaPlugin) InstallPluginPaths(pluginPaths []string) error {
	if len(pluginPaths) == 0 {
		return nil
	}
	var failures []error
	for _, pluginPath := range pluginPaths {
		if err := p.InstallPlugin(pluginPath); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", pluginPath, err))
		}
	}
	return errors.Join(failures...)

}

// UninstallPlugin uninstalls a plugin by packageID
func (p *WaPlugin) UninstallPlugin(packageID string) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	if err := p.installer.UninstallPlugin(packageID); err != nil {
		return err
	}
	return p.loadPluginsLocked()
}

// TogglePlugin enables or disables a plugin
func (p *WaPlugin) TogglePlugin(packageID string, enabled bool) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	state, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool { return item.PackageID == packageID })
	if !found {
		return fmt.Errorf("plugin not found: %s", packageID)
	}
	if err := db.GetWaDB().UpdatePluginEnabled(p.ctx, packageID, enabled); err != nil {
		return err
	}
	state.Enabled = enabled
	return nil
}

// GetStorage gets a value from plugin storage by key
func (p *WaPlugin) GetStorage(packageID string, key string) (interface{}, error) {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found || !plugin.Enabled {
		return nil, fmt.Errorf("plugin is missing or disabled: %s", packageID)
	}

	if plugin.Storage == nil {
		return nil, nil
	}

	value, exists := plugin.Storage[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// SetStorage sets a value in plugin storage by key
func (p *WaPlugin) SetStorage(packageID string, key string, value interface{}) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found || !plugin.Enabled {
		return fmt.Errorf("plugin is missing or disabled: %s", packageID)
	}

	next := make(map[string]interface{}, len(plugin.Storage)+1)
	for k, v := range plugin.Storage {
		next[k] = v
	}
	next[key] = value
	if err := p.saveStorage(packageID, next); err != nil {
		return err
	}
	plugin.Storage = next
	return nil

}

// RemoveStorage removes a key from plugin storage
func (p *WaPlugin) RemoveStorage(packageID string, key string) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found || !plugin.Enabled {
		return fmt.Errorf("plugin is missing or disabled: %s", packageID)
	}

	if plugin.Storage == nil {
		return nil
	}

	next := make(map[string]interface{}, len(plugin.Storage))
	for k, v := range plugin.Storage {
		if k != key {
			next[k] = v
		}
	}
	if err := p.saveStorage(packageID, next); err != nil {
		return err
	}
	plugin.Storage = next
	return nil
}

// ClearStorage clears all storage for a plugin
func (p *WaPlugin) ClearStorage(packageID string) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found || !plugin.Enabled {
		return fmt.Errorf("plugin is missing or disabled: %s", packageID)
	}

	next := make(map[string]interface{})
	if err := p.saveStorage(packageID, next); err != nil {
		return err
	}
	plugin.Storage = next
	return nil
}

// ListStorageKeys returns all keys in plugin storage
func (p *WaPlugin) ListStorageKeys(packageID string) ([]string, error) {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found || !plugin.Enabled {
		return nil, fmt.Errorf("plugin is missing or disabled: %s", packageID)
	}

	if plugin.Storage == nil {
		return []string{}, nil
	}

	keys := make([]string, 0, len(plugin.Storage))
	for key := range plugin.Storage {
		keys = append(keys, key)
	}

	return keys, nil
}

// saveStorage persists storage to database (must be called with lock held)
func (p *WaPlugin) saveStorage(packageID string, storage map[string]interface{}) error {
	dbInstance := db.GetWaDB()

	// marshal storage to JSON string
	storageJSON, err := json.Marshal(storage)
	if err != nil {
		return fmt.Errorf("failed to marshal storage: %w", err)
	}

	return dbInstance.UpdatePluginStorage(p.ctx, packageID, string(storageJSON))
}

// IsEnabled is used by resource handlers; disabled packages cannot start new loads.
func (p *WaPlugin) IsEnabled(packageID string) bool {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	for _, state := range p.pluginStates {
		if state.PackageID == packageID {
			return state.Enabled
		}
	}
	return false
}
