package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/samber/mo"
	"modernc.org/sqlite"

	"watools/config"
	"watools/internal/command/application"
	"watools/pkg/db"
	"watools/pkg/models"
)

// Store independent snapshots and commit each batch atomically. In particular,
// callers cannot accidentally make a failed mutation succeed through aliasing.
type reconcileRepository struct {
	mu                                         sync.Mutex
	items                                      []*models.ApplicationCommand
	queryErr, insertErr, updateErr, deleteErr  error
	queries, inserts, updates, deletes, usages int
	afterCommit                                func()
}

func cloneReconcileCommands(items []*models.ApplicationCommand) []*models.ApplicationCommand {
	clones := make([]*models.ApplicationCommand, len(items))
	for i, item := range items {
		copy := *item
		clones[i] = &copy
	}
	return clones
}

func (r *reconcileRepository) GetCommands(ctx context.Context) ([]*models.ApplicationCommand, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.queryErr != nil {
		return nil, r.queryErr
	}
	return cloneReconcileCommands(r.items), nil
}

func (r *reconcileRepository) BatchInsertCommands(ctx context.Context, items []*models.ApplicationCommand) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inserts++
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.insertErr != nil {
		return r.insertErr
	}
	next := cloneReconcileCommands(r.items)
	for _, item := range items {
		for _, old := range next {
			if old.ID == item.ID || old.Path == item.Path {
				return errors.New("duplicate application identity or path")
			}
		}
		next = append(next, cloneReconcileCommands([]*models.ApplicationCommand{item})[0])
	}
	r.items = next
	if r.afterCommit != nil {
		r.afterCommit()
	}
	return nil
}

func (r *reconcileRepository) BatchUpdateCommands(ctx context.Context, items []*models.ApplicationCommand) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates++
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.updateErr != nil {
		return r.updateErr
	}
	next := cloneReconcileCommands(r.items)
	for _, item := range items {
		found := false
		for i, old := range next {
			if old.ID == item.ID {
				next[i] = cloneReconcileCommands([]*models.ApplicationCommand{item})[0]
				found = true
			}
		}
		if !found {
			return errors.New("update missing application")
		}
	}
	r.items = next
	if r.afterCommit != nil {
		r.afterCommit()
	}
	return nil
}

func (r *reconcileRepository) DeleteCommands(ctx context.Context, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deletes++
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.deleteErr != nil {
		return r.deleteErr
	}
	next := make([]*models.ApplicationCommand, 0, len(r.items))
	for _, item := range r.items {
		remove := false
		for _, id := range ids {
			if item.ID == id {
				remove = true
			}
		}
		if !remove {
			next = append(next, item)
		}
	}
	r.items = next
	if r.afterCommit != nil {
		r.afterCommit()
	}
	return nil
}

func (r *reconcileRepository) BatchUpdateApplicationUsage(ctx context.Context, updates []models.ApplicationUsageUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usages++
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, update := range updates {
		for _, item := range r.items {
			if item.ID == update.ID {
				item.UsedCount = int64(update.UsedCount)
				item.LastUsedAt = mo.Some(update.LastUsedAt)
			}
		}
	}
	return nil
}

func (r *reconcileRepository) snapshot() []*models.ApplicationCommand {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneReconcileCommands(r.items)
}

type reconcileSource struct {
	paths    []application.AppPathInfo
	pathsErr error
	parse    func(string) (*models.ApplicationCommand, error)
	parses   atomic.Int64
}

