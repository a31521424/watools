package host

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"net/http"
	"net/url"
	"watools/internal/api"
)

type SystemService struct{ core *api.WaApi }

func NewSystemService(core *api.WaApi) *SystemService { return &SystemService{core: core} }
func (s *SystemService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.core.OnStartup(ctx)
	return nil
}
func (s *SystemService) ServiceShutdown() error { s.core.Shutdown(); return nil }
func (s *SystemService) OpenFolder(path string) error {
	if path == "" {
		return invalid("文件夹路径不能为空")
	}
	return failure("system.openFolder", s.core.OpenFolderWithPath(path))
}
func (s *SystemService) SaveBase64Image(data string) (string, error) {
	v, e := s.core.SaveBase64Image(data)
	return v, failure("system.saveImage", e)
}
func (s *SystemService) CopyBase64ImageToClipboard(data string) error {
	return failure("system.copyImage", s.core.CopyBase64ImageToClipboard(data))
}
func (s *SystemService) HttpProxyApi(r api.HttpProxyRequest) (*api.HttpProxyResponse, error) {
	u, e := url.Parse(r.URL)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, invalid("url 必须是有效 HTTP(S) 地址")
	}
	if r.Timeout < 0 || r.Timeout > 60000 {
		return nil, invalid("timeout 必须是 0 到 60000 的整数毫秒")
	}
	if len(r.Body) > 4<<20 {
		return nil, invalid("请求体不能超过 4 MiB")
	}
	if _, e := http.NewRequest(r.Method, r.URL, nil); e != nil {
		return nil, invalid("HTTP method 格式不正确")
	}
	v, e := s.core.HttpProxy(r)
	return v, failure("system.http", e)
}
func (s *SystemService) CancelHttpProxyApi(id string) { s.core.CancelHttpProxy(id) }
