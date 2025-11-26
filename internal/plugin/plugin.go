package plugin

import (
	"context"
	"sync"
	"watools/pkg/db"
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
