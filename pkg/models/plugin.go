package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/utils"

	"github.com/samber/mo"
)

type PluginMetadata struct {
	PackageID   string `json:"packageId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	UIEnabled   bool   `json:"uiEnabled"`
	Entry       string `json:"entry"`
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
	manifestPath := path.Join(config.ProjectCacheDir(), "plugins", p.PackageID, "manifest.json")
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

	return &metadata, nil
}

func (p *PluginState) getPluginHomeUrl() string {
	return path.Join("/api/plugin/", p.PackageID)
}

func (p *PluginState) GetFullInfo() map[string]interface{} {
	var mapData map[string]interface{}
	metadata, err := p.GetMetadata()
	if err != nil {
		logger.Error(err, "Failed to get plugin metadata")
		return mapData
	}
	mapData, err = utils.MergeStructToMap([]interface{}{
		p,
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
