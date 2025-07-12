package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"watools/config"
	"watools/pkg/logger"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

func New() (*Database, error) {
	sqliteFolderPath := filepath.Join(config.ProjectCacheDir(), "data")
	sqliteFilePath := filepath.Join(sqliteFolderPath, fmt.Sprintf("%s.db", config.ProjectName()))
	err := os.MkdirAll(sqliteFolderPath, 0755)
	if err != nil {
		logger.Error(err, "Failed to create sqlite folder")
	}
	db, err := sql.Open("sqlite", sqliteFilePath)
	if err != nil {
		logger.Error(err, "Failed to open sqlite file")
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Database{
		db: db,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
