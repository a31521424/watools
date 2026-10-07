package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/samber/mo"

	"watools/internal/command/application"
	"watools/pkg/models"
)

type failingApplicationRepository struct {
	commands                                   []*models.ApplicationCommand
	queryErr, insertErr, updateErr, deleteErr  error
	queries, inserts, updates, deletes, usages int
}

func (r *failingApplicationRepository) GetCommands(context.Context) ([]*models.ApplicationCommand, error) {
	r.queries++
	return r.commands, r.queryErr
}
func (r *failingApplicationRepository) BatchInsertCommands(context.Context, []*models.ApplicationCommand) error {
	r.inserts++
	return r.insertErr
}
func (r *failingApplicationRepository) BatchUpdateCommands(context.Context, []*models.ApplicationCommand) error {
	r.updates++
	return r.updateErr
}
func (r *failingApplicationRepository) DeleteCommands(context.Context, []string) error {
	r.deletes++
	return r.deleteErr
}
func (r *failingApplicationRepository) BatchUpdateApplicationUsage(context.Context, []models.ApplicationUsageUpdate) error {
	r.usages++
	return nil
}

type controlledApplicationSource struct {
	commands             []*models.ApplicationCommand
	err                  error
	pathInfos            []application.AppPathInfo
	onScan               func()
	scans, paths, parses int
}

func (s *controlledApplicationSource) Scan() ([]*models.ApplicationCommand, error) {
	s.scans++
	if s.onScan != nil {
		s.onScan()
	}
	return s.commands, s.err
}
func (s *controlledApplicationSource) Paths() []application.AppPathInfo {
	s.paths++
	return s.pathInfos
}
func (s *controlledApplicationSource) Parse(path string) (*models.ApplicationCommand, error) {
	s.parses++
	if len(s.commands) > 0 && s.err == nil {
		copy := *s.commands[0]
		copy.Path = path
		return &copy, nil
	}
	return nil, s.err
}

type retainedApplicationRunner struct{ triggers int }

func (r *retainedApplicationRunner) GetTriggerID() string         { return "retained" }
func (r *retainedApplicationRunner) OnTrigger() error             { r.triggers++; return nil }
func (r *retainedApplicationRunner) GetMetadata() *models.Command { return &models.Command{} }

func testApplicationCommand(id string) *models.ApplicationCommand {
	return &models.ApplicationCommand{Command: models.Command{TriggerID: id, Name: id, Category: models.CategoryApplication},
		ID: id, Path: "/" + id + ".app", IconPath: mo.Some("/fixture-icon.png")}
}

func TestApplicationQueryFailureDoesNotScanWriteNotifyOrReplaceRunners(t *testing.T) {
	cause := errors.New("application query failed")
	repository := &failingApplicationRepository{commands: []*models.ApplicationCommand{testApplicationCommand("partial")}, queryErr: cause}
	source := &controlledApplicationSource{}
	launch := NewWaLaunchWithDependencies(repository, source)
	notifications := 0
	launch.notify = func() { notifications++ }
	retained := &retainedApplicationRunner{}
	launch.applicationRunner = []models.CommandRunner{retained}
	items, err := launch.GetApplicationCommands()
	if items != nil || !errors.Is(err, cause) {
		t.Fatalf("failed query became commands or lost cause: %+v, %v", items, err)
	}
	if err := launch.updateApplications(); !errors.Is(err, cause) {
		t.Fatalf("reconciliation lost query failure: %v", err)
	}
	if source.scans+source.paths+source.parses != 0 || repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 {
		t.Fatal("query failure initiated scan, mutation or notification")
	}
	if err := launch.TriggerCommand("retained", models.CategoryApplication); err != nil || retained.triggers != 1 {
		t.Fatalf("failed load replaced old runners: %v", err)
	}
}

