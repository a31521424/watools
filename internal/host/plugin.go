package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"sync"
	"watools/internal/plugin"
	"watools/pkg/models"
)

type PluginService struct {
	core       *plugin.WaPlugin
	pickFiles  func() ([]string, error)
	ctx        context.Context
	mutationMu sync.Mutex
	windows    pluginWindows
}

func NewPluginService(core *plugin.WaPlugin, pickFiles func() ([]string, error), factory ...PluginWindowFactory) *PluginService {
	s := &PluginService{core: core, pickFiles: pickFiles}
	if len(factory) > 0 {
		s.windows.factory = factory[0]
	}
	return s
}
func (s *PluginService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	s.core.OnStartup(ctx)
	return nil
}
func (s *PluginService) ServiceShutdown() error {
	s.windows.shutdown()
	s.core.OnShutdown(s.ctx)
	return nil
}
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
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
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
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	return failure("plugin.install", s.core.InstallPluginPaths(paths))
}
func (s *PluginService) UninstallPluginApi(id string) error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if err := validatePackage(id); err != nil {
		return err
	}
	s.windows.closePackage(id)
	return failure("plugin.uninstall", s.core.UninstallPlugin(id))
}
func (s *PluginService) TogglePluginApi(id string, enabled *bool) error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	if err := validatePackage(id); err != nil {
		return err
	}
	if enabled == nil {
		return invalid("enabled 必须是布尔值")
	}
	if !*enabled {
		s.windows.closePackage(id)
	}
	return failure("plugin.toggle", s.core.TogglePlugin(id, *enabled))
}
