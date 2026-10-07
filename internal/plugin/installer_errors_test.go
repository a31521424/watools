package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type repositoryFailure struct{ cause error }

func (e *repositoryFailure) Error() string { return e.cause.Error() }
func (e *repositoryFailure) Unwrap() error { return e.cause }

type uninstallFailureRepository struct {
	*testPluginRepository
	beforeDelete func()
}

func (r *uninstallFailureRepository) DeletePlugin(ctx context.Context, id string) error {
	r.beforeDelete()
	return r.testPluginRepository.DeletePlugin(ctx, id)
}

func TestRevisionCommitAndRegistrationRollbackErrorsRemainDiscoverable(t *testing.T) {
	rollbackCause := errors.New("registration rollback unavailable")
	rollbackFailure := &repositoryFailure{cause: rollbackCause}
	repo := &testPluginRepository{deleteErr: rollbackFailure}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	pluginDir := filepath.Join(pi.pluginsDir, "watools.plugin.test")
	// A directory in place of the revision pointer makes the actual rename fail.
	if err := os.MkdirAll(filepath.Join(pluginDir, "current.json"), 0700); err != nil {
		t.Fatal(err)
	}
	err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "new"))
	var commitFailure *os.LinkError
	if !errors.As(err, &commitFailure) || commitFailure.Op != "rename" {
		t.Fatalf("revision commit failure missing from error chain: %v", err)
	}
	if !errors.Is(err, commitFailure.Err) || !errors.Is(err, rollbackCause) {
		t.Fatalf("errors.Is cannot find both failure causes: %v", err)
	}
	var gotRollback *repositoryFailure
	if !errors.As(err, &gotRollback) || gotRollback != rollbackFailure {
		t.Fatalf("registration rollback failure missing from error chain: %v", err)
	}
	if repo.insertCalls != 1 || repo.deleteCalls != 1 || len(repo.states) != 1 {
		t.Fatal("expected registration followed by a failed rollback")
	}
	versions, readErr := os.ReadDir(filepath.Join(pluginDir, "versions"))
	if readErr != nil || len(versions) != 0 {
		t.Fatalf("failed commit left an unpublished revision: %v", readErr)
	}
}

func TestUninstallDeleteAndRestoreErrorsRemainDiscoverable(t *testing.T) {
	deleteCause := errors.New("database delete unavailable")
	deleteFailure := &repositoryFailure{cause: deleteCause}
	repo := &uninstallFailureRepository{testPluginRepository: &testPluginRepository{}}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "old")); err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(pi.pluginsDir, "watools.plugin.test")
	repo.deleteErr = deleteFailure
	repo.beforeDelete = func() {
		// The plugin has already moved to trash. A nonempty conflicting directory
		// prevents the real restore rename after the database delete fails.
		if err := os.Mkdir(pluginDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "conflict"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err := pi.UninstallPlugin("watools.plugin.test")
	var gotDelete *repositoryFailure
	var restoreFailure *os.LinkError
	if !errors.As(err, &gotDelete) || gotDelete != deleteFailure || !errors.As(err, &restoreFailure) {
		t.Fatalf("errors.As cannot find both uninstall failures: %v", err)
	}
	if !errors.Is(err, deleteCause) || !errors.Is(err, restoreFailure.Err) {
		t.Fatalf("errors.Is cannot find both uninstall failure causes: %v", err)
	}
	if len(repo.states) != 1 || repo.deleteCalls != 1 {
		t.Fatal("failed uninstall lost the database record")
	}
	removed, globErr := filepath.Glob(filepath.Join(pi.pluginsDir, ".removed-*", "versions", "*", "app.js"))
	if globErr != nil || len(removed) != 1 {
		t.Fatalf("failed restore lost recoverable plugin files: %v", globErr)
	}
}
