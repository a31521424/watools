package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"watools/config"

	_ "modernc.org/sqlite"
)

type Database struct {
	db *sql.DB
}

func New() (*Database, error) {
	projectName := config.ProjectName()
	if config.ProjectName() == "" {
		return nil, fmt.Errorf("project name is empty")
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	sqliteFolderPath := filepath.Join(userCacheDir, projectName, "data")
	sqliteFilePath := filepath.Join(sqliteFolderPath, config.ProjectName()+".db")
	err = os.MkdirAll(sqliteFolderPath, 0755)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sqliteFilePath)
	if err != nil {
		return nil, err
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
