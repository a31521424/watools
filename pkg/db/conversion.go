package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"watools/pkg/logger"
	"watools/pkg/models"
)

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func ConvertApplicationCommand(command Application) *models.ApplicationCommand {
	var dirUpdatedAt time.Time
	var err error
	dirUpdatedAt, err = time.Parse(time.DateTime, command.DirUpdatedAt)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to parse dirUpdatedAt: %s", command.DirUpdatedAt))
	}
	return models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, command.ID, dirUpdatedAt)
}

func ConvertPluginState(plugin PluginState) *models.PluginState {
	var lastUsedTime time.Time
	var err error
	if plugin.LastUsedTime != "" {
		lastUsedTime, err = time.Parse(time.DateTime, plugin.LastUsedTime)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to parse lastUsedTime: %s", plugin.LastUsedTime))
		}
	}
	var storage map[string]interface{}
	if plugin.Storage != "" {
		err = json.Unmarshal([]byte(plugin.Storage), &storage)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to unmarshal storage: %s", plugin.Storage))
		}
	}
	return &models.PluginState{
		PackageID:    plugin.PackageID,
		Enabled:      plugin.Enabled,
		Storage:      storage,
		LastUsedTime: lastUsedTime,
		UsedCount:    int(plugin.UsedCount),
	}
}
