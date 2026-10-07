//go:build darwin || linux

package workflow

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAtomicCommitDefersCancellationAcceptedAtItsBoundary(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "committed")
	r := New(root, io.Discard, io.Discard)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// install already checked cancellation before announcing the commit. A
	// cancellation arriving between that announcement and spawn is deferred.
	if err := r.atomic(ctx, "/bin/sh", "-c", `printf committed > "$1"`, "sh", marker); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "committed" {
		t.Fatalf("announced commit did not finish: %q, %v", content, err)
	}
}

type shortCommitOutputWriter struct{}

func (shortCommitOutputWriter) Write(content []byte) (int, error) { return len(content) - 1, nil }
func TestAtomicShortOutputAndNonzeroExitBothRemainInspectable(t *testing.T) {
	r := New(t.TempDir(), io.Discard, io.Discard)
	err := r.command(context.Background(), r.Root, true, shortCommitOutputWriter{}, io.Discard, "/bin/sh", "-c", "printf helper-output; exit 7")
	var exitErr *exec.ExitError
	if !errors.Is(err, io.ErrShortWrite) || !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("short output or helper failure was lost: %v", err)
	}
	if errors.Is(err, ErrInstallationCommitted) {
		t.Fatal("failed helper was reported as committed")
	}
}

type commitOutputWriter struct{ cause error }

func (w commitOutputWriter) Write([]byte) (int, error) { return 0, w.cause }
func TestAtomicOutputFailureKeepsActualCommitOutcome(t *testing.T) {
	for _, code := range []string{"0", "7"} {
		t.Run(code, func(t *testing.T) {
			root := t.TempDir()
			r := New(root, io.Discard, io.Discard)
			cause := errors.New("commit output write failure")
			err := r.command(context.Background(), root, true, commitOutputWriter{cause}, io.Discard, "/bin/sh", "-c", "printf helper-output; exit "+code)
			if !errors.Is(err, cause) {
				t.Fatalf("lost mandatory output failure: %v", err)
			}
			if committed := errors.Is(err, ErrInstallationCommitted); committed != (code == "0") {
				t.Fatalf("incorrect committed outcome for exit %s: %v", code, err)
			}
		})
	}
}
