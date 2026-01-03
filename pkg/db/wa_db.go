package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/lo"
	"github.com/samber/mo"
	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type WaDB struct {
	db    *sql.DB
	query *Queries
}

func runMigrations(db *sql.DB) error {
	logger.Info("Running migrations")

	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	databaseDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", databaseDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	logger.Info("Migrations completed")
	return nil
}

func NewWaDB() (*WaDB, error) {
	sqliteFolderPath := filepath.Join(config.ProjectCacheDir(), "data")
	sqliteFilePath := filepath.Join(sqliteFolderPath, fmt.Sprintf("%s.db", config.ProjectName()))
	err := os.MkdirAll(sqliteFolderPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite folder: %w", err)
	}
	db, err := sql.Open("sqlite", sqliteFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite file: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite file: %w", err)
	}
	if _, err = db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}
	if _, err = db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	err = runMigrations(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run db migrations: %w", err)
	}

	return &WaDB{
		db:    db,
		query: New(db),
	}, nil
}

var (
	waDBInstance *WaDB
	waDBOnce     sync.Once
)

func GetWaDB() *WaDB {
	waDBOnce.Do(func() {
		var err error
		waDBInstance, err = NewWaDB()
		if err != nil || waDBInstance == nil {
			panic(err)
		}
	})
	return waDBInstance
}

func (d *WaDB) Close() error {
	return d.db.Close()
}

func (d *WaDB) withTx(ctx context.Context, f func(tx *sql.Tx) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	return f(tx)
}

func (d *WaDB) GetCommands(ctx context.Context) []*models.ApplicationCommand {
	dbCommands, err := d.query.GetApplications(ctx)
	if err != nil {
		logger.Error(err, "Failed to get commands")
		return nil
	}
	return lo.Map(dbCommands, func(item Application, _ int) *models.ApplicationCommand {
		return ConvertApplicationCommand(item)
	})
}

func (d *WaDB) BatchInsertCommands(ctx context.Context, commands []*models.ApplicationCommand) error {
	logger.Info(fmt.Sprintf("Starting to insert %d commands", len(commands)))
	return d.withTx(ctx, func(tx *sql.Tx) error {
		txQuery := d.query.WithTx(tx)
		for _, command := range commands {
			if _, err := txQuery.CreateApplication(ctx, CreateApplicationParams{
				ID:           command.ID,
				Name:         command.Name,
				Description:  command.Description,
				Category:     string(command.Category),
				Path:         command.Path,
				IconPath:     command.IconPath,
				DirUpdatedAt: command.DirUpdatedAt,
			}); err != nil {
				return err
			}
		}
		logger.Info(fmt.Sprintf("Inserted %d commands", len(commands)))
		return tx.Commit()
	})
}

func (d *WaDB) GetExpiredCommands(ctx context.Context) []*models.ApplicationCommand {
	expiredTime := time.Now().Add(-time.Hour * 24)
	dbCommands, err := d.query.GetExpiredApplications(ctx, expiredTime)
	if err != nil {
		logger.Error(err, "Failed to get expired commands")
		return nil
	}
	return lo.Map(dbCommands, func(item Application, _ int) *models.ApplicationCommand {
		return ConvertApplicationCommand(item)
	})
}

func (d *WaDB) BatchUpdateCommands(ctx context.Context, commands []*models.ApplicationCommand) error {
	logger.Info(fmt.Sprintf("Updating %d commands", len(commands)))
	return d.withTx(ctx, func(tx *sql.Tx) error {
		txQuery := d.query.WithTx(tx)
		for _, command := range commands {
			if err := txQuery.UpdateApplicationPartial(ctx, UpdateApplicationPartialParams{
				ID:           command.ID,
				DirUpdatedAt: command.DirUpdatedAt,
				Name:         mo.Some(command.Name),
				Description:  command.Description,
				Category:     mo.Some(string(command.Category)),
				Path:         mo.Some(command.Path),
				IconPath:     command.IconPath,
			}); err != nil {
				return fmt.Errorf("failed to update command: %w", err)
			}
		}
		return tx.Commit()
	})
}

