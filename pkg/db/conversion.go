package db

import (
	"database/sql"
	"fmt"
	"time"
	"watools/pkg/logger"
	"watools/pkg/models"
)

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func ConvertApplicationCommand(command Command) *models.ApplicationCommand {
	var dirUpdatedAt time.Time
	var err error
	dirUpdatedAt, err = time.Parse(time.DateTime, command.DirUpdatedAt)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to parse dirUpdatedAt: %s", command.DirUpdatedAt))
	}
	return models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, command.ID, dirUpdatedAt)
}
