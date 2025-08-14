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
	hotkeyParts := strings.Split(c.Hotkey, "+")
	var modifiers []hotkey.Modifier
	var pureKey *hotkey.Key
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
	for {
		if !l.IsRegistered() {
			logger.Info("Hotkey listener unregistered")
			return
		}
		select {
		case <-l.quit:
			return
		case <-l.hk.Keydown():
			logger.Info(fmt.Sprintf("[Hotkey] pressed %s", l.ID))
			l.OnTrigger()
		}
	}
}

func (l *HotkeyListener) Register() error {
	l.quit = make(chan struct{}, 1)
	l.hk = hotkey.New(l.Modifiers, l.Key)
	err := l.hk.Register()
	if err != nil {
		logger.Error(err, "Failed to register hotkey")
		return err
	}
	go l.listen()
	return nil
}

func (l *HotkeyListener) Unregister() error {
	err := l.hk.Unregister()
	if err != nil {
		logger.Error(err, "Failed to unregister hotkey")
		return err
	}
	l.quit <- struct{}{}
	l.hk = nil
	l.quit = nil
	return nil
}

func (l *HotkeyListener) IsRegistered() bool {
	return l.hk != nil
}

func GetHotkeyListeners() []*HotkeyListener {
	return defaultListener
}
