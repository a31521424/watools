package app

import (
	"context"
	"sync"
	"watools/config"
	"watools/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

// HotkeyConfigAPI represents a hotkey configuration for API responses
type HotkeyConfigAPI struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
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

func (a *WaApp) Shutdown(ctx context.Context) {
	a.unregisterHotkeys()
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
	hm := GetHotkeyManager()
	if err := hm.LoadConfigs(); err != nil {
		logger.Error(err, "Failed to load hotkey configs")
		return
	}

	if err := hm.RegisterAll(); err != nil {
		logger.Error(err, "Failed to register hotkeys")
	}
}

func (a *WaApp) unregisterHotkeys() {
	hm := GetHotkeyManager()
	hm.UnregisterAll()
}

// GetHotkeys returns all hotkey configurations (Wails binding method)
func (a *WaApp) GetHotkeys() []HotkeyConfigAPI {
	hm := GetHotkeyManager()
	configs := hm.GetAllConfigs()

	// Convert to API format
	apiConfigs := make([]HotkeyConfigAPI, len(configs))
	for i, cfg := range configs {
		apiConfigs[i] = HotkeyConfigAPI{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Hotkey: cfg.Hotkey,
		}
	}

	return apiConfigs
}

// UpdateHotkey updates a hotkey configuration (Wails binding method)
func (a *WaApp) UpdateHotkey(apiConfig HotkeyConfigAPI) error {
	hm := GetHotkeyManager()

	// Create app layer config
	cfg := HotkeyConfig{
		ID:     apiConfig.ID,
		Name:   apiConfig.Name,
		Hotkey: apiConfig.Hotkey,
	}

	// Validate and set config
	if err := hm.SetConfig(cfg); err != nil {
		logger.Error(err, "Failed to set hotkey config", "id", cfg.ID)
		return err
	}

	// Save config
	if err := hm.SaveConfigs(); err != nil {
		logger.Error(err, "Failed to save hotkey configs")
		return err
	}

	// Re-register all hotkeys
	if err := hm.RegisterAll(); err != nil {
		logger.Error(err, "Failed to re-register hotkeys")
		return err
	}

	logger.Info("Hotkey updated successfully")
	return nil
}
