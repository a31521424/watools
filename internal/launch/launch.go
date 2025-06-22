package launch

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
		runtime.LogErrorf(w.ctx, fmt.Sprintf("Failed to run application: %s", err.Error()))
	}
}
