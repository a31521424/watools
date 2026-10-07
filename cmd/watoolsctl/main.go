// watoolsctl is the shared Go entry for development, checks, builds and installation.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"watools/internal/workflow"
)

func main() {
	if err := run(); err != nil {
		code := exitCode(err)
		if code != 130 {
			fmt.Fprintln(os.Stderr, "WaTools:", err)
		}
		os.Exit(code)
	}
}

// The local installer distinguishes an acknowledged cancellation from a build
// or stop failure. Mixed failures must keep the failure exit code.
func exitCode(err error) int {
	if errors.Is(err, workflow.ErrInstallationCommitted) {
		return 131
	}
	if cancellationOnly(err) {
		return 130
	}
	return 1
}

func cancellationOnly(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !cancellationOnly(cause) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return cancellationOnly(wrapped.Unwrap())
	}
	return err == context.Canceled
}
func run() error {
	options, err := workflow.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	if options.Task == "help" {
		fmt.Print(workflow.Help)
		return nil
	}
	root, err := workflow.FindRoot()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// go run may exit on TERM before its executable. Observe parent loss so the
	// actual CLI cancels its children and owns cleanup even in that case.
	parent := os.Getppid()
	if parent > 1 {
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if os.Getppid() != parent {
						cancel()
						return
					}
				}
			}
		}()
	}
	return workflow.New(root, os.Stdout, os.Stderr).Execute(ctx, os.Args[1:])
}
