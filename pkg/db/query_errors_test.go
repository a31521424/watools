package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/samber/mo"
	"modernc.org/sqlite"

	"watools/pkg/models"
)

func applicationQueryFixture(t *testing.T) *WaDB {
	t.Helper()
	connection, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "queries.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	if err := runMigrations(connection); err != nil {
		t.Fatal(err)
	}
	return &WaDB{db: connection, query: New(connection)}
}

func TestApplicationQueriesDistinguishEmptyResultsFromMissingCommand(t *testing.T) {
	database := applicationQueryFixture(t)
	ctx := context.Background()
	for _, read := range []func(context.Context) ([]*models.ApplicationCommand, error){database.GetCommands, database.GetExpiredCommands} {
		items, err := read(ctx)
		if err != nil || len(items) != 0 {
			t.Fatalf("empty database = %v, %v", items, err)
		}
	}
	if item, err := database.GetCommandIsUpdatedDir(ctx, "/missing.app", time.Now()); item != nil || err != nil {
		t.Fatalf("no matching row = %+v, %v", item, err)
	}
	updatedAt := time.Now().Truncate(time.Second)
	command := models.NewApplicationCommand("fixture", mo.None[string](), "/fixture.app", mo.None[string](), mo.Some("fixture"), updatedAt)
	if err := database.BatchInsertCommands(ctx, []*models.ApplicationCommand{command}); err != nil {
		t.Fatal(err)
	}
	if item, err := database.GetCommandIsUpdatedDir(ctx, command.Path, updatedAt); item != nil || err != nil {
		t.Fatalf("unchanged timestamp = %+v, %v", item, err)
	}
	item, err := database.GetCommandIsUpdatedDir(ctx, command.Path, updatedAt.Add(time.Second))
	if err != nil || item == nil || item.ID != command.ID {
		t.Fatalf("changed timestamp = %+v, %v", item, err)
	}
}

func assertFailedApplicationQueries(t *testing.T, database *WaDB, ctx context.Context, check func(error) bool) {
	t.Helper()
	for _, read := range []struct {
		name string
		fn   func(context.Context) ([]*models.ApplicationCommand, error)
	}{{"commands", database.GetCommands}, {"expired", database.GetExpiredCommands}} {
		items, err := read.fn(ctx)
		if items != nil || err == nil || !check(err) {
			t.Fatalf("%s failure became a successful list or lost cause: %v, %v", read.name, items, err)
		}
	}
	item, err := database.GetCommandIsUpdatedDir(ctx, "/fixture.app", time.Now())
	if item != nil || err == nil || !check(err) {
		t.Fatalf("single-row failure became a zero command or lost cause: %+v, %v", item, err)
	}
}

func TestApplicationQueryContextFailuresKeepCancellationAndDeadline(t *testing.T) {
	database := applicationQueryFixture(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	assertFailedApplicationQueries(t, database, cancelled, func(err error) bool { return errors.Is(err, context.Canceled) })
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	assertFailedApplicationQueries(t, database, expired, func(err error) bool { return errors.Is(err, context.DeadlineExceeded) })
}

func TestApplicationQueriesPropagateClosedDatabaseAndSQLiteTableFailures(t *testing.T) {
	t.Run("closed database", func(t *testing.T) {
		database := applicationQueryFixture(t)
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		_, cause := database.db.QueryContext(context.Background(), "SELECT 1")
		if cause == nil {
			t.Fatal("closed database unexpectedly accepted query")
		}
		assertFailedApplicationQueries(t, database, context.Background(), func(err error) bool { return errors.Is(err, cause) })
	})
	t.Run("missing table", func(t *testing.T) {
		database := applicationQueryFixture(t)
		if _, err := database.db.Exec("DROP TABLE application"); err != nil {
			t.Fatal(err)
		}
		assertFailedApplicationQueries(t, database, context.Background(), func(err error) bool {
			var cause *sqlite.Error
			return errors.As(err, &cause)
		})
	})
}
