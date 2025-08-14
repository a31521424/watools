package app

import (
	"testing"
)

func TestHotkeyConfig_ParseHotkey(t *testing.T) {
	tests := []struct {
		name        string
		hotkeyStr   string
		wantErr     bool
	}{
		{"Valid cmd+space", "cmd+space", false},
		{"Valid cmd+shift+r", "cmd+shift+r", false},
		{"Valid ctrl+alt+a", "ctrl+alt+a", false},
		{"Invalid no modifier", "a", true},
		{"Invalid no key", "cmd", true},
		{"Valid with aliases", "command+option+return", false},
		{"Case insensitive", "CMD+SPACE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &HotkeyConfig{
				Hotkey: tt.hotkeyStr,
			}
			_, _, err := c.ParseHotkey()
			if (err != nil) != tt.wantErr {
				t.Errorf("HotkeyConfig.ParseHotkey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}