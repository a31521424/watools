package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"testing"
	"time"
	"watools/pkg/models"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestWithTxCommitsSuccessfulWorkAndRollsBackPartialWork(t *testing.T) {
	connection, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "transactions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	database := &WaDB{db: connection, query: New(connection)}
	if _, err := connection.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := database.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO items (id) VALUES (1)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("second operation failed")
	err = database.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO items (id) VALUES (2)"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("callback failure was lost: %v", err)
	}
	if errors.Is(err, ErrTransactionOutcomeUnknown) {
		t.Fatalf("successful rollback was marked as unconfirmed: %v", err)
	}
	var count, onlyID int
	if err := connection.QueryRow("SELECT COUNT(*), MIN(id) FROM items").Scan(&count, &onlyID); err != nil {
		t.Fatal(err)
	}
	if count != 1 || onlyID != 1 {
		t.Fatalf("transaction did not commit or roll back atomically: count=%d, id=%d", count, onlyID)
	}
}

func TestDeferredConstraintCommitFailureHasUnconfirmedOutcome(t *testing.T) {
	connection, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "deferred-constraint.db"))
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { connection.Close() })
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"CREATE TABLE parent (id INTEGER PRIMARY KEY)",
		"CREATE TABLE child (parent_id INTEGER REFERENCES parent(id) DEFERRABLE INITIALLY DEFERRED)",
	} {
		if _, err := connection.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	database := &WaDB{db: connection}
	inserted := false
	err = database.withTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO child (parent_id) VALUES (1)")
		inserted = err == nil
		return err
	})
	var constraint *sqlite.Error
	if !inserted || !errors.Is(err, ErrTransactionOutcomeUnknown) || !errors.As(err, &constraint) || constraint.Code()&0xff != sqlite3.SQLITE_CONSTRAINT {
		t.Fatalf("deferred commit failure lost its marker or driver cause: inserted=%t, error=%v", inserted, err)
	}
}

func TestBatchMutationsCommitAndFailedInsertIsAtomic(t *testing.T) {
	connection, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "batch.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	if err := runMigrations(connection); err != nil {
		t.Fatal(err)
	}
	database := &WaDB{db: connection, query: New(connection)}
	ctx := context.Background()
	command := func(id string) *models.ApplicationCommand {
		return &models.ApplicationCommand{
			Command: models.Command{Name: id, Category: models.CategoryApplication},
			ID:      id, Path: "/apps/" + id, DirUpdatedAt: time.Now(),
		}
	}
	first, second := command("first"), command("second")
	if err := database.BatchInsertCommands(ctx, []*models.ApplicationCommand{first, second}); err != nil {
		t.Fatal(err)
	}
	if err := database.BatchInsertCommands(ctx, []*models.ApplicationCommand{command("rolled-back"), first}); err == nil {
		t.Fatal("duplicate insert unexpectedly succeeded")
	}
	var count int
	if err := connection.QueryRow("SELECT COUNT(*) FROM application").Scan(&count); err != nil || count != 2 {
		t.Fatalf("failed insert committed partial work: count=%d, error=%v", count, err)
	}
	first.Name = "renamed"
	if err := database.BatchUpdateCommands(ctx, []*models.ApplicationCommand{first}); err != nil {
		t.Fatal(err)
	}
	if err := database.BatchUpdateApplicationUsage(ctx, []models.ApplicationUsageUpdate{{ID: first.ID, LastUsedAt: time.Now(), UsedCount: 3}}); err != nil {
		t.Fatal(err)
	}
	if err := database.InsertPlugin(ctx, InsertPluginParams{PackageID: "watools.plugin.test", Enabled: true, Storage: "{}"}); err != nil {
		t.Fatal(err)
	}
	if err := database.BatchUpdatePluginUsage(ctx, []models.PluginUsageUpdate{{PackageID: "watools.plugin.test", LastUsedAt: time.Now(), UsedCount: 5}}); err != nil {
		t.Fatal(err)
	}
	if err := database.DeleteCommands(ctx, []string{second.ID}); err != nil {
		t.Fatal(err)
	}
	var name string
	var applicationUsage, pluginUsage int
	if err := connection.QueryRow("SELECT name, used_count FROM application WHERE id = ?", first.ID).Scan(&name, &applicationUsage); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow("SELECT COUNT(*) FROM application").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRow("SELECT used_count FROM plugin_state WHERE package_id = ?", "watools.plugin.test").Scan(&pluginUsage); err != nil {
		t.Fatal(err)
	}
	if name != "renamed" || applicationUsage != 3 || pluginUsage != 5 || count != 1 {
		t.Fatalf("batch mutation was not committed: name=%q, appUsage=%d, pluginUsage=%d, count=%d", name, applicationUsage, pluginUsage, count)
	}
}

