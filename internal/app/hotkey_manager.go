package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"watools/config"
	"watools/pkg/logger"
)

type HotkeyManager struct {
	listeners map[string]*HotkeyListener
	configs   map[string]HotkeyConfig
	configDir string
}

var (
	hotkeyManagerInstance *HotkeyManager
	defaultConfigs        = map[string]HotkeyConfig{
		"show-hide-window": {
			ID:     "show-hide-window",
			Name:   "显示/隐藏窗口",
			Hotkey: "cmd+Space",
		},
		"reload": {
			ID:     "reload",
			Name:   "重新加载",
			Hotkey: "cmd+R",
		},
		"reload-app": {
			ID:     "reload-app",
			Name:   "重新加载应用",
			Hotkey: "cmd+shift+R",
		},
	}
)

func GetHotkeyManager() *HotkeyManager {
	if hotkeyManagerInstance == nil {
		hotkeyManagerInstance = &HotkeyManager{
			listeners: make(map[string]*HotkeyListener),
			configs:   make(map[string]HotkeyConfig),
			configDir: filepath.Join(config.ProjectCacheDir(), "hotkeys"),
		}
	}
	return hotkeyManagerInstance
}

func (hm *HotkeyManager) LoadConfigs() error {
	// 创建配置目录
	if err := os.MkdirAll(hm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create hotkey config directory: %w", err)
	}

	// 加载配置文件
	configFile := filepath.Join(hm.configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，使用默认配置并保存
			hm.configs = defaultConfigs
			return hm.SaveConfigs()
		}
		return fmt.Errorf("failed to read hotkey config file: %w", err)
	}

	// 解析配置
	var configs []HotkeyConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse hotkey config file: %w", err)
	}

	// 转换为map
	hm.configs = make(map[string]HotkeyConfig)
	for _, cfg := range configs {
		hm.configs[cfg.ID] = cfg
	}

	// 确保所有默认配置都存在
	for id, defaultCfg := range defaultConfigs {
		if _, exists := hm.configs[id]; !exists {
			hm.configs[id] = defaultCfg
		}
	}

	return nil
}

func (hm *HotkeyManager) SaveConfigs() error {
	// 转换为slice
	configs := make([]HotkeyConfig, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		configs = append(configs, cfg)
	}

	// 序列化
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hotkey configs: %w", err)
	}

	// 保存到文件
	configFile := filepath.Join(hm.configDir, "config.json")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write hotkey config file: %w", err)
	}

	return nil
}

func (hm *HotkeyManager) GetConfig(id string) (HotkeyConfig, bool) {
	cfg, exists := hm.configs[id]
	return cfg, exists
}

func (hm *HotkeyManager) SetConfig(cfg HotkeyConfig) error {
	// 验证热键格式
	if _, _, err := cfg.ParseHotkey(); err != nil {
		return fmt.Errorf("invalid hotkey format: %w", err)
	}

	hm.configs[cfg.ID] = cfg
	return nil
}

func (hm *HotkeyManager) GetAllConfigs() []HotkeyConfig {
	configs := make([]HotkeyConfig, 0, len(hm.configs))
	for _, cfg := range hm.configs {
		configs = append(configs, cfg)
	}
	return configs
}

func (hm *HotkeyManager) RegisterAll() error {
	waApp := GetWaApp()
	
	// 清除现有的监听器
	hm.unregisterAll()

	// 为每个配置创建监听器
	for _, cfg := range hm.configs {
		modifiers, key, err := cfg.ParseHotkey()
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to parse hotkey, id: %s, error: %v", cfg.ID, err))
			continue
		}

		listener := &HotkeyListener{
			ID:        cfg.ID,
			Modifiers: modifiers,
			Key:       key,
		}

		// 根据ID设置对应的触发函数
		switch cfg.ID {
		case "show-hide-window":
			listener.OnTrigger = func() {
				waApp.HideOrShowApp()
			}
		case "reload":
			listener.OnTrigger = func() {
				waApp.Reload()
			}
		case "reload-app":
			listener.OnTrigger = func() {
				waApp.ReloadAPP()
			}
		default:
			// 对于未知的热键ID，跳过注册
			logger.Info(fmt.Sprintf("Unknown hotkey ID, skipping registration, id: %s", cfg.ID))
			continue
		}

		// 注册热键
		if err := listener.Register(); err != nil {
			logger.Info(fmt.Sprintf("Failed to register hotkey, id: %s, error: %v", cfg.ID, err))
			continue
		}

		hm.listeners[cfg.ID] = listener
		logger.Info(fmt.Sprintf("Hotkey registered, id: %s, hotkey: %s", cfg.ID, cfg.Hotkey))
	}

	return nil
}

func (hm *HotkeyManager) UnregisterAll() {
	hm.unregisterAll()
}

func (hm *HotkeyManager) unregisterAll() {
	for id, listener := range hm.listeners {
		if err := listener.Unregister(); err != nil {
			logger.Error(err, "Failed to unregister hotkey", "id", id)
		}
		delete(hm.listeners, id)
	}
}

func (hm *HotkeyManager) IsRegistered(id string) bool {
	listener, exists := hm.listeners[id]
	if !exists {
		return false
	}
	return listener.IsRegistered()
}