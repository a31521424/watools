package launch

import (
	"context"
	"fmt"
	"sync"
	"time"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"
)

type WaLaunchApp struct {
	ctx     context.Context
	scanner AppScanner
}

func NewWaLaunchApp() *WaLaunchApp {
	return &WaLaunchApp{}
}

func (w *WaLaunchApp) Startup(ctx context.Context) {
	w.ctx = ctx
	w.scanner = NewAppScanner()
	w.initCommandsUpdater()
}

func (w *WaLaunchApp) initCommandsUpdater() {
	go func() {
		dbInstance := db.GetWaDB()
		for {

			commands := dbInstance.FindExpiredCommands(w.ctx)
			logger.Info(fmt.Sprintf("Found %d expired commands", len(commands)))
			if len(commands) > 0 {
				var updateCommands []*models.Command
				for _, command := range commands {
					id := command.ID
					command, err := w.scanner.ParseApplication(command.Path)
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

func (w *WaLaunchApp) GetApplications() []*models.Command {
	ApiMutex.Lock()
	defer ApiMutex.Unlock()
	dbInstance := db.GetWaDB()
	commands := dbInstance.GetCommands(w.ctx)
	if len(commands) == 0 {
		commands, err := w.scanner.GetApplications()
		if err != nil {
			logger.Error(err, "Failed to get application")
			return []*models.Command{}
		}
		err = dbInstance.BatchInsertCommands(w.ctx, commands)
		if err != nil {
			logger.Error(err, "Failed to batch insert commands")
		}
	}
	for _, command := range commands {
		if command.IconPath == "" {
			command.IconPath = w.scanner.GetDefaultIconPath()
		}
	}
	return commands
}

func (w *WaLaunchApp) RunApplication(path string) {
	err := w.scanner.RunApplication(path)
	if err != nil {
		logger.Error(err, "Failed to run application")
	}
}
