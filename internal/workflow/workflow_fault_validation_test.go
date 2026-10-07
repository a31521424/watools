package workflow

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"watools/internal/filelock"
)

type faultValidationWriter struct{ cause error }

func (w faultValidationWriter) Write([]byte) (int, error) { return 0, w.cause }

func TestFaultValidationCommandAnnouncementWriteFailurePreventsSideEffect(t *testing.T) {
	cause := errors.New("output write fault")
	for _, atomic := range []bool{false, true} {
		t.Run(map[bool]string{false: "build", true: "commit"}[atomic], func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "executed")
			r := New(root, faultValidationWriter{cause}, &bytes.Buffer{})
			r.SetEnv("PATH", "/bin:/usr/bin")
			var err error
			if atomic {
				err = r.atomic(context.Background(), "/bin/sh", "-c", `touch "$1"`, "sh", marker)
			} else {
				err = r.Run(context.Background(), root, "/bin/sh", "-c", `touch "$1"`, "sh", marker)
			}
			if !errors.Is(err, cause) {
				t.Fatalf("lost mandatory output failure: %v", err)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("command ran after output failure: %v", err)
			}
		})
	}
}

func TestFaultValidationPermissionRestoreKeepsTraversalAndChmodFailures(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "service.ts")
	if err := os.WriteFile(file, []byte("export {}"), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	walkCause, chmodCause := errors.New("directory read fault"), errors.New("permission update fault")
	walk := func(path string, callback fs.WalkDirFunc) error {
		if err := callback(file, entries[0], nil); err != nil {
			return err
		}
		return callback(filepath.Join(path, "unreadable"), nil, walkCause)
	}
	err = restoreBindingPermissions(root, walk, func(string, fs.FileMode) error { return chmodCause })
	if !errors.Is(err, walkCause) || !errors.Is(err, chmodCause) {
		t.Fatalf("permission restore lost independent failures: %v", err)
	}
}

func TestFaultValidationWailsCommandAndRequiredPermissionFailuresAreJoined(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nexit 13\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("binding traversal fault")
	r := New(root, &bytes.Buffer{}, &bytes.Buffer{})
	r.SetEnv("PATH", bin)
	r.walkDir = func(string, fs.WalkDirFunc) error { return cause }
	err := r.wails(context.Background(), Toolchain{Wails: "unused"}, "generate", "bindings")
	var exitErr *exec.ExitError
	if !errors.Is(err, cause) || !errors.As(err, &exitErr) || exitErr.ExitCode() != 13 {
		t.Fatalf("lost generation or required permission failure: %v", err)
	}
}

func TestFaultValidationBestEffortTemporaryCleanupDoesNotReverseSuccess(t *testing.T) {
	f := newFixture(t)
	cause := errors.New("cleanup fault")
	var retained []string
	f.r.removeAll = func(path string) error { retained = append(retained, path); return cause }
	t.Cleanup(func() {
		for _, path := range retained {
			_ = os.RemoveAll(path)
		}
	})
	if err := f.r.Execute(context.Background(), []string{"install", "--scope", "plugins", "--no-launch"}); err != nil {
		t.Fatalf("harmless cleanup reversed completed install: %v", err)
	}
	if len(retained) == 0 {
		t.Fatal("temporary cleanup was not attempted")
	}
	diagnostics := f.r.Err.(*bytes.Buffer).String()
	if !strings.Contains(diagnostics, cause.Error()) || !strings.Contains(diagnostics, retained[0]) {
		t.Fatalf("cleanup failure lacks residual path diagnostic: %q", diagnostics)
	}
}

func TestFaultValidationStopNotificationDeliveryFailuresPreserveAllCauses(t *testing.T) {
	stopCause, outputCause := errors.New("task stop fault"), errors.New("task notification output fault")
	r := New(t.TempDir(), faultValidationWriter{outputCause}, &bytes.Buffer{})
	path := t.TempDir() // A directory cannot be replaced by the notification file.
	r.SetEnv(StopFailureFileEnv, path)
	err := r.reportStopFailure(stopCause)
	var fileErr *os.LinkError
	if !errors.Is(err, ErrTaskStopFailed) || !errors.Is(err, stopCause) || !errors.Is(err, outputCause) || !errors.As(err, &fileErr) || fileErr.New != path {
		t.Fatalf("lost stop or mandatory notification cause: %v", err)
	}
}

