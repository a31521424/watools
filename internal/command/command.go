package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"watools/internal/command/application"
	"watools/internal/command/operator"
	"watools/internal/command/watcher"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/lo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	launchAppInstance *WaLaunchApp
	launchAppOnce     sync.Once
)

type WaLaunchApp struct {
	ctx               context.Context
	applicationRunner []models.CommandRunner
	operationRunner   []models.CommandRunner
	watchManager      watcher.AppWatchManager
}

func GetWaLaunch() *WaLaunchApp {
	launchAppOnce.Do(func() {
		launchAppInstance = &WaLaunchApp{}
	})
	return launchAppInstance
}

func (w *WaLaunchApp) Startup(ctx context.Context) {
	w.ctx = ctx
	w.initAppWatcher()
	w.asyncUpdateApplications(30 * time.Second)
}

func (w *WaLaunchApp) Shutdown(ctx context.Context) {
	if w.watchManager != nil {
		if err := w.watchManager.Stop(); err != nil {
			logger.Error(err, "Failed to stop app watch manager")
		}
	}
}

func (w *WaLaunchApp) asyncUpdateApplications(delay time.Duration) {
	time.Sleep(delay)
	go w.updateApplications()
}

func (w *WaLaunchApp) updateApplications() {
	dbInstance := db.GetWaDB()
	commands := dbInstance.GetCommands(w.ctx)
	seen := make(map[string]struct{})
	var updateCommands []*models.ApplicationCommand
	var insertCommands []*models.ApplicationCommand
	for _, command := range commands {
		seen[command.Path] = struct{}{}
		id := command.ID
		fi, err := os.Stat(command.Path)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to stat command %s", command.Path))
			continue
		}
		if fi.ModTime().Format(time.DateTime) == command.DirUpdatedAt.Format(time.DateTime) {
			continue
		}
		logger.Info(fmt.Sprintf("Update dir updated for command: %s, %s", command.Name, command.Path))
		command, err := application.ParseApplication(command.Path)
		if err != nil {
			logger.Error(err, "Failed to parse application")
			err := dbInstance.DeleteCommand(w.ctx, id)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Failed to delete command %s", id))
			}
			continue
		}
		command.ID = id
		updateCommands = append(updateCommands, command)
	}
	logger.Info(fmt.Sprintf("Update commands result: updated %d / total %d", len(updateCommands), len(commands)))
	err := dbInstance.BatchUpdateCommands(w.ctx, updateCommands)
	if err != nil {
		logger.Error(err, "Failed to batch update updated commands to db")
	}

	logger.Info("Checking commands from disk")
	appPathInfos := application.GetAppPathInfos()
	for _, appPathInfo := range appPathInfos {
		if _, exists := seen[appPathInfo.Path]; exists {
			continue
		}
		logger.Info(fmt.Sprintf("Adding command from path: %s", appPathInfo.Path))
		command, err := application.ParseApplication(appPathInfo.Path)
		if err != nil {
			logger.Error(err, "Failed to parse application")
			continue
		}
		insertCommands = append(insertCommands, command)
	}
	logger.Info(fmt.Sprintf("Scan disk commands result: added %d / total %d", len(insertCommands), len(appPathInfos)))
	err = dbInstance.BatchInsertCommands(w.ctx, insertCommands)
	if err != nil {
		logger.Error(err, "Failed to batch insert new commands to db")
	}
	if len(insertCommands)+len(updateCommands) > 0 {
		runtime.EventsEmit(w.ctx, "watools.applicationChanged")
	}
}

var ApiMutex sync.Mutex

func (w *WaLaunchApp) getApplicationCommands() []*models.ApplicationCommand {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	dbInstance := db.GetWaDB()
	commands := dbInstance.GetCommands(w.ctx)
	if len(commands) == 0 {
		commands, err := application.GetApplications()
		if err != nil {
			logger.Error(err, "Failed to get application")
			return []*models.ApplicationCommand{}
		}
		err = dbInstance.BatchInsertCommands(w.ctx, commands)
		if err != nil {
			logger.Error(err, "Failed to batch insert commands")
		}
	}
	for _, command := range commands {
		if command.IconPath == "" {
			command.IconPath = application.GetDefaultIconPath()
		}
	}
	return commands
}

func (w *WaLaunchApp) GetApplicationCommands() []interface{} {
	var commands []interface{}
	w.applicationRunner = nil

	apps := w.getApplicationCommands()
	w.applicationRunner = append(w.applicationRunner, lo.Map(apps, func(app *models.ApplicationCommand, _ int) models.CommandRunner { return app })...)
	commands = append(commands, lo.Map(apps, func(app *models.ApplicationCommand, _ int) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(app)
		_ = json.Unmarshal(data, &m)
		return m
	})...)

	return commands
}

func (w *WaLaunchApp) GetOperationCommands() []interface{} {
	var commands []interface{}
	w.operationRunner = nil

	operations := operator.GetOperations()
	w.operationRunner = append(w.operationRunner, lo.Map(operations, func(operation *models.OperationCommand, _ int) models.CommandRunner { return operation })...)
	commands = append(commands, lo.Map(operations, func(operation *models.OperationCommand, _ int) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(operation)
		_ = json.Unmarshal(data, &m)
		return m
	})...)

	return lo.Map(w.operationRunner, func(runner models.CommandRunner, _ int) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(runner)
		_ = json.Unmarshal(data, &m)
		return m
	})
}

func (w *WaLaunchApp) TriggerCommand(uniqueTriggerID string, triggerCategory models.CommandCategory) {
	var runners []models.CommandRunner
	if triggerCategory == models.CategoryApplication {
		runners = w.applicationRunner
	} else if triggerCategory == models.CategoryOperation {
		runners = w.operationRunner
	} else {
		logger.Error(fmt.Errorf("trigger category is not valid: %s", triggerCategory))
	}

	for _, runner := range runners {
		if runner.GetTriggerID() == uniqueTriggerID {
			err := runner.OnTrigger()
			if err != nil {
				logger.Error(err, fmt.Sprintf("cant trigger runner: %s", uniqueTriggerID))
			} else {
				logger.Info(fmt.Sprintf("trigger runner success: %s", uniqueTriggerID))
			}
			return
		}
	}

	logger.Error(fmt.Errorf("not find runner: %s", uniqueTriggerID))
}

func (w *WaLaunchApp) initAppWatcher() {
	eventHandler := watcher.NewDefaultAppEventHandler(w.ctx)

	watchManager, err := watcher.NewAppWatchManager(eventHandler, w.ctx)
	if err != nil {
		logger.Error(err, "Failed to create app watch manager")
		return
	}

	w.watchManager = watchManager

	if err := w.watchManager.Start(); err != nil {
		logger.Error(err, "Failed to start app watch manager")
		w.watchManager = nil
		return
	}

	logger.Info("App watcher initialized successfully")
}

func (w *WaLaunchApp) GetWatchStatus() map[string]interface{} {
	status := make(map[string]interface{})

	if w.watchManager == nil {
		status["enabled"] = false
		status["error"] = "watch manager not initialized"
		return status
	}

	status["enabled"] = true
	status["running"] = w.watchManager.IsRunning()

	status["watchDirs"] = w.watchManager.GetWatchDirs()
	status["config"] = w.watchManager.GetConfig()
	status["metrics"] = w.watchManager.GetMetrics()

	return status
}

func (w *WaLaunchApp) GetWatchMetrics() *watcher.WatcherMetrics {
	if w.watchManager == nil {
		return watcher.NewWatcherMetrics()
	}
	return w.watchManager.GetMetrics()
}
