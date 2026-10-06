//go:build darwin || linux

package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"watools/internal/filelock"
)

func TestCancellationStopsUncooperativeTasksBeforeUnlocking(t *testing.T) {
	for _, mode := range []string{"direct", "descendant"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			ready := filepath.Join(t.TempDir(), "ready")
			leader := ready + ".leader"
			f.r.SetEnv("WATOOLS_TEST_READY", ready)
			f.r.SetEnv("WATOOLS_TEST_LEADER", leader)
			tool, err := FindExecutable("go", EnvValue(f.r.Env, "PATH"))
			if err != nil {
				t.Fatal(err)
			}
			// Remove the fixture hardlink before replacing this one tool.
			if err := os.Remove(tool); err != nil {
				t.Fatal(err)
			}
			body := "trap '' TERM\necho $$ > \"$WATOOLS_TEST_READY\"\nwhile :; do /bin/sleep 1; done\n"
			if mode == "descendant" {
				body = "/bin/sh -c 'trap \"\" TERM; echo $$ > \"$WATOOLS_TEST_READY\"; while :; do /bin/sleep 1; done' &\nwait\n"
			}
			script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 'go version go1.26.8 darwin/arm64'; exit 0; fi\necho $$ > \"$WATOOLS_TEST_LEADER\"\n" + body
			if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- f.r.Execute(ctx, []string{"setup"}) }()
			readPID := func(file string) int {
				t.Helper()
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					if value, err := os.ReadFile(file); err == nil {
						if pid, err := strconv.Atoi(strings.TrimSpace(string(value))); err == nil {
							return pid
						}
					}
					time.Sleep(10 * time.Millisecond)
				}
				t.Fatal("task did not start")
				return 0
			}
			pid := readPID(leader)
			child := readPID(ready)
			t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
			cancel()
			time.Sleep(100 * time.Millisecond)
			lock := filepath.Join(f.r.Root, ".cache/installation/source.lock")
			if unlock, err := filelock.Acquire(lock); err == nil {
				unlock()
				t.Fatal("source lock released while cancelled task was still running")
			}
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("unexpected cancellation result: %v", err)
				}
			case <-time.After(25 * time.Second):
				t.Fatal("uncooperative task blocked cancellation")
			}
			if err := syscall.Kill(-pid, 0); err != syscall.ESRCH {
				t.Fatalf("task group survived cancellation: %v", err)
			}
			if err := syscall.Kill(child, 0); err != syscall.ESRCH {
				t.Fatalf("task child survived cancellation: %v", err)
			}
			if !f.r.cleanupSafe {
				t.Fatal("fully stopped task left unsafe cleanup")
			}
			unlock, err := filelock.Acquire(lock)
			if err != nil {
				t.Fatal("source lock was not released after tasks stopped", err)
			}
			unlock()
		})
	}
}
