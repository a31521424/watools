//go:build darwin || windows

package watcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"watools/internal/command/application"
	"watools/pkg/models"
)

type watcherEventFixture struct {
	*FSWatcher
	events chan fsnotify.Event
}

func isolatedFSWatcher(t *testing.T, debounce, capacity int) *watcherEventFixture {
	t.Helper()
	config := DefaultWatcherConfig()
	config.DebounceInterval = debounce
	config.EventBufferSize = capacity
	fw, err := NewFSWatcherWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	fw.watchDirs = nil
	// Supply a separate native event stream. fsnotify owns and closes its own
	// channel, so tests must not become another producer on that channel.
	events := make(chan fsnotify.Event)
	fw.running = true
	go fw.handleEvents(events, nil)
	t.Cleanup(func() {
		if err := fw.Stop(); err != nil {
			t.Error(err)
		}
	})
	return &watcherEventFixture{FSWatcher: fw, events: events}
}

func changedTestPath(t *testing.T, exists bool) string {
	t.Helper()
	name := "Example.app"
	if runtime.GOOS == "windows" {
		name = "Example.lnk"
	}
	path := filepath.Join(t.TempDir(), name)
	if exists {
		if err := os.WriteFile(path, []byte("example"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func sendNativeEvent(t *testing.T, fw *watcherEventFixture, path string, op fsnotify.Op) {
	t.Helper()
	select {
	case fw.events <- fsnotify.Event{Name: path, Op: op}:
	case <-time.After(time.Second):
		t.Fatal("watcher stopped consuming native events")
	}
}

func receiveChange(t *testing.T, fw *watcherEventFixture, timeout time.Duration) AppChangeEvent {
	t.Helper()
	select {
	case event, ok := <-fw.EventChannel():
		if !ok {
			t.Fatal("watcher closed before publishing event")
		}
		return event
	case <-time.After(timeout):
		t.Fatal("event was not published within configured debounce")
		return AppChangeEvent{}
	}
}

func TestConfiguredDebounceMergesRepeatedAndCombinedEvents(t *testing.T) {
	fw := isolatedFSWatcher(t, 120, 7)
	path := changedTestPath(t, true)
	sendNativeEvent(t, fw, path, fsnotify.Create)
	time.Sleep(70 * time.Millisecond)
	lastEvent := time.Now()
	sendNativeEvent(t, fw, path, fsnotify.Write|fsnotify.Remove|fsnotify.Create)
	sendNativeEvent(t, fw, path, fsnotify.Write)
	select {
	case event := <-fw.EventChannel():
		t.Fatalf("published before trailing debounce: %+v", event)
	case <-time.After(65 * time.Millisecond):
	}
	event := receiveChange(t, fw, 300*time.Millisecond)
	if time.Since(lastEvent) < 110*time.Millisecond {
		t.Fatal("configured debounce did not delay the final event")
	}
	if event.Path != path || event.Type != AppAdded {
		t.Fatalf("replacement final state was lost: %+v", event)
	}
	select {
	case event := <-fw.EventChannel():
		t.Fatalf("duplicate event after debounce: %+v", event)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestConfiguredEventBufferHoldsEvents(t *testing.T) {
	fw := isolatedFSWatcher(t, 0, 3)
	for i := 0; i < 3; i++ {
		if !fw.processEvent(fsnotify.Event{Name: changedTestPath(t, false), Op: fsnotify.Create}) {
			t.Fatal("watcher canceled unexpectedly")
		}
	}
	blocked := make(chan bool, 1)
	path := changedTestPath(t, false)
	go func() {
		blocked <- fw.processEvent(fsnotify.Event{Name: path, Op: fsnotify.Create})
	}()
	select {
	case <-blocked:
		t.Fatal("configured capacity did not apply backpressure")
	case <-time.After(30 * time.Millisecond):
	}
	receiveChange(t, fw, time.Second)
	select {
	case ok := <-blocked:
		if !ok {
			t.Fatal("event was canceled")
		}
	case <-time.After(time.Second):
		t.Fatal("event remained blocked after buffer was drained")
	}
}

func TestZeroDebounceAndFinalMissingState(t *testing.T) {
	fw := isolatedFSWatcher(t, 0, 1)
	path := changedTestPath(t, false)
	sendNativeEvent(t, fw, path, fsnotify.Create|fsnotify.Write)
	if event := receiveChange(t, fw, 250*time.Millisecond); event.Type != AppRemoved {
		t.Fatalf("missing final state resurrected: %+v", event)
	}
}

func TestStopDiscardsDebouncingEvents(t *testing.T) {
	fw := isolatedFSWatcher(t, 200, 1)
	sendNativeEvent(t, fw, changedTestPath(t, true), fsnotify.Create)
	if err := fw.Stop(); err != nil {
		t.Fatal(err)
	}
	for event := range fw.EventChannel() {
		t.Fatalf("canceled debounce published a late event: %+v", event)
	}
}

func TestCanceledWatcherDoesNotPublishIntoAvailableBuffer(t *testing.T) {
	fw := isolatedFSWatcher(t, 0, 1)
	fw.cancel()
	if fw.processEvent(fsnotify.Event{Name: changedTestPath(t, false), Op: fsnotify.Create}) {
		t.Fatal("canceled event was accepted")
	}
	if err := fw.Stop(); err != nil {
		t.Fatal(err)
	}
	for event := range fw.EventChannel() {
		t.Fatalf("canceled watcher published: %+v", event)
	}
}

type pathRecordingHandler struct {
	calls  []string
	onPath func()
}

func (h *pathRecordingHandler) ReconcileApplicationPath(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	h.calls = append(h.calls, path)
	if h.onPath != nil {
		h.onPath()
	}
	return nil
}
func (h *pathRecordingHandler) OnAppAdded(*models.ApplicationCommand) error {
	panic("path handler must own parsing")
}
func (h *pathRecordingHandler) OnAppRemoved(string) error { panic("path handler must own removal") }
func (h *pathRecordingHandler) OnAppModified(*models.ApplicationCommand) error {
	panic("path handler must own parsing")
}

func isolatedManager(t *testing.T, handler AppEventHandler, config *WatcherConfig) *appWatchManager {
	t.Helper()
	manager, err := NewAppWatchManagerWithConfig(handler, context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	awm := manager.(*appWatchManager)
	awm.fsWatcher.watchDirs = nil
	t.Cleanup(func() {
		if err := awm.Stop(); err != nil {
			t.Error(err)
		}
	})
	return awm
}

func TestManagerOwnsConfigAndWiresFSOptions(t *testing.T) {
	config := DefaultWatcherConfig()
	config.DebounceInterval = 11
	config.EventBufferSize = 2
	config.CustomWatchDirs = []string{t.TempDir()}
	originalDir := config.CustomWatchDirs[0]
	awm := isolatedManager(t, &pathRecordingHandler{}, config)
	config.CustomWatchDirs[0] = "changed"
	config.DebounceInterval = 999
	if awm.GetConfig().CustomWatchDirs[0] != originalDir || awm.fsWatcher.debounce != 11*time.Millisecond || cap(awm.fsWatcher.eventCh) != 2 {
		t.Fatal("constructor did not own and apply configuration")
	}
	returned := awm.GetConfig()
	returned.CustomWatchDirs[0] = "changed again"
	returned.RetryConfig.MaxRetries = 999
	if awm.GetConfig().CustomWatchDirs[0] != originalDir || awm.GetConfig().RetryConfig.MaxRetries == 999 {
		t.Fatal("GetConfig exposed manager configuration")
	}
}

func TestManagerRoutesSettledPathsWithoutParsing(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{AppAddedDelay: 25, AppModifiedDelay: 15}
	handler := &pathRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	for _, eventType := range []AppChangeType{AppAdded, AppModified, AppRemoved} {
		started := time.Now()
		awm.processAppEvent(AppChangeEvent{Type: eventType, Path: "coordinator-owned-path"})
		if eventType == AppAdded && time.Since(started) < 20*time.Millisecond {
			t.Fatal("path reconciliation bypassed install delay")
		}
	}
	if len(handler.calls) != 3 || awm.GetMetrics().EventsProcessed != 3 {
		t.Fatalf("path handler not used: calls=%v metrics=%+v", handler.calls, awm.GetMetrics())
	}
}

func TestManagerCancellationDuringDelaySkipsHandler(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay.AppAddedDelay = 1000
	handler := &pathRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	done := make(chan struct{})
	go func() {
		awm.processAppEvent(AppChangeEvent{Type: AppAdded, Path: "pending-path"})
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for awm.GetMetrics().EventsByType["added"] == 0 {
		if time.Now().After(deadline) {
			t.Fatal("manager did not begin processing")
		}
		time.Sleep(time.Millisecond)
	}
	awm.cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt delay")
	}
	if len(handler.calls) != 0 || awm.GetMetrics().EventsProcessed != 0 {
		t.Fatal("canceled installation produced a late success")
	}
}

func TestManagerHandlerCancellationDoesNotCountSuccess(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{}
	handler := &pathRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	handler.onPath = awm.cancel
	if err := awm.handlePathChange(AppChangeEvent{Type: AppRemoved, Path: "path"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("late success was not invalidated: %v", err)
	}
	if len(handler.calls) != 1 || awm.GetMetrics().EventsProcessed != 0 {
		t.Fatal("unexpected late success metrics")
	}
}

type legacyRecordingHandler struct {
	added, removed, modified int
}

func (h *legacyRecordingHandler) OnAppAdded(*models.ApplicationCommand) error {
	h.added++
	return nil
}
func (h *legacyRecordingHandler) OnAppRemoved(string) error {
	h.removed++
	return nil
}
func (h *legacyRecordingHandler) OnAppModified(*models.ApplicationCommand) error {
	h.modified++
	return nil
}

func TestLegacyHandlerRemovesOnlyConfirmedMissingRoot(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{}
	handler := &legacyRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	if err := awm.handlePathChange(AppChangeEvent{Type: AppAdded, Path: changedTestPath(t, false)}); err != nil {
		t.Fatal(err)
	}
	if handler.removed != 1 {
		t.Fatal("missing root was not removed after stale create")
	}
	if runtime.GOOS != "darwin" {
		return
	}
	for _, eventType := range []AppChangeType{AppAdded, AppModified, AppRemoved} {
		path := filepath.Join(t.TempDir(), "Broken.app")
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		err := awm.handlePathChange(AppChangeEvent{Type: eventType, Path: path})
		if !errors.Is(err, application.ErrInvalidMetadata) {
			t.Fatalf("existing root's missing metadata lost its cause: %v", err)
		}
		if handler.removed != 1 || handler.added != 0 || handler.modified != 0 {
			t.Fatal("invalid metadata changed the previous command")
		}
	}
}

func TestLegacyHandlerKeepsUnreadableExistingApplication(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS bundle metadata fixture")
	}
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{}
	handler := &legacyRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	path := filepath.Join(t.TempDir(), "Unreadable.app")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	// A loop makes metadata unreadable even under privileged test users.
	if err := os.Symlink("Contents", filepath.Join(path, "Contents")); err != nil {
		t.Fatal(err)
	}
	err := awm.handlePathChange(AppChangeEvent{Type: AppModified, Path: path})
	if !errors.Is(err, application.ErrApplicationUnreadable) {
		t.Fatalf("unreadable metadata lost its cause: %v", err)
	}
	if handler.removed != 0 || handler.added != 0 || handler.modified != 0 {
		t.Fatal("unreadable metadata changed the previous command")
	}
}

func TestManagerLateHandlerSuccessDoesNotUpdateMetrics(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{}
	handler := &pathRecordingHandler{}
	awm := isolatedManager(t, handler, config)
	handler.onPath = awm.cancel
	awm.processAppEvent(AppChangeEvent{Type: AppRemoved, Path: "path"})
	if len(handler.calls) != 1 || awm.GetMetrics().EventsProcessed != 0 || awm.GetMetrics().ErrorsCount != 0 {
		t.Fatal("late handler success produced success or error statistics")
	}
}

type cancellationPathHandler struct {
	pathRecordingHandler
	started, completed chan struct{}
}

func (h *cancellationPathHandler) ReconcileApplicationPath(ctx context.Context, path string) error {
	close(h.started)
	<-ctx.Done()
	close(h.completed)
	return nil // A late successful return must not count as a committed event.
}

func TestManagerStopCancelsPathHandlerAndWaitsForWorker(t *testing.T) {
	config := DefaultWatcherConfig()
	config.ProcessDelay = ProcessDelayConfig{}
	handler := &cancellationPathHandler{started: make(chan struct{}), completed: make(chan struct{})}
	awm := isolatedManager(t, handler, config)
	if err := awm.Start(); err != nil {
		t.Fatal(err)
	}
	awm.fsWatcher.eventCh <- AppChangeEvent{Type: AppRemoved, Path: "path"}
	select {
	case <-handler.started:
	case <-time.After(time.Second):
		t.Fatal("path handler did not start")
	}
	if err := awm.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-handler.completed:
	default:
		t.Fatal("Stop returned before the path handler completed")
	}
	if awm.GetMetrics().EventsProcessed != 0 || awm.GetMetrics().ErrorsCount != 0 {
		t.Fatal("canceled worker published late success or error statistics")
	}
}

func TestInterleavedPathsKeepIndependentFinalStates(t *testing.T) {
	fw := isolatedFSWatcher(t, 30, 2)
	replacement := changedTestPath(t, true)
	removed := changedTestPath(t, false)
	sendNativeEvent(t, fw, replacement, fsnotify.Create)
	sendNativeEvent(t, fw, removed, fsnotify.Create|fsnotify.Write)
	sendNativeEvent(t, fw, replacement, fsnotify.Remove|fsnotify.Create)
	sendNativeEvent(t, fw, removed, fsnotify.Remove)
	changes := make(map[string]AppChangeType)
	for i := 0; i < 2; i++ {
		event := receiveChange(t, fw, time.Second)
		if _, duplicate := changes[event.Path]; duplicate {
			t.Fatalf("interleaved path published twice: %+v", event)
		}
		changes[event.Path] = event.Type
	}
	if changes[replacement] != AppAdded || changes[removed] != AppRemoved {
		t.Fatalf("interleaved final states were lost: %v", changes)
	}
	select {
	case event := <-fw.EventChannel():
		t.Fatalf("interleaved events were not coalesced: %+v", event)
	case <-time.After(60 * time.Millisecond):
	}
}
