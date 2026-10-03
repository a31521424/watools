package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"time"
	"watools/internal/update"
)

type UpdateService struct {
	core *update.Service
	quit func()
	ctx  context.Context
}

func NewUpdateService(core *update.Service, quit func()) *UpdateService {
	return &UpdateService{core: core, quit: quit, ctx: context.Background()}
}
func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}
func (s *UpdateService) ServiceShutdown() error { return nil }
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
