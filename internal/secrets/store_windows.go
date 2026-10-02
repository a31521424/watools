package secrets

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
	"watools/config"
)

var fileMu sync.Mutex

func secretPath(account string) string {
	return filepath.Join(config.ProjectDataDir(), "secrets", account+".dpapi")
}
func crypt(data []byte, decrypt bool) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	var err error
	if decrypt {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	return append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...), nil
}
func get(account string) (string, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	data, err := os.ReadFile(secretPath(account))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	plain, err := crypt(data, true)
	return string(plain), err
}
func set(account, value string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	sealed, err := crypt([]byte(value), false)
	if err != nil {
		return err
	}
	path := secretPath(account)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".secret-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(sealed)
	closeErr := file.Close()
	if err != nil {
		return err
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
