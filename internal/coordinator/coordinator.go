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
	waPlugin    *plugin.WaPlugin
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
			waPlugin:    plugin.GetWaPluginInstance(),
		}
	})
	return waAppCoordinatorInstance
}

func (w *WaAppCoordinator) Startup(ctx context.Context) {
	w.ctx = ctx

	config.InitWithWailsContext(ctx)

	w.waApp.OnStartup(ctx)
	w.waLaunchApp.OnStartup(ctx)
	w.waPlugin.OnStartup(ctx)
}

func (w *WaAppCoordinator) Shutdown(ctx context.Context) {
	w.waApp.Shutdown(ctx)
	w.waLaunchApp.Shutdown(ctx)
	w.waPlugin.Shutdown(ctx)
}

// region app

func (w *WaAppCoordinator) HideAppApi() {
	w.waApp.HideApp()
}

func (w *WaAppCoordinator) HideOrShowAppApi() {
	w.waApp.HideOrShowApp()
}

func (w *WaAppCoordinator) ReloadApi() {
	w.waApp.Reload()
}

func (w *WaAppCoordinator) ReloadAppApi() {
	w.waApp.ReloadAPP()
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

func (w *WaAppCoordinator) GetPluginsApi() []*models.Plugin {
	return w.waPlugin.GetPlugins()
}

func (w *WaAppCoordinator) GetPluginApi(id string) *models.Plugin {
	return w.waPlugin.GetPlugin(id)
}

func (w *WaAppCoordinator) GetPluginExecEntryApi(id string) string {
	return w.waPlugin.GetPluginExecEntry(id)
}

// end region plugin
