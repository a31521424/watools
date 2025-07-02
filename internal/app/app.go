package app

import (
	"context"
	"fmt"
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

func (a *WaApp) InitWindowSize() {
	screen, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		return
	}
	width := 800
	height := 56
	if len(screen) > 0 {
		width = screen[0].Size.Width / 3
	}
	runtime.WindowSetSize(a.ctx, width, height)
}

func (a *WaApp) Startup(ctx context.Context) {
	a.ctx = ctx
	a.InitWindowSize()
	a.RegisterHotkeys()
	a.ListenHotkeys()
}

func (a *WaApp) ListenHotkeys() {
	if a.hk == nil {
		runtime.LogErrorf(a.ctx, "Hotkey is not registered")
		return
	}
	go func() {
		for {
			select {
			case <-a.hk.Keydown():
				fmt.Println("Global Hotkey pressed")
				a.HideOrShow()
			case <-a.ctx.Done():
				return
			}
		}
	}()
}

func (a *WaApp) HideOrShow() {
	if a.isHidden {
		runtime.WindowShow(a.ctx)
		a.isHidden = false
	} else {
		runtime.WindowHide(a.ctx)
		a.isHidden = true
	}
}

func (a *WaApp) RegisterHotkeys() {
	if a.hk == nil {
		a.hk = hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeySpace)
	} else {
		err := a.hk.Unregister()
		if err != nil {
			runtime.LogErrorf(a.ctx, "Failed to unregister hotkey: %s", err.Error())
			return
		}
	}

	err := a.hk.Register()
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to register hotkey: %s", err.Error())
		return
	}
}
