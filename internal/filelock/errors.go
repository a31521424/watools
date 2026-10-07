package filelock

import "errors"

// ErrBusy identifies contention, rather than a failure to open or lock a file.
var ErrBusy = errors.New("another installation is in progress")
