package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"time"
	"watools/internal/command"
	"watools/pkg/models"
)

type CommandService struct {
	core   *command.WaLaunchApp
	notify func()
	ctx    context.Context
}

func NewCommandService(core *command.WaLaunchApp, notify func()) *CommandService {
	return &CommandService{core: core, notify: notify}
}
func (s *CommandService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	s.core.OnStartup(ctx, s.notify)
	return nil
}
func (s *CommandService) ServiceShutdown() error { s.core.Shutdown(s.ctx); return nil }

type CommandInfo struct {
	TriggerID   string                 `json:"triggerId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    models.CommandCategory `json:"category"`
	LastUsedAt  *time.Time             `json:"lastUsedAt"`
	UsedCount   int64                  `json:"usedCount"`
}
type ApplicationCommand struct {
	CommandInfo
	ID        string `json:"id"`
	Path      string `json:"path"`
	IconPath  string `json:"iconPath"`
	IsUserApp bool   `json:"isUserApp"`
}
type OperationCommand struct {
	CommandInfo
	Icon string `json:"icon"`
}

func commandInfo(c models.Command) CommandInfo {
	v := CommandInfo{TriggerID: c.TriggerID, Name: c.Name, Description: c.Description.OrEmpty(), Category: c.Category, UsedCount: c.UsedCount}
	if t, ok := c.LastUsedAt.Get(); ok {
		v.LastUsedAt = &t
	}
	return v
}
func (s *CommandService) GetApplicationCommandsApi() ([]ApplicationCommand, error) {
	items, err := s.core.GetApplicationCommands()
	if err != nil {
		return nil, failure("command.applications", err)
	}
	result := make([]ApplicationCommand, 0, len(items))
	for _, c := range items {
		result = append(result, ApplicationCommand{CommandInfo: commandInfo(c.Command), ID: c.ID, Path: c.Path, IconPath: c.IconPath.OrEmpty(), IsUserApp: c.IsUserApp})
	}
	return result, nil
}
func (s *CommandService) GetOperatorCommandsApi() []OperationCommand {
	items := s.core.GetOperationCommands()
	result := make([]OperationCommand, 0, len(items))
	for _, c := range items {
		result = append(result, OperationCommand{CommandInfo: commandInfo(c.Command), Icon: c.Icon})
	}
	return result
}
func (s *CommandService) TriggerCommandApi(id, category string) error {
	if id == "" {
		return invalid("triggerId 不能为空")
	}
	c, err := models.ParseCommandCategory(category)
	if err != nil {
		return invalid("category 必须是 Application 或 Operation")
	}
	return failure("command.trigger", s.core.TriggerCommand(id, c))
}
func (s *CommandService) UpdateApplicationUsageApi(items []ApplicationUsage) error {
	updates := make([]models.ApplicationUsageUpdate, len(items))
	for i, v := range items {
		if err := validateUsage(v.ID, v.LastUsedAt, v.UsedCount); err != nil {
			return err
		}
		updates[i] = models.ApplicationUsageUpdate{ID: v.ID, LastUsedAt: *v.LastUsedAt, UsedCount: int(*v.UsedCount)}
	}
	return failure("command.usage", s.core.UpdateApplicationUsage(updates))
}
