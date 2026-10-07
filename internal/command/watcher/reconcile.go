package watcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"watools/internal/command/application"
	"watools/pkg/models"
)

// PreserveApplicationIdentity merges disk metadata with persisted identity and
// usage. Neither the parsed candidate nor the repository snapshot is mutated.
func PreserveApplicationIdentity(candidate, existing *models.ApplicationCommand) *models.ApplicationCommand {
	merged := *candidate
	merged.ID = existing.ID
	merged.TriggerID = existing.TriggerID
	merged.Path = existing.Path
	merged.UsedCount = existing.UsedCount
	merged.LastUsedAt = existing.LastUsedAt
	return &merged
}

// ReconcileApplicationPath must run under the caller's application ownership
// lock, covering the read, fresh parse and commit. Event kinds are only hints:
// only a confirmed missing application path allows deletion.
func ReconcileApplicationPath(ctx, owner context.Context, repository ApplicationRepository, parse func(string) (*models.ApplicationCommand, error), path string) (bool, error) {
	checkCancellation := func() error {
		if err := owner.Err(); err != nil {
			return err
		}
		return ctx.Err()
	}
	if err := checkCancellation(); err != nil {
		return false, err
	}
	commands, err := repository.GetCommands(ctx)
	if err != nil {
		return false, fmt.Errorf("read application before reconcile %q: %w", path, err)
	}
	var existing *models.ApplicationCommand
	for _, command := range commands {
		if application.PathKey(command.Path) == application.PathKey(path) {
			existing = command
			break
		}
	}
	if err := checkCancellation(); err != nil {
		return false, err
	}
	_, statErr := os.Stat(path)
	var candidate *models.ApplicationCommand
	if statErr == nil {
		candidate, err = parse(path)
	} else if os.IsNotExist(statErr) {
		err = errors.Join(application.ErrApplicationNotFound, statErr)
	} else {
		err = errors.Join(application.ErrApplicationUnreadable, statErr)
	}
	if cancelErr := checkCancellation(); cancelErr != nil {
		return false, cancelErr
	}
	if errors.Is(err, application.ErrApplicationNotFound) {
		// An installer can recreate the root while parsing was in flight. A
		// stale absence must not discard its persisted identity and usage.
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			if statErr != nil {
				return false, fmt.Errorf("confirm application removal %q: %w", path, errors.Join(application.ErrApplicationUnreadable, statErr))
			}
			return false, fmt.Errorf("application reappeared during reconcile %q: %w", path, err)
		}
		if existing == nil {
			return false, nil
		}
		if err := checkCancellation(); err != nil {
			return false, err
		}
		if err := repository.DeleteCommands(ctx, []string{existing.ID}); err != nil {
			return false, fmt.Errorf("delete missing application %q: %w", path, err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("parse application for reconcile %q: %w", path, err)
	}
	if candidate == nil {
		return false, fmt.Errorf("parse application for reconcile %q: %w", path, application.ErrInvalidMetadata)
	}
	if err := checkCancellation(); err != nil {
		return false, err
	}
	if existing != nil {
		candidate = PreserveApplicationIdentity(candidate, existing)
		if candidate.Name == existing.Name && candidate.Description.Equal(existing.Description) &&
			candidate.Category == existing.Category && candidate.IconPath.Equal(existing.IconPath) &&
			candidate.DirUpdatedAt.Equal(existing.DirUpdatedAt) {
			return false, nil
		}
		if err := repository.BatchUpdateCommands(ctx, []*models.ApplicationCommand{candidate}); err != nil {
			return false, fmt.Errorf("update application %q: %w", path, err)
		}
	} else if err := repository.BatchInsertCommands(ctx, []*models.ApplicationCommand{candidate}); err != nil {
		return false, fmt.Errorf("insert application %q: %w", path, err)
	}
	return true, nil
}
