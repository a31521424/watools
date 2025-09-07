package plugin

import (
	"context"
	"sync"
	"watools/pkg/models"

	"github.com/samber/lo"
)

type WaPlugin struct {
	ctx context.Context
}

var (
	waPluginInstance *WaPlugin
	waPluginOnce     sync.Once
)

func GetWaPluginInstance() *WaPlugin {
	waPluginOnce.Do(func() {
		waPluginInstance = &WaPlugin{}
	})
	return waPluginInstance
}

func (w *WaPlugin) OnStartup(ctx context.Context) {
	w.ctx = ctx
}

var allPlugins []*models.Plugin = []*models.Plugin{
	&models.Plugin{
		ID:          "e6c8cc94-27ba-42b7-9ad2-4543ab02635b",
		PackageID:   "watools.calculator",
		Name:        "calculator",
		Version:     "0.0.1",
		Description: "A simple calculator plugin",
	},
}

func (w *WaPlugin) GetPlugins() []*models.Plugin {
	return allPlugins
}

func (w *WaPlugin) GetPlugin(id string) *models.Plugin {
	return lo.FindOrElse(allPlugins, nil, func(plugin *models.Plugin) bool { return plugin.ID == id })
}

func (w *WaPlugin) GetPluginExecEntry(id string) string {
	plugin := w.GetPlugin(id)
	if plugin == nil {
		return ""
	}
	return plugin.GetExecEntry()
}
