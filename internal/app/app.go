package app

import (
	"context"
	"sync"
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

func (a *WaApp) positionWindow() {
	windowWidth, windowHeight := runtime.WindowGetSize(a.ctx)
	if a.positionWindowOnScreen(windowWidth, windowHeight) {
		return
	}

	runtime.WindowCenter(a.ctx)

	screen, ok := a.getPrimaryScreen()
	if !ok {
		return
	}

	currentX, _ := runtime.WindowGetPosition(a.ctx)
	runtime.WindowSetPosition(a.ctx, currentX, clampWindowTopOffset(screen.Size.Height, windowHeight))
}

func clampWindowWidth(screenWidth int) int {
	if screenWidth <= 0 {
		return 1280
	}

	width := int(float64(screenWidth) * 0.42)
	if width < 920 {
		width = 920
	}
	if width > 1320 {
		width = 1320
	}

	maxAllowedWidth := screenWidth - 96
	if maxAllowedWidth < 840 {
		maxAllowedWidth = screenWidth
	}
	if width > maxAllowedWidth {
		width = maxAllowedWidth
	}
	if width < 840 {
		width = 840
	}

	return width
}

func clampWindowTopOffset(screenHeight int, windowHeight int) int {
	if screenHeight <= 0 {
		return 72
	}

	offset := int(float64(screenHeight) * 0.12)
	if offset < 72 {
		offset = 72
	}
	if offset > 132 {
		offset = 132
	}

	maxOffset := screenHeight - windowHeight - 32
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	return offset
}

func (a *WaApp) getPrimaryScreen() (runtime.Screen, bool) {
	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		logger.Error(err, "Failed to get screens")
		return runtime.Screen{}, false
	}
	if len(screens) == 0 {
		return runtime.Screen{}, false
	}

	primaryScreen := screens[0]
	for _, screen := range screens {
		if screen.IsPrimary {
			primaryScreen = screen
			break
		}
	}

	return primaryScreen, true
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
	width := 1280
	height := 64
	if primaryScreen, ok := a.getPrimaryScreen(); ok {
		a.lastScreenWidth = primaryScreen.Size.Width
		a.lastScreenHeight = primaryScreen.Size.Height
		width = clampWindowWidth(primaryScreen.Size.Width)
	}
	runtime.WindowSetSize(a.ctx, width, height)
	a.positionWindow()
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
	primaryScreen, ok := a.getPrimaryScreen()
	if !ok {
		return
	}

	// Check if screen dimensions changed
	if primaryScreen.Size.Width != a.lastScreenWidth || primaryScreen.Size.Height != a.lastScreenHeight {
		logger.Info("Screen configuration changed, repositioning window")

		// Update stored dimensions
		a.lastScreenWidth = primaryScreen.Size.Width
		a.lastScreenHeight = primaryScreen.Size.Height

		// Recalculate and set window size
		width := clampWindowWidth(primaryScreen.Size.Width)
		height := 64

		runtime.WindowSetSize(a.ctx, width, height)
		a.positionWindow()
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
