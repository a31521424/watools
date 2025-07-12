package launch

import (
	"context"
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
}

func (w *WaLaunchApp) GetApplication() []models.Command {
	commands, _ := w.scanner.GetApplication()
	return commands
}

func (w *WaLaunchApp) RunApplication(path string) {
	err := w.scanner.RunApplication(path)
	if err != nil {
		logger.Error(err, "Failed to run application")
	}
}
