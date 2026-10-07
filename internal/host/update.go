package host

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"watools/config"
	"watools/internal/localinstall"
	"watools/internal/update"
	"watools/pkg/logger"
)

type UpdateService struct {
	core  *update.Service
	quit  func()
	ctx   context.Context
	local localInstallation
}

// localInstallation is the job boundary; native construction uses Manager while
// tests can exercise failure mapping without spawning an installer.
type localInstallation interface {
	Status() (localinstall.Status, error)
	Configure(localinstall.Configuration) error
	Start(context.Context, localinstall.Request) error
	Cancel() error
	Shutdown() error
}

func NewUpdateService(core *update.Service, quit func()) *UpdateService {
	return &UpdateService{core: core, quit: quit, ctx: context.Background(), local: localinstall.NewManager(config.ProjectDataDir(), config.ProjectCacheDir())}
}
func (s *UpdateService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}
func (s *UpdateService) ServiceShutdown() error {
	err := s.local.Shutdown()
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		logger.Error(err, "Failed to shut down local installation")
	}
	return installationFailure("installation.shutdown", err)
}
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
	return installationFailure("installation.configure", s.local.Configure(value))
}
func (s *UpdateService) StartLocalInstallationApi(request localinstall.Request) error {
	return installationFailure("installation.start", s.local.Start(s.ctx, request))
}
func (s *UpdateService) CancelLocalInstallationApi() error {
	return installationFailure("installation.cancel", s.local.Cancel())
}

func installationFailure(operation string, err error) error {
	if err == nil {
		return nil
	}
	// Cancellation and timeout remain operational context outcomes even if a
	// future implementation wraps them in a validation or busy error.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return failure(operation, err)
	}
	var validation *localinstall.ValidationError
	if errors.As(err, &validation) {
		return &Error{Code: "INVALID_ARGUMENT", Message: validation.Message, Operation: operation, cause: err}
	}
	if errors.Is(err, localinstall.ErrBusy) {
		return &Error{Code: "UNAVAILABLE", Message: "安装任务正在执行或提交，请稍后重试", Operation: operation, cause: err}
	}
	return failure(operation, err)
}
