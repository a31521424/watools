package app

import (
	"fmt"
	"strings"
	"watools/pkg/logger"

	"golang.design/x/hotkey"
)

type HotkeyConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
}

func (c *HotkeyConfig) ParseHotkey() ([]hotkey.Modifier, hotkey.Key, error) {
	hotkeyParts := strings.Split(strings.ToLower(c.Hotkey), "+")
	var modifiers []hotkey.Modifier
	var pureKey *hotkey.Key

	// Normalize modifier key names
	for i, part := range hotkeyParts {
		switch part {
		case "cmd", "command", "⌘":
			hotkeyParts[i] = "cmd"
		case "ctrl", "control", "^":
			hotkeyParts[i] = "ctrl"
		case "alt", "option", "opt", "⌥":
			hotkeyParts[i] = "alt"
		case "shift", "⇧":
			hotkeyParts[i] = "shift"
		}
	}

	for _, part := range hotkeyParts {
		if mod, ok := modifierMap[part]; ok {
			modifiers = append(modifiers, mod)
		} else if key, ok := keyMap[part]; ok && pureKey == nil {
			pureKey = &key
		}
	}
	if len(modifiers) == 0 || pureKey == nil {
		return nil, 0, fmt.Errorf("invalid hotkey: %s", c.Hotkey)
	}
	return modifiers, *pureKey, nil
}

type HotkeyListener struct {
	OnTrigger func()
	Modifiers []hotkey.Modifier
	Key       hotkey.Key
	ID        string
	hk        *hotkey.Hotkey
	quit      chan struct{}
}

func (l *HotkeyListener) listen() {
	// Ensure channel is initialized
	if l.quit == nil {
		l.quit = make(chan struct{}, 1)
	}

	for {
		// Check if registered
		if !l.IsRegistered() {
			logger.Info(fmt.Sprintf("Hotkey listener unregistered, id: %s", l.ID))
			return
		}

		select {
		case <-l.quit:
			logger.Info(fmt.Sprintf("Hotkey listener stopped, id: %s", l.ID))
			return
		case <-l.hk.Keydown():
			logger.Info(fmt.Sprintf("Hotkey pressed, id: %s", l.ID))
			if l.OnTrigger != nil {
				l.OnTrigger()
			} else {
				logger.Info(fmt.Sprintf("Hotkey trigger function is nil, id: %s", l.ID))
			}
		}
	}
}

func (l *HotkeyListener) Register() error {
	// If already registered, unregister first
	if l.IsRegistered() {
		if err := l.Unregister(); err != nil {
			logger.Info(fmt.Sprintf("Failed to unregister existing hotkey, id: %s, error: %v", l.ID, err))
		}
	}

	l.quit = make(chan struct{}, 1)
	l.hk = hotkey.New(l.Modifiers, l.Key)

	err := l.hk.Register()
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to register hotkey, id: %s, error: %v", l.ID, err))
		// Clean up resources
		l.hk = nil
		l.quit = nil
		return err
	}

	go l.listen()
	logger.Info(fmt.Sprintf("Hotkey registered successfully, id: %s", l.ID))
	return nil
}

func (l *HotkeyListener) Unregister() error {
	// Check if registered
	if !l.IsRegistered() {
		return nil // Already unregistered or never registered
	}

	err := l.hk.Unregister()
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to unregister hotkey, id: %s, error: %v", l.ID, err))
		// Clean up resources even if error occurs
	}

	// Send stop signal
	if l.quit != nil {
		select {
		case l.quit <- struct{}{}:
		default:
			// Channel is full, no need to handle
		}
	}

	// Clean up resources
	l.hk = nil
	l.quit = nil

	logger.Info(fmt.Sprintf("Hotkey unregistered successfully, id: %s", l.ID))
	return err
}

func (l *HotkeyListener) IsRegistered() bool {
	return l.hk != nil
}
