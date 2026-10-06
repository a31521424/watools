package host

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"watools/config"
	"watools/pkg/utils"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type PluginInput struct {
	Value                string `json:"value"`
	ValueType            string `json:"valueType"`
	ClipboardContentType string `json:"clipboardContentType,omitempty"`
}
type PluginClipboard struct {
	ContentType string   `json:"contentType"`
	Text        *string  `json:"text"`
	ImageBase64 *string  `json:"imageBase64"`
	Files       []string `json:"files"`
}
type PluginLaunchContext struct {
	Input     PluginInput      `json:"input"`
	Clipboard *PluginClipboard `json:"clipboard"`
}
type DetachPluginRequest struct {
	PackageID string              `json:"packageId"`
	File      string              `json:"file"`
	HomeURL   string              `json:"homeUrl"`
	Context   PluginLaunchContext `json:"context"`
	State     json.RawMessage     `json:"state"`
}
type PluginWindowSession struct {
	ID         string `json:"id"`
	WindowName string `json:"windowName"`
	Title      string `json:"title"`
	DetachPluginRequest
}
type PluginWindowStatus struct {
	Ready bool `json:"ready"`
}

// The native window is the only replaceable boundary. No cache lock is held
// while creating, showing or closing a window on the application thread.
type PluginWindow interface {
	Show()
	Close()
}
type PluginWindowFactory func(PluginWindowSession, func()) (PluginWindow, error)
type nativePluginWindow struct{ window *application.WebviewWindow }

func (w nativePluginWindow) Show()  { w.window.Show(); w.window.Focus() }
func (w nativePluginWindow) Close() { w.window.Close() }

func NewPluginWindowFactory(desktop func() *application.App, cancelRequests func(string)) PluginWindowFactory {
	return func(session PluginWindowSession, closed func()) (PluginWindow, error) {
		if application.System.IsServer() {
			return nil, &Error{Code: "UNAVAILABLE", Message: "浏览器模式不支持原生插件小窗"}
		}
		window := desktop().Window.NewWithOptions(application.WebviewWindowOptions{
			Name: session.WindowName, Title: session.Title, URL: "/?pluginSession=" + session.ID,
			Width: 960, Height: 760, MinWidth: 480, MinHeight: 320, Hidden: true,
			BackgroundColour: application.NewRGB(255, 255, 255),
		})
		windowName := session.WindowName
		window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { cancelRequests(windowName + ":"); closed() })
		return nativePluginWindow{window}, nil
	}
}

type pluginWindowRecord struct {
	session PluginWindowSession
	window  PluginWindow
	ready   bool
	timer   *time.Timer
}
type pluginWindows struct {
	mu      sync.Mutex
	items   map[string]*pluginWindowRecord
	factory PluginWindowFactory
	closed  bool
}

func sourceWindow(ctx context.Context) string {
	if window, ok := ctx.Value(application.WindowKey).(interface{ Name() string }); ok {
		return window.Name()
	}
	return ""
}
func unavailableSession() error {
	return &Error{Code: "UNAVAILABLE", Message: "插件窗口会话不可用，请返回主窗口重试"}
}

func (w *pluginWindows) create(request DetachPluginRequest, title string) (string, error) {
	w.mu.Lock()
	if w.closed || w.factory == nil {
		w.mu.Unlock()
		return "", &Error{Code: "UNAVAILABLE", Message: "当前运行模式不支持插件小窗"}
	}
	if len(w.items) >= 32 {
		w.mu.Unlock()
		return "", &Error{Code: "UNAVAILABLE", Message: "请先关闭不再使用的插件窗口"}
	}
	for _, item := range w.items {
		if !item.ready {
			w.mu.Unlock()
			return "", &Error{Code: "UNAVAILABLE", Message: "已有插件正在分离，请稍候"}
		}
	}
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	record := &pluginWindowRecord{session: PluginWindowSession{ID: id, WindowName: "plugin-" + id, Title: title, DetachPluginRequest: request}}
	if w.items == nil {
		w.items = make(map[string]*pluginWindowRecord)
	}
	w.items[id] = record
	w.mu.Unlock()
	window, err := w.factory(record.session, func() { w.remove(id, false) })
	if err != nil {
		w.remove(id, false)
		return "", err
	}
	w.mu.Lock()
	if w.closed || w.items[id] != record {
		w.mu.Unlock()
		window.Close()
		return "", unavailableSession()
	}
	record.window = window
	if !record.ready {
		record.timer = time.AfterFunc(30*time.Second, func() { w.expire(id) })
	}
	w.mu.Unlock()
	return id, nil
}
func (w *pluginWindows) get(id, source string) (PluginWindowSession, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item := w.items[id]
	if item == nil || source != item.session.WindowName {
		return PluginWindowSession{}, unavailableSession()
	}
	return item.session, nil
}
func (w *pluginWindows) status(id string) (PluginWindowStatus, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item := w.items[id]
	if item == nil {
		return PluginWindowStatus{}, unavailableSession()
	}
	return PluginWindowStatus{Ready: item.ready}, nil
}
func (w *pluginWindows) ready(id, source string) error {
	w.mu.Lock()
	item := w.items[id]
	if item == nil || item.window == nil || source != item.session.WindowName {
		w.mu.Unlock()
		return unavailableSession()
	}
	if item.ready {
		w.mu.Unlock()
		return nil
	}
	item.ready = true
	if item.timer != nil {
		item.timer.Stop()
	}
	window := item.window
	w.mu.Unlock()
	window.Show()
	return nil
}
func (w *pluginWindows) remove(id string, closeWindow bool) {
	w.mu.Lock()
	item := w.items[id]
	delete(w.items, id)
	if item != nil && item.timer != nil {
		item.timer.Stop()
	}
	w.mu.Unlock()
	if item != nil && closeWindow && item.window != nil {
		item.window.Close()
	}
}

