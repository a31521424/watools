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
	hotkey    string
	newHotkey func([]hotkey.Modifier, hotkey.Key) registeredHotkey
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
	create := l.newHotkey
	if create == nil {
		create = newPlatformHotkey
	}
	hk := create(l.Modifiers, l.Key)
	err := hk.Register()
	if err != nil {
		resources := false
		if state, ok := hk.(interface{ HasResources() bool }); ok {
			resources = state.HasResources()
		} else if state, ok := hk.(interface{ IsRegistered() bool }); ok {
			resources = state.IsRegistered()
		}
		if !resources {
			return err
		}
	}
	quit := make(chan struct{})
	l.hk = hk
	l.quit = quit
	go l.listen(hk, quit)
	return err
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
func (l *HotkeyListener) IsRegistered() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hk == nil {
		return false
	}
	// A failed native cleanup can leave a handler after the key itself was
	// released. Keep ownership for retry, but report the actual key state.
	if state, ok := l.hk.(interface{ IsRegistered() bool }); ok {
		return state.IsRegistered()
	}
	return true
}

func (l *HotkeyListener) hasResources() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.hk != nil
}