func TestEmptyApplicationQueryScansAndPropagatesScanOrInsertFailures(t *testing.T) {
	for _, stage := range []string{"scan", "insert", "success"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New(stage + " failed")
			repository := &failingApplicationRepository{}
			source := &controlledApplicationSource{commands: []*models.ApplicationCommand{testApplicationCommand("scanned")}}
			if stage == "scan" {
				source.err = cause
			}
			if stage == "insert" {
				repository.insertErr = cause
			}
			launch := NewWaLaunchWithDependencies(repository, source)
			retained := &retainedApplicationRunner{}
			launch.applicationRunner = []models.CommandRunner{retained}
			notifications := 0
			launch.notify = func() { notifications++ }
			items, err := launch.GetApplicationCommands()
			if stage == "success" {
				if err != nil || len(items) != 1 || len(launch.applicationRunner) != 1 || launch.applicationRunner[0].GetTriggerID() != "scanned" {
					t.Fatalf("successful scan did not publish committed runners: %+v, %v", items, err)
				}
			} else {
				if items != nil || !errors.Is(err, cause) {
					t.Fatalf("%s failure = %+v, %v", stage, items, err)
				}
				if len(launch.applicationRunner) != 1 || launch.applicationRunner[0] != retained {
					t.Fatal("failed initialization replaced retained runners")
				}
			}
			wantInserts := 1
			if stage == "scan" {
				wantInserts = 0
			}
			if source.scans != 1 || repository.inserts != wantInserts || repository.updates+repository.deletes != 0 || notifications != 0 {
				t.Fatalf("unexpected side effects: source=%+v repository=%+v notifications=%d", source, repository, notifications)
			}
		})
	}
}

func TestExistingApplicationQueryDoesNotScanOrRewriteDatabase(t *testing.T) {
	repository := &failingApplicationRepository{commands: []*models.ApplicationCommand{testApplicationCommand("existing")}}
	source := &controlledApplicationSource{}
	launch := NewWaLaunchWithDependencies(repository, source)
	items, err := launch.GetApplicationCommands()
	if err != nil || len(items) != 1 || items[0].ID != "existing" || source.scans != 0 || repository.inserts+repository.updates+repository.deletes != 0 {
		t.Fatalf("existing query changed data or scanned disk: %+v, %v", items, err)
	}
}

func TestApplicationScanCancelledBeforeCompletionKeepsRunnersAndDoesNotWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repository := &failingApplicationRepository{}
	source := &controlledApplicationSource{onScan: cancel}
	launch := NewWaLaunchWithDependencies(repository, source)
	launch.ctx = ctx
	retained := &retainedApplicationRunner{}
	launch.applicationRunner = []models.CommandRunner{retained}
	items, err := launch.GetApplicationCommands()
	if items != nil || !errors.Is(err, context.Canceled) || repository.inserts != 0 || len(launch.applicationRunner) != 1 || launch.applicationRunner[0] != retained {
		t.Fatalf("late cancelled scan replaced runners or wrote data: %+v, %v", items, err)
	}
}

func TestReconciliationWriteFailureStopsLaterStagesAndOnlyNotifiesCommittedChanges(t *testing.T) {
	for _, stage := range []string{"update", "delete", "insert"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New(stage + " failed")
			repository := &failingApplicationRepository{}
			source := &controlledApplicationSource{commands: []*models.ApplicationCommand{testApplicationCommand("parsed")},
				pathInfos: []application.AppPathInfo{{Path: filepath.Join(t.TempDir(), "new.app")}}}
			if stage != "insert" {
				path := filepath.Join(t.TempDir(), "existing.app")
				if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
				existing := testApplicationCommand("existing")
				existing.Path = path
				existing.DirUpdatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				missing := testApplicationCommand("missing")
				missing.Path = filepath.Join(t.TempDir(), "missing.app")
				repository.commands = []*models.ApplicationCommand{existing, missing}
			}
			switch stage {
			case "update":
				repository.updateErr = cause
			case "delete":
				repository.deleteErr = cause
			case "insert":
				repository.insertErr = cause
			}
			launch := NewWaLaunchWithDependencies(repository, source)
			notifications := 0
			launch.notify = func() { notifications++ }
			err := launch.updateApplications()
			if !errors.Is(err, cause) {
				t.Fatalf("lost %s cause: %v", stage, err)
			}
			wantUpdates, wantDeletes, wantInserts, wantPaths, wantNotifications := 0, 0, 0, 0, 0
			switch stage {
			case "update":
				wantUpdates = 1
			case "delete":
				wantUpdates, wantDeletes, wantNotifications = 1, 1, 1
			case "insert":
				wantPaths, wantInserts = 1, 1
			}
			if repository.updates != wantUpdates || repository.deletes != wantDeletes || repository.inserts != wantInserts || source.paths != wantPaths || notifications != wantNotifications {
				t.Fatalf("%s failure continued later stages or reported uncommitted changes: repository=%+v source=%+v notifications=%d", stage, repository, source, notifications)
			}
		})
	}
}

