package coordinator

import (
	"context"
	"sync"
	"time"
	"watools/config"
	"watools/internal/api"
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
	waApi       *api.WaApi
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
			waApi:       api.GetWaApi(),
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

func (w *WaAppCoordinator) GetClipboardContentApi() (*app.ClipboardContent, error) {
	return w.waApp.GetClipboardContent()
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

func (w *WaAppCoordinator) UpdateApplicationUsageApi(usageUpdates []map[string]interface{}) error {
	updates := make([]models.ApplicationUsageUpdate, len(usageUpdates))
	for i, update := range usageUpdates {
		id, _ := update["id"].(string)
		lastUsedAtStr, _ := update["lastUsedAt"].(string)
		usedCount, _ := update["usedCount"].(float64)

		lastUsedAt, err := time.Parse(time.RFC3339, lastUsedAtStr)
		if err != nil {
			logger.Error(err, "Failed to parse lastUsedAt")
			continue
		}

		updates[i] = models.ApplicationUsageUpdate{
			ID:         id,
			LastUsedAt: lastUsedAt,
			UsedCount:  int(usedCount),
		}
	}

	return w.waLaunchApp.UpdateApplicationUsage(updates)
}

// end region command

// region plugin

func (w *WaAppCoordinator) GetPluginsApi() []map[string]interface{} {
	return w.waPluginApp.GetPlugins()
}

func (w *WaAppCoordinator) GetPluginJsEntryUrlApi(packageID string) string {
	return w.waPluginApp.GetJsEntryUrl(packageID)
}

func (w *WaAppCoordinator) UpdatePluginUsageApi(usageUpdates []map[string]interface{}) error {
	updates := make([]models.PluginUsageUpdate, len(usageUpdates))
	for i, update := range usageUpdates {
		packageID, _ := update["packageId"].(string)
		lastUsedAtStr, _ := update["lastUsedAt"].(string)
		usedCount, _ := update["usedCount"].(float64)

		lastUsedAt, err := time.Parse(time.RFC3339, lastUsedAtStr)
		if err != nil {
			logger.Error(err, "Failed to parse lastUsedAt")
			continue
		}

		updates[i] = models.PluginUsageUpdate{
			PackageID:  packageID,
			LastUsedAt: lastUsedAt,
			UsedCount:  int(usedCount),
		}
	}

	return w.waPluginApp.UpdatePluginUsage(updates)
}

// end region plugin

// region api

func (w *WaAppCoordinator) OpenFolder(path string) {
	w.waApi.OpenFolderWithPath(path)
}

// end region api
