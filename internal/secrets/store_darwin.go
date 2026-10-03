package secrets

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"watools/config"
)

// macOS stores plugin credentials locally by user preference. These files are
// plaintext with owner-only permissions; this backend does not use Keychain.
var fileMu sync.Mutex

func secretPath(account string) string {
	return filepath.Join(config.ProjectDataDir(), "secrets", account+".local")
}

func get(account string) (string, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	file, err := os.Open(secretPath(account))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 64<<10 {
		return "", fmt.Errorf("secret exceeds size limit")
	}
	return string(data), nil
}

func set(account, value string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	path := secretPath(account)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".secret-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(value)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}

func remove(account string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	err := os.Remove(secretPath(account))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
