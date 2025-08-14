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
	
	// 标准化修饰键名称
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
	// 确保通道已初始化
	if l.quit == nil {
		l.quit = make(chan struct{}, 1)
	}
	
	for {
		// 检查是否已注册
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
	// 如果已经注册，先注销
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
		// 清理资源
		l.hk = nil
		l.quit = nil
		return err
	}
	
	go l.listen()
	logger.Info(fmt.Sprintf("Hotkey registered successfully, id: %s", l.ID))
	return nil
}

func (l *HotkeyListener) Unregister() error {
	// 检查是否已注册
	if !l.IsRegistered() {
		return nil // 已经注销或从未注册
	}
	
	err := l.hk.Unregister()
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to unregister hotkey, id: %s, error: %v", l.ID, err))
		// 即使出错也要清理资源
	}
	
	// 发送停止信号
	if l.quit != nil {
		select {
		case l.quit <- struct{}{}:
		default:
			// 通道已满，无需处理
		}
	}
	
	// 清理资源
	l.hk = nil
	l.quit = nil
	
	logger.Info(fmt.Sprintf("Hotkey unregistered successfully, id: %s", l.ID))
	return err
}

func (l *HotkeyListener) IsRegistered() bool {
	return l.hk != nil
}