func (s *reconcileSource) Scan() ([]*models.ApplicationCommand, error) {
	var result []*models.ApplicationCommand
	for _, path := range s.paths {
		item, err := s.Parse(path.Path)
		if err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, s.pathsErr
}
func (s *reconcileSource) Paths() []application.AppPathInfo { return s.paths }
func (s *reconcileSource) PathsWithError() ([]application.AppPathInfo, error) {
	return s.paths, s.pathsErr
}
func (s *reconcileSource) Parse(path string) (*models.ApplicationCommand, error) {
	s.parses.Add(1)
	return s.parse(path)
}

func reconcileFixture(t *testing.T, name string) *models.ApplicationCommand {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".app")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	return &models.ApplicationCommand{
		Command: models.Command{TriggerID: "stable-trigger-" + name, Name: name, Category: models.CategoryApplication,
			UsedCount: 7, LastUsedAt: mo.Some(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))},
		ID: "stable-id-" + name, Path: path, IconPath: mo.Some("fixture-icon"),
		DirUpdatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func reconcileCandidate(path, name string) *models.ApplicationCommand {
	return &models.ApplicationCommand{Command: models.Command{TriggerID: "new-trigger", Name: name,
		Category: models.CategoryApplication}, ID: "new-id", Path: path, IconPath: mo.Some("new-icon"),
		DirUpdatedAt: time.Now()}
}

func assertReconcileIdentity(t *testing.T, got, old *models.ApplicationCommand) {
	t.Helper()
	if got.ID != old.ID || got.TriggerID != old.TriggerID || got.UsedCount != old.UsedCount || got.LastUsedAt != old.LastUsedAt {
		t.Fatalf("reconciliation changed identity or usage: got=%+v old=%+v", got, old)
	}
}

func TestReconcileParseFaultRetainsExistingApplication(t *testing.T) {
	for _, entry := range []struct {
		name  string
		cause error
	}{
		{"unreadable", application.ErrApplicationUnreadable},
		{"invalid-metadata", application.ErrInvalidMetadata},
		{"not-application", application.ErrNotApplication},
		{"unknown-io", errors.New("transient read failure")},
	} {
		for _, mode := range []string{"periodic", "watcher"} {
			t.Run(entry.name+"/"+mode, func(t *testing.T) {
				old := reconcileFixture(t, "existing")
				repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
				source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
					return nil, fmt.Errorf("read metadata %s: %w", path, entry.cause)
				}}
				launch := NewWaLaunchWithDependencies(repository, source)
				notifications := 0
				launch.notify = func() { notifications++ }
				var err error
				if mode == "periodic" {
					err = launch.updateApplications()
				} else {
					err = launch.OnAppPathChanged(old.Path)
				}
				if !errors.Is(err, entry.cause) {
					t.Fatalf("parse cause lost: %v", err)
				}
				got := repository.snapshot()
				if len(got) != 1 || got[0].Name != old.Name || repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 {
					t.Fatalf("parse fault changed committed application: %+v notifications=%d", got, notifications)
				}
				assertReconcileIdentity(t, got[0], old)
			})
		}
	}
}

func TestReconcileOnlyConfirmedMissingApplicationIsDeleted(t *testing.T) {
	for _, cause := range []string{"missing-root", "parse-not-found"} {
		for _, mode := range []string{"periodic", "watcher"} {
			t.Run(cause+"/"+mode, func(t *testing.T) {
				old := reconcileFixture(t, "existing")
				if cause == "missing-root" {
					if err := os.Remove(old.Path); err != nil {
						t.Fatal(err)
					}
				}
				repository := &reconcileRepository{items: []*models.ApplicationCommand{old}}
				source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
					if cause == "parse-not-found" {
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
					}
					return nil, fmt.Errorf("confirmed absent: %w", application.ErrApplicationNotFound)
				}}
				launch := NewWaLaunchWithDependencies(repository, source)
				notifications := 0
				launch.notify = func() { notifications++ }
				var err error
				if mode == "periodic" {
					err = launch.updateApplications()
				} else {
					err = launch.OnAppPathChanged(old.Path)
				}
				if err != nil {
					t.Fatalf("confirmed deletion failed: %v", err)
				}
				if len(repository.snapshot()) != 0 || repository.deletes != 1 || notifications != 1 {
					t.Fatalf("confirmed absence did not delete and notify: %+v notifications=%d", repository.snapshot(), notifications)
				}
				if cause == "missing-root" && source.parses.Load() != 0 {
					t.Fatal("missing root unexpectedly parsed")
				}
			})
		}
	}
}

