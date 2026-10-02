package app

import (
	"testing"
)

func TestHotkeyRegistrationFailureIsReturned(t *testing.T) {
	manager := &HotkeyManager{listeners: map[string]*HotkeyListener{}, configs: map[string]HotkeyConfig{"unknown": {ID: "unknown", Hotkey: "ctrl+a"}}}
	if err := manager.RegisterAll(); err == nil {
		t.Fatal("unknown shortcut reported as registered")
	}
	manager.configs = map[string]HotkeyConfig{"show-hide-window": {ID: "show-hide-window", Hotkey: "invalid"}}
	if err := manager.RegisterAll(); err == nil {
		t.Fatal("invalid shortcut reported as registered")
	}
}
func TestUnknownHotkeyCannotBeSaved(t *testing.T) {
	manager := &HotkeyManager{configs: map[string]HotkeyConfig{}}
	if err := manager.SetConfig(HotkeyConfig{ID: "unknown", Hotkey: "ctrl+a"}); err == nil {
		t.Fatal("unknown hotkey accepted")
	}
	if len(manager.configs) != 0 {
		t.Fatal("rejected config was persisted")
	}
}
