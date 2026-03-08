package app

func defaultHotkeyConfigs() map[string]HotkeyConfig {
	return map[string]HotkeyConfig{
		"show-hide-window": {
			ID:     "show-hide-window",
			Name:   "Show/Hide Window",
			Hotkey: "ctrl+Space",
		},
	}
}
