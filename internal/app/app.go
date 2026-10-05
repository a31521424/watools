package app

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"watools/pkg/logger"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	waAppInstance *WaApp
	waAppOnce     sync.Once
)

type WaApp struct {
	clipboard        clipboardSnapshots
	hotkeyUpdateMu   sync.Mutex
	desktop          *application.App
	window           *application.WebviewWindow
	isHidden         atomic.Bool
	hotkeyListener   []*HotkeyListener
	lastScreenWidth  int
	lastScreenHeight int
}

func (a *WaApp) positionWindow() {
	windowWidth, windowHeight := a.window.Size()
	if a.positionWindowOnScreen(windowWidth, windowHeight) {
		return
	}

	a.window.Center()

	screen, ok := a.getPrimaryScreen()
	if !ok {
		return
	}

	currentX, _ := a.window.Position()
	a.window.SetPosition(currentX, clampWindowTopOffset(screen.Size.Height, windowHeight))
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

func (a *WaApp) getPrimaryScreen() (application.Screen, bool) {
	screen := a.desktop.Screen.GetPrimary()
	if screen == nil {
		return application.Screen{}, false
	}
	return *screen, true
}

// HotkeyConfigAPI represents a hotkey configuration for API responses
type HotkeyConfigAPI struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
}

type HotkeyEnvironmentStatus struct {
	SecureEventInputEnabled bool   `json:"secureEventInputEnabled"`
	Note                    string `json:"note"`
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
	a.window.SetSize(width, height)
	a.positionWindow()
}

func (a *WaApp) ConfigureDesktop(desktop *application.App, window *application.WebviewWindow) {
	a.desktop, a.window = desktop, window
}

// OnWindowReady runs once after the native window exists.
func (a *WaApp) OnWindowReady() {
	a.initWindowSize()
	a.registerHotkeys()
}

func (a *WaApp) Shutdown(ctx context.Context) {
	a.clipboard.close()
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

		a.window.SetSize(width, height)
		a.positionWindow()
	}
}
func (a *WaApp) Reload() {
	a.window.Reload()
}

func (a *WaApp) ReloadAPP() {
	a.window.ForceReload()
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

	if a.IsSecureEventInputEnabled() {
		logger.Warning("macOS Secure Event Input is enabled; global hotkeys may be temporarily unavailable while another app owns a password field")
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
	return runHotkeyLifecycle(func() error { return a.updateHotkey(apiConfig) })
}

func (a *WaApp) updateHotkey(apiConfig HotkeyConfigAPI) error {
	a.hotkeyUpdateMu.Lock()
	defer a.hotkeyUpdateMu.Unlock()
	hm := GetHotkeyManager()

	// Create app layer config
	cfg := HotkeyConfig{
		ID:     apiConfig.ID,
		Name:   apiConfig.Name,
		Hotkey: apiConfig.Hotkey,
	}

	old, exists := hm.GetConfig(cfg.ID)
	if !exists {
		return fmt.Errorf("unknown hotkey ID: %s", cfg.ID)
	}
	if err := hm.SetConfig(cfg); err != nil {
		return err
	}
	rollback := func() {
		_ = hm.SetConfig(old)
		if err := hm.RegisterAll(); err != nil {
			logger.Error(err, "Failed to restore hotkeys")
		}
	}
	if err := hm.RegisterAll(); err != nil {
		rollback()
		return err
	}
	if err := hm.SaveConfigs(); err != nil {
		rollback()
		return err
	}

	logger.Info("Hotkey updated successfully")
	return nil
}

func (a *WaApp) GetHotkeyEnvironmentStatus() HotkeyEnvironmentStatus {
	status := HotkeyEnvironmentStatus{
		SecureEventInputEnabled: a.IsSecureEventInputEnabled(),
	}
	if status.SecureEventInputEnabled {
		status.Note = "Secure Event Input is enabled by another macOS app. Global hotkeys may not fire while a password field or other secure text input is focused."
	}
	return status
}
