package config

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"watools/pkg/utils"
)

func TestExplicitDataDirectoryDoesNotImportLegacyData(t *testing.T) {
	legacy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(legacy, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(legacy, "data", "private.data")
	if err := os.WriteFile(legacyFile, []byte("user-data"), 0600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "development")
	t.Setenv("WATOOLS_DATA_DIR", root)
	t.Setenv("WATOOLS_CACHE_DIR", legacy)
	if got := ProjectDataDir(); got != root {
		t.Fatalf("explicit directory: %q", got)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("explicit directory imported legacy data: %v, %v", entries, err)
	}
	if got, err := os.ReadFile(legacyFile); err != nil || string(got) != "user-data" {
		t.Fatalf("legacy data changed: %q, %v", got, err)
	}
	// Resolving an explicit data directory must not even create a cache.
	cache := filepath.Join(t.TempDir(), "unused-cache")
	t.Setenv("WATOOLS_CACHE_DIR", cache)
	ProjectDataDir()
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("explicit data directory touched the cache: %v", err)
	}
}

func TestDataMigrationPreservesLegacyAndDoesNotOverwriteCurrent(t *testing.T) {
	legacy, parent := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(legacy, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(legacy, "data", "test.data")
	if err := os.WriteFile(file, []byte("old-data"), 0600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "app")
	if err := prepareDataDir(root, legacy); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(root, "data", "test.data")
	if got, err := os.ReadFile(current); err != nil || string(got) != "old-data" {
		t.Fatalf("migration: %s %v", got, err)
	}
	if err := os.WriteFile(current, []byte("new-data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareDataDir(root, legacy); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(current); string(got) != "new-data" {
		t.Fatal("current data overwritten")
	}
	if got, _ := os.ReadFile(file); string(got) != "old-data" {
		t.Fatal("legacy backup changed")
	}
}

func TestMigrationSnapshotsCommittedWAL(t *testing.T) {
	legacy := t.TempDir()
	source := filepath.Join(legacy, "data")
	os.MkdirAll(source, 0700)
	database, err := sql.Open("sqlite", utils.FileURI(filepath.Join(source, "state.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE state(value TEXT); INSERT INTO state VALUES ('preserved')"); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "new-state")
	if err := prepareDataDir(root, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "state.db-wal")); err == nil {
		t.Fatal("copied a live WAL beside the snapshot")
	}
	migrated, err := sql.Open("sqlite", utils.FileURI(filepath.Join(root, "data", "state.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer migrated.Close()
	var value string
	if err := migrated.QueryRow("SELECT value FROM state").Scan(&value); err != nil || value != "preserved" {
		t.Fatalf("WAL changes lost: %q, %v", value, err)
	}
}

func TestCorruptMigrationDoesNotPublishPartialData(t *testing.T) {
	legacy := t.TempDir()
	os.MkdirAll(filepath.Join(legacy, "data"), 0700)
	file := filepath.Join(legacy, "data", "broken.db")
	os.WriteFile(file, []byte("not a database"), 0600)
	target := filepath.Join(t.TempDir(), "state")
	if err := prepareDataDir(target, legacy); err == nil {
		t.Fatal("corrupt database migrated")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("partial migration published")
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "not a database" {
		t.Fatal("legacy data changed")
	}
}
