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
	ctx              context.Context
	isHidden         bool
	hotkeyListener   []*HotkeyListener
	lastScreenWidth  int
	lastScreenHeight int
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
	width := 1024
	height := 56
	if len(screen) > 0 {
		// Find primary screen and record its size
		primaryScreen := screen[0]
		for _, s := range screen {
			if s.IsPrimary {
				primaryScreen = s
				break
			}
		}
		a.lastScreenWidth = primaryScreen.Size.Width
		a.lastScreenHeight = primaryScreen.Size.Height
		width = primaryScreen.Size.Width / 3
	}
	runtime.WindowSetSize(a.ctx, width, height)
	if config.IsDevMode() {
		// for fronted debug
		runtime.WindowSetPosition(a.ctx, width, 0)
		return
	}
	runtime.WindowCenter(a.ctx)
}

func (a *WaApp) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.initWindowSize()
	a.registerHotkeys()
}

func (a *WaApp) Shutdown(ctx context.Context) {
	a.unregisterHotkeys()
}

// checkAndRepositionIfNeeded checks for screen changes and repositions window if needed
func (a *WaApp) checkAndRepositionIfNeeded() {
	screen, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		logger.Error(err, "Failed to get screen for reposition check")
		return
	}

	if len(screen) == 0 {
		return
	}

	// Find current primary screen
	primaryScreen := screen[0]
	for _, s := range screen {
		if s.IsPrimary {
			primaryScreen = s
			break
		}
	}

	// Check if screen dimensions changed
	if primaryScreen.Size.Width != a.lastScreenWidth || primaryScreen.Size.Height != a.lastScreenHeight {
		logger.Info("Screen configuration changed, repositioning window")

		// Update stored dimensions
		a.lastScreenWidth = primaryScreen.Width
		a.lastScreenHeight = primaryScreen.Height

		// Recalculate and set window size
		width := primaryScreen.Width / 3
		if width < 400 {
			width = 400
		}
		if width > 1024 {
			width = 1024
		}
		height := 56

		runtime.WindowSetSize(a.ctx, width, height)

		if config.IsDevMode() {
			runtime.WindowSetPosition(a.ctx, width, 0)
		} else {
			runtime.WindowCenter(a.ctx)
		}
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
