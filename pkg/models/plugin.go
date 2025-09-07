package models

import (
	"path"
	"watools/config"
)

type Plugin struct {
	ID          string
	PackageID   string
	Name        string
	Version     string
	Description string
}

func (p *Plugin) GetExecEntry() string {
	return path.Join(config.ProjectCacheDir(), "plugins", p.PackageID, "index.js")
}
