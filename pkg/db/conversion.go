package db

import (
	"database/sql"
	"time"
	"watools/pkg/models"
)

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func ConvertApplicationCommand(command Command) *models.ApplicationCommand {
	var dirUpdatedAt *time.Time
	if parsed, err := time.Parse(time.DateTime, command.DirUpdatedAt); err != nil {
		dirUpdatedAt = nil
	} else {
		dirUpdatedAt = &parsed
	}
	return models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, command.ID, dirUpdatedAt)
}
