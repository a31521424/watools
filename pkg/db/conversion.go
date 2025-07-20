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

func DBTimeToTime(t string) *time.Time {
	result, err := time.Parse(time.DateTime, t)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to parse time %s", t))
		return nil
	}
	return &result
}

func TimeToDBTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	result := t.Format(time.DateTime)
	return &result
}

func ConvertApplicationCommand(command Command) *models.ApplicationCommand {
	return models.NewApplicationCommand(command.Name, command.Description, command.Path, command.IconPath, command.ID)
}
