//go:build darwin || windows

package watcher

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"watools/pkg/db"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Use a genuine driver error: matching text or a fake Code method would not
// establish that retry decisions work with the database errors used in production.
func sqliteBusyError(t *testing.T) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "busy.db")
	writer, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { writer.Close() })
	if _, err := writer.Exec("CREATE TABLE items (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tx, err := writer.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	if _, err := tx.Exec("INSERT INTO items VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	contender, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { contender.Close() })
	_, err = contender.Exec("INSERT INTO items VALUES (2)")
	var dbErr *sqlite.Error
	if !errors.As(err, &dbErr) || dbErr.Code()&0xff != sqlite3.SQLITE_BUSY {
		t.Fatalf("expected SQLite contention, got %v", err)
	}
	return err
}

func TestRetryStopsOnPermanentErrorsAndReportsActualAttempts(t *testing.T) {
	for _, cause := range []error{errors.New("invalid application"), os.ErrPermission, context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			attempts := 0
			config := &RetryConfig{MaxRetries: 3}
			op := NewRetryableOperation(func() error { attempts++; return fmt.Errorf("operation: %w", cause) }, nil, config)
			err := op.ExecuteContext(context.Background())
			if attempts != 1 || !errors.Is(err, cause) || !strings.Contains(err.Error(), "after 1 attempts") {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
		})
	}
}

func TestSQLiteBusyRetrySucceedsAndExhaustionPreservesCause(t *testing.T) {
	cause := sqliteBusyError(t)
	for _, succeeds := range []bool{false, true} {
		attempts := 0
		op := NewRetryableOperation(func() error {
			attempts++
			if succeeds && attempts == 2 {
				return nil
			}
			return fmt.Errorf("write applications: %w", cause)
		}, nil, &RetryConfig{MaxRetries: 2})
		err := op.ExecuteContext(context.Background())
		if succeeds {
			if err != nil || attempts != 2 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
		} else if attempts != 3 || !errors.Is(err, cause) || !strings.Contains(err.Error(), "after 3 attempts") {
			t.Fatalf("attempts=%d err=%v", attempts, err)
		}
	}
}

func TestRetryCancellationDuringBackoff(t *testing.T) {
	cause := sqliteBusyError(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	attempts := 0
	op := NewRetryableOperation(func() error { attempts++; close(started); return cause }, nil, &RetryConfig{MaxRetries: 3, RetryInterval: 60000})
	result := make(chan error, 1)
	go func() { result <- op.ExecuteContext(ctx) }()
	<-started
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) || attempts != 1 {
			t.Fatalf("attempts=%d err=%v", attempts, err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt retry delay")
	}
}

func TestSQLiteBusyWithFailedRollbackIsNotRetried(t *testing.T) {
	busy := sqliteBusyError(t)
	rollback := errors.New("rollback failed; transaction outcome unknown")
	attempts := 0
	op := NewRetryableOperation(func() error {
		attempts++
		return errors.Join(fmt.Errorf("update applications: %w", busy), fmt.Errorf("rollback: %w", rollback))
	}, nil, &RetryConfig{MaxRetries: 3})
	err := op.ExecuteContext(context.Background())
	if attempts != 1 || !errors.Is(err, busy) || !errors.Is(err, rollback) {
		t.Fatalf("unsafe retry or lost cause: attempts=%d err=%v", attempts, err)
	}
}

func TestSQLiteBusyWithUnknownTransactionOutcomeIsNotRetried(t *testing.T) {
	busy := sqliteBusyError(t)
	attempts := 0
	op := NewRetryableOperation(func() error {
		attempts++
		return errors.Join(fmt.Errorf("commit or rollback: %w", busy), db.ErrTransactionOutcomeUnknown)
	}, nil, &RetryConfig{MaxRetries: 3})
	err := op.ExecuteContext(context.Background())
	if attempts != 1 || !errors.Is(err, busy) || !errors.Is(err, db.ErrTransactionOutcomeUnknown) {
		t.Fatalf("unknown transaction outcome was retried: attempts=%d err=%v", attempts, err)
	}
}

type alwaysRetry struct{ *DefaultErrorHandler }

func (alwaysRetry) ShouldRetry(error, int) bool { return true }

func TestCancellationCannotBeRetriedByCustomHandler(t *testing.T) {
	attempts := 0
	op := NewRetryableOperation(func() error { attempts++; return context.Canceled }, alwaysRetry{}, &RetryConfig{MaxRetries: 3})
	if err := op.ExecuteContext(context.Background()); attempts != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
}

func TestNilRetryConfigurationPerformsOneAttempt(t *testing.T) {
	attempts := 0
	cause := errors.New("failure")
	op := NewRetryableOperation(func() error { attempts++; return cause }, nil, nil)
	if err := op.ExecuteContext(context.Background()); attempts != 1 || !errors.Is(err, cause) {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
}
