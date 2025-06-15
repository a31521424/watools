package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func initWindowSize(ctx context.Context) {
	screen, err := runtime.ScreenGetAll(ctx)
	if err != nil {
		println(err.Error())
	}
	width := 800
	height := 56
	if len(screen) > 0 {
		width = screen[0].Size.Width / 3
	}
	runtime.WindowSetSize(ctx, width, height)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	initWindowSize(ctx)
}
