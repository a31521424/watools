package command

import (
	"context"
	"errors"
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
	repository        ApplicationRepository
	source            ApplicationSource
}

// ApplicationRepository is the persistence boundary for application commands.
type ApplicationRepository interface {
	watcher.ApplicationRepository
	BatchUpdateApplicationUsage(context.Context, []models.ApplicationUsageUpdate) error
}

// ApplicationSource supplies the platform application scan and parsing boundary.
type ApplicationSource interface {
	Scan() ([]*models.ApplicationCommand, error)
	Paths() []application.AppPathInfo
	Parse(string) (*models.ApplicationCommand, error)
}

type platformApplicationSource struct{}

func (platformApplicationSource) Scan() ([]*models.ApplicationCommand, error) {
	return application.GetApplications()
}
func (platformApplicationSource) Paths() []application.AppPathInfo {
	return application.GetAppPathInfos()
}
func (platformApplicationSource) Parse(path string) (*models.ApplicationCommand, error) {
	return application.ParseApplication(path)
}

func NewWaLaunch() *WaLaunchApp { return NewWaLaunchWithDependencies(nil, nil) }

// NewWaLaunchWithDependencies supports isolated command consumers without
// starting native watchers. Nil dependencies retain the platform defaults.
func NewWaLaunchWithDependencies(repository ApplicationRepository, source ApplicationSource) *WaLaunchApp {
	if source == nil {
		source = platformApplicationSource{}
	}
	return &WaLaunchApp{ctx: context.Background(), repository: repository, source: source}
}

func (w *WaLaunchApp) database() ApplicationRepository {
	if w.repository != nil {
		return w.repository
	}
	return db.GetWaDB()
}

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
		w.refreshApplications()
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.refreshApplications()
			}
		}
	}()
}

func (w *WaLaunchApp) refreshApplications() {
	if err := w.updateApplications(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error(err, "Failed to refresh applications; keeping the last available runners")
	}
}

func (w *WaLaunchApp) updateApplications() error {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	w.reconcileMu.Lock()
	defer w.reconcileMu.Unlock()
	if w.ctx.Err() != nil {
		return w.ctx.Err()
	}
	dbInstance := w.database()
	commands, err := dbInstance.GetCommands(w.ctx)
	if err != nil {
		return fmt.Errorf("read application commands for refresh: %w", err)
	}
	changed := false
	defer func() {
		if changed && w.notify != nil {
			w.notify()
		}
	}()
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
		parsedCommand, err := w.source.Parse(command.Path)
		if err != nil {
			logger.Error(err, "Failed to parse application")
			removeCommands = append(removeCommands, command)
			continue
		}
		parsedCommand.ID = id
		updateCommands = append(updateCommands, parsedCommand)
	}
	if len(updateCommands) > 0 {
		if err := dbInstance.BatchUpdateCommands(w.ctx, updateCommands); err != nil {
			return fmt.Errorf("persist updated application commands: %w", err)
		}
		changed = true
		logger.Info(fmt.Sprintf("Updated %d application commands", len(updateCommands)))
	}
	if len(removeCommands) > 0 {
		if err := dbInstance.DeleteCommands(w.ctx, lo.Map(removeCommands, func(command *models.ApplicationCommand, _ int) string {
			return command.ID
		})); err != nil {
			return fmt.Errorf("delete missing application commands: %w", err)
		}
		changed = true
		logger.Info(fmt.Sprintf("Removed %d application commands", len(removeCommands)))
	}

	logger.Info("Checking commands from disk")
	appPathInfos := w.source.Paths()
	for _, appPathInfo := range appPathInfos {
		if _, exists := seen[appPathInfo.Path]; exists {
			continue
		}
		logger.Info(fmt.Sprintf("Adding command from path: %s", appPathInfo.Path))
		command, err := w.source.Parse(appPathInfo.Path)
		if err != nil {
			logger.Error(err, "Failed to parse application")
			continue
		}
		insertCommands = append(insertCommands, command)
	}
	if len(insertCommands) > 0 {
		if err := dbInstance.BatchInsertCommands(w.ctx, insertCommands); err != nil {
			return fmt.Errorf("persist new application commands: %w", err)
		}
		changed = true
		logger.Info(fmt.Sprintf("Added %d application commands", len(insertCommands)))
	}
	return nil
}

var ApiMutex sync.Mutex

func (w *WaLaunchApp) getApplicationCommands() ([]*models.ApplicationCommand, error) {
	if err := w.ctx.Err(); err != nil {
		return nil, err
	}
	dbInstance := w.database()
	commands, err := dbInstance.GetCommands(w.ctx)
	if err != nil {
		return nil, fmt.Errorf("read application commands: %w", err)
	}
	if len(commands) == 0 {
		scanned, err := w.source.Scan()
		if err != nil {
			return nil, fmt.Errorf("scan application commands: %w", err)
		}
		if err := w.ctx.Err(); err != nil {
			return nil, err
		}
		if len(scanned) > 0 {
			if err := dbInstance.BatchInsertCommands(w.ctx, scanned); err != nil {
				return nil, fmt.Errorf("initialize application commands: %w", err)
			}
		}
		commands = scanned
	}
	for _, command := range commands {
		if command.IconPath.IsNone() {
			command.IconPath = mo.Some(application.GetDefaultIconPath())
		}
	}
	return commands, nil
}

func (w *WaLaunchApp) GetApplicationCommands() ([]*models.ApplicationCommand, error) {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	var runners []models.CommandRunner

	commands, err := w.getApplicationCommands()
	if err != nil {
		return nil, err
	}
	if err := w.ctx.Err(); err != nil {
		return nil, err
	}
	apps := lo.UniqBy(commands, func(app *models.ApplicationCommand) string {
		if app.ID != "" {
			return app.ID
		}
		return app.Path
	})
	runners = append(runners, lo.Map(apps, func(app *models.ApplicationCommand, _ int) models.CommandRunner { return app })...)

	w.runnersMu.Lock()
	w.applicationRunner = runners
	w.runnersMu.Unlock()
	return apps, nil
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
	eventHandler := watcher.NewDefaultAppEventHandlerWithRepository(w.ctx, w.notify, w.database())

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
	return w.database().BatchUpdateApplicationUsage(w.ctx, usageUpdates)
}
