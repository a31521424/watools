package app

import (
	"errors"
	"fmt"
	"watools/pkg/logger"
)

// HotkeyUpdateError reports an update that required recovery. A nil RecoveryErr
// means both the old configuration and its actual registration state were restored.
type HotkeyUpdateError struct {
	UpdateErr   error
	RecoveryErr error
}

func (e *HotkeyUpdateError) Error() string {
	if e.RecoveryErr == nil {
		return fmt.Sprintf("hotkey update failed; previous state restored: %v", e.UpdateErr)
	}
	return fmt.Sprintf("hotkey update failed and recovery was incomplete: %v", errors.Join(e.UpdateErr, e.RecoveryErr))
}

func (e *HotkeyUpdateError) Unwrap() error {
	return errors.Join(e.UpdateErr, e.RecoveryErr)
}

// The update owns sequencing; these boundaries allow configuration and native
// failures to be tested without a desktop application or real registrations.
type hotkeyUpdateManager interface {
	GetHotkeyStates() []HotkeyConfigAPI
	SetConfig(HotkeyConfig) error
	RegisterAll() error
	registerConfigs([]HotkeyConfig) error
	SaveConfigs() error
}

// Caller holds hotkeyUpdateMu after entering the platform lifecycle dispatcher.
func (a *WaApp) updateHotkeyWithManager(apiConfig HotkeyConfigAPI, hm hotkeyUpdateManager) error {
	cfg := HotkeyConfig{ID: apiConfig.ID, Name: apiConfig.Name, Hotkey: apiConfig.Hotkey}
	var old HotkeyConfig
	var exists bool
	var registered []HotkeyConfig
	for _, state := range hm.GetHotkeyStates() {
		if state.ID == cfg.ID {
			old = HotkeyConfig{ID: state.ID, Name: state.Name, Hotkey: state.Hotkey}
			exists = true
		}
		if state.Registered {
			registered = append(registered, HotkeyConfig{ID: state.ID, Name: state.Name, Hotkey: state.RegisteredHotkey})
		}
	}
	if !exists {
		return fmt.Errorf("unknown hotkey ID: %s", cfg.ID)
	}
	if err := hm.SetConfig(cfg); err != nil {
		return err
	}
	rollback := func(updateErr error) error {
		var failures []error
		if err := hm.SetConfig(old); err != nil {
			failures = append(failures, fmt.Errorf("restore hotkey configuration: %w", err))
		}
		// Use the captured native state even when configuration recovery failed.
		if err := hm.registerConfigs(registered); err != nil {
			failures = append(failures, fmt.Errorf("restore hotkey registrations: %w", err))
		}
		return &HotkeyUpdateError{UpdateErr: updateErr, RecoveryErr: errors.Join(failures...)}
	}
	if err := hm.RegisterAll(); err != nil {
		return rollback(fmt.Errorf("register updated hotkeys: %w", err))
	}
	if err := hm.SaveConfigs(); err != nil {
		// SaveConfigs replaces the file only after all required writes and Close
		// succeed, so a failed save leaves the previous persistent configuration.
		return rollback(fmt.Errorf("save hotkey configuration: %w", err))
	}
	logger.Info("Hotkey updated successfully")
	return nil
}