func (d *WaDB) DeleteCommands(ctx context.Context, ids []string) error {
	logger.Info(fmt.Sprintf("Deleting command %v", ids))
	return d.withTx(ctx, func(tx *sql.Tx) error {
		txQuery := d.query.WithTx(tx)
		if err := txQuery.DeleteApplication(ctx, ids); err != nil {
			return fmt.Errorf("failed to delete command: %w", err)
		}
		return tx.Commit()
	})
}

func (d *WaDB) GetCommandIsUpdatedDir(ctx context.Context, path string, dirUpdatedAt time.Time) *models.ApplicationCommand {
	command, err := d.query.GetApplicationIsUpdatedDir(ctx, GetApplicationIsUpdatedDirParams{
		Path:         path,
		DirUpdatedAt: dirUpdatedAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logger.Error(err, "Failed to get command is updated dir")
	}
	return ConvertApplicationCommand(command)
}

func (d *WaDB) GetPlugins(ctx context.Context) []*models.PluginState {
	dbPlugins, err := d.query.GetPlugins(ctx)
	if err != nil {
		logger.Error(err, "Failed to get plugins")
		return nil
	}
	return lo.Map(dbPlugins, func(item PluginState, _ int) *models.PluginState {
		return ConvertPluginState(item)
	})
}

func (d *WaDB) InsertPlugin(ctx context.Context, params InsertPluginParams) error {
	return d.query.InsertPlugin(ctx, params)
}

func (d *WaDB) DeletePlugin(ctx context.Context, packageID string) error {
	return d.query.DeletePlugin(ctx, packageID)
}

func (d *WaDB) UpdatePluginEnabled(ctx context.Context, packageID string, enabled bool) error {
	return d.query.UpdatePluginEnabled(ctx, UpdatePluginEnabledParams{
		Enabled:   enabled,
		PackageID: packageID,
	})
}

func (d *WaDB) UpdatePluginStorage(ctx context.Context, packageID string, storage string) error {
	return d.query.UpdatePluginStorage(ctx, UpdatePluginStorageParams{
		Storage:   storage,
		PackageID: packageID,
	})
}

func (d *WaDB) BatchUpdateApplicationUsage(ctx context.Context, usageUpdates []models.ApplicationUsageUpdate) error {
	logger.Info(fmt.Sprintf("Updating usage for %d applications", len(usageUpdates)))
	return d.withTx(ctx, func(tx *sql.Tx) error {
		txQuery := d.query.WithTx(tx)
		for _, update := range usageUpdates {
			if err := txQuery.UpdateApplicationUsage(ctx, UpdateApplicationUsageParams{
				ID:         update.ID,
				LastUsedAt: models.ToOptionTime(update.LastUsedAt),
				UsedCount:  int64(update.UsedCount),
			}); err != nil {
				return fmt.Errorf("failed to update application usage: %w", err)
			}
		}
		return tx.Commit()
	})
}

func (d *WaDB) BatchUpdatePluginUsage(ctx context.Context, usageUpdates []models.PluginUsageUpdate) error {
	logger.Info(fmt.Sprintf("Updating usage for %d plugins", len(usageUpdates)))
	return d.withTx(ctx, func(tx *sql.Tx) error {
		txQuery := d.query.WithTx(tx)
		for _, update := range usageUpdates {
			if err := txQuery.UpdatePluginUsage(ctx, UpdatePluginUsageParams{
				PackageID:  update.PackageID,
				LastUsedAt: models.ToOptionTime(update.LastUsedAt),
				UsedCount:  int64(update.UsedCount),
			}); err != nil {
				return fmt.Errorf("failed to update plugin usage: %w", err)
			}
		}
		return tx.Commit()
	})
}
