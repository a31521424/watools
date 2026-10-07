package localinstall

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"watools/internal/filelock"
)

func TestValidationAndConfigurationFailuresHaveSemanticCauses(t *testing.T) {
	m, value := managerFixture(t, "exit 0\n")
	err := m.Start(context.Background(), Request{Scope: "invalid"})
	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("invalid scope = %v", err)
	}
	value.SourceRoot = filepath.Join(t.TempDir(), "missing-source")
	_, err = ValidateConfiguration(value)
	if !errors.As(err, &invalid) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing source lost validation or cause: %v", err)
	}
	if err := os.WriteFile(filepath.Join(m.dataDir, "installation.json"), []byte("{malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	err = m.Start(context.Background(), Request{Scope: "all"})
	if err == nil || errors.As(err, &invalid) {
		t.Fatalf("corrupt persisted configuration treated as a caller error: %v", err)
	}
}

func TestNonDirectorySourceAndFilesystemFailuresRemainDistinct(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ValidateSource(file)
	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("non-directory source is not a correctable validation error: %v", err)
	}
	for _, cause := range []error{os.ErrPermission, errors.New("filesystem failure")} {
		pathError := &os.PathError{Op: "stat", Path: file, Err: cause}
		err = sourcePathError("源码目录不可用", pathError)
		if errors.As(err, &invalid) || !errors.Is(err, pathError) || !errors.Is(err, cause) {
			t.Fatalf("source I/O failure lost operational cause: %v", err)
		}
	}
}

func TestConfigurationLockContentionIsBusy(t *testing.T) {
	m, value := managerFixture(t, "exit 0\n")
	unlock, err := filelock.Acquire(filepath.Join(m.dataDir, ".local-install.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	err = m.Configure(value)
	if !errors.Is(err, ErrBusy) || !errors.Is(err, filelock.ErrBusy) {
		t.Fatalf("lock contention lost semantic causes: %v", err)
	}
}

func TestCancelledStartDoesNotInspectConfigurationOrSpawn(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "missing"), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Start(ctx, Request{Scope: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled start = %v", err)
	}
	if m.status.JobID != 0 || m.process != nil {
		t.Fatal("cancelled request spawned a process")
	}
}

func TestStartFailureRetainsCauseAndCleansBootstrap(t *testing.T) {
	m, value := managerFixture(t, "exit 0\n")
	if err := os.WriteFile(value.GoExecutable, []byte("#!/nonexistent-synthetic-credential\n"), 0700); err != nil {
		t.Fatal(err)
	}
	err := m.Start(context.Background(), Request{Scope: "all"})
	var pathError *os.PathError
	var invalid *ValidationError
	if !errors.As(err, &pathError) || errors.As(err, &invalid) {
		t.Fatalf("process start failure = %v", err)
	}
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil || len(entries) != 0 || m.process != nil || m.status.JobID != 0 {
		t.Fatal("failed start left process state or bootstrap behind")
	}
}

func TestCancelFailureDoesNotClaimSuccessfulCancellation(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("POSIX exited-process fixture")
	}
	command := exec.Command("/bin/sh", "-c", "exit 0")
	configureProcess(command)
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	m := NewManager(t.TempDir(), t.TempDir())
	m.process = command
	m.status = Status{State: "running", CanCancel: true}
	if err := m.Cancel(); err == nil || errors.Is(err, ErrBusy) {
		t.Fatalf("stopping exited process = %v", err)
	}
	if m.cancelled || !m.status.CanCancel || m.status.Message != "" {
		t.Fatal("failed cancellation changed the job to cancelling")
	}
}
