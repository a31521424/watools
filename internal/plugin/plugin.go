package plugin

import (
	"context"
	"sync"
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
