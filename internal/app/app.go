package app

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey"
)

type WaApp struct {
	ctx      context.Context
	hk       *hotkey.Hotkey
	isHidden bool
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
	a.RegisterHotkeys(ctx)
}

func (a *WaApp) RegisterHotkeys(ctx context.Context) {
	if a.hk == nil {
		a.hk = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeySpace)
	} else {
		err := a.hk.Unregister()
		if err != nil {
			runtime.LogErrorf(ctx, "Failed to unregister hotkey: %s", err.Error())
			return
		}
	}

	err := a.hk.Register()
	if err != nil {
		return
	}
	go func() {
		for {
			select {
			case <-a.hk.Keydown():
				if a.isHidden {
					runtime.WindowShow(ctx)
					a.isHidden = false
				} else {
					runtime.WindowHide(ctx)
					a.isHidden = true
				}
			case <-a.ctx.Done():
				return
			}
		}
	}()
}
