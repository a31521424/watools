package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/utils"

	"github.com/google/uuid"
	"github.com/samber/mo"
)

type PluginMetadata struct {
	PackageID       string   `json:"packageId"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Version         string   `json:"version"`
	Author          string   `json:"author"`
	Usage           string   `json:"usage"`
	TriggerKeywords []string `json:"triggerKeywords"`
	UsageExamples   []string `json:"usageExamples"`
	Shortcuts       []string `json:"shortcuts"`
	Notes           []string `json:"notes"`
	UIEnabled       bool     `json:"uiEnabled"`
	Entry           string   `json:"entry"`
}

type PluginState struct {
	PackageID  string                 `json:"packageId"`
	Enabled    bool                   `json:"enabled"`
	Storage    map[string]interface{} `json:"storage"`
	LastUsedAt mo.Option[time.Time]   `json:"lastUsedAt"`
	UsedCount  int64                  `json:"usedCount"`
}

type PluginUsageUpdate struct {
	PackageID  string
	LastUsedAt time.Time
	UsedCount  int
}

func (p *PluginState) GetMetadata() (*PluginMetadata, error) {
	var metadata PluginMetadata
	root, _, err := ResolveInstalledPlugin(filepath.Join(config.ProjectDataDir(), "plugins"), p.PackageID)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin manifest not found: %s", manifestPath)
	}

	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode plugin manifest %s: %w", manifestPath, err)
	}
	if metadata.PackageID != p.PackageID {
		return nil, fmt.Errorf("plugin manifest packageId mismatch")
	}

	return &metadata, nil
}

func (p *PluginState) getPluginHomeUrl() string {
	_, revision, err := ResolveInstalledPlugin(filepath.Join(config.ProjectDataDir(), "plugins"), p.PackageID)
	if err == nil && revision != "" {
		return path.Join("/api/plugin/", p.PackageID, "_v", revision)
	}
	return path.Join("/api/plugin/", p.PackageID)
}

// ResolveInstalledPlugin supports legacy flat packages and immutable revision trees.
func ResolveInstalledPlugin(base, packageID string) (string, string, error) {
	if err := utils.ValidatePluginPackageID(packageID); err != nil {
		return "", "", err
	}
	root, err := utils.ResolvePathWithinBase(base, packageID)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(filepath.Join(root, "current.json"))
	if os.IsNotExist(err) {
		return root, "", nil
	}
	if err != nil {
		return "", "", err
	}
	var current struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(data, &current); err != nil {
		return "", "", err
	}
	if _, err := uuid.Parse(current.Revision); err != nil {
		return "", "", fmt.Errorf("invalid plugin revision")
	}
	return filepath.Join(root, "versions", current.Revision), current.Revision, nil
}

func (p *PluginState) GetFullInfo() map[string]interface{} {
	mapData := map[string]interface{}{
		"packageId": p.PackageID, "enabled": p.Enabled, "usedCount": p.UsedCount,
		"lastUsedAt": p.LastUsedAt, "homeUrl": p.getPluginHomeUrl(),
	}
	metadata, err := p.GetMetadata()
	if err != nil {
		logger.Error(err, "Failed to get plugin metadata")
		mapData["name"] = p.PackageID
		mapData["loadError"] = "插件元数据损坏，请重新安装或卸载"
		return mapData
	}
	mapData, err = utils.MergeStructToMap([]interface{}{
		mapData,
		metadata,
		map[string]interface{}{
			"homeUrl": p.getPluginHomeUrl(),
		},
	})
	if err != nil {
		logger.Error(err, "Failed to marshal plugin state")
	}
	return mapData
}

func (p *PluginState) GetJsEntryUrl() string {
	metadata, err := p.GetMetadata()
	if err != nil {
		logger.Error(err, "Failed to get plugin metadata")
		return ""
	}
	homeUrl := p.getPluginHomeUrl()
	return path.Join(homeUrl, metadata.Entry)
}
