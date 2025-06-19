package app

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WaApp struct {
	ctx context.Context
}

func NewWaApp() *WaApp {
	return &WaApp{}
}

func (a *WaApp) InitWindowSize(ctx context.Context) {
	screen, err := runtime.ScreenGetAll(ctx)
	if err != nil {
		return
	}
	width := 800
	height := 56
	if len(screen) > 0 {
		width = screen[0].Size.Width / 3
	}
	runtime.WindowSetSize(ctx, width, height)
}

func (a *WaApp) Startup(ctx context.Context) {
	a.ctx = ctx
	a.InitWindowSize(ctx)
}
