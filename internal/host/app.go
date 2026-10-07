package host

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"

	"watools/config"
	"watools/internal/app"
	"watools/pkg/logger"
)

type AppService struct {
	core   *app.WaApp
	debug  func() bool
	window func() WindowGeometry
}

func NewAppService(core *app.WaApp, debug func() bool, window func() WindowGeometry) *AppService {
	return &AppService{core: core, debug: debug, window: window}
}
func (s *AppService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	config.InitEnvironment(s.debug())
	return nil
}
func (s *AppService) ServiceShutdown() error { s.core.Shutdown(context.Background()); return nil }
func (s *AppService) HideAppApi()            { s.core.HideApp() }
func (s *AppService) HideOrShowAppApi()      { s.core.HideOrShowApp() }
func (s *AppService) GetClipboardContentApi() (*app.ClipboardContent, error) {
	v, e := s.core.GetClipboardContent()
	return v, failure("app.clipboard", e)
}
func (s *AppService) SetClipboardTextApi(text string) error {
	return failure("app.clipboardWrite", s.core.SetClipboardText(text))
}
func (s *AppService) GetAutomaticClipboardApi() (*app.AutomaticClipboard, error) {
	v, e := s.core.GetAutomaticClipboard()
	return v, failure("app.automaticClipboard", e)
}
func (s *AppService) GetClipboardSnapshotTextApi(id string) (string, error) {
	if len(id) != 32 {
		return "", invalid("剪贴板快照标识无效")
	}
	v, e := s.core.GetClipboardSnapshotText(id)
	if errors.Is(e, app.ErrClipboardSnapshotExpired) {
		return "", &Error{Code: "UNAVAILABLE", Message: "剪贴板快照已释放或过期，请重新复制文本", Operation: "app.clipboardSnapshot", cause: e}
	}
	return v, failure("app.clipboardSnapshot", e)
}
func (s *AppService) ReleaseClipboardSnapshotApi(id string) { s.core.ReleaseClipboardSnapshot(id) }
func (s *AppService) RetainClipboardSnapshotApi(id string) error {
	if len(id) != 32 {
		return invalid("剪贴板快照标识无效")
	}
	e := s.core.RetainClipboardSnapshot(id)
	if errors.Is(e, app.ErrClipboardSnapshotExpired) {
		return &Error{Code: "UNAVAILABLE", Message: "剪贴板快照已释放或过期，请重新复制文本", Operation: "app.clipboardSnapshot", cause: e}
	}
	return failure("app.clipboardSnapshot", e)
}
func (s *AppService) GetHotkeyEnvironmentStatusApi() app.HotkeyEnvironmentStatus {
	return s.core.GetHotkeyEnvironmentStatus()
}

type HotkeyUpdateRequest struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hotkey string `json:"hotkey"`
}

func (s *AppService) GetHotkeysApi() []app.HotkeyConfigAPI {
	return s.core.GetHotkeys()
}

func (s *AppService) UpdateHotkeyApi(request HotkeyUpdateRequest) error {
	if request.ID != "show-hide-window" {
		return invalid("未知快捷键标识")
	}
	if len(request.Name) > 256 || len(request.Hotkey) > 128 {
		return invalid("快捷键名称或组合过长")
	}
	cfg := app.HotkeyConfig{ID: request.ID, Name: request.Name, Hotkey: request.Hotkey}
	if _, _, err := cfg.ParseHotkey(); err != nil {
		return &Error{Code: "INVALID_ARGUMENT", Message: "快捷键组合格式无效", Operation: "app.hotkeyUpdate", cause: err}
	}
	err := s.core.UpdateHotkey(app.HotkeyConfigAPI{ID: cfg.ID, Name: cfg.Name, Hotkey: cfg.Hotkey})
	if err != nil {
		logger.Error(err, "Failed to update hotkey")
	}
	return hotkeyFailure(err)
}

func hotkeyFailure(err error) error {
	if err == nil {
		return nil
	}
	var updateErr *app.HotkeyUpdateError
	if errors.As(err, &updateErr) {
		message := "快捷键更新失败，旧配置和注册已恢复"
		if updateErr.RecoveryErr != nil {
			message = "快捷键更新失败，恢复也失败，请核对配置和实际注册状态并查看日志"
		}
		return &Error{Code: "OPERATION_FAILED", Message: message, Operation: "app.hotkeyUpdate", cause: err}
	}
	return failure("app.hotkeyUpdate", err)
}

// WindowGeometry is the native window boundary needed by the launcher layout.
// Wails WebviewWindow and test doubles both implement it.
type WindowGeometry interface {
	Size() (int, int)
	SetSize(int, int) application.Window
}
type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (s *AppService) GetLauncherSizeApi() (WindowSize, error) {
	if s.window == nil || s.window() == nil {
		return WindowSize{}, &Error{Code: "UNAVAILABLE", Message: "启动器窗口不可用"}
	}
	w, h := s.window().Size()
	return WindowSize{Width: w, Height: h}, nil
}
func (s *AppService) SetLauncherSizeApi(width, height int) error {
	if width <= 0 || width > 20000 || height <= 0 || height > 20000 {
		return invalid("窗口尺寸必须是 1 到 20000 的整数")
	}
	if s.window == nil || s.window() == nil {
		return &Error{Code: "UNAVAILABLE", Message: "启动器窗口不可用"}
	}
	s.window().SetSize(width, height)
	return nil
}