func TestFaultValidationRunnerReportsStopFailureWithoutReleasingLiveSourceLock(t *testing.T) {
	f := newFixture(t)
	ready, gate, notification := filepath.Join(f.r.Root, "ready"), filepath.Join(f.r.Root, "release"), filepath.Join(f.r.Root, "stop-failure")
	tool, err := FindExecutable("go", EnvValue(f.r.Env, "PATH"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(tool); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 'go version go1.26.8 darwin/arm64'; exit 0; fi\necho ready > \"$WATOOLS_TEST_READY\"\nwhile [ ! -f \"$WATOOLS_TEST_GATE\" ]; do /bin/sleep 0.01; done\nexit 0\n"
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	f.r.SetEnv("WATOOLS_TEST_READY", ready)
	f.r.SetEnv("WATOOLS_TEST_GATE", gate)
	f.r.SetEnv(StopFailureFileEnv, notification)
	stopCause, groupCause := errors.New("runner signal fault"), errors.New("runner group wait fault")
	f.r.stopProcess = func(*exec.Cmd) error { return stopCause }
	f.r.waitProcessGroup = func(int) error { return groupCause }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { defer close(done); done <- f.r.Execute(ctx, []string{"setup"}) }()
	t.Cleanup(func() {
		_ = os.WriteFile(gate, nil, 0600)
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})
	awaitFile := func(path string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(path); err == nil {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("missing workflow handshake %s", path)
	}
	awaitFile(ready)
	cancel()
	awaitFile(notification)
	if unlock, err := filelock.Acquire(filepath.Join(f.r.Root, ".cache/installation/source.lock")); err == nil {
		unlock()
		t.Fatal("failed stop released live source lock")
	} else if !errors.Is(err, filelock.ErrBusy) {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("reported stop failure as completed task: %v", err)
	default:
	}
	if err := os.WriteFile(gate, nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrTaskStopFailed) || !errors.Is(err, stopCause) || !errors.Is(err, groupCause) {
			t.Fatalf("lost owned-stop failures: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not finish after actual task exit")
	}
	if !strings.Contains(f.r.Out.(*bytes.Buffer).String(), StopFailureMessage) {
		t.Fatal("missing safe output notification")
	}
	unlock, err := filelock.Acquire(filepath.Join(f.r.Root, ".cache/installation/source.lock"))
	if err != nil {
		t.Fatalf("actual task exit did not release source lock: %v", err)
	}
	unlock()
}

type faultValidationChildOutputWriter struct {
	writes int
	cause  error
}

func (w *faultValidationChildOutputWriter) Write(content []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		return len(content), nil
	} // The task announcement succeeds.
	return 0, w.cause
}

func TestFaultValidationChildOutputWriteAndExitFailuresAreJoined(t *testing.T) {
	cause := errors.New("child output write fault")
	r := New(t.TempDir(), &faultValidationChildOutputWriter{cause: cause}, &bytes.Buffer{})
	err := r.Run(context.Background(), r.Root, "/bin/sh", "-c", "printf child-output; exit 11")
	var exitErr *exec.ExitError
	if !errors.Is(err, cause) || !errors.As(err, &exitErr) || exitErr.ExitCode() != 11 {
		t.Fatalf("child exit hid output write failure: %v", err)
	}
}

func TestFaultValidationAtomicCommitFactSurvivesOutputFailureOnlyAfterZeroExit(t *testing.T) {
	for _, exitCode := range []int{0, 11} {
		t.Run(map[int]string{0: "committed", 11: "commit_failed"}[exitCode], func(t *testing.T) {
			cause := errors.New("atomic helper output write fault")
			r := New(t.TempDir(), &faultValidationChildOutputWriter{cause: cause}, &bytes.Buffer{})
			code := "0"
			if exitCode != 0 {
				code = "11"
			}
			err := r.atomic(context.Background(), "/bin/sh", "-c", "printf child-output; exit "+code)
			if !errors.Is(err, cause) {
				t.Fatalf("commit diagnostic lost: %v", err)
			}
			if got := errors.Is(err, ErrInstallationCommitted); got != (exitCode == 0) {
				t.Fatalf("committed fact=%t for helper exit=%d: %v", got, exitCode, err)
			}
			if exitCode != 0 {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != exitCode {
					t.Fatalf("lost real commit failure: %v", err)
				}
			}
		})
	}
}
