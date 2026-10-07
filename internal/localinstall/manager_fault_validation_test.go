//go:build darwin || linux

package localinstall

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"watools/internal/workflow"
)

func faultValidationAwaitOutput(t *testing.T, m *Manager, expected string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, err := m.Status()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.Join(status.Output, "\n"), expected) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job never produced %q", expected)
}

func TestFaultValidationCancelSignalFailureDoesNotClaimCancelled(t *testing.T) {
	m, _ := managerFixture(t, "echo ready\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 9\n")
	release := faultValidationReleaseGate(t, m)
	cause := errors.New("signal denied")
	m.signalProcess = func(*exec.Cmd, bool) error { return cause }
	if err := m.Start(context.Background(), Request{Scope: "application"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	if err := m.Cancel(); !errors.Is(err, cause) {
		t.Fatalf("lost signal cause: %v", err)
	}
	status, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "running" || !status.CanCancel || m.cancelled {
		t.Fatalf("failed cancellation changed actual state: %+v", status)
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("late process result did not finish")
	}
	status, _ = m.Status()
	if status.State != "failed" || status.RestartRequired {
		t.Fatalf("late process failure became cancellation/success: %+v", status)
	}
}

func TestFaultValidationAcceptedCancellationInvalidatesLateBootstrapSuccess(t *testing.T) {
	m, value := managerFixture(t, "exit 99\n")
	if err := os.WriteFile(value.GoExecutable, []byte("#!/bin/sh\necho ready\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	release := faultValidationReleaseGate(t, m)
	// Simulate an acknowledged cancellation whose process reports success later.
	m.signalProcess = func(*exec.Cmd, bool) error { return nil }
	if err := m.Start(context.Background(), Request{Scope: "application"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	if err := m.Cancel(); err != nil {
		t.Fatal(err)
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("late success did not drain")
	}
	status, _ := m.Status()
	if status.State != "cancelled" || status.RestartRequired {
		t.Fatalf("cancelled build accepted late success: %+v", status)
	}
}

func TestFaultValidationShutdownWaitsForFinalCleanupAfterProcessCleared(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir())
	m.done = make(chan struct{})
	returned := make(chan error, 1)
	go func() { returned <- m.Shutdown() }()
	select {
	case err := <-returned:
		t.Fatalf("shutdown returned before final cleanup: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(m.done)
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not return after final cleanup")
	}
}

type faultValidationReadCloser struct {
	io.ReadCloser
	cause                  error
	drainCause, closeCause error
	reads                  int
}

func (r *faultValidationReadCloser) Read([]byte) (int, error) {
	r.reads++
	if r.reads > 1 && r.drainCause != nil {
		return 0, r.drainCause
	}
	return 0, r.cause
}
func (r *faultValidationReadCloser) Close() error {
	return errors.Join(r.ReadCloser.Close(), r.closeCause)
}

func TestFaultValidationOutputReadSignalAndExitFailuresRemainInspectable(t *testing.T) {
	m := NewManager(t.TempDir(), t.TempDir())
	readCause, signalCause := errors.New("output read fault"), errors.New("stop signal fault")
	drainCause, closeCause := errors.New("output drain fault"), errors.New("output close fault")
	m.signalProcess = func(*exec.Cmd, bool) error { return signalCause }
	m.outputDrainTimeout = 100 * time.Millisecond
	cmd := exec.Command("/bin/sh", "-c", "sleep 0.1; exit 7")
	pipe, err := startCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer pipe.Close()
	m.process = cmd
	m.status.State = "running"
	m.status.CanCancel = true
	err = m.collect(cmd, &faultValidationReadCloser{ReadCloser: pipe, cause: readCause, drainCause: drainCause, closeCause: closeCause})
	var exitErr *exec.ExitError
	if !errors.Is(err, readCause) || !errors.Is(err, drainCause) || !errors.Is(err, closeCause) || !errors.Is(err, signalCause) || !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("lost output/stop/exit cause: %v", err)
	}
}

func TestFaultValidationManagerCleanupFailurePreservesSuccessfulInstall(t *testing.T) {
	m, _ := managerFixture(t, "exit 0\n")
	cause := errors.New("installer cleanup fault")
	var retained []string
	m.removeAll = func(path string) error { retained = append(retained, path); return cause }
	t.Cleanup(func() {
		for _, path := range retained {
			_ = os.RemoveAll(path)
		}
	})
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("successful job did not finish cleanup")
	}
	status, _ := m.Status()
	if status.State != "succeeded" || len(retained) == 0 {
		t.Fatalf("cleanup reversed install or was skipped: %+v", status)
	}
	output := strings.Join(status.Output, "\n")
	if !strings.Contains(output, filepath.Base(retained[0])) {
		t.Fatalf("cleanup failure lacks diagnostic: %q", output)
	}
}

func TestFaultValidationTrustedInstallZeroExitAfterCancelReportsCompletedInstallation(t *testing.T) {
	m, _ := managerFixture(t, "echo ready\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 0\n")
	release := faultValidationReleaseGate(t, m)
	m.signalProcess = func(*exec.Cmd, bool) error { return nil }
	if err := m.Start(context.Background(), Request{Scope: "application"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	if err := m.Cancel(); err != nil {
		t.Fatal(err)
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("committed install did not drain")
	}
	status, _ := m.Status()
	if status.State != "succeeded" || !status.RestartRequired {
		t.Fatalf("trusted commit completion became cancelled: %+v", status)
	}
}

func TestFaultValidationCleanupMustFinishBeforeAnotherJobCanStart(t *testing.T) {
	m, _ := managerFixture(t, "exit 0\n")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	m.removeAll = func(path string) error { once.Do(func() { close(entered) }); <-release; return os.RemoveAll(path) }
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("cleanup did not start")
	}
	err := m.Start(context.Background(), Request{Scope: "plugins"})
	close(release)
	if !m.wait(5 * time.Second) {
		t.Fatal("cleanup did not finish")
	}
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("new job started before previous ownership released: %v", err)
	}
}

func TestFaultValidationBootstrapParentExitWithInheritedOutputIsBounded(t *testing.T) {
	m, value := managerFixture(t, "exit 99\n")
	if err := os.WriteFile(value.GoExecutable, []byte("#!/bin/sh\nsleep 0.2 &\necho parent-exit\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	m.outputDrainTimeout = 25 * time.Millisecond
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("inherited output pipe held completed bootstrap open")
	}
	status, _ := m.Status()
	if status.State != "failed" {
		t.Fatalf("output drain failure allowed installation: %+v", status)
	}
	if err := m.Shutdown(); !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("lost output drain failure: %v", err)
	}
}

func TestFaultValidationFailedBootstrapStopReportsAndRetainsOwnershipUntilActualExit(t *testing.T) {
	m, value := managerFixture(t, "exit 99\n")
	if err := os.WriteFile(value.GoExecutable, []byte("#!/bin/sh\necho ready\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	release := faultValidationReleaseGate(t, m)
	signalCause, groupCause := errors.New("bootstrap signal fault"), errors.New("bootstrap group wait fault")
	m.signalProcess = func(*exec.Cmd, bool) error { return signalCause }
	m.waitProcessGroup = func(int) error { return groupCause }
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	if err := m.Cancel(); !errors.Is(err, signalCause) {
		t.Fatalf("lost cancellation signal cause: %v", err)
	}
	returned := make(chan error, 1)
	go func() { returned <- m.Shutdown() }()
	select {
	case err := <-returned:
		if !errors.Is(err, signalCause) || !errors.Is(err, groupCause) {
			t.Fatalf("shutdown lost bounded stop causes: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("failed stop blocked shutdown diagnostics")
	}
	m.mu.Lock()
	process, done, state := m.process, m.done, m.status.State
	m.mu.Unlock()
	if process == nil || state != "running" {
		t.Fatalf("failed stop released running ownership: %s", state)
	}
	select {
	case <-done:
		t.Fatal("failed stop published job completion before actual exit")
	default:
	}
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("failed stop removed live bootstrap artifacts: %v", err)
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("ownership did not drain after actual child exit")
	}
	status, _ := m.Status()
	if status.State != "failed" {
		t.Fatalf("real stop failure became success/cancelled: %+v", status)
	}
}

func faultValidationReleaseGate(t *testing.T, m *Manager) func() {
	t.Helper()
	release := func() {
		if err := os.WriteFile(filepath.Join(m.cacheDir, "release"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(m.cacheDir, "release"), nil, 0600)
		_ = m.wait(time.Second)
	})
	return release
}

func TestFaultValidationStopFailureFileReportsEvenWithoutOutputMarker(t *testing.T) {
	program := "printf '%s' 'WATOOLS_TASK_STOP_FAILED' > \"$WATOOLS_INSTALL_STOP_FAILURE_FILE\"\necho ready\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 1\n"
	m, _ := managerFixture(t, program)
	release := faultValidationReleaseGate(t, m)
	m.signalProcess = func(*exec.Cmd, bool) error { return nil }
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	returned := make(chan error, 1)
	go func() { returned <- m.Shutdown() }()
	select {
	case err := <-returned:
		if !errors.Is(err, workflow.ErrTaskStopFailed) {
			t.Fatalf("file notification failure cause missing: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop failure hidden by incomplete output/child exit")
	}
	m.mu.Lock()
	process, done, state := m.process, m.done, m.status.State
	m.mu.Unlock()
	if process == nil || state != "running" {
		t.Fatalf("stop file released actual process ownership: %s", state)
	}
	select {
	case <-done:
		t.Fatal("stop file published completion before process exit")
	default:
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("file-reported job did not drain after actual exit")
	}
	status, _ := m.Status()
	if status.State != "failed" {
		t.Fatalf("mixed stop/exit failure became cancelled: %+v", status)
	}
}

func TestFaultValidationOutputReadFailureReportsShutdownBeforeActualInstallerExit(t *testing.T) {
	program := "echo ready\n/usr/bin/awk 'BEGIN { for (i=0;i<70000;i++) printf \"x\"; print \"\" }'\nwhile [ ! -f \"$WATOOLS_CACHE_DIR/release\" ]; do sleep 0.01; done\nexit 1\n"
	m, _ := managerFixture(t, program)
	release := faultValidationReleaseGate(t, m)
	m.signalProcess = func(*exec.Cmd, bool) error { return nil }
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	faultValidationAwaitOutput(t, m, "ready")
	returned := make(chan error, 1)
	go func() { returned <- m.Shutdown() }()
	select {
	case err := <-returned:
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Fatalf("read failure missing from shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("read failure blocked shutdown on live installer")
	}
	m.mu.Lock()
	process, done, state := m.process, m.done, m.status.State
	m.mu.Unlock()
	if process == nil || state != "running" {
		t.Fatalf("read failure released live installer: %s", state)
	}
	select {
	case <-done:
		t.Fatal("read failure prematurely completed installer")
	default:
	}
	release()
	if !m.wait(5 * time.Second) {
		t.Fatal("read-failed installer did not finish after actual exit")
	}
	status, _ := m.Status()
	if status.State != "failed" {
		t.Fatalf("read failure became cancelled/success: %+v", status)
	}
}

func TestFaultValidationCommittedInstallDiagnosticExitPreservesSuccessAndShutdownError(t *testing.T) {
	m, _ := managerFixture(t, "exit 131\n")
	if err := m.Start(context.Background(), Request{Scope: "application"}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("committed install diagnostic did not drain")
	}
	status, _ := m.Status()
	if status.State != "succeeded" || !status.RestartRequired || status.CanCancel {
		t.Fatalf("diagnostic exit reversed completed installation: %+v", status)
	}
	var exitErr *exec.ExitError
	if err := m.Shutdown(); !errors.As(err, &exitErr) || exitErr.ExitCode() != 131 {
		t.Fatalf("committed install lost shutdown diagnostic: %v", err)
	}
}
