package localinstall

import "errors"

// ErrBusy means an installation owns the job or commit lock. It is not an
// indication that the request itself is invalid.
var ErrBusy = errors.New("local installation is busy")

// ValidationError carries a safe explanation for a correctable configuration or
// selection problem. Cause is retained for diagnostics without sending it to UI.
type ValidationError struct {
	Message string
	cause   error
}

func (e *ValidationError) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}
func (e *ValidationError) Unwrap() error { return e.cause }

func validation(message string, cause error) error {
	return &ValidationError{Message: message, cause: cause}
}
