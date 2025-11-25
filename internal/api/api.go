package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"watools/pkg/logger"
)

var (
	waApiInstance *WaApi
	waApiOnce     sync.Once
)

type WaApi struct {
	httpClient *http.Client
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

func (a *WaApi) SaveBase64Image(base64Data string) string {
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return ""
	}
	downloadFolder := []string{"Downloads", "downloads", "Download", "download"}
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	for _, ddn := range downloadFolder {
		downloadPath := path.Join(userHomeDir, ddn)
		if _, err := os.Stat(downloadPath); err == nil {
			filePath := path.Join(downloadPath, fmt.Sprint("wa-image-", time.Now().Unix(), ".png"))
			err = os.WriteFile(filePath, imgBytes, 0644)
			if err != nil {
				continue
			}
			return filePath
		}
	}
	return ""
}

// HttpProxyRequest represents a generic HTTP request
type HttpProxyRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // Timeout in milliseconds
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
func (a *WaApi) HttpProxy(req HttpProxyRequest) (*HttpProxyResponse, error) {
	// Validate request
	if req.URL == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}
	if req.Method == "" {
		req.Method = "GET" // Default to GET
	}

	// Create HTTP request
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to create HTTP request: %s %s", req.Method, req.URL))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set custom timeout if provided
	client := a.httpClient
	if req.Timeout > 0 {
		client = &http.Client{
			Timeout: time.Duration(req.Timeout) * time.Millisecond,
		}
	}

	// Send request
	logger.Info(fmt.Sprintf("Proxying HTTP request: %s %s (timeout: %v)", req.Method, req.URL, client.Timeout))

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Error(err, fmt.Sprintf("HTTP request failed: %s", req.URL))
		return &HttpProxyResponse{
			Error: fmt.Sprintf("request failed: %v", err),
		}, err
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "Failed to read response body")
		return &HttpProxyResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("failed to read response: %v", err),
		}, err
	}

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	// Build response
	response := &HttpProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(bodyBytes),
	}

	logger.Info(fmt.Sprintf("HTTP proxy response received: status=%d, size=%d bytes", resp.StatusCode, len(bodyBytes)))

	return response, nil
}
