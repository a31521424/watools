package models

import (
	"path"
	"watools/config"
)

type Plugin struct {
	ID          string `json:"id"`
	PackageID   string `json:"packageID"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Internal    bool   `json:"internal"`
}

func (p *Plugin) GetExecEntry() string {
	return path.Join(config.ProjectCacheDir(), "plugins", p.PackageID, "index.js")
}
