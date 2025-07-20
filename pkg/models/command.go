package models

type CommandCategory string

const (
	CategoryApplication     CommandCategory = "Application"
	CategorySystemOperation CommandCategory = "SystemOperation"
)

type Command struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    CommandCategory `json:"category"`
}

type ApplicationCommand struct {
	Command
	Path     string `json:"path"`
	IconPath string `json:"iconPath"`
	ID       int64  `json:"id"`
}

func NewApplicationCommand(name string, description string, category CommandCategory, path string, iconPath string, id int64) *ApplicationCommand {
	return &ApplicationCommand{
		Command: Command{
			Name:        name,
			Description: description,
			Category:    category,
		},
		Path:     path,
		IconPath: iconPath,
		ID:       id,
	}
}
