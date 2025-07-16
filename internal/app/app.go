package app

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"sync"
	"watools/config"
	"watools/pkg/logger"
)

var (
	waAppInstance *WaApp
	waAppOnce     sync.Once
)

type WaApp struct {
	ctx            context.Context
	isHidden       bool
	hotkeyListener []*HotkeyListener
}

func GetWaApp() *WaApp {
	waAppOnce.Do(func() {
		waAppInstance = &WaApp{}
	})
	return waAppInstance
}

func (a *WaApp) initWindowSize() {
	screen, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		logger.Error(err, "Failed to get screen when init window size")
		return
	}
	width := 800
	height := 56
	if len(screen) > 0 {
		width = screen[0].Size.Width / 3
	}
	runtime.WindowSetSize(a.ctx, width, height)
	if config.IsDevMode() {
		// for fronted debug
		runtime.WindowSetPosition(a.ctx, width, 0)
		return
	}
	runtime.WindowCenter(a.ctx)
}

func (a *WaApp) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initWindowSize()
	a.registerHotkeys()
}

func (a *WaApp) HideApp() {
	if !a.isHidden {
		runtime.WindowHide(a.ctx)
		a.isHidden = true
	}
}

func (a *WaApp) ShowApp() {
	if a.isHidden {
		runtime.WindowShow(a.ctx)
		a.isHidden = false
	}
}

func (a *WaApp) HideOrShowApp() {
	if a.isHidden {
		a.ShowApp()
	} else {
		a.HideApp()
	}
}

func (a *WaApp) Reload() {
	runtime.WindowReload(a.ctx)
}

func (a *WaApp) ReloadAPP() {
	runtime.WindowReloadApp(a.ctx)
}

func (a *WaApp) registerHotkeys() {
	if a.hotkeyListener != nil {
		return
	}
	a.hotkeyListener = GetHotkeyListeners()
	for _, l := range a.hotkeyListener {
		err := l.Register()
		if err != nil {
			logger.Error(err, "Failed to register hotkey")
		}
	}
}

func (a *WaApp) unregisterHotkeys() {
	if a.hotkeyListener == nil {
		return
	}
	for _, l := range a.hotkeyListener {
		err := l.Unregister()
		if err != nil {
			logger.Error(err, "Failed to unregister hotkey")
		}
	}
}
