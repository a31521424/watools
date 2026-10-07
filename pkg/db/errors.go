package db

import "errors"

// ErrTransactionOutcomeUnknown marks a failed commit or rollback whose recovery
// is not confirmed. Callers must not blindly retry the entire transaction.
var ErrTransactionOutcomeUnknown = errors.New("transaction outcome or recovery is not confirmed")