func TestPeriodicMissingRootRecreatedDuringAnotherParseIsRetained(t *testing.T) {
	recreated, changing := reconcileFixture(t, "recreated"), reconcileFixture(t, "changing")
	if err := os.Remove(recreated.Path); err != nil {
		t.Fatal(err)
	}
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{recreated, changing})}
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		if err := os.Mkdir(recreated.Path, 0700); err != nil {
			t.Fatal(err)
		}
		return reconcileCandidate(path, "changed"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.updateApplications(); err != nil {
		t.Fatal(err)
	}
	got := repository.snapshot()
	if len(got) != 2 || got[0].Name != recreated.Name || got[1].Name != "changed" || repository.deletes != 0 || notifications != 1 {
		t.Fatalf("stale absence deleted a replacement root: %+v deletes=%d notifications=%d", got, repository.deletes, notifications)
	}
	assertReconcileIdentity(t, got[0], recreated)
}

func TestPathParseNotFoundCannotDeleteRootThatHasReappeared(t *testing.T) {
	for _, mode := range []string{"periodic", "watcher"} {
		t.Run(mode, func(t *testing.T) {
			old := reconcileFixture(t, "existing")
			repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
			source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				return nil, fmt.Errorf("root disappeared during parse: %w", application.ErrApplicationNotFound)
			}}
			launch := NewWaLaunchWithDependencies(repository, source)
			notifications := 0
			launch.notify = func() { notifications++ }
			var err error
			if mode == "periodic" {
				err = launch.updateApplications()
			} else {
				err = launch.OnAppPathChanged(old.Path)
			}
			if err != nil && !errors.Is(err, application.ErrApplicationNotFound) {
				t.Fatalf("late absence lost its parse cause: %v", err)
			}
			got := repository.snapshot()
			if len(got) != 1 || got[0].Name != old.Name || repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 {
				t.Fatalf("late parse absence erased a replacement root: %+v notifications=%d", got, notifications)
			}
			assertReconcileIdentity(t, got[0], old)
		})
	}
}