// The database/sql driver boundary allows deterministic begin/commit/rollback
// failures without relying on timing or a SQLite connection's recovery policy.
type transactionFaults struct {
	beginErr, commitErr, rollbackErr error
	commits, rollbacks               int
}

type transactionConnector struct{ faults *transactionFaults }

func (c transactionConnector) Connect(context.Context) (driver.Conn, error) {
	return transactionConnection{faults: c.faults}, nil
}
func (c transactionConnector) Driver() driver.Driver { return transactionDriver{faults: c.faults} }

type transactionDriver struct{ faults *transactionFaults }

func (d transactionDriver) Open(string) (driver.Conn, error) {
	return transactionConnection{faults: d.faults}, nil
}

type transactionConnection struct{ faults *transactionFaults }

func (c transactionConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("statements are unsupported by the transaction fault driver")
}
func (c transactionConnection) Close() error { return nil }
func (c transactionConnection) Begin() (driver.Tx, error) {
	if c.faults.beginErr != nil {
		return nil, c.faults.beginErr
	}
	return faultTransaction{faults: c.faults}, nil
}

type faultTransaction struct{ faults *transactionFaults }

func (t faultTransaction) Commit() error {
	t.faults.commits++
	return t.faults.commitErr
}
func (t faultTransaction) Rollback() error {
	t.faults.rollbacks++
	return t.faults.rollbackErr
}

func TestWithTxPreservesLifecycleFailures(t *testing.T) {
	beginFailure := errors.New("begin unavailable")
	callbackFailure := errors.New("query failed")
	commitFailure := errors.New("commit unavailable")
	rollbackFailure := errors.New("rollback unavailable")
	for _, test := range []struct {
		name               string
		faults             transactionFaults
		callbackErr        error
		wantErrors         []error
		wantCallback       bool
		wantUnknown        bool
		commits, rollbacks int
	}{
		{name: "begin", faults: transactionFaults{beginErr: beginFailure}, wantErrors: []error{beginFailure}},
		{name: "callback", callbackErr: callbackFailure, wantErrors: []error{callbackFailure}, wantCallback: true, rollbacks: 1},
		{name: "commit", faults: transactionFaults{commitErr: commitFailure}, wantErrors: []error{commitFailure}, wantCallback: true, wantUnknown: true, commits: 1},
		{name: "rollback", faults: transactionFaults{rollbackErr: rollbackFailure}, callbackErr: callbackFailure, wantErrors: []error{callbackFailure, rollbackFailure}, wantCallback: true, wantUnknown: true, rollbacks: 1},
		{name: "success", wantCallback: true, commits: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection := sql.OpenDB(transactionConnector{faults: &test.faults})
			t.Cleanup(func() { connection.Close() })
			database := &WaDB{db: connection}
			called := false
			err := database.withTx(context.Background(), func(*sql.Tx) error {
				called = true
				return test.callbackErr
			})
			if called != test.wantCallback || test.faults.commits != test.commits || test.faults.rollbacks != test.rollbacks {
				t.Fatalf("unexpected lifecycle: callback=%t, commits=%d, rollbacks=%d", called, test.faults.commits, test.faults.rollbacks)
			}
			for _, cause := range test.wantErrors {
				if !errors.Is(err, cause) {
					t.Fatalf("failure cause %v was lost: %v", cause, err)
				}
			}
			if errors.Is(err, ErrTransactionOutcomeUnknown) != test.wantUnknown {
				t.Fatalf("unexpected unconfirmed transaction marker: %v", err)
			}
			if len(test.wantErrors) == 0 && err != nil {
				t.Fatalf("successful commit reported an error: %v", err)
			}
		})
	}
}
