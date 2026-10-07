package application

import (
	"errors"
	"fmt"
)

var (
	// ErrApplicationNotFound means the application root path no longer exists.
	ErrApplicationNotFound = errors.New("application not found")
	// ErrNotApplication means an existing path is not a supported application.
	ErrNotApplication = errors.New("path is not an application")
	// ErrApplicationUnreadable means required application state could not be read.
	ErrApplicationUnreadable = errors.New("application unreadable")
	// ErrInvalidMetadata includes missing or malformed required metadata.
	ErrInvalidMetadata = errors.New("invalid application metadata")
)

func applicationError(kind error, operation, path string, cause error) error {
	return fmt.Errorf("%s %q: %w", operation, path, errors.Join(kind, cause))
}
