package schemas

type CommandCategory string

const (
	CategoryApplication     CommandCategory = "Application"
	CategorySystemOperation CommandCategory = "SystemOperation"
)

type Command struct {
	Name        string
	Description string
	Category    CommandCategory
	Path        string
	IconPath    string
}

type CommandGroup struct {
	Category CommandCategory
	Commands []Command
}
