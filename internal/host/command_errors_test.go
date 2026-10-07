package host

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"watools/internal/command"
	"watools/internal/command/application"
	"watools/pkg/models"
)

type hostApplicationRepository struct {
	err       error
	mutations int
}

func (r *hostApplicationRepository) GetCommands(context.Context) ([]*models.ApplicationCommand, error) {
	return nil, r.err
}
func (r *hostApplicationRepository) BatchInsertCommands(context.Context, []*models.ApplicationCommand) error {
	r.mutations++
	return nil
}
func (r *hostApplicationRepository) BatchUpdateCommands(context.Context, []*models.ApplicationCommand) error {
	r.mutations++
	return nil
}
func (r *hostApplicationRepository) DeleteCommands(context.Context, []string) error {
	r.mutations++
	return nil
}
func (r *hostApplicationRepository) BatchUpdateApplicationUsage(context.Context, []models.ApplicationUsageUpdate) error {
	r.mutations++
	return nil
}

type hostApplicationSource struct{ scans int }

func (s *hostApplicationSource) Scan() ([]*models.ApplicationCommand, error) {
	s.scans++
	return nil, nil
}
func (s *hostApplicationSource) Paths() []application.AppPathInfo                 { return nil }
func (s *hostApplicationSource) Parse(string) (*models.ApplicationCommand, error) { return nil, nil }

func TestApplicationCommandHostPropagatesQueryFailuresAsSafeTypedErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		cause error
		code  string
	}{
		{"database", errors.New("synthetic credential in database path"), "OPERATION_FAILED"},
		{"cancelled", context.Canceled, "CANCELLED"},
		{"deadline", context.DeadlineExceeded, "TIMEOUT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &hostApplicationRepository{err: fmt.Errorf("query: %w", test.cause)}
			source := &hostApplicationSource{}
			service := NewCommandService(command.NewWaLaunchWithDependencies(repository, source), nil)
			items, err := service.GetApplicationCommandsApi()
			errorCode(t, err, test.code)
			if items != nil || !errors.Is(err, test.cause) || strings.Contains(string(MarshalError(err)), "credential") || strings.Contains(err.Error(), "credential") {
				t.Fatalf("query failure lost cause, exposed diagnostics or returned list: %+v, %v", items, err)
			}
			if source.scans != 0 || repository.mutations != 0 {
				t.Fatal("host query failure scanned or mutated data")
			}
		})
	}
}

func TestApplicationCommandHostEmptySuccessHasNoError(t *testing.T) {
	repository := &hostApplicationRepository{}
	source := &hostApplicationSource{}
	service := NewCommandService(command.NewWaLaunchWithDependencies(repository, source), nil)
	items, err := service.GetApplicationCommandsApi()
	if err != nil || len(items) != 0 || source.scans != 1 {
		t.Fatalf("successful empty load = %+v, %v, scans=%d", items, err, source.scans)
	}
}