// A timer already in flight must not close a window that just acknowledged
// restoration. Check and remove atomically, then perform native work unlocked.
func (w *pluginWindows) expire(id string) {
	w.mu.Lock()
	item := w.items[id]
	if item == nil || item.ready {
		w.mu.Unlock()
		return
	}
	delete(w.items, id)
	w.mu.Unlock()
	if item.window != nil {
		item.window.Close()
	}
}
func (w *pluginWindows) closePackage(packageID string) {
	w.mu.Lock()
	ids := []string{}
	for id, item := range w.items {
		if item.session.PackageID == packageID {
			ids = append(ids, id)
		}
	}
	w.mu.Unlock()
	for _, id := range ids {
		w.remove(id, true)
	}
}
func (w *pluginWindows) shutdown() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	for _, item := range w.items {
		if item.timer != nil {
			item.timer.Stop()
		}
	}
	w.items = nil // The application owns native destruction during shutdown.
}

func (s *PluginService) DetachPluginApi(ctx context.Context, request DetachPluginRequest) (string, error) {
	if sourceWindow(ctx) != "launcher" {
		return "", invalid("只能从主窗口分离插件")
	}
	if err := validatePackage(request.PackageID); err != nil {
		return "", err
	}
	if request.Context.Input.ValueType != "text" && request.Context.Input.ValueType != "clipboard" {
		return "", invalid("插件输入上下文无效")
	}
	if len(request.State) > 64<<20 || !json.Valid(request.State) {
		return "", invalid("插件窗口状态必须是有效 JSON，且不超过 64 MiB")
	}
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()
	plugins, err := s.core.GetPlugins()
	if err != nil {
		return "", failure("plugin.detach", err)
	}
	for _, plugin := range plugins {
		if plugin.PackageID != request.PackageID || !plugin.Enabled {
			continue
		}
		// Running immutable revisions may be detached even after an upgrade.
		base := path.Join("/api/plugin", request.PackageID)
		root := filepath.Join(config.ProjectDataDir(), "plugins", request.PackageID)
		if request.HomeURL != base {
			if !strings.HasPrefix(request.HomeURL, base+"/_v/") {
				return "", invalid("插件资源版本无效")
			}
			revision := strings.TrimPrefix(request.HomeURL, base+"/_v/")
			if _, err := uuid.Parse(revision); err != nil {
				return "", invalid("插件资源版本无效")
			}
			root = filepath.Join(root, "versions", revision)
		}
		if request.File == "" || strings.ContainsAny(request.File, "\\?#") || path.Clean(request.File) != request.File || strings.HasPrefix(request.File, "/") {
			return "", invalid("插件页面路径无效")
		}
		file, err := utils.ResolvePathWithinBase(root, request.File)
		if err != nil {
			return "", invalid("插件页面路径无效")
		}
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			return "", invalid("插件页面不存在")
		}
		return s.windows.create(request, plugin.Name)
	}
	return "", &Error{Code: "PLUGIN_DISABLED", Message: "插件未安装或已停用"}
}
func (s *PluginService) GetPluginWindowApi(ctx context.Context, id string) (PluginWindowSession, error) {
	return s.windows.get(id, sourceWindow(ctx))
}
func (s *PluginService) GetPluginWindowStatusApi(ctx context.Context, id string) (PluginWindowStatus, error) {
	if sourceWindow(ctx) != "launcher" {
		return PluginWindowStatus{}, unavailableSession()
	}
	return s.windows.status(id)
}
func (s *PluginService) PluginWindowReadyApi(ctx context.Context, id string) error {
	return s.windows.ready(id, sourceWindow(ctx))
}
func (s *PluginService) ClosePluginWindowApi(ctx context.Context, id string) error {
	source := sourceWindow(ctx)
	if source != "launcher" {
		if _, err := s.windows.get(id, source); err != nil {
			return err
		}
	}
	s.windows.remove(id, true)
	return nil
}
