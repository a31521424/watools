//go:build darwin || windows

package watcher

import (
	"context"
	"errors"
	"testing"

	"watools/pkg/models"
)

type watcherFailureRepository struct {
	commands                           []*models.ApplicationCommand
	queryErr, mutationErr              error
	queries, inserts, updates, deletes int
}

func (r *watcherFailureRepository) GetCommands(context.Context) ([]*models.ApplicationCommand, error) {
	r.queries++
	return r.commands, r.queryErr
}
func (r *watcherFailureRepository) BatchInsertCommands(context.Context, []*models.ApplicationCommand) error {
	r.inserts++
	return r.mutationErr
}
func (r *watcherFailureRepository) BatchUpdateCommands(context.Context, []*models.ApplicationCommand) error {
	r.updates++
	return r.mutationErr
}
func (r *watcherFailureRepository) DeleteCommands(context.Context, []string) error {
	r.deletes++
	return r.mutationErr
}

func TestWatcherQueryFailureDoesNotMutateOrNotify(t *testing.T) {
	for _, operation := range []string{"added", "modified", "removed"} {
		t.Run(operation, func(t *testing.T) {
			cause := errors.New("query failed")
			repository := &watcherFailureRepository{queryErr: cause,
				commands: []*models.ApplicationCommand{{ID: "existing", Path: "/fixture.app"}}}
			notifications := 0
			handler := &defaultAppEventHandler{db: repository, ctx: context.Background(), notify: func() { notifications++ }}
			candidate := &models.ApplicationCommand{ID: "incoming", Path: "/fixture.app"}
			var err error
			switch operation {
			case "added":
				err = handler.OnAppAdded(candidate)
			case "modified":
				err = handler.OnAppModified(candidate)
			case "removed":
				err = handler.OnAppRemoved(candidate.Path)
			}
			if !errors.Is(err, cause) || repository.inserts+repository.updates+repository.deletes != 0 || notifications != 0 || candidate.ID != "incoming" {
				t.Fatalf("query failure mutated data, notified success or lost cause: %+v, %v, notifications=%d", repository, err, notifications)
			}
		})
	}
}

func TestWatcherMutationFailureDoesNotNotifySuccess(t *testing.T) {
	for _, operation := range []string{"added", "modified", "removed"} {
		t.Run(operation, func(t *testing.T) {
			cause := errors.New("mutation failed")
			repository := &watcherFailureRepository{mutationErr: cause}
			if operation != "added" {
				repository.commands = []*models.ApplicationCommand{{ID: "existing", Path: "/fixture.app"}}
			}
			notifications := 0
			handler := &defaultAppEventHandler{db: repository, ctx: context.Background(), notify: func() { notifications++ }}
			candidate := &models.ApplicationCommand{ID: "incoming", Path: "/fixture.app"}
			var err error
			switch operation {
			case "added":
				err = handler.OnAppAdded(candidate)
			case "modified":
				err = handler.OnAppModified(candidate)
			case "removed":
				err = handler.OnAppRemoved(candidate.Path)
			}
			if !errors.Is(err, cause) || notifications != 0 || repository.inserts+repository.updates+repository.deletes != 1 {
				t.Fatalf("mutation failure notified success or lost cause: %+v, %v, notifications=%d", repository, err, notifications)
			}
		})
	}
}
