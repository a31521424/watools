// Package localinstall owns the shared source-install configuration and job runner.
// It executes the repository workflow rather than exposing build privileges to plugins.
package localinstall

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"watools/internal/workflow"
	"watools/pkg/models"
)

type Configuration struct {
	SourceRoot      string `json:"sourceRoot"`
	GoExecutable    string `json:"goExecutable"`
	ToolPath        string `json:"toolPath"`
	ApplicationPath string `json:"applicationPath"`
	InstalledAt     string `json:"installedAt"`
	Version         string `json:"version"`
}

type OfficialPlugin struct {
	PackageID string `json:"packageId"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

func ReadConfiguration(dataDir string) (Configuration, error) {
	var value Configuration
	content, err := os.ReadFile(filepath.Join(dataDir, "installation.json"))
	if os.IsNotExist(err) {
		return value, nil
	}
	if err == nil {
		err = json.Unmarshal(content, &value)
	}
	return value, err
}

func SaveConfiguration(dataDir string, value Configuration) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dataDir, ".installation-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(content); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dataDir, "installation.json"))
}

func ValidateSource(root string) ([]OfficialPlugin, error) {
	if !filepath.IsAbs(root) {
		return nil, validation("源码目录必须是绝对路径", nil)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, sourcePathError("找不到 WaTools 源码目录", err)
	}
	if !info.IsDir() {
		return nil, validation("源码目录必须是文件夹", nil)
	}
	project, err := os.ReadFile(filepath.Join(root, "wails.json"))
	if err != nil {
		return nil, sourcePathError("找不到 WaTools 源码目录", err)
	}
	var identity struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(project, &identity); err != nil {
		return nil, validation("WaTools 项目文件格式不正确", err)
	}
	if identity.Name != "watools" {
		return nil, validation("目录不是 WaTools 源码", nil)
	}
	for _, path := range []string{"cmd/watoolsctl/main.go", "go.mod", "frontend/pnpm-lock.yaml"} {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil {
			return nil, sourcePathError("源码缺少 "+path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, validation("源码缺少 "+path, nil)
		}
	}
	files, err := filepath.Glob(filepath.Join(root, "plugins/official/*/plugin/manifest.json"))
	if err != nil {
		return nil, err
	}
	result := make([]OfficialPlugin, 0, len(files))
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var manifest models.PluginMetadata
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, validation("官方插件 manifest 格式不正确", err)
		}
		result = append(result, OfficialPlugin{manifest.PackageID, manifest.Name, manifest.Version})
	}
	if len(result) == 0 {
		return nil, validation("源码缺少官方插件", nil)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PackageID < result[j].PackageID })
	return result, nil
}

// Missing source files are a correctable choice of source tree. Permission and
// other filesystem failures are operational errors, even during validation.
func sourcePathError(message string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return validation(message, err)
	}
	return fmt.Errorf("%s: %w", message, err)
}

func ValidateConfiguration(value Configuration) (Configuration, error) {
	sourceRoot := strings.TrimSpace(value.SourceRoot)
	if !filepath.IsAbs(sourceRoot) {
		return value, validation("源码目录必须是绝对路径", nil)
	}
	root, err := filepath.EvalSymlinks(sourceRoot)
	if err != nil {
		return value, sourcePathError("找不到 WaTools 源码目录", err)
	}
	value.SourceRoot = root
	if _, err := ValidateSource(root); err != nil {
		return value, err
	}
	if value.ToolPath == "" {
		value.ToolPath = os.Getenv("PATH")
	}
	if value.GoExecutable == "" {
		value.GoExecutable, err = workflow.FindExecutable("go", value.ToolPath)
		if err != nil {
			return value, validation("找不到 Go，请填写 Go 可执行文件的绝对路径", err)
		}
	}
	if !filepath.IsAbs(value.GoExecutable) {
		return value, validation("Go 必须使用绝对路径", nil)
	}
	info, err := os.Stat(value.GoExecutable)
	if err != nil {
		return value, sourcePathError("Go 路径不可执行", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return value, validation("Go 路径不可执行", nil)
	}
	if !strings.HasPrefix(value.ToolPath, filepath.Dir(value.GoExecutable)+string(os.PathListSeparator)) {
		value.ToolPath = filepath.Dir(value.GoExecutable) + string(os.PathListSeparator) + value.ToolPath
	}
	if value.ApplicationPath == "" {
		value.ApplicationPath = "/Applications/watools.app"
	}
	if !filepath.IsAbs(value.ApplicationPath) || filepath.Base(value.ApplicationPath) != "watools.app" {
		return value, validation("应用位置必须是绝对路径且以 watools.app 结尾", nil)
	}
	return value, nil
}
