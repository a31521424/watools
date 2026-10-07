//go:build darwin || windows

package watcher

import (
	"context"
	"fmt"
	"os"
	"sync"
	"watools/internal/command/application"
	"watools/pkg/db"
	"watools/pkg/models"
)

// ApplicationRepository is the application persistence boundary.
type ApplicationRepository interface {
	GetCommands(context.Context) ([]*models.ApplicationCommand, error)
	BatchInsertCommands(context.Context, []*models.ApplicationCommand) error
	BatchUpdateCommands(context.Context, []*models.ApplicationCommand) error
	DeleteCommands(context.Context, []string) error
}

type defaultAppEventHandler struct {
	notify func()
	db     ApplicationRepository
	ctx    context.Context
	mu     sync.Mutex
}

func NewDefaultAppEventHandler(ctx context.Context, notify func()) DefaultAppEventHandler {
	return NewDefaultAppEventHandlerWithRepository(ctx, notify, db.GetWaDB())
}

func NewDefaultAppEventHandlerWithRepository(ctx context.Context, notify func(), repository ApplicationRepository) DefaultAppEventHandler {
	return &defaultAppEventHandler{db: repository, ctx: ctx, notify: notify}
}

// ReconcileApplicationPath reparses under ownership rather than trusting event metadata.
func (h *defaultAppEventHandler) ReconcileApplicationPath(ctx context.Context, path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(h.ctx, cancel)
	defer stop()
	defer cancel()
	changed, err := ReconcileApplicationPath(ctx, h.ctx, h.db, application.ParseApplication, path)
	if changed && ctx.Err() == nil {
		h.emitApplicationChanged()
	}
	return err
}

func (h *defaultAppEventHandler) OnAppAdded(command *models.ApplicationCommand) error {
	return h.upsert(command)
}

func (h *defaultAppEventHandler) OnAppModified(command *models.ApplicationCommand) error {
	return h.upsert(command)
}

// upsert retains the candidate-based interface for consumers without a path
// source. The native manager uses ReconcileApplicationPath and fresh disk metadata.
func (h *defaultAppEventHandler) upsert(command *models.ApplicationCommand) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.ctx.Err(); err != nil {
		return err
	}
	commands, err := h.db.GetCommands(h.ctx)
	if err != nil {
		return fmt.Errorf("read application before upsert: %w", err)
	}
	var existing *models.ApplicationCommand
	for _, item := range commands {
		if application.PathKey(item.Path) == application.PathKey(command.Path) {
			existing = item
			break
		}
	}
	candidate := *command
	if existing != nil {
		candidate = *PreserveApplicationIdentity(command, existing)
	}
	if err := h.ctx.Err(); err != nil {
		return err
	}
	if existing != nil {
		err = h.db.BatchUpdateCommands(h.ctx, []*models.ApplicationCommand{&candidate})
	} else {
		err = h.db.BatchInsertCommands(h.ctx, []*models.ApplicationCommand{&candidate})
	}
	if err != nil {
		return fmt.Errorf("persist application candidate: %w", err)
	}
	h.emitApplicationChanged()
	return nil
}

func (h *defaultAppEventHandler) OnAppRemoved(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.ctx.Err(); err != nil {
		return err
	}
	commands, err := h.db.GetCommands(h.ctx)
	if err != nil {
		return fmt.Errorf("read application before remove: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("confirm application removal %q: %w", path, err)
	}
	for _, command := range commands {
		if application.PathKey(command.Path) == application.PathKey(path) {
			if err := h.ctx.Err(); err != nil {
				return err
			}
			if err := h.db.DeleteCommands(h.ctx, []string{command.ID}); err != nil {
				return fmt.Errorf("delete missing application: %w", err)
			}
			h.emitApplicationChanged()
			return nil
		}
	}
	return nil
}

func (h *defaultAppEventHandler) emitApplicationChanged() {
	if h.ctx.Err() == nil && h.notify != nil {
		h.notify()
	}
}