func TestRepeatedPathEventsKeepIdentityAndUsage(t *testing.T) {
	old := reconcileFixture(t, "existing")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		return reconcileCandidate(path, "replacement"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	for i := 0; i < 4; i++ {
		if err := launch.OnAppPathChanged(old.Path); err != nil {
			t.Fatal(err)
		}
	}
	got := repository.snapshot()
	if len(got) != 1 || got[0].Name != "replacement" || repository.inserts != 0 {
		t.Fatalf("duplicate event changed path membership: %+v", got)
	}
	assertReconcileIdentity(t, got[0], old)
}

func TestPathEventQueryFailureDoesNotParseWriteOrNotify(t *testing.T) {
	cause := errors.New("database read unavailable")
	repository := &reconcileRepository{queryErr: cause}
	source := &reconcileSource{parse: func(string) (*models.ApplicationCommand, error) {
		t.Fatal("query failure parsed path")
		return nil, nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.OnAppPathChanged(filepath.Join(t.TempDir(), "existing.app")); !errors.Is(err, cause) {
		t.Fatalf("lost query cause: %v", err)
	}
	if repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 {
		t.Fatal("query failure mutated or notified")
	}
}

func TestReconcilePathEnumerationFaultPreservesRecords(t *testing.T) {
	old := reconcileFixture(t, "existing")
	info, err := os.Stat(old.Path)
	if err != nil {
		t.Fatal(err)
	}
	old.DirUpdatedAt = info.ModTime()
	cause := errors.New("application directory unreadable")
	repository := &reconcileRepository{items: []*models.ApplicationCommand{old}}
	source := &reconcileSource{pathsErr: cause, parse: func(string) (*models.ApplicationCommand, error) { t.Fatal("unchanged record parsed"); return nil, nil }}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.updateApplications(); !errors.Is(err, cause) {
		t.Fatalf("enumeration failure became success: %v", err)
	}
	if len(repository.snapshot()) != 1 || repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 {
		t.Fatal("enumeration failure changed stored records")
	}
}

func TestReconcilePartialEnumerationCommitsValidPathAndReturnsFailure(t *testing.T) {
	old := reconcileFixture(t, "existing")
	info, err := os.Stat(old.Path)
	if err != nil {
		t.Fatal(err)
	}
	old.DirUpdatedAt = info.ModTime()
	fresh := reconcileFixture(t, "fresh")
	cause := errors.New("another application directory unreadable")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
	source := &reconcileSource{paths: []application.AppPathInfo{{Path: fresh.Path}, {Path: fresh.Path}}, pathsErr: cause,
		parse: func(path string) (*models.ApplicationCommand, error) { return reconcileCandidate(path, "fresh"), nil }}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.updateApplications(); !errors.Is(err, cause) {
		t.Fatalf("partial enumeration lost failure: %v", err)
	}
	got := repository.snapshot()
	if len(got) != 2 || got[0].Name != old.Name || repository.inserts != 1 || notifications != 1 || source.parses.Load() != 1 {
		t.Fatalf("partial enumeration failed to preserve/commit deduplicated paths: %+v notifications=%d", got, notifications)
	}
	assertReconcileIdentity(t, got[0], old)
}

func TestReconcileParseFailureRetainsBadRecordWhileCommittingGoodRecord(t *testing.T) {
	good, bad := reconcileFixture(t, "good"), reconcileFixture(t, "bad")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{good, bad})}
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		if path == bad.Path {
			return nil, fmt.Errorf("decode damaged plist: %w", application.ErrInvalidMetadata)
		}
		return reconcileCandidate(path, "updated"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.updateApplications(); !errors.Is(err, application.ErrInvalidMetadata) {
		t.Fatalf("partial parse failure lost cause: %v", err)
	}
	got := repository.snapshot()
	if len(got) != 2 || got[0].Name != "updated" || got[1].Name != bad.Name || repository.updates != 1 || repository.deletes != 0 || notifications != 1 {
		t.Fatalf("parse failure prevented valid commit or removed invalid record: %+v notifications=%d", got, notifications)
	}
	assertReconcileIdentity(t, got[0], good)
	assertReconcileIdentity(t, got[1], bad)
}

func awaitReconcile(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("reconciliation did not finish")
		return nil
	}
}

func TestPeriodicRefreshAndPathEventReparseUnderOneOwner(t *testing.T) {
	old := reconcileFixture(t, "existing")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
	paused, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	defer release()
	var calls atomic.Int64
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		if calls.Add(1) == 1 {
			close(paused)
			<-resume
			return reconcileCandidate(path, "periodic-snapshot"), nil
		}
		return reconcileCandidate(path, "latest-disk-state"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	periodic, event := make(chan error, 1), make(chan error, 1)
	go func() { periodic <- launch.updateApplications() }()
	select {
	case <-paused:
	case <-time.After(3 * time.Second):
		t.Fatal("periodic parse did not pause")
	}
	unprotected := ApiMutex.TryLock()
	if unprotected {
		ApiMutex.Unlock()
	}
	go func() { event <- launch.OnAppPathChanged(old.Path) }()
	if unprotected {
		if err := awaitReconcile(t, event); err != nil {
			t.Fatal(err)
		}
	}
	release()
	if err := awaitReconcile(t, periodic); err != nil {
		t.Fatal(err)
	}
	if !unprotected {
		if err := awaitReconcile(t, event); err != nil {
			t.Fatal(err)
		}
	}
	got := repository.snapshot()
	if len(got) != 1 || got[0].Name != "latest-disk-state" || repository.inserts != 0 {
		t.Fatalf("stale periodic snapshot defeated path event: %+v", got)
	}
	assertReconcileIdentity(t, got[0], old)
}

func TestConcurrentUsageSurvivesReconcileSnapshot(t *testing.T) {
	for _, mode := range []string{"periodic", "watcher"} {
		t.Run(mode, func(t *testing.T) {
			old := reconcileFixture(t, "existing")
			repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
			paused, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			defer release()
			source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
				close(paused)
				<-resume
				return reconcileCandidate(path, "replacement"), nil
			}}
			launch := NewWaLaunchWithDependencies(repository, source)
			refresh, usage := make(chan error, 1), make(chan error, 1)
			go func() {
				if mode == "periodic" {
					refresh <- launch.updateApplications()
				} else {
					refresh <- launch.OnAppPathChanged(old.Path)
				}
			}()
			select {
			case <-paused:
			case <-time.After(3 * time.Second):
				t.Fatal("parse did not pause")
			}
			unprotected := ApiMutex.TryLock()
			if unprotected {
				ApiMutex.Unlock()
			}
			usedAt := time.Date(2026, 10, 7, 1, 2, 3, 0, time.UTC)
			go func() {
				usage <- launch.UpdateApplicationUsage([]models.ApplicationUsageUpdate{{ID: old.ID, UsedCount: 29, LastUsedAt: usedAt}})
			}()
			if unprotected {
				if err := awaitReconcile(t, usage); err != nil {
					t.Fatal(err)
				}
			}
			release()
			if err := awaitReconcile(t, refresh); err != nil {
				t.Fatal(err)
			}
			if !unprotected {
				if err := awaitReconcile(t, usage); err != nil {
					t.Fatal(err)
				}
			}
			got := repository.snapshot()
			if len(got) != 1 || got[0].UsedCount != 29 || got[0].LastUsedAt != mo.Some(usedAt) || got[0].ID != old.ID || got[0].TriggerID != old.TriggerID {
				t.Fatalf("concurrent usage was lost: %+v", got)
			}
		})
	}
}

