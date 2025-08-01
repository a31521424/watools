package models

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"os/exec"
)

type CommandCategory string

const (
	CategoryApplication     CommandCategory = "Application"
	CategorySystemOperation CommandCategory = "SystemOperation"
)

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
	IconPath string `json:"iconPath,omitempty"`
	Path     string `json:"path"`
	ID       string `json:"id"`
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

func NewApplicationCommand(name string, description string, path string, iconPath string, id string) *ApplicationCommand {
	category := CategoryApplication
	if id == "" {
		id = uuid.New().String()
	}
	return &ApplicationCommand{
		Command: Command{
			TriggerID:   fmt.Sprintf("%s-%s-%d", category, name, id),
			Name:        name,
			Description: description,
			Category:    category,
		},
		IconPath: iconPath,
		Path:     path,
		ID:       id,
	}
}
