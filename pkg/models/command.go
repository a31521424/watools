package models

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

type CommandCategory string

const (
	CategoryApplication CommandCategory = "Application"
	CategoryOperation   CommandCategory = "Operation"
)

func ParseCommandCategory(category string) (CommandCategory, error) {
	switch category {
	case string(CategoryApplication):
		return CategoryApplication, nil
	case string(CategoryOperation):
		return CategoryOperation, nil
	default:
		return CategoryApplication, fmt.Errorf("cant parse command category")
	}
}

type CommandRunner interface {
	GetTriggerID() string
	OnTrigger() error
	GetMetadata() *Command
}

type Command struct {
	TriggerID   string          `json:"triggerId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    CommandCategory `json:"category"`
}

type ApplicationCommand struct {
	Command
	IconPath     string    `json:"iconPath,omitempty"`
	Path         string    `json:"path"`
	ID           string    `json:"id"`
	DirUpdatedAt time.Time `json:"dirUpdatedAt"`
}

func (a *ApplicationCommand) GetTriggerID() string {
	return a.TriggerID
}

func (a *ApplicationCommand) OnTrigger() error {
	path := a.Path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("failed to find application file '%s': %w", path, err)
	}
	cmd := exec.Command("open", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run application: %w\n%s", err, output)
	}
	return nil
}

func (a *ApplicationCommand) GetMetadata() *Command {
	return &a.Command
}

func NewApplicationCommand(name string, description string, path string, iconPath string, id string, dirUpdatedAt time.Time) *ApplicationCommand {
	category := CategoryApplication
	if id == "" {
		id = uuid.New().String()
	}
	return &ApplicationCommand{
		Command: Command{
			TriggerID:   fmt.Sprintf("%s-%s-%s", category, name, id),
			Name:        name,
			Description: description,
			Category:    category,
		},
		IconPath:     iconPath,
		Path:         path,
		ID:           id,
		DirUpdatedAt: dirUpdatedAt,
	}
}

type OperationCommand struct {
	Command
	Icon      string `json:"icon"`
	onTrigger func() error
}

func (o *OperationCommand) GetTriggerID() string {
	return o.TriggerID
}

func (o *OperationCommand) OnTrigger() error {
	return o.onTrigger()
}

func (o *OperationCommand) GetMetadata() *Command {
	return &o.Command
}

func NewOperationCommand(name string, description string, icon string, onTrigger func() error) *OperationCommand {
	category := CategoryOperation
	return &OperationCommand{
		Command: Command{
			TriggerID:   fmt.Sprintf("%s-%s", category, name),
			Name:        name,
			Description: description,
			Category:    category,
		},
		Icon:      icon,
		onTrigger: onTrigger,
	}
}