func TestPeriodicDiscoveryAndPathEventCannotInsertTheSamePathTwice(t *testing.T) {
	fresh := reconcileFixture(t, "fresh")
	repository := &reconcileRepository{}
	paused, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	defer release()
	var calls atomic.Int64
	source := &reconcileSource{paths: []application.AppPathInfo{{Path: fresh.Path}}, parse: func(path string) (*models.ApplicationCommand, error) {
		if calls.Add(1) == 1 {
			close(paused)
			<-resume
			return reconcileCandidate(path, "discovered"), nil
		}
		return reconcileCandidate(path, "latest"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	periodic, event := make(chan error, 1), make(chan error, 1)
	go func() { periodic <- launch.updateApplications() }()
	select {
	case <-paused:
	case <-time.After(3 * time.Second):
		t.Fatal("discovery parse did not pause")
	}
	unprotected := ApiMutex.TryLock()
	if unprotected {
		ApiMutex.Unlock()
	}
	go func() { event <- launch.OnAppPathChanged(fresh.Path) }()
	if unprotected {
		if err := awaitReconcile(t, event); err != nil {
			t.Fatal(err)
		}
	}
	release()
	if err := awaitReconcile(t, periodic); err != nil {
		t.Fatalf("concurrent discovery inserted an already committed path: %v", err)
	}
	if !unprotected {
		if err := awaitReconcile(t, event); err != nil {
			t.Fatal(err)
		}
	}
	got := repository.snapshot()
	if len(got) != 1 || got[0].Name != "latest" || repository.inserts != 1 {
		t.Fatalf("discovery/path event duplicated or lost latest path: %+v inserts=%d", got, repository.inserts)
	}
}

func TestCancelledParseCannotWriteOrNotifyLateResult(t *testing.T) {
	for _, mode := range []string{"periodic", "watcher"} {
		t.Run(mode, func(t *testing.T) {
			old := reconcileFixture(t, "existing")
			repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
				cancel()
				return reconcileCandidate(path, "late"), nil
			}}
			launch := NewWaLaunchWithDependencies(repository, source)
			launch.ctx = ctx
			notifications := 0
			launch.notify = func() { notifications++ }
			var err error
			if mode == "periodic" {
				err = launch.updateApplications()
			} else {
				err = launch.OnAppPathChanged(old.Path)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("late parse lost cancellation: %v", err)
			}
			if repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 || repository.snapshot()[0].Name != old.Name {
				t.Fatal("cancelled parse wrote or notified")
			}
		})
	}
}

func TestCancellationAfterCommitSuppressesChangeNotification(t *testing.T) {
	old := reconcileFixture(t, "existing")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old}), afterCommit: cancel}
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		return reconcileCandidate(path, "committed"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	launch.ctx = ctx
	notifications := 0
	launch.notify = func() { notifications++ }
	_ = launch.OnAppPathChanged(old.Path)
	if repository.snapshot()[0].Name != "committed" || notifications != 0 {
		t.Fatal("cancelled service notified after a committed mutation")
	}
}

