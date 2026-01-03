package coordinator

import (
	"context"
	"fmt"
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

// InstallPluginApi installs a plugin from a .wt file path
func (w *WaAppCoordinator) InstallPluginApi(wtFilePath string) error {
	return w.waPluginApp.InstallPlugin(wtFilePath)
}

// InstallPluginByFileDialogApi installs a plugin from a .wt file by file dialog
func (w *WaAppCoordinator) InstallPluginByFileDialogApi() error {
	return w.waPluginApp.InstallPluginByFileDialog()
}

// UninstallPluginApi uninstalls a plugin by packageID
func (w *WaAppCoordinator) UninstallPluginApi(packageID string) error {
	return w.waPluginApp.UninstallPlugin(packageID)
}

// TogglePluginApi enables or disables a plugin
func (w *WaAppCoordinator) TogglePluginApi(packageID string, enabled bool) error {
	return w.waPluginApp.TogglePlugin(packageID, enabled)
}

// end region plugin

// region api

func (w *WaAppCoordinator) OpenFolder(path string) {
	w.waApi.OpenFolderWithPath(path)
}

func (w *WaAppCoordinator) SaveBase64Image(base64Data string) string {
	return w.waApi.SaveBase64Image(base64Data)
}

// end region api

// region proxy

// HttpProxyApi provides generic HTTP proxy functionality for plugins
// This allows plugins to make HTTP requests without CORS restrictions
func (w *WaAppCoordinator) HttpProxyApi(requestMap map[string]interface{}) (map[string]interface{}, error) {
	// Parse request from frontend
	url, _ := requestMap["url"].(string)
	method, _ := requestMap["method"].(string)
	body, _ := requestMap["body"].(string)
	timeout, _ := requestMap["timeout"].(float64)

	// Parse headers
	headers := make(map[string]string)
	if headersMap, ok := requestMap["headers"].(map[string]interface{}); ok {
		for key, value := range headersMap {
			if strValue, ok := value.(string); ok {
				headers[key] = strValue
			}
		}
	}

	req := api.HttpProxyRequest{
		URL:     url,
		Method:  method,
		Headers: headers,
		Body:    body,
		Timeout: int(timeout),
	}

	// Call API service
	result, err := w.waApi.HttpProxy(req)
	if err != nil {
		logger.Error(err, "HTTP proxy request failed")
		return map[string]interface{}{
			"error":       err.Error(),
			"status_code": 0,
		}, err
	}

	// Return result as map for frontend
	return map[string]interface{}{
		"status_code": result.StatusCode,
		"headers":     result.Headers,
		"body":        result.Body,
		"error":       result.Error,
	}, nil
}

// end region proxy

// region plugin storage

// PluginStorageGetApi retrieves a value from plugin storage
func (w *WaAppCoordinator) PluginStorageGetApi(requestMap map[string]interface{}) (interface{}, error) {
	packageID, _ := requestMap["packageId"].(string)
	key, _ := requestMap["key"].(string)

	if packageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	return w.waPluginApp.GetStorage(packageID, key)
}

// PluginStorageSetApi sets a value in plugin storage
func (w *WaAppCoordinator) PluginStorageSetApi(requestMap map[string]interface{}) error {
	packageID, _ := requestMap["packageId"].(string)
	key, _ := requestMap["key"].(string)
	value := requestMap["value"]

	if packageID == "" {
		return fmt.Errorf("packageId is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	return w.waPluginApp.SetStorage(packageID, key, value)
}

// PluginStorageRemoveApi removes a key from plugin storage
func (w *WaAppCoordinator) PluginStorageRemoveApi(requestMap map[string]interface{}) error {
	packageID, _ := requestMap["packageId"].(string)
	key, _ := requestMap["key"].(string)

	if packageID == "" {
		return fmt.Errorf("packageId is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	return w.waPluginApp.RemoveStorage(packageID, key)
}

// PluginStorageClearApi clears all storage for a plugin
func (w *WaAppCoordinator) PluginStorageClearApi(requestMap map[string]interface{}) error {
	packageID, _ := requestMap["packageId"].(string)

	if packageID == "" {
		return fmt.Errorf("packageId is required")
	}

	return w.waPluginApp.ClearStorage(packageID)
}

// PluginStorageKeysApi returns all keys in plugin storage
func (w *WaAppCoordinator) PluginStorageKeysApi(requestMap map[string]interface{}) ([]string, error) {
	packageID, _ := requestMap["packageId"].(string)

	if packageID == "" {
		return nil, fmt.Errorf("packageId is required")
	}

	return w.waPluginApp.ListStorageKeys(packageID)
}

// end region plugin storage
