package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"watools/internal/command/application"
	"watools/internal/command/operator"
	"watools/internal/command/watcher"
	"watools/pkg/db"
	"watools/pkg/generics"
	"watools/pkg/logger"
	"watools/pkg/models"
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
	w.initCommandsUpdater()
	w.initAppWatcher()
}

func (w *WaLaunchApp) Shutdown(ctx context.Context) {
	if w.watchManager != nil {
		if err := w.watchManager.Stop(); err != nil {
			logger.Error(err, "Failed to stop app watch manager")
		}
	}
}

func (w *WaLaunchApp) initCommandsUpdater() {
	go func() {
		dbInstance := db.GetWaDB()

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				commands := dbInstance.FindExpiredCommands(w.ctx)
				logger.Info(fmt.Sprintf("Found %d expired commands", len(commands)))
				if len(commands) > 0 {
					var updateCommands []*models.ApplicationCommand
					for _, command := range commands {
						id := command.ID
						command, err := application.ParseApplication(command.Path)
						if err != nil {
							logger.Error(err, "Failed to parse application")
							err := dbInstance.DeleteCommand(w.ctx, id)
							if err != nil {
								logger.Error(err, fmt.Sprintf("Failed to delete command %s", id))
								return
							}
							continue
						}
						command.ID = id
						updateCommands = append(updateCommands, command)
					}
					err := dbInstance.BatchUpdateCommands(w.ctx, updateCommands)
					if err != nil {
						logger.Error(err, "Failed to batch update commands")
					}
				}
			}
		}
	}()
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
	w.applicationRunner = append(w.applicationRunner, generics.Map(apps, func(app *models.ApplicationCommand) models.CommandRunner { return app })...)
	commands = append(commands, generics.Map(apps, func(app *models.ApplicationCommand) interface{} {
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
	w.operationRunner = append(w.operationRunner, generics.Map(operations, func(operation *models.OperationCommand) models.CommandRunner { return operation })...)
	commands = append(commands, generics.Map(operations, func(operation *models.OperationCommand) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(operation)
		_ = json.Unmarshal(data, &m)
		return m
	})...)

	return generics.Map(w.operationRunner, func(runner models.CommandRunner) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(runner)
		_ = json.Unmarshal(data, &m)
		return m
	})
}

func (w *WaLaunchApp) TriggerCommand(uniqueTriggerID string) error {
	for _, runner := range w.applicationRunner {
		if runner.GetTriggerID() == uniqueTriggerID {
			return runner.OnTrigger()
		}
	}
	logger.Info(fmt.Sprintf("Command not found: %s", uniqueTriggerID))
	return fmt.Errorf("command not found")
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