func TestCancelledWatcherTaskCannotWriteWhileServiceRemainsLive(t *testing.T) {
	old := reconcileFixture(t, "existing")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
	paused, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	defer release()
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		close(paused)
		<-resume
		return reconcileCandidate(path, "late-task-result"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	var notifications atomic.Int64
	launch.notify = func() { notifications.Add(1) }
	taskCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- launch.ReconcileApplicationPath(taskCtx, old.Path) }()
	select {
	case <-paused:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher task parse did not pause")
	}
	cancel()
	release()
	if err := awaitReconcile(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("task cancellation lost: %v", err)
	}
	if launch.ctx.Err() != nil || repository.inserts+repository.updates+repository.deletes != 0 || notifications.Load() != 0 {
		t.Fatal("cancelled watcher task wrote or notified while the service remained live")
	}
	got := repository.snapshot()
	if len(got) != 1 || got[0].Name != old.Name {
		t.Fatal("late watcher result replaced the committed application")
	}
}

func TestCancelledServiceCannotWriteWhileWatcherTaskRemainsLive(t *testing.T) {
	old := reconcileFixture(t, "existing")
	repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old})}
	serviceCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := &reconcileSource{parse: func(path string) (*models.ApplicationCommand, error) {
		cancel()
		return reconcileCandidate(path, "late-service-result"), nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	launch.ctx = serviceCtx
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.ReconcileApplicationPath(context.Background(), old.Path); !errors.Is(err, context.Canceled) {
		t.Fatalf("service cancellation lost: %v", err)
	}
	if repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 || repository.snapshot()[0].Name != old.Name {
		t.Fatal("live task committed a result after service cancellation")
	}
}

func TestReconcileBatchFailureReportsOnlyCommittedState(t *testing.T) {
	for _, stage := range []string{"update", "delete", "insert"} {
		t.Run(stage, func(t *testing.T) {
			old := reconcileFixture(t, "existing")
			missing := reconcileFixture(t, "missing")
			if err := os.Remove(missing.Path); err != nil {
				t.Fatal(err)
			}
			fresh := reconcileFixture(t, "fresh")
			cause := errors.New(stage + " commit rejected")
			repository := &reconcileRepository{items: cloneReconcileCommands([]*models.ApplicationCommand{old, missing})}
			switch stage {
			case "update":
				repository.updateErr = cause
			case "delete":
				repository.deleteErr = cause
			case "insert":
				repository.insertErr = cause
			}
			source := &reconcileSource{paths: []application.AppPathInfo{{Path: fresh.Path}}, parse: func(path string) (*models.ApplicationCommand, error) {
				return reconcileCandidate(path, "replacement"), nil
			}}
			launch := NewWaLaunchWithDependencies(repository, source)
			notifications := 0
			var notified []*models.ApplicationCommand
			launch.notify = func() { notifications++; notified = repository.snapshot() }
			if err := launch.updateApplications(); !errors.Is(err, cause) {
				t.Fatalf("lost commit cause: %v", err)
			}
			got := repository.snapshot()
			wantCount, wantName, wantNotifications := 2, old.Name, 0
			if stage != "update" {
				wantName, wantNotifications = "replacement", 1
			}
			if stage == "insert" {
				wantCount = 1
			}
			if len(got) != wantCount || got[0].Name != wantName || notifications != wantNotifications {
				t.Fatalf("failure reported wrong committed state: got=%+v notifications=%d", got, notifications)
			}
			if notifications > 0 && (len(notified) != len(got) || notified[0].Name != got[0].Name) {
				t.Fatal("notification preceded commit")
			}
			assertReconcileIdentity(t, got[0], old)
		})
	}
}

func newReconcileSQLite(t *testing.T) (*db.WaDB, *sql.DB) {
	t.Helper()
	data := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", data)
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	repository, err := db.NewWaDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Error(err)
		}
	})
	control, err := sql.Open("sqlite", filepath.Join(data, "data", config.ProjectName()+".db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := control.Close(); err != nil {
			t.Error(err)
		}
	})
	return repository, control
}