// Pause the first successful read at the next cancellation check, before its
// runners are published. This exposes the read/publish interleaving without
// relying on scheduler delays or a platform application scan.
type applicationPublicationContext struct {
	context.Context
	armed          atomic.Bool
	paused, resume chan struct{}
}

func (c *applicationPublicationContext) Err() error {
	if c.armed.CompareAndSwap(true, false) {
		close(c.paused)
		<-c.resume
	}
	return c.Context.Err()
}

type sequencedApplicationRepository struct {
	failingApplicationRepository
	ctx   *applicationPublicationContext
	reads int
}

func (r *sequencedApplicationRepository) GetCommands(context.Context) ([]*models.ApplicationCommand, error) {
	r.reads++
	if r.reads == 1 {
		r.ctx.armed.Store(true)
		return []*models.ApplicationCommand{testApplicationCommand("old")}, nil
	}
	return []*models.ApplicationCommand{testApplicationCommand("new")}, nil
}

func TestConcurrentApplicationLoadsCannotPublishAnOlderSnapshotLast(t *testing.T) {
	ctx := &applicationPublicationContext{Context: context.Background(), paused: make(chan struct{}), resume: make(chan struct{})}
	repository := &sequencedApplicationRepository{ctx: ctx}
	launch := NewWaLaunchWithDependencies(repository, &controlledApplicationSource{})
	launch.ctx = ctx
	type result struct {
		items []*models.ApplicationCommand
		err   error
	}
	oldResult, newResult := make(chan result, 1), make(chan result, 1)
	var resumeOnce sync.Once
	resume := func() { resumeOnce.Do(func() { close(ctx.resume) }) }
	defer resume()
	go func() { items, err := launch.GetApplicationCommands(); oldResult <- result{items, err} }()
	select {
	case <-ctx.paused:
	case <-time.After(3 * time.Second):
		t.Fatal("first read did not reach publication")
	}
	// If read ownership ended too early, force the newer request to finish
	// before allowing the old publisher to continue. Otherwise let the first
	// owner finish, after which the newer read must publish last.
	unprotected := ApiMutex.TryLock()
	if unprotected {
		ApiMutex.Unlock()
	}
	go func() { items, err := launch.GetApplicationCommands(); newResult <- result{items, err} }()
	await := func(results <-chan result) result {
		t.Helper()
		select {
		case value := <-results:
			return value
		case <-time.After(3 * time.Second):
			t.Fatal("application load did not finish")
			return result{}
		}
	}
	var newer result
	if unprotected {
		newer = await(newResult)
	}
	resume()
	older := await(oldResult)
	if !unprotected {
		newer = await(newResult)
	}
	if older.err != nil || newer.err != nil || len(older.items) != 1 || len(newer.items) != 1 || older.items[0].ID != "old" || newer.items[0].ID != "new" {
		t.Fatalf("unexpected concurrent loads: old=%+v new=%+v", older, newer)
	}
	launch.runnersMu.RLock()
	defer launch.runnersMu.RUnlock()
	if len(launch.applicationRunner) != 1 || launch.applicationRunner[0].GetTriggerID() != "new" {
		t.Fatal("older query overwrote the runners published by a newer query")
	}
}
