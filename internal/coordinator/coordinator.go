package coordinator

import (
	"context"
	"sync"
	"watools/config"
	"watools/internal/app"
	"watools/internal/command"
	"watools/internal/plugin"
	"watools/pkg/logger"
	"watools/pkg/models"
)

type WaAppCoordinator struct {
	ctx         context.Context
	waApp       *app.WaApp
	waLaunchApp *command.WaLaunchApp
	waPluginApp *plugin.WaPlugin
}

var (
	waAppCoordinatorInstance *WaAppCoordinator
	waAppCoordinatorOnce     sync.Once
)

func GetWaAppCoordinator() *WaAppCoordinator {
	waAppCoordinatorOnce.Do(func() {
		waAppCoordinatorInstance = &WaAppCoordinator{
			waApp:       app.GetWaApp(),
			waLaunchApp: command.GetWaLaunch(),
			waPluginApp: plugin.GetWaPlugin(),
		}
	})
	return waAppCoordinatorInstance
}

func (w *WaAppCoordinator) Startup(ctx context.Context) {
	w.ctx = ctx

	config.InitWithWailsContext(ctx)

	w.waApp.OnStartup(ctx)
	w.waLaunchApp.OnStartup(ctx)
	w.waPluginApp.OnStartup(ctx)
}

func (w *WaAppCoordinator) Shutdown(ctx context.Context) {
	w.waApp.Shutdown(ctx)
	w.waLaunchApp.Shutdown(ctx)
	w.waPluginApp.OnShutdown(ctx)
}

// region app

func (w *WaAppCoordinator) HideAppApi() {
	w.waApp.HideApp()
}

func (w *WaAppCoordinator) HideOrShowAppApi() {
	w.waApp.HideOrShowApp()
}

// end region app

// region command

func (w *WaAppCoordinator) GetApplicationCommandsApi() []interface{} {
	return w.waLaunchApp.GetApplicationCommands()
}

func (w *WaAppCoordinator) GetOperatorCommandsApi() []interface{} {
	return w.waLaunchApp.GetOperationCommands()
}

func (w *WaAppCoordinator) TriggerCommandApi(uniqueTriggerID string, triggerCategory string) {
	category, err := models.ParseCommandCategory(triggerCategory)
	if err != nil {
		logger.Error(err)
	}
	w.waLaunchApp.TriggerCommand(uniqueTriggerID, category)
}

// end region command

// region plugin

func (w *WaAppCoordinator) GetPluginsApi() []map[string]interface{} {
	return w.waPluginApp.GetPlugins()
}

// end region plugin
