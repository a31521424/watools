package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/lo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	waPluginInstance *WaPlugin
	waPluginOnce     sync.Once
)

type WaPlugin struct {
	ctx          context.Context
	pluginStates []*models.PluginState
	installer    *PluginInstaller
	storageMutex sync.RWMutex // protect storage operations
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
	dbInstance := db.GetWaDB()
	p.pluginStates = dbInstance.GetPlugins(p.ctx)
}

func (p *WaPlugin) GetPlugins() []map[string]interface{} {
	return lo.Map(p.pluginStates, func(item *models.PluginState, index int) map[string]interface{} {
		return item.GetFullInfo()
	})
}

func (p *WaPlugin) GetJsEntryUrl(packageID string) string {
	plugin, find := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !find {
		return ""
	}
	return plugin.GetJsEntryUrl()
}

func (p *WaPlugin) UpdatePluginUsage(updates []models.PluginUsageUpdate) error {
	dbInstance := db.GetWaDB()
	return dbInstance.BatchUpdatePluginUsage(p.ctx, updates)
}

// InstallPlugin installs a plugin from a .wt file
func (p *WaPlugin) InstallPlugin(wtFilePath string) error {
	if err := p.installer.InstallFromWtFile(wtFilePath); err != nil {
		return err
	}
	// Reload plugins after installation
	p.loadPlugins()
	return nil
}

// InstallPluginByFileDialog InstallPlugin install a plugin from a.wt file by file dialog
func (p *WaPlugin) InstallPluginByFileDialog() error {
	pluginPaths, err := runtime.OpenMultipleFilesDialog(p.ctx, runtime.OpenDialogOptions{
		Title:            "Select Plugin .wt File(s)",
		Filters:          []runtime.FileFilter{{DisplayName: "WaTools Plugin Files", Pattern: "*.wt"}},
		DefaultDirectory: "",
	})
	if err != nil {
		return err
	}
	if len(pluginPaths) == 0 {
		return nil
	}
	needRefresh := false
	for _, pluginPath := range pluginPaths {
		if err := p.installer.InstallFromWtFile(pluginPath); err != nil {
			logger.Error(err, "Failed to install plugin from file", "file", pluginPath)
			continue
		}
		needRefresh = true
	}
	if needRefresh {
		p.loadPlugins()
	}
	return nil
}

// UninstallPlugin uninstalls a plugin by packageID
func (p *WaPlugin) UninstallPlugin(packageID string) error {
	if err := p.installer.UninstallPlugin(packageID); err != nil {
		return err
	}
	// Reload plugins after uninstallation
	p.loadPlugins()
	return nil
}

// TogglePlugin enables or disables a plugin
func (p *WaPlugin) TogglePlugin(packageID string, enabled bool) error {
	dbInstance := db.GetWaDB()
	if err := dbInstance.UpdatePluginEnabled(p.ctx, packageID, enabled); err != nil {
		return err
	}
	// Update local state
	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if found {
		plugin.Enabled = enabled
	}
	return nil
}

// GetStorage gets a value from plugin storage by key
func (p *WaPlugin) GetStorage(packageID string, key string) (interface{}, error) {
	p.storageMutex.RLock()
	defer p.storageMutex.RUnlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found {
		return nil, fmt.Errorf("plugin not found: %s", packageID)
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
	p.storageMutex.Lock()
	defer p.storageMutex.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found {
		return fmt.Errorf("plugin not found: %s", packageID)
	}

	// initialize storage if nil
	if plugin.Storage == nil {
		plugin.Storage = make(map[string]interface{})
	}

	// set value
	plugin.Storage[key] = value

	// persist to database
	return p.saveStorage(packageID, plugin.Storage)
}

// RemoveStorage removes a key from plugin storage
func (p *WaPlugin) RemoveStorage(packageID string, key string) error {
	p.storageMutex.Lock()
	defer p.storageMutex.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found {
		return fmt.Errorf("plugin not found: %s", packageID)
	}

	if plugin.Storage == nil {
		return nil
	}

	delete(plugin.Storage, key)

	// persist to database
	return p.saveStorage(packageID, plugin.Storage)
}

// ClearStorage clears all storage for a plugin
func (p *WaPlugin) ClearStorage(packageID string) error {
	p.storageMutex.Lock()
	defer p.storageMutex.Unlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found {
		return fmt.Errorf("plugin not found: %s", packageID)
	}

	plugin.Storage = make(map[string]interface{})

	// persist to database
	return p.saveStorage(packageID, plugin.Storage)
}

// ListStorageKeys returns all keys in plugin storage
func (p *WaPlugin) ListStorageKeys(packageID string) ([]string, error) {
	p.storageMutex.RLock()
	defer p.storageMutex.RUnlock()

	plugin, found := lo.Find(p.pluginStates, func(item *models.PluginState) bool {
		return item.PackageID == packageID
	})
	if !found {
		return nil, fmt.Errorf("plugin not found: %s", packageID)
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
