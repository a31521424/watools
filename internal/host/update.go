package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"time"
	"watools/config"
	"watools/internal/localinstall"
	"watools/internal/update"
)

type UpdateService struct {
	core  *update.Service
	quit  func()
	ctx   context.Context
	local *localinstall.Manager
}

func NewUpdateService(core *update.Service, quit func()) *UpdateService {
	return &UpdateService{core: core, quit: quit, ctx: context.Background(), local: localinstall.NewManager(config.ProjectDataDir(), config.ProjectCacheDir())}
}
func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}
func (s *UpdateService) ServiceShutdown() error { s.local.Shutdown(); return nil }
func (s *UpdateService) CheckForUpdatesApi() (update.UpdateInfo, error) {
	v, e := s.core.Check(s.ctx)
	return v, failure("update.check", e)
}
func (s *UpdateService) DownloadUpdateApi() (update.UpdateInfo, error) {
	v, e := s.core.Download(s.ctx)
	return v, failure("update.download", e)
}
func (s *UpdateService) InstallUpdateApi(path string) (update.InstallResult, error) {
	if path == "" {
		return update.InstallResult{}, invalid("更新包路径不能为空")
	}
	v, e := s.core.Install(path)
	if e != nil {
		return v, failure("update.install", e)
	}
	if v.ShouldQuit && s.quit != nil {
		go func() {
			timer := time.NewTimer(400 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
				s.quit()
			case <-s.ctx.Done():
			}
		}()
	}
	return v, nil
}

func (s *UpdateService) GetLocalInstallationApi() (localinstall.Status, error) {
	value, err := s.local.Status()
	return value, failure("installation.status", err)
}
func (s *UpdateService) ConfigureLocalInstallationApi(value localinstall.Configuration) error {
	if err := s.local.Configure(value); err != nil {
		return invalid(err.Error())
	}
	return nil
}
func (s *UpdateService) StartLocalInstallationApi(request localinstall.Request) error {
	if err := s.local.Start(s.ctx, request); err != nil {
		return invalid(err.Error())
	}
	return nil
}
func (s *UpdateService) CancelLocalInstallationApi() error {
	if err := s.local.Cancel(); err != nil {
		return invalid(err.Error())
	}
	return nil
}
