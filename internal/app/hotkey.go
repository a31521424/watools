package app

import (
	"fmt"
	"strings"
	"sync"

	"golang.design/x/hotkey"
)

type HotkeyConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
}

func (c *HotkeyConfig) ParseHotkey() ([]hotkey.Modifier, hotkey.Key, error) {
	modifiersKeyMap := make(map[string]*struct{})
	var modifiers []hotkey.Modifier
	var pureKey *hotkey.Key

	// Normalize modifier key names
	for _, part := range strings.Split(strings.ToLower(c.Hotkey), "+") {
		switch part {
		case "cmd", "command", "meta", "super", "⌘":
			modifiersKeyMap["cmd"] = &struct{}{}
		case "win":
			modifiersKeyMap["win"] = &struct{}{}
		case "ctrl", "control", "^":
			modifiersKeyMap["ctrl"] = &struct{}{}
		case "alt", "option", "opt", "⌥":
			modifiersKeyMap["alt"] = &struct{}{}
		case "shift", "⇧":
			modifiersKeyMap["shift"] = &struct{}{}
		default:
			part = strings.ToLower(part)
			if key, ok := keyMap[part]; ok {
				pureKey = &key
			}
		}
	}
	for part := range modifiersKeyMap {
		if mod, ok := modifierMap[part]; ok {
			modifiers = append(modifiers, mod)
		}
	}
	if len(modifiers) == 0 || pureKey == nil {
		return nil, 0, fmt.Errorf("invalid hotkey: %s", c.Hotkey)
	}
	return modifiers, *pureKey, nil
}

type registeredHotkey interface {
	Register() error
	Unregister() error
	Keydown() <-chan hotkey.Event
}

type HotkeyListener struct {
	OnTrigger func()
	Modifiers []hotkey.Modifier
	Key       hotkey.Key
	ID        string
	mu        sync.Mutex
	hk        registeredHotkey
	quit      chan struct{}
}

// Each listener captures its own channels. Unregister cannot replace the state
// read by a goroutine belonging to an earlier registration.
func (l *HotkeyListener) listen(hk registeredHotkey, quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			return
		case _, ok := <-hk.Keydown():
			if !ok {
				return
			}
			select {
			case <-quit:
				return
			default:
			}
			if l.OnTrigger != nil {
				l.OnTrigger()
			}
		}
	}
}
func (l *HotkeyListener) Register() error {
	return runHotkeyLifecycle(l.register)
}

func (l *HotkeyListener) register() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hk != nil {
		return fmt.Errorf("hotkey is already registered")
	}
	hk := newPlatformHotkey(l.Modifiers, l.Key)
	if err := hk.Register(); err != nil {
		return err
	}
	quit := make(chan struct{})
	l.hk = hk
	l.quit = quit
	go l.listen(hk, quit)
	return nil
}
func (l *HotkeyListener) Unregister() error {
	return runHotkeyLifecycle(l.unregister)
}

func (l *HotkeyListener) unregister() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hk == nil {
		return nil
	}
	if err := l.hk.Unregister(); err != nil {
		return err
	}
	close(l.quit)
	l.hk = nil
	l.quit = nil
	return nil
}
func (l *HotkeyListener) IsRegistered() bool { l.mu.Lock(); defer l.mu.Unlock(); return l.hk != nil }
