package handler

import (
	"encoding/json"
	"net/http"
	"watools/internal/app"
)

type HotkeyAPI struct{}

func NewHotkeyAPI() *HotkeyAPI {
	return &HotkeyAPI{}
}

// HotkeyConfig represents a hotkey configuration
type HotkeyConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
}

// GetHotkeys returns all hotkey configurations
func (h *HotkeyAPI) GetHotkeys(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hm := app.GetHotkeyManager()
	configs := hm.GetAllConfigs()

	// 转换为API格式
	apiConfigs := make([]HotkeyConfig, len(configs))
	for i, cfg := range configs {
		apiConfigs[i] = HotkeyConfig{
			ID:     cfg.ID,
			Name:   cfg.Name,
			Hotkey: cfg.Hotkey,
		}
	}

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(apiConfigs)
}

// UpdateHotkey updates a hotkey configuration
func (h *HotkeyAPI) UpdateHotkey(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var apiConfig HotkeyConfig
	if err := json.NewDecoder(req.Body).Decode(&apiConfig); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	hm := app.GetHotkeyManager()
	
	// 创建应用层配置
	cfg := app.HotkeyConfig{
		ID:     apiConfig.ID,
		Name:   apiConfig.Name,
		Hotkey: apiConfig.Hotkey,
	}

	// 验证并设置配置
	if err := hm.SetConfig(cfg); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// 保存配置
	if err := hm.SaveConfigs(); err != nil {
		http.Error(res, "Failed to save configs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新注册所有热键
	if err := hm.RegisterAll(); err != nil {
		http.Error(res, "Failed to re-register hotkeys: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(map[string]string{"status": "success"})
}