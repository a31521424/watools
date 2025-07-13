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
	logger.Info(fmt.Sprintf("Converting time %s to %s", t, result))
	return &result
}

func ConvertCommand(command Command) *models.Command {
	var category models.CommandCategory
	switch command.Category {
	case "Application":
		category = models.CategoryApplication
	case "SystemOperation":
		category = models.CategorySystemOperation
	}
	return &models.Command{
		Name:        command.Name,
		Description: command.Description,
		Category:    category,
		Path:        command.Path,
		IconPath:    command.IconPath,
		ID:          command.ID,
	}
}
