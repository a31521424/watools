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
func (platformApplicationSource) PathsWithError() ([]application.AppPathInfo, error) {
	return application.GetAppPathInfosWithError()
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
		if changed && w.ctx.Err() == nil && w.notify != nil {
			w.notify()
		}
	}()
	seen := make(map[string]struct{})
	var failures []error
	var updateCommands []*models.ApplicationCommand
	var insertCommands []*models.ApplicationCommand
	var removeCommands []*models.ApplicationCommand
	removeParseErrors := make(map[string]error)
	for _, command := range commands {
		if err := w.ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		seen[application.PathKey(command.Path)] = struct{}{}
		fi, err := os.Stat(command.Path)
		if err != nil {
			if os.IsNotExist(err) {
				removeCommands = append(removeCommands, command)
			} else {
				failures = append(failures, fmt.Errorf("stat application %q: %w", command.Path, errors.Join(application.ErrApplicationUnreadable, err)))
			}
			continue
		}
		if fi.ModTime().Equal(command.DirUpdatedAt) {
			continue
		}
		parsedCommand, err := w.source.Parse(command.Path)
		if cancelErr := w.ctx.Err(); cancelErr != nil {
			return errors.Join(append(failures, cancelErr)...)
		}
		if err != nil {
			if errors.Is(err, application.ErrApplicationNotFound) {
				removeCommands = append(removeCommands, command)
				removeParseErrors[command.Path] = err
			} else {
				failures = append(failures, fmt.Errorf("refresh application %q: %w", command.Path, err))
			}
			continue
		}
		if parsedCommand == nil {
			failures = append(failures, fmt.Errorf("refresh application %q: %w", command.Path, application.ErrInvalidMetadata))
			continue
		}
		parsedCommand = watcher.PreserveApplicationIdentity(parsedCommand, command)
		updateCommands = append(updateCommands, parsedCommand)
	}
	if len(updateCommands) > 0 {
		if err := w.ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := dbInstance.BatchUpdateCommands(w.ctx, updateCommands); err != nil {
			return errors.Join(append(failures, fmt.Errorf("persist updated application commands: %w", err))...)
		}
		changed = true
		logger.Info(fmt.Sprintf("Updated %d application commands", len(updateCommands)))
	}
	if len(removeCommands) > 0 {
		// Filesystem changes do not take ApiMutex. Recheck the roots after
		// parsing the other candidates, just before the deletion batch.
		confirmedMissing := make([]*models.ApplicationCommand, 0, len(removeCommands))
		for _, command := range removeCommands {
			if _, err := os.Stat(command.Path); os.IsNotExist(err) {
				confirmedMissing = append(confirmedMissing, command)
			} else if err != nil {
				failures = append(failures, fmt.Errorf("confirm application removal %q: %w", command.Path, errors.Join(application.ErrApplicationUnreadable, err)))
			} else if err := removeParseErrors[command.Path]; err != nil {
				failures = append(failures, fmt.Errorf("application reappeared during refresh %q: %w", command.Path, err))
			}
		}
		removeCommands = confirmedMissing
	}
	if len(removeCommands) > 0 {
		if err := w.ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := dbInstance.DeleteCommands(w.ctx, lo.Map(removeCommands, func(command *models.ApplicationCommand, _ int) string {
			return command.ID
		})); err != nil {
			return errors.Join(append(failures, fmt.Errorf("delete missing application commands: %w", err))...)
		}
		changed = true
		logger.Info(fmt.Sprintf("Removed %d application commands", len(removeCommands)))
	}

	logger.Info("Checking commands from disk")
	if err := w.ctx.Err(); err != nil {
		return errors.Join(append(failures, err)...)
	}
	appPathInfos, pathErr := w.applicationPaths()
	if pathErr != nil {
		failures = append(failures, fmt.Errorf("enumerate applications: %w", pathErr))
	}
	for _, appPathInfo := range appPathInfos {
		if err := w.ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		key := application.PathKey(appPathInfo.Path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		command, err := w.source.Parse(appPathInfo.Path)
		if cancelErr := w.ctx.Err(); cancelErr != nil {
			return errors.Join(append(failures, cancelErr)...)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("discover application %q: %w", appPathInfo.Path, err))
			continue
		}
		if command == nil {
			failures = append(failures, fmt.Errorf("discover application %q: %w", appPathInfo.Path, application.ErrInvalidMetadata))
			continue
		}
		insertCommands = append(insertCommands, command)
	}
	if len(insertCommands) > 0 {
		if err := w.ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := dbInstance.BatchInsertCommands(w.ctx, insertCommands); err != nil {
			return errors.Join(append(failures, fmt.Errorf("persist new application commands: %w", err))...)
		}
		changed = true
		logger.Info(fmt.Sprintf("Added %d application commands", len(insertCommands)))
	}
	return errors.Join(append(failures, w.ctx.Err())...)
}

// applicationPaths supports sources that report partial enumeration failures.
func (w *WaLaunchApp) applicationPaths() ([]application.AppPathInfo, error) {
	if source, ok := w.source.(interface {
		PathsWithError() ([]application.AppPathInfo, error)
	}); ok {
		return source.PathsWithError()
	}
	return w.source.Paths(), nil
}

// OnAppPathChanged resolves the current disk state under the same ownership as
// scans, queries and usage writes. Event-time metadata is deliberately discarded.
func (w *WaLaunchApp) OnAppPathChanged(path string) error {
	return w.ReconcileApplicationPath(w.ctx, path)
}

// ReconcileApplicationPath also honors the watcher task context, which is
// canceled by Stop even if the service context remains live.
func (w *WaLaunchApp) ReconcileApplicationPath(ctx context.Context, path string) error {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	if err := w.ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(w.ctx, cancel)
	defer stop()
	defer cancel()
	changed, err := watcher.ReconcileApplicationPath(ctx, w.ctx, w.database(), w.source.Parse, path)
	if changed && ctx.Err() == nil && w.ctx.Err() == nil && w.notify != nil {
		w.notify()
	}
	return err
}

// ApiMutex owns application read/parse/write sequences and runner publication.
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
	eventHandler := w

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
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	if err := w.ctx.Err(); err != nil {
		return err
	}
	return w.database().BatchUpdateApplicationUsage(w.ctx, usageUpdates)
}

func (w *WaLaunchApp) OnAppAdded(command *models.ApplicationCommand) error {
	return w.OnAppPathChanged(command.Path)
}
func (w *WaLaunchApp) OnAppModified(command *models.ApplicationCommand) error {
	return w.OnAppPathChanged(command.Path)
}
func (w *WaLaunchApp) OnAppRemoved(path string) error {
	return w.OnAppPathChanged(path)
}
