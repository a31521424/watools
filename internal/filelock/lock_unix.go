//go:build darwin || linux

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Acquire is nonblocking: UI operations fail clearly instead of waiting on another installer.
func Acquire(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("%w: %w", ErrBusy, err)
		}
		return nil, fmt.Errorf("acquire installation lock: %w", err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
