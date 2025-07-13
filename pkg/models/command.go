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
	Path        string          `json:"path"`
	IconPath    string          `json:"iconPath"`
	ID          int64           `json:"id"`
}

type CommandGroup struct {
	Category CommandCategory `json:"category"`
	Commands []Command       `json:"commands"`
}
