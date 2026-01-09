package app

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ClipboardContentType represents the primary type of content in clipboard
type ClipboardContentType string

const (
	ClipboardTypeText  ClipboardContentType = "text"
	ClipboardTypeImage ClipboardContentType = "image"
	ClipboardTypeFiles ClipboardContentType = "files"
	ClipboardTypeEmpty ClipboardContentType = "empty"
)

// ClipboardContent contains comprehensive clipboard data with automatic type detection
type ClipboardContent struct {
	Types       []string             `json:"types"`
	ContentType ClipboardContentType `json:"contentType"`
	HasText     bool                 `json:"hasText"`
	HasImage    bool                 `json:"hasImage"`
	HasFiles    bool                 `json:"hasFiles"`
	Text        string               `json:"text,omitempty"`
	ImageBase64 string               `json:"imageBase64,omitempty"`
	Files       []string             `json:"files,omitempty"`
}

func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("powershell failed: %w\n%s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

// ==================== Window Management ====================

func (a *WaApp) showAppAsPanel() {
	time.Sleep(10 * time.Millisecond)
	a.checkAndRepositionIfNeeded()
	runtime.WindowShow(a.ctx)
	a.isHidden = false
}

func (a *WaApp) hideAppWithFocusReturn() {
	if !a.isHidden {
		runtime.WindowHide(a.ctx)
		a.isHidden = true
	}
}

func (a *WaApp) HideApp() {
	a.hideAppWithFocusReturn()
}

func (a *WaApp) ShowApp() {
	a.showAppAsPanel()
}

func (a *WaApp) HideOrShowApp() {
	if a.isHidden {
		a.ShowApp()
	} else {
		a.HideApp()
	}
}

// ==================== Clipboard API ====================

// GetClipboardTypes returns available clipboard type identifiers on Windows
func (a *WaApp) GetClipboardTypes() ([]string, error) {
	script := "$types=@(); try{ $null=Get-Clipboard -Format Text -ErrorAction Stop; $types+='text' } catch {}; " +
		"try{ $null=Get-Clipboard -Format Image -ErrorAction Stop; $types+='image' } catch {}; " +
		"try{ $null=Get-Clipboard -Format FileDropList -ErrorAction Stop; $types+='files' } catch {}; " +
		"@($types) | ConvertTo-Json -Compress"
	out, err := runPowerShell(script)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	var types []string
	if err := json.Unmarshal([]byte(out), &types); err != nil {
		return nil, fmt.Errorf("failed to parse clipboard types: %w", err)
	}
	return types, nil
}

// HasClipboardType checks if clipboard contains a specific type identifier
func (a *WaApp) HasClipboardType(typeStr string) bool {
	types, err := a.GetClipboardTypes()
	if err != nil {
		return false
	}
	typeStr = strings.ToLower(typeStr)
	for _, t := range types {
		if strings.ToLower(t) == typeStr {
			return true
		}
	}
	return false
}

// GetClipboardText returns plain text from clipboard
func (a *WaApp) GetClipboardText() (string, error) {
	script := "try { Get-Clipboard -Raw -Format Text } catch { '' }"
	out, err := runPowerShell(script)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("no text in clipboard")
	}
	return out, nil
}

// GetClipboardImage returns clipboard image as base64-encoded PNG
func (a *WaApp) GetClipboardImage() (string, error) {
	script := "Add-Type -AssemblyName System.Drawing; " +
		"$img = Get-Clipboard -Format Image -ErrorAction Stop; " +
		"$ms = New-Object System.IO.MemoryStream; " +
		"$img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png); " +
		"[Convert]::ToBase64String($ms.ToArray())"
	out, err := runPowerShell(script)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("no image in clipboard")
	}
	return out, nil
}

// GetClipboardFiles returns absolute file paths from clipboard
func (a *WaApp) GetClipboardFiles() ([]string, error) {
	script := "try { $files = Get-Clipboard -Format FileDropList -ErrorAction Stop; @($files) | ConvertTo-Json -Compress } catch { '[]' }"
	out, err := runPowerShell(script)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	var files []string
	if err := json.Unmarshal([]byte(out), &files); err != nil {
		return nil, fmt.Errorf("failed to parse clipboard files: %w", err)
	}
	return files, nil
}

// GetClipboardContent performs automatic type detection and returns all available content
// Priority order for ContentType: Files > Image > Text > Empty
func (a *WaApp) GetClipboardContent() (*ClipboardContent, error) {
	content := &ClipboardContent{}

	types, err := a.GetClipboardTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get clipboard types: %w", err)
	}
	content.Types = types

	if len(types) == 0 {
		content.ContentType = ClipboardTypeEmpty
		return content, nil
	}

	for _, t := range types {
		switch strings.ToLower(t) {
		case "text":
			content.HasText = true
		case "image":
			content.HasImage = true
		case "files":
			content.HasFiles = true
		}
	}

	if content.HasFiles {
		content.ContentType = ClipboardTypeFiles
	} else if content.HasImage {
		content.ContentType = ClipboardTypeImage
	} else if content.HasText {
		content.ContentType = ClipboardTypeText
	} else {
		content.ContentType = ClipboardTypeEmpty
	}

	if content.HasText {
		if text, err := a.GetClipboardText(); err == nil {
			content.Text = text
		}
	}

	if content.HasImage {
		if imageBase64, err := a.GetClipboardImage(); err == nil {
			content.ImageBase64 = imageBase64
		}
	}

	if content.HasFiles {
		if files, err := a.GetClipboardFiles(); err == nil {
			content.Files = files
		}
	}

	return content, nil
}
