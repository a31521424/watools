package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestContentionAndFilesystemFailureRemainDistinct(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation.lock")
	unlock, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if next, err := Acquire(path); !errors.Is(err, ErrBusy) {
		if next != nil {
			next()
		}
		t.Fatalf("contention error = %v", err)
	}
	_, err = Acquire(t.TempDir())
	var pathError *os.PathError
	if errors.Is(err, ErrBusy) || !errors.As(err, &pathError) {
		t.Fatalf("filesystem failure classified as busy or lost cause: %v", err)
	}
}