func seedReconcileSQLite(t *testing.T, repository *db.WaDB, items ...*models.ApplicationCommand) {
	t.Helper()
	for _, item := range items {
		item.TriggerID = string(models.CategoryApplication) + "-" + item.ID
	}
	if err := repository.BatchInsertCommands(context.Background(), items); err != nil {
		t.Fatal(err)
	}
	var usage []models.ApplicationUsageUpdate
	for _, item := range items {
		usage = append(usage, models.ApplicationUsageUpdate{ID: item.ID, UsedCount: int(item.UsedCount), LastUsedAt: item.LastUsedAt.MustGet()})
	}
	if err := repository.BatchUpdateApplicationUsage(context.Background(), usage); err != nil {
		t.Fatal(err)
	}
}

func readReconcileSQLite(t *testing.T, repository *db.WaDB) []*models.ApplicationCommand {
	t.Helper()
	items, err := repository.GetCommands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestSQLiteReconcileCorruptMetadataRecoveryAndRootDeletion(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("real bundle metadata is macOS specific")
	}
	repository, _ := newReconcileSQLite(t)
	old := reconcileFixture(t, "existing")
	old.Description = mo.Some("previous copyright")
	contents := filepath.Join(old.Path, "Contents")
	if err := os.Mkdir(contents, 0700); err != nil {
		t.Fatal(err)
	}
	plistPath := filepath.Join(contents, "Info.plist")
	if err := os.WriteFile(plistPath, []byte("corrupt plist"), 0600); err != nil {
		t.Fatal(err)
	}
	seedReconcileSQLite(t, repository, old)
	source := &reconcileSource{parse: application.ParseApplication}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	for _, mode := range []string{"periodic", "watcher"} {
		var err error
		if mode == "periodic" {
			err = launch.updateApplications()
		} else {
			err = launch.OnAppPathChanged(old.Path)
		}
		if !errors.Is(err, application.ErrInvalidMetadata) {
			t.Fatalf("%s corrupt plist cause=%v", mode, err)
		}
		got := readReconcileSQLite(t, repository)
		if len(got) != 1 || got[0].Name != old.Name || !got[0].Description.Equal(old.Description) || !got[0].IconPath.Equal(old.IconPath) || notifications != 0 {
			t.Fatalf("corrupt plist erased SQLite record: %+v", got)
		}
		assertReconcileIdentity(t, got[0], old)
	}
	if err := os.Remove(plistPath); err != nil {
		t.Fatal(err)
	}
	if err := launch.OnAppPathChanged(old.Path); !errors.Is(err, application.ErrInvalidMetadata) {
		t.Fatalf("missing metadata classified as missing root: %v", err)
	}
	if len(readReconcileSQLite(t, repository)) != 1 || notifications != 0 {
		t.Fatal("missing plist erased application")
	}
	validPlist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleName</key><string>Recovered Application</string></dict></plist>`
	if err := os.WriteFile(plistPath, []byte(validPlist), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := launch.OnAppPathChanged(old.Path); err != nil {
			t.Fatal(err)
		}
	}
	got := readReconcileSQLite(t, repository)
	if len(got) != 1 || got[0].Name != "Recovered Application" || got[0].Description.IsSome() || got[0].IconPath.IsSome() || notifications != 1 {
		t.Fatalf("recovery or duplicate event changed identity/notification: %+v notifications=%d", got, notifications)
	}
	assertReconcileIdentity(t, got[0], old)
	if err := os.RemoveAll(old.Path); err != nil {
		t.Fatal(err)
	}
	if err := launch.OnAppPathChanged(old.Path); err != nil {
		t.Fatal(err)
	}
	if len(readReconcileSQLite(t, repository)) != 0 || notifications != 2 {
		t.Fatal("confirmed root absence was not deleted")
	}
}

func TestSQLiteReconcileBatchFaultRollsBackAndOnlyNotifiesCommittedChanges(t *testing.T) {
	for _, stage := range []string{"update", "delete", "insert"} {
		t.Run(stage, func(t *testing.T) {
			repository, control := newReconcileSQLite(t)
			old := reconcileFixture(t, "existing")
			other := reconcileFixture(t, "other")
			missing := reconcileFixture(t, "missing")
			fresh := reconcileFixture(t, "fresh")
			seedReconcileSQLite(t, repository, old, other, missing)
			if err := os.Remove(missing.Path); err != nil {
				t.Fatal(err)
			}
			var trigger string
			switch stage {
			case "update":
				trigger = `CREATE TRIGGER reconcile_fault BEFORE UPDATE ON application WHEN NEW.id='stable-id-other' BEGIN SELECT RAISE(ABORT,'update rejected'); END`
			case "delete":
				trigger = `CREATE TRIGGER reconcile_fault BEFORE DELETE ON application BEGIN SELECT RAISE(ABORT,'delete rejected'); END`
			case "insert":
				trigger = `CREATE TRIGGER reconcile_fault BEFORE INSERT ON application BEGIN SELECT RAISE(ABORT,'insert rejected'); END`
			}
			if _, err := control.Exec(trigger); err != nil {
				t.Fatal(err)
			}
			source := &reconcileSource{paths: []application.AppPathInfo{{Path: fresh.Path}}, parse: func(path string) (*models.ApplicationCommand, error) {
				return reconcileCandidate(path, "replacement"), nil
			}}
			launch := NewWaLaunchWithDependencies(repository, source)
			notifications := 0
			var notified []*models.ApplicationCommand
			launch.notify = func() { notifications++; notified = readReconcileSQLite(t, repository) }
			err := launch.updateApplications()
			var sqlErr *sqlite.Error
			if !errors.As(err, &sqlErr) {
				t.Fatalf("SQLite failure cause lost: %v", err)
			}
			got := readReconcileSQLite(t, repository)
			byID := make(map[string]*models.ApplicationCommand)
			for _, item := range got {
				byID[item.ID] = item
			}
			wantName, wantNotifications, wantCount := old.Name, 0, 3
			if stage != "update" {
				wantName, wantNotifications = "replacement", 1
			}
			if stage == "insert" {
				wantCount = 2
			}
			if len(got) != wantCount || byID[old.ID] == nil || byID[old.ID].Name != wantName || notifications != wantNotifications {
				t.Fatalf("failed SQLite batch changed/report wrong state: %+v notifications=%d", got, notifications)
			}
			if stage == "update" && byID[other.ID].Name != other.Name {
				t.Fatal("failed update batch was not rolled back")
			}
			if notifications > 0 && len(notified) != len(got) {
				t.Fatal("notification did not reflect committed SQLite state")
			}
			assertReconcileIdentity(t, byID[old.ID], old)
		})
	}
}

func TestSQLitePathReadFailureLeavesPersistedRecordAndSkipsParsing(t *testing.T) {
	repository, control := newReconcileSQLite(t)
	old := reconcileFixture(t, "existing")
	seedReconcileSQLite(t, repository, old)
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	source := &reconcileSource{parse: func(string) (*models.ApplicationCommand, error) {
		t.Fatal("closed database query initiated application parsing")
		return nil, nil
	}}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	if err := launch.OnAppPathChanged(old.Path); err == nil {
		t.Fatal("closed database read became successful path reconciliation")
	}
	if notifications != 0 || source.parses.Load() != 0 {
		t.Fatal("database read failure parsed or notified")
	}
	var id, name string
	var count int64
	if err := control.QueryRow("SELECT id,name,used_count FROM application WHERE path=?", old.Path).Scan(&id, &name, &count); err != nil {
		t.Fatal(err)
	}
	if id != old.ID || name != old.Name || count != old.UsedCount {
		t.Fatal("read failure changed the independently persisted SQLite record")
	}
}
