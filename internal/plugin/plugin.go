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
}

func GetWaPlugin() *WaPlugin {
	waPluginOnce.Do(func() {
		waPluginInstance = &WaPlugin{}
	})
	return waPluginInstance
}
func (p *WaPlugin) OnStartup(ctx context.Context) {
	p.ctx = ctx
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
