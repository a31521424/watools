package config

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"watools/pkg/utils"
)

var dataDirMu sync.Mutex

// ProjectDataDir contains durable user data. The legacy cache is kept as a backup.
func ProjectDataDir() string {
	dataDirMu.Lock()
	defer dataDirMu.Unlock()
	root := os.Getenv("WATOOLS_DATA_DIR")
	if root != "" {
		// An explicit directory is a separate installation, not a migration
		// destination. In particular, development must not import user data.
		if err := os.MkdirAll(root, 0700); err != nil {
			panic(fmt.Errorf("prepare explicit application data directory: %w", err))
		}
		return root
	}
	base, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Errorf("resolve application data directory: %w", err))
	}
	root = filepath.Join(base, ProjectName())
	if err := prepareDataDir(root, ProjectCacheDir()); err != nil {
		panic(err)
	}
	return root
}

func prepareDataDir(root, legacy string) error {
	if info, err := os.Stat(root); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("application data path is not a directory")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(root), ".watools-data-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, name := range []string{"data", "plugins", "hotkeys"} {
		source := filepath.Join(legacy, name)
		if _, err := os.Stat(source); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		var err error
		if name == "data" {
			err = copyDatabaseDirectory(source, filepath.Join(stage, name))
		} else {
			err = os.CopyFS(filepath.Join(stage, name), os.DirFS(source))
		}
		if err != nil {
			return fmt.Errorf("migrate %s; legacy data was preserved: %w", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(stage, "migration.json"), []byte(`{"schemaVersion":1,"legacyCachePreserved":true}`), 0600); err != nil {
		return err
	}
	return os.Rename(stage, root)
}

func copyDatabaseDirectory(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		if strings.HasSuffix(path, ".db-wal") || strings.HasSuffix(path, ".db-shm") || strings.HasSuffix(path, ".db-journal") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("data migration encountered a special file")
		}
		if strings.HasSuffix(path, ".db") {
			return snapshotSQLite(path, dest)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = io.Copy(output, input)
		closeErr := output.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}
func snapshotSQLite(source, target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := sql.Open("sqlite", utils.FileURI(source)+"?mode=ro&_pragma=busy_timeout%285000%29")
	if err != nil {
		return err
	}
	defer database.Close()
	connection, err := database.Conn(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	err = connection.Raw(func(native interface{}) error {
		creator, ok := native.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return fmt.Errorf("SQLite driver does not support online backup")
		}
		backup, err := creator.NewBackup(utils.FileURI(target))
		if err != nil {
			return err
		}
		for {
			if err := ctx.Err(); err != nil {
				backup.Finish()
				return err
			}
			more, err := backup.Step(256)
			if err != nil {
				backup.Finish()
				return err
			}
			if !more {
				return backup.Finish()
			}
		}
	})
	if err != nil {
		return err
	}
	return os.Chmod(target, 0600)
}
