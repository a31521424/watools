package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"watools/internal/workflow"
)

func TestExitCodeSeparatesAcknowledgedCancellationAndFailures(t *testing.T) {
	failure := errors.New("stop or build fault")
	for _, test := range []struct {
		err  error
		code int
	}{
		{context.Canceled, 130},
		{fmt.Errorf("task: %w", context.Canceled), 130},
		{errors.Join(context.Canceled, fmt.Errorf("child: %w", context.Canceled)), 130},
		{errors.Join(context.Canceled, failure), 1},
		{context.DeadlineExceeded, 1},
		{failure, 1},
		{errors.Join(workflow.ErrInstallationCommitted, failure), 131},
	} {
		if got := exitCode(test.err); got != test.code {
			t.Fatalf("exitCode(%v) = %d, want %d", test.err, got, test.code)
		}
	}
}
