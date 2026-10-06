package localinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"watools/internal/filelock"
	"watools/internal/plugin"
	"watools/pkg/db"
)

type ApplyRequest struct {
	Configuration Configuration
	Bundle        string
	Packages      []string
}

// Apply prepares the bundle next to its destination, retains the previous bundle,
// then joins plugins and the receipt to the rollback boundary. Running processes
// keep their executable and old immutable plugin revisions.
func Apply(ctx context.Context, dataDir string, request ApplyRequest) (string, error) {
	value, err := ValidateConfiguration(request.Configuration)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", err
	}
	unlock, err := filelock.Acquire(filepath.Join(dataDir, ".local-install.lock"))
	if err != nil {
		return "", err
	}
	defer unlock()
	installer := plugin.NewPluginInstaller(ctx)
	if len(request.Packages) > 0 {
		database, err := db.NewWaDB()
		if err != nil {
			return "", fmt.Errorf("installation database: %w", err)
		}
		defer database.Close()
		installer = plugin.NewPluginInstallerWithDatabase(ctx, database)
	}
	target := value.ApplicationPath
	var staged, backup string
	if request.Bundle != "" {
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "", err
		}
		appUnlock, err := filelock.Acquire(filepath.Join(filepath.Dir(target), ".watools-install.lock"))
		if err != nil {
			return "", err
		}
		defer appUnlock()
		if err := verifyBundle(request.Bundle); err != nil {
			return "", err
		}
		if _, err := os.Lstat(target); err == nil {
			if err := verifyBundle(target); err != nil {
				return "", fmt.Errorf("拒绝覆盖其他应用: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
		staged, err = os.MkdirTemp(filepath.Dir(target), ".watools-install-*.app")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(staged)
		if err := copyBundle(request.Bundle, staged); err != nil {
			return "", err
		}
		if err := verifyBundle(staged); err != nil {
			return "", err
		}
		backup = filepath.Join(filepath.Dir(target), ".watools-backup-"+time.Now().Format("20060102-150405.000000000")+".app")
		if err := os.Rename(target, backup); err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
			backup = ""
		}
		if err := os.Rename(staged, target); err != nil {
			if backup != "" {
				err = errors.Join(err, os.Rename(backup, target))
			}
			return "", err
		}
	}
	if request.Bundle == "" {
		previous, err := ReadConfiguration(dataDir)
		if err != nil {
			return "", err
		}
		value.Version = previous.Version
	}
	value.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	err = installer.InstallBatch(request.Packages, func() error { return SaveConfiguration(dataDir, value) })
	if err != nil && request.Bundle != "" {
		// Keep the failed candidate for inspection; restore the previous install.
		if moveErr := os.Rename(target, staged); moveErr != nil {
			return backup, errors.Join(err, fmt.Errorf("application rollback: %w", moveErr))
		}
		if backup != "" {
			err = errors.Join(err, os.Rename(backup, target))
		}
	}
	return backup, err
}
