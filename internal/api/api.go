package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	waApiInstance *WaApi
	waApiOnce     sync.Once
)

type WaApi struct {
	httpClient *http.Client
	requestMu  sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	slots      chan struct{}
	requests   map[string]context.CancelFunc
}

func GetWaApi() *WaApi {
	waApiOnce.Do(func() {
		waApiInstance = &WaApi{
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	})
	return waApiInstance
}

func (a *WaApi) SaveBase64Image(base64Data string) (string, error) {
	if len(base64Data) > 32<<20 {
		return "", fmt.Errorf("image exceeds size limit")
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("invalid PNG: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, name := range []string{"Downloads", "downloads", "Download", "download"} {
		dir := filepath.Join(home, name)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		return saveImageFile(dir, data)
	}
	return "", fmt.Errorf("downloads directory is unavailable")
}
func saveImageFile(dir string, data []byte) (string, error) {
	file, err := os.CreateTemp(dir, "wa-image-*.png")
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func (a *WaApi) CopyBase64ImageToClipboard(base64Data string) error {
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 image: %w", err)
	}

	return a.copyImageBytesToClipboard(imgBytes)
}

// HttpProxyRequest represents a generic HTTP request
type HttpProxyRequest struct {
	RequestID string            `json:"requestId,omitempty"`
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	Timeout   int               `json:"timeout,omitempty"` // Timeout in milliseconds
}

// HttpProxyResponse represents the HTTP response
type HttpProxyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// HttpProxy performs a generic HTTP request and returns the response
// This allows plugins to make HTTP requests without CORS restrictions
const maxProxyBody = 8 << 20

func (a *WaApi) OnStartup(parent context.Context) {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()
	a.ctx, a.cancel = context.WithCancel(parent)
	a.slots = make(chan struct{}, 8)
	a.requests = make(map[string]context.CancelFunc)
}
func (a *WaApi) Shutdown() {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}
func (a *WaApi) CancelHttpProxy(id string) {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()
	if cancel := a.requests[id]; cancel != nil {
		cancel()
	}
}
func (a *WaApi) HttpProxy(req HttpProxyRequest) (*HttpProxyResponse, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}
	if len(req.Body) > 4<<20 {
		return nil, fmt.Errorf("request body exceeds 4 MiB")
	}
	if req.Timeout < 0 || req.Timeout > 60000 {
		return nil, fmt.Errorf("timeout must be between 0 and 60000 ms")
	}
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Millisecond
	}
	a.requestMu.Lock()
	if a.ctx == nil {
		a.ctx, a.cancel = context.WithCancel(context.Background())
		a.slots = make(chan struct{}, 8)
		a.requests = make(map[string]context.CancelFunc)
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	slots := a.slots
	if req.RequestID != "" {
		if _, exists := a.requests[req.RequestID]; exists {
			a.requestMu.Unlock()
			cancel()
			return nil, fmt.Errorf("requestId is already in use")
		}
		a.requests[req.RequestID] = cancel
	}
	a.requestMu.Unlock()
	defer func() { cancel(); a.requestMu.Lock(); delete(a.requests, req.RequestID); a.requestMu.Unlock() }()
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	request, err := http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		return nil, fmt.Errorf("only HTTP(S) requests are supported")
	}
	for key, value := range req.Headers {
		request.Header.Set(key, value)
	}
	client := a.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProxyBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxProxyBody {
		return nil, fmt.Errorf("response body exceeds 8 MiB")
	}
	headers := make(map[string]string)
	for key, values := range response.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return &HttpProxyResponse{StatusCode: response.StatusCode, Headers: headers, Body: string(body)}, nil
}

// Plugin Storage API

// PluginStorageGetRequest represents a get storage request
type PluginStorageGetRequest struct {
	PackageID string `json:"packageId"`
	Key       string `json:"key"`
}

// PluginStorageSetRequest represents a set storage request
type PluginStorageSetRequest struct {
	PackageID string      `json:"packageId"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
}

// PluginStorageGet retrieves a value from plugin storage
func (a *WaApi) PluginStorageGet(req PluginStorageGetRequest) (interface{}, error) {
	if req.PackageID == "" {
		return nil, fmt.Errorf("packageId cannot be empty")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	// Will be injected by coordinator
	return nil, fmt.Errorf("not implemented - must be called through coordinator")
}

// PluginStorageSet sets a value in plugin storage
func (a *WaApi) PluginStorageSet(req PluginStorageSetRequest) error {
	if req.PackageID == "" {
		return fmt.Errorf("packageId cannot be empty")
	}
	if req.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Will be injected by coordinator
	return fmt.Errorf("not implemented - must be called through coordinator")
}

// PluginStorageRemove removes a key from plugin storage
func (a *WaApi) PluginStorageRemove(req PluginStorageGetRequest) error {
	if req.PackageID == "" {
		return fmt.Errorf("packageId cannot be empty")
	}
	if req.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Will be injected by coordinator
	return fmt.Errorf("not implemented - must be called through coordinator")
}

// PluginStorageClear clears all storage for a plugin
func (a *WaApi) PluginStorageClear(packageID string) error {
	if packageID == "" {
		return fmt.Errorf("packageId cannot be empty")
	}

	// Will be injected by coordinator
	return fmt.Errorf("not implemented - must be called through coordinator")
}

// PluginStorageKeys returns all keys in plugin storage
func (a *WaApi) PluginStorageKeys(packageID string) ([]string, error) {
	if packageID == "" {
		return nil, fmt.Errorf("packageId cannot be empty")
	}

	// Will be injected by coordinator
	return nil, fmt.Errorf("not implemented - must be called through coordinator")
}
