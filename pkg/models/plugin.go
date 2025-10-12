package models

import "time"

type PluginState struct {
	PackageID    string                 `json:"package_id"`
	Enabled      bool                   `json:"enabled"`
	Storage      map[string]interface{} `json:"storage"`
	LastUsedTime time.Time              `json:"last_used_time"`
	UsedCount    int                    `json:"used_count"`
}
