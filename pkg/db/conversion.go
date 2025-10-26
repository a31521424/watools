package db

import (
	"encoding/json"
	"fmt"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/mo"
)

func ConvertApplicationCommand(command Application) *models.ApplicationCommand {
	return models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, mo.Some(command.ID), command.DirUpdatedAt)
}

func ConvertPluginState(plugin PluginState) *models.PluginState {
	var storage map[string]interface{}
	if plugin.Storage != "" {
		err := json.Unmarshal([]byte(plugin.Storage), &storage)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to unmarshal storage: %s", plugin.Storage))
		}
	}
	return &models.PluginState{
		PackageID:    plugin.PackageID,
		Enabled:      plugin.Enabled,
		Storage:      storage,
		LastUsedTime: plugin.LastUsedAt,
		UsedCount:    plugin.UsedCount,
	}
}
