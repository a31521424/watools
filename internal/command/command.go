package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"watools/internal/command/application"
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
	ctx     context.Context
	runners []models.CommandRunner
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
}

func (w *WaLaunchApp) Shutdown(ctx context.Context) {}

func (w *WaLaunchApp) initCommandsUpdater() {
	go func() {
		dbInstance := db.GetWaDB()
		for {

			commands := dbInstance.FindExpiredCommands(w.ctx)
			logger.Info(fmt.Sprintf("Found %d expired commands", len(commands)))
			if len(commands) > 0 {
				var updateCommands []*models.ApplicationCommand
				for _, command := range commands {
					id := command.ID
					command, err := application.Scanner.ParseApplication(command.Path)
					if err != nil {
						logger.Error(err, "Failed to parse application")
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
			time.Sleep(time.Minute * 5)
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
		commands, err := application.Scanner.GetApplications()
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
			command.IconPath = application.Scanner.GetDefaultIconPath()
		}
	}
	return commands
}

func (w *WaLaunchApp) GetAllCommands() []interface{} {
	var commands []interface{}
	w.runners = nil

	apps := w.getApplicationCommands()
	w.runners = append(w.runners, generics.Map(apps, func(app *models.ApplicationCommand) models.CommandRunner { return app })...)
	commands = append(commands, generics.Map(apps, func(app *models.ApplicationCommand) interface{} {
		var m map[string]interface{}
		data, _ := json.Marshal(app)
		_ = json.Unmarshal(data, &m)
		return m
	})...)

	return commands
}

func (w *WaLaunchApp) TriggerCommand(uniqueTriggerID string) error {
	for _, runner := range w.runners {
		if runner.GetTriggerID() == uniqueTriggerID {
			return runner.OnTrigger()
		}
	}
	logger.Info(fmt.Sprintf("Command not found: %s", uniqueTriggerID))
	return fmt.Errorf("command not found")

}
