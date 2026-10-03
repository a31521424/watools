package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"watools/internal/plugin"
	"watools/pkg/models"
)

type PluginService struct {
	core      *plugin.WaPlugin
	pickFiles func() ([]string, error)
	ctx       context.Context
}

func NewPluginService(core *plugin.WaPlugin, pickFiles func() ([]string, error)) *PluginService {
	return &PluginService{core: core, pickFiles: pickFiles}
}
func (s *PluginService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	s.core.OnStartup(ctx)
	return nil
}
func (s *PluginService) ServiceShutdown() error { s.core.OnShutdown(s.ctx); return nil }
func (s *PluginService) GetPluginsApi() ([]models.PluginInfo, error) {
	v, e := s.core.GetPlugins()
	return v, failure("plugin.list", e)
}
func (s *PluginService) GetPluginJsEntryUrlApi(id string) (string, error) {
	if err := validatePackage(id); err != nil {
		return "", err
	}
	return s.core.GetJsEntryUrl(id), nil
}
func (s *PluginService) UpdatePluginUsageApi(items []PluginUsage) error {
	updates := make([]models.PluginUsageUpdate, len(items))
	for i, v := range items {
		if err := validatePackage(v.PackageID); err != nil {
			return err
		}
		if err := validateUsage(v.PackageID, v.LastUsedAt, v.UsedCount); err != nil {
			return err
		}
		updates[i] = models.PluginUsageUpdate{PackageID: v.PackageID, LastUsedAt: *v.LastUsedAt, UsedCount: int(*v.UsedCount)}
	}
	return failure("plugin.usage", s.core.UpdatePluginUsage(updates))
}
func (s *PluginService) InstallPluginApi(path string) error {
	if path == "" {
		return invalid("插件文件路径不能为空")
	}
	return failure("plugin.install", s.core.InstallPlugin(path))
}
func (s *PluginService) InstallPluginByFileDialogApi() error {
	if s.pickFiles == nil {
		return &Error{Code: "UNAVAILABLE", Message: "文件选择对话框不可用"}
	}
	paths, err := s.pickFiles()
	if err != nil {
		return failure("plugin.pickFiles", err)
	}
	return failure("plugin.install", s.core.InstallPluginPaths(paths))
}
func (s *PluginService) UninstallPluginApi(id string) error {
	if err := validatePackage(id); err != nil {
		return err
	}
	return failure("plugin.uninstall", s.core.UninstallPlugin(id))
}
func (s *PluginService) TogglePluginApi(id string, enabled *bool) error {
	if err := validatePackage(id); err != nil {
		return err
	}
	if enabled == nil {
		return invalid("enabled 必须是布尔值")
	}
	return failure("plugin.toggle", s.core.TogglePlugin(id, *enabled))
}
