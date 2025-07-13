package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"watools/config"
	"watools/pkg/generics"
	"watools/pkg/logger"
	"watools/pkg/models"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // driver for reading migrations from files
)

var dbMutex sync.Mutex

type WaDB struct {
	db    *sql.DB
	query *Queries
}

func runMigrations(db *sql.DB) error {
	logger.Info("Running migrations")
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		logger.Error(err, "Failed to create sqlite driver")
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://pkg/db/migrations",
		"sqlite",
		driver,
	)
	if err != nil {
		logger.Error(err, "Failed to create migrate instance")
		return err
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error(err, "Failed to run migrations")
		return err
	}
	logger.Info("Migrations completed")
	return nil
}

func NewWaDB() *WaDB {
	sqliteFolderPath := filepath.Join(config.ProjectCacheDir(), "data")
	sqliteFilePath := filepath.Join(sqliteFolderPath, fmt.Sprintf("%s.db", config.ProjectName()))
	err := os.MkdirAll(sqliteFolderPath, 0755)
	if err != nil {
		logger.Error(err, "Failed to create sqlite folder")
		return nil
	}
	db, err := sql.Open("sqlite", sqliteFilePath)
	if err != nil {
		logger.Error(err, "Failed to open sqlite file")
		return nil
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		logger.Error(err, "Failed to ping sqlite file")
		return nil
	}
	if _, err = db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		logger.Error(err, "Failed to set journal mode")
		return nil
	}
	if _, err = db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		db.Close()
		logger.Error(err, "Failed to set busy timeout")
		return nil
	}

	err = runMigrations(db)
	if err != nil {
		db.Close()
		logger.Error(err, "Failed to run db migrations")
		return nil
	}

	return &WaDB{
		db:    db,
		query: New(db),
	}
}

var (
	waDBInstance *WaDB
	waDBOnce     sync.Once
)

func GetWaDB() *WaDB {
	waDBOnce.Do(func() {
		waDBInstance = NewWaDB()
		if waDBInstance == nil {
			panic("Failed to get WaDB instance")
		}
	})
	return waDBInstance
}

func (d *WaDB) Close() error {
	return d.db.Close()
}

func (d *WaDB) GetCommands(ctx context.Context) []*models.Command {
	dbCommands, err := d.query.GetCommands(ctx)
	if err != nil {
		logger.Error(err, "Failed to get commands")
		return nil
	}
	return generics.Map(dbCommands, ConvertCommand)
}

func (d *WaDB) BatchInsertCommands(ctx context.Context, commands []*models.Command) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	logger.Info(fmt.Sprintf("Starting to insert %d commands", len(commands)))
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, command := range commands {
		if _, err := d.query.CreateCommand(ctx, CreateCommandParams{
			Name:        command.Name,
			Description: command.Description,
			Category:    string(command.Category),
			Path:        command.Path,
			IconPath:    command.IconPath,
		}); err != nil {
			logger.Error(err, "Failed to create command")
			return err
		}
	}
	logger.Info(fmt.Sprintf("Inserted %d commands", len(commands)))
	return tx.Commit()
}

func (d *WaDB) FindExpiredCommands(ctx context.Context) []*models.Command {
	expiredTime := time.Now().Add(-time.Hour * 24)
	dbCommands, err := d.query.FindExpiredCommands(ctx, *TimeToDBTime(&expiredTime))
	if err != nil {
		logger.Error(err, "Failed to get expired commands")
		return nil
	}
	return generics.Map(dbCommands, ConvertCommand)
}

func (d *WaDB) BatchUpdateCommands(ctx context.Context, commands []*models.Command) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	logger.Info(fmt.Sprintf("Updating %d commands", len(commands)))
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, command := range commands {
		if err := d.query.UpdateCommandPartial(ctx, UpdateCommandPartialParams{
			ID:          command.ID,
			Name:        nullString(command.Name),
			Path:        nullString(command.Path),
			IconPath:    nullString(command.IconPath),
			Category:    nullString(string(command.Category)),
			Description: nullString(command.Description),
		}); err != nil {
			logger.Error(err, "Failed to update command")
			return err
		}
	}
	return tx.Commit()
}
