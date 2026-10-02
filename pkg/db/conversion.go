package db

import (
	"encoding/json"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/mo"
)

func ConvertApplicationCommand(command Application) *models.ApplicationCommand {
	result := models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, mo.Some(command.ID), command.DirUpdatedAt)
	result.UsedCount = command.UsedCount
	result.LastUsedAt = command.LastUsedAt
	return result
}

func ConvertPluginState(plugin PluginState) *models.PluginState {
	var storage map[string]interface{}
	if plugin.Storage != "" {
		err := json.Unmarshal([]byte(plugin.Storage), &storage)
		if err != nil {
			logger.Error(err, "Failed to unmarshal plugin storage", "packageId", plugin.PackageID)
		}
	}
	return &models.PluginState{
		PackageID:  plugin.PackageID,
		Enabled:    plugin.Enabled,
		Storage:    storage,
		LastUsedAt: plugin.LastUsedAt,
		UsedCount:  plugin.UsedCount,
	}
}
