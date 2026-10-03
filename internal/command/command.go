package command

import (
	"context"
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
	"github.com/samber/mo"
)

type WaLaunchApp struct {
	notify            func()
	ctx               context.Context
	runnersMu         sync.RWMutex
	reconcileMu       sync.Mutex
	background        sync.WaitGroup
	cancel            context.CancelFunc
	applicationRunner []models.CommandRunner
	operationRunner   []models.CommandRunner
	watchManager      watcher.AppWatchManager
}

func NewWaLaunch() *WaLaunchApp { return &WaLaunchApp{} }

func (w *WaLaunchApp) OnStartup(ctx context.Context, notify func()) {
	w.notify = notify
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.initAppWatcher()
	w.asyncUpdateApplications(30 * time.Second)
}

func (w *WaLaunchApp) Shutdown(ctx context.Context) {
	if w.cancel != nil {
		w.cancel()
	}
	if w.watchManager != nil {
		if err := w.watchManager.Stop(); err != nil {
			logger.Error(err, "Failed to stop app watch manager")
		}
	}
	w.background.Wait()
}

func (w *WaLaunchApp) asyncUpdateApplications(delay time.Duration) {
	w.background.Add(1)
	go func() {
		defer w.background.Done()
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
		}
		w.updateApplications()
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.updateApplications()
			}
		}
	}()
}

func (w *WaLaunchApp) updateApplications() {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	w.reconcileMu.Lock()
	defer w.reconcileMu.Unlock()
	if w.ctx.Err() != nil {
		return
	}
	dbInstance := db.GetWaDB()
	commands := dbInstance.GetCommands(w.ctx)
	seen := make(map[string]struct{})
	var updateCommands []*models.ApplicationCommand
	var insertCommands []*models.ApplicationCommand
	var removeCommands []*models.ApplicationCommand
	for _, command := range commands {
		seen[command.Path] = struct{}{}
		id := command.ID
		fi, err := os.Stat(command.Path)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Error(err, fmt.Sprintf("Command is not exists %s", command.Path))
				removeCommands = append(removeCommands, command)
			} else {
				logger.Error(err, fmt.Sprintf("Failed to stat command %s", command.Path))
			}
			continue
		}
		if fi.ModTime().Format(time.DateTime) == command.DirUpdatedAt.Format(time.DateTime) {
			continue
		}
		logger.Info(fmt.Sprintf("Update dir updated for command: %s, %s", command.Name, command.Path))
		parsedCommand, err := application.ParseApplication(command.Path)
		if err != nil {
			logger.Error(err, "Failed to parse application")
			removeCommands = append(removeCommands, command)
			continue
		}
		parsedCommand.ID = id
		updateCommands = append(updateCommands, parsedCommand)
	}
	logger.Info(fmt.Sprintf("Update commands result: updated %d / total %d", len(updateCommands), len(commands)))
	err := dbInstance.BatchUpdateCommands(w.ctx, updateCommands)
	if err != nil {
		logger.Error(err, "Failed to batch update updated commands to db")
	}
	logger.Info(fmt.Sprintf("Delete commands result: removed %d / total %d", len(removeCommands), len(commands)))
	err = dbInstance.DeleteCommands(w.ctx, lo.Map(removeCommands, func(command *models.ApplicationCommand, _ int) string {
		return command.ID
	}))
	if err != nil {
		logger.Error(err, "Failed to delete commands")
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
	if len(insertCommands)+len(updateCommands)+len(removeCommands) > 0 {
		if w.notify != nil {
			w.notify()
		}
	}
}

var ApiMutex sync.Mutex

func (w *WaLaunchApp) getApplicationCommands() []*models.ApplicationCommand {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	dbInstance := db.GetWaDB()
	commands := dbInstance.GetCommands(w.ctx)
	if len(commands) == 0 {
		scanned, err := application.GetApplications()
		commands = scanned
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
		if command.IconPath.IsNone() {
			command.IconPath = mo.Some(application.GetDefaultIconPath())
		}
	}
	return commands
}

func (w *WaLaunchApp) GetApplicationCommands() []*models.ApplicationCommand {
	var runners []models.CommandRunner

	apps := lo.UniqBy(w.getApplicationCommands(), func(app *models.ApplicationCommand) string {
		if app.ID != "" {
			return app.ID
		}
		return app.Path
	})
	runners = append(runners, lo.Map(apps, func(app *models.ApplicationCommand, _ int) models.CommandRunner { return app })...)

	w.runnersMu.Lock()
	w.applicationRunner = runners
	w.runnersMu.Unlock()
	return apps
}

func (w *WaLaunchApp) GetOperationCommands() []*models.OperationCommand {
	var runners []models.CommandRunner

	operations := operator.GetOperations()
	uniqueOperations := lo.UniqBy(operations, func(operation *models.OperationCommand) string {
		return operation.TriggerID
	})
	runners = append(runners, lo.Map(uniqueOperations, func(operation *models.OperationCommand, _ int) models.CommandRunner { return operation })...)

	w.runnersMu.Lock()
	w.operationRunner = runners
	w.runnersMu.Unlock()
	return uniqueOperations
}

func (w *WaLaunchApp) TriggerCommand(uniqueTriggerID string, triggerCategory models.CommandCategory) error {
	w.runnersMu.RLock()
	var runners []models.CommandRunner
	if triggerCategory == models.CategoryApplication {
		runners = w.applicationRunner
	} else if triggerCategory == models.CategoryOperation {
		runners = w.operationRunner
	} else {
		w.runnersMu.RUnlock()
		return fmt.Errorf("trigger category is not valid: %s", triggerCategory)
	}

	runners = append([]models.CommandRunner(nil), runners...)
	w.runnersMu.RUnlock()
	for _, runner := range runners {
		if runner.GetTriggerID() == uniqueTriggerID {
			err := runner.OnTrigger()
			if err != nil {
				logger.Error(err, fmt.Sprintf("cant trigger runner: %s", uniqueTriggerID))
			} else {
				logger.Info(fmt.Sprintf("trigger runner success: %s", uniqueTriggerID))
			}
			return err
		}
	}

	return fmt.Errorf("not find runner: %s", uniqueTriggerID)
}

func (w *WaLaunchApp) initAppWatcher() {
	eventHandler := watcher.NewDefaultAppEventHandler(w.ctx, w.notify)

	watchManager, err := watcher.NewAppWatchManager(eventHandler, w.ctx)
	if err != nil {
		logger.Error(err, "Failed to create app watch manager")
		return
	}

	if watchManager == nil {
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

func (w *WaLaunchApp) UpdateApplicationUsage(usageUpdates []models.ApplicationUsageUpdate) error {
	dbInstance := db.GetWaDB()
	return dbInstance.BatchUpdateApplicationUsage(w.ctx, usageUpdates)
}
