package app

import (
	"golang.design/x/hotkey"
)

var modifierMap = map[string]hotkey.Modifier{
	"command": hotkey.ModCmd,
	"cmd":     hotkey.ModCmd,
	"⌘":       hotkey.ModCmd,
	"control": hotkey.ModCtrl,
	"ctrl":    hotkey.ModCtrl,
	"^":       hotkey.ModCtrl,
	"option":  hotkey.ModOption,
	"opt":     hotkey.ModOption,
	"alt":     hotkey.ModOption,
	"⌥":       hotkey.ModOption,
	"shift":   hotkey.ModShift,
	"⇧":       hotkey.ModShift,
}

var keyMap = map[string]hotkey.Key{
	"Space":  hotkey.KeySpace,
	"1":      hotkey.Key1,
	"2":      hotkey.Key2,
	"3":      hotkey.Key3,
	"4":      hotkey.Key4,
	"5":      hotkey.Key5,
	"6":      hotkey.Key6,
	"7":      hotkey.Key7,
	"8":      hotkey.Key8,
	"9":      hotkey.Key9,
	"0":      hotkey.Key0,
	"A":      hotkey.KeyA,
	"B":      hotkey.KeyB,
	"C":      hotkey.KeyC,
	"D":      hotkey.KeyD,
	"E":      hotkey.KeyE,
	"F":      hotkey.KeyF,
	"G":      hotkey.KeyG,
	"H":      hotkey.KeyH,
	"I":      hotkey.KeyI,
	"J":      hotkey.KeyJ,
	"K":      hotkey.KeyK,
	"L":      hotkey.KeyL,
	"M":      hotkey.KeyM,
	"N":      hotkey.KeyN,
	"O":      hotkey.KeyO,
	"P":      hotkey.KeyP,
	"Q":      hotkey.KeyQ,
	"R":      hotkey.KeyR,
	"S":      hotkey.KeyS,
	"T":      hotkey.KeyT,
	"U":      hotkey.KeyU,
	"V":      hotkey.KeyV,
	"W":      hotkey.KeyW,
	"X":      hotkey.KeyX,
	"Y":      hotkey.KeyY,
	"Z":      hotkey.KeyZ,
	"Return": hotkey.KeyReturn,
	"Escape": hotkey.KeyEscape,
	"Delete": hotkey.KeyDelete,
	"Tab":    hotkey.KeyTab,
	"Left":   hotkey.KeyLeft,
	"Right":  hotkey.KeyRight,
	"Up":     hotkey.KeyUp,
	"Down":   hotkey.KeyDown,
	"F1":     hotkey.KeyF1,
	"F2":     hotkey.KeyF2,
	"F3":     hotkey.KeyF3,
	"F4":     hotkey.KeyF4,
	"F5":     hotkey.KeyF5,
	"F6":     hotkey.KeyF6,
	"F7":     hotkey.KeyF7,
	"F8":     hotkey.KeyF8,
	"F9":     hotkey.KeyF9,
	"F10":    hotkey.KeyF10,
	"F11":    hotkey.KeyF11,
	"F12":    hotkey.KeyF12,
	"F13":    hotkey.KeyF13,
	"F14":    hotkey.KeyF14,
	"F15":    hotkey.KeyF15,
	"F16":    hotkey.KeyF16,
	"F17":    hotkey.KeyF17,
	"F18":    hotkey.KeyF18,
	"F19":    hotkey.KeyF19,
	"F20":    hotkey.KeyF20,
}

var defaultListener = []*HotkeyListener{
	&HotkeyListener{
		Modifiers: []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift},
		Key:       hotkey.KeySpace,
		OnTrigger: func() {
			GetWaApp().HideOrShowApp()
		},
		ID: "show-hide-window",
	},
	&HotkeyListener{
		Modifiers: []hotkey.Modifier{hotkey.ModCmd},
		Key:       hotkey.KeyR,
		OnTrigger: func() {
			GetWaApp().Reload()
		},
		ID: "reload",
	},
	&HotkeyListener{
		Modifiers: []hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift},
		Key:       hotkey.KeyR,
		OnTrigger: func() {
			GetWaApp().ReloadAPP()
		},
		ID: "reload-app",
	},
}
