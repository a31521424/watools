// Package localinstall owns the shared source-install configuration and job runner.
// It executes the repository workflow rather than exposing build privileges to plugins.
package localinstall

import (
	"encoding/json"
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
		return nil, fmt.Errorf("源码目录必须是绝对路径")
	}
	project, err := os.ReadFile(filepath.Join(root, "wails.json"))
	if err != nil {
		return nil, fmt.Errorf("找不到 WaTools 源码目录: %w", err)
	}
	var identity struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(project, &identity); err != nil || identity.Name != "watools" {
		return nil, fmt.Errorf("目录不是 WaTools 源码")
	}
	for _, path := range []string{"cmd/watoolsctl/main.go", "go.mod", "frontend/pnpm-lock.yaml"} {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("源码缺少 %s", path)
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
			return nil, err
		}
		result = append(result, OfficialPlugin{manifest.PackageID, manifest.Name, manifest.Version})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("源码缺少官方插件")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PackageID < result[j].PackageID })
	return result, nil
}

func ValidateConfiguration(value Configuration) (Configuration, error) {
	root, err := filepath.EvalSymlinks(strings.TrimSpace(value.SourceRoot))
	if err != nil {
		return value, err
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
			return value, fmt.Errorf("找不到 Go，请填写 Go 可执行文件的绝对路径")
		}
	}
	if !filepath.IsAbs(value.GoExecutable) {
		return value, fmt.Errorf("Go 必须使用绝对路径")
	}
	info, err := os.Stat(value.GoExecutable)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return value, fmt.Errorf("Go 路径不可执行")
	}
	if !strings.HasPrefix(value.ToolPath, filepath.Dir(value.GoExecutable)+string(os.PathListSeparator)) {
		value.ToolPath = filepath.Dir(value.GoExecutable) + string(os.PathListSeparator) + value.ToolPath
	}
	if value.ApplicationPath == "" {
		value.ApplicationPath = "/Applications/watools.app"
	}
	if !filepath.IsAbs(value.ApplicationPath) || filepath.Base(value.ApplicationPath) != "watools.app" {
		return value, fmt.Errorf("应用位置必须是绝对路径且以 watools.app 结尾")
	}
	return value, nil
}
