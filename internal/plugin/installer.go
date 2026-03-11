package plugin

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"watools/config"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"
	"watools/pkg/utils"
)

type PluginInstaller struct {
	ctx        context.Context
	pluginsDir string
}

func NewPluginInstaller(ctx context.Context) *PluginInstaller {
	pluginsDir := filepath.Join(config.ProjectCacheDir(), "plugins")
	return &PluginInstaller{
		ctx:        ctx,
		pluginsDir: pluginsDir,
	}
}

// InstallFromWtFile installs a plugin from a .wt file (zip format)
func (pi *PluginInstaller) InstallFromWtFile(wtFilePath string) error {
	logger.Info(fmt.Sprintf("Installing plugin from: %s", wtFilePath))

	// 1. 验证文件存在
	if _, err := os.Stat(wtFilePath); os.IsNotExist(err) {
		return fmt.Errorf("plugin file not found: %s", wtFilePath)
	}

	// 2. 创建临时解压目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("watools_plugin_%d", time.Now().Unix()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. 解压.wt文件
	if err := pi.unzipFile(wtFilePath, tempDir); err != nil {
		return fmt.Errorf("failed to unzip plugin: %w", err)
	}

	// 4. 读取并验证manifest.json
	manifestPath, err := pi.findManifestPath(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}
	manifest, err := pi.readManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// 5. 验证必需字段
	if err := pi.validateManifest(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	pluginRoot := filepath.Dir(manifestPath)
	if _, err := pi.resolvePluginFile(pluginRoot, manifest.Entry); err != nil {
		return fmt.Errorf("invalid plugin entry: %w", err)
	}

	// 6. 同包插件安装时，只有版本不低于已安装版本才允许覆盖安装
	if installedPlugin, found := pi.findInstalledPlugin(manifest.PackageID); found {
		installedManifest, err := installedPlugin.GetMetadata()
		if err != nil {
			return fmt.Errorf("failed to read installed plugin manifest: %w", err)
		}

		versionComparison, err := comparePluginVersions(manifest.Version, installedManifest.Version)
		if err != nil {
			return fmt.Errorf("failed to compare plugin versions: %w", err)
		}
		if versionComparison < 0 {
			return fmt.Errorf(
				"plugin %s version %s is older than installed version %s",
				manifest.PackageID,
				manifest.Version,
				installedManifest.Version,
			)
		}

		if err := pi.UninstallPlugin(manifest.PackageID); err != nil {
			return fmt.Errorf("failed to replace installed plugin %s: %w", manifest.PackageID, err)
		}
	}

	// 7. 创建插件目录
	pluginDir, err := utils.ResolvePathWithinBase(pi.pluginsDir, manifest.PackageID)
	if err != nil {
		return fmt.Errorf("invalid package installation path: %w", err)
	}
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// 8. 复制文件到插件目录
	if err := pi.copyDir(pluginRoot, pluginDir); err != nil {
		os.RemoveAll(pluginDir) // 清理失败的安装
		return fmt.Errorf("failed to copy plugin files: %w", err)
	}

	// 9. 插入数据库记录
	if err := pi.registerPlugin(manifest, pluginDir); err != nil {
		os.RemoveAll(pluginDir) // 清理失败的安装
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	logger.Info(fmt.Sprintf("Plugin installed successfully: %s", manifest.PackageID))
	return nil
}

// UninstallPlugin uninstalls a plugin
func (pi *PluginInstaller) UninstallPlugin(packageID string) error {
	logger.Info(fmt.Sprintf("Uninstalling plugin: %s", packageID))
	if err := utils.ValidatePluginPackageID(packageID); err != nil {
		return fmt.Errorf("invalid packageId: %w", err)
	}

	dbInstance := db.GetWaDB()

	// 1. 检查插件是否存在
	plugins := dbInstance.GetPlugins(pi.ctx)
	var found bool
	for _, p := range plugins {
		if p.PackageID == packageID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("plugin not found: %s", packageID)
	}

	// 2. 检查是否为内置插件 (约定: 内置插件以 watools.plugin. 开头且在 fronted-plugin 目录)
	// 简化: 所有已安装的插件都可以卸载

	// 3. 删除插件目录
	pluginDir, err := utils.ResolvePathWithinBase(pi.pluginsDir, packageID)
	if err != nil {
		return fmt.Errorf("invalid package uninstall path: %w", err)
	}
	if err := os.RemoveAll(pluginDir); err != nil {
		logger.Error(err, fmt.Sprintf("Failed to remove plugin directory: %s", pluginDir))
	}

	// 4. 从数据库删除
	if err := dbInstance.DeletePlugin(pi.ctx, packageID); err != nil {
		return fmt.Errorf("failed to delete plugin from database: %w", err)
	}

	logger.Info(fmt.Sprintf("Plugin uninstalled successfully: %s", packageID))
	return nil
}

// unzipFile extracts a zip file to a destination directory
func (pi *PluginInstaller) unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath, err := utils.ResolvePathWithinBase(dest, f.Name)
		if err != nil {
			return fmt.Errorf("invalid file path %q: %w", f.Name, err)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func (pi *PluginInstaller) findManifestPath(root string) (string, error) {
	var manifestPaths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "manifest.json") {
			manifestPaths = append(manifestPaths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(manifestPaths) == 0 {
		return "", fmt.Errorf("manifest.json not found")
	}
	if len(manifestPaths) > 1 {
		return "", fmt.Errorf("multiple manifest.json files found")
	}
	return manifestPaths[0], nil
}

// readManifest reads and parses manifest.json
func (pi *PluginInstaller) readManifest(manifestPath string) (*models.PluginMetadata, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest models.PluginMetadata
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// validateManifest validates required fields in manifest
func (pi *PluginInstaller) validateManifest(manifest *models.PluginMetadata) error {
	if manifest.PackageID == "" {
		return fmt.Errorf("packageId is required")
	}
	if err := utils.ValidatePluginPackageID(manifest.PackageID); err != nil {
		return err
	}
	if manifest.Name == "" {
		return fmt.Errorf("name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("version is required")
	}
	if _, err := parsePluginVersion(manifest.Version); err != nil {
		return fmt.Errorf("version must be a valid numeric semantic version: %w", err)
	}
	if manifest.Entry == "" {
		return fmt.Errorf("entry is required")
	}
	return nil
}

func (pi *PluginInstaller) findInstalledPlugin(packageID string) (*models.PluginState, bool) {
	plugins := db.GetWaDB().GetPlugins(pi.ctx)
	for _, plugin := range plugins {
		if plugin.PackageID == packageID {
			return plugin, true
		}
	}
	return nil, false
}

type pluginVersion struct {
	core       []int
	prerelease []string
}

func parsePluginVersion(raw string) (pluginVersion, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	if trimmed == "" {
		return pluginVersion{}, fmt.Errorf("empty version")
	}

	withoutBuild := strings.SplitN(trimmed, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	corePart := parts[0]
	if corePart == "" {
		return pluginVersion{}, fmt.Errorf("missing version core")
	}

	coreSegments := strings.Split(corePart, ".")
	core := make([]int, 0, len(coreSegments))
	for _, segment := range coreSegments {
		if segment == "" {
			return pluginVersion{}, fmt.Errorf("invalid core segment in %q", raw)
		}

		value, err := strconv.Atoi(segment)
		if err != nil || value < 0 {
			return pluginVersion{}, fmt.Errorf("invalid numeric segment %q", segment)
		}
		core = append(core, value)
	}

	version := pluginVersion{core: core}
	if len(parts) == 1 {
		return version, nil
	}

	prereleaseSegments := strings.Split(parts[1], ".")
	for _, segment := range prereleaseSegments {
		if segment == "" {
			return pluginVersion{}, fmt.Errorf("invalid prerelease segment in %q", raw)
		}
		version.prerelease = append(version.prerelease, segment)
	}

	return version, nil
}

func comparePluginVersions(nextVersion string, installedVersion string) (int, error) {
	next, err := parsePluginVersion(nextVersion)
	if err != nil {
		return 0, fmt.Errorf("invalid new version %q: %w", nextVersion, err)
	}

	current, err := parsePluginVersion(installedVersion)
	if err != nil {
		return 0, fmt.Errorf("invalid installed version %q: %w", installedVersion, err)
	}

	maxCoreLen := len(next.core)
	if len(current.core) > maxCoreLen {
		maxCoreLen = len(current.core)
	}
	for i := 0; i < maxCoreLen; i++ {
		nextPart := 0
		if i < len(next.core) {
			nextPart = next.core[i]
		}

		currentPart := 0
		if i < len(current.core) {
			currentPart = current.core[i]
		}

		if nextPart > currentPart {
			return 1, nil
		}
		if nextPart < currentPart {
			return -1, nil
		}
	}

	return comparePrereleaseIdentifiers(next.prerelease, current.prerelease), nil
}

func comparePrereleaseIdentifiers(next []string, current []string) int {
	if len(next) == 0 && len(current) == 0 {
		return 0
	}
	if len(next) == 0 {
		return 1
	}
	if len(current) == 0 {
		return -1
	}

	maxLen := len(next)
	if len(current) > maxLen {
		maxLen = len(current)
	}
	for i := 0; i < maxLen; i++ {
		if i >= len(next) {
			return -1
		}
		if i >= len(current) {
			return 1
		}

		nextIdentifier := next[i]
		currentIdentifier := current[i]
		nextNumeric, nextNumericErr := strconv.Atoi(nextIdentifier)
		currentNumeric, currentNumericErr := strconv.Atoi(currentIdentifier)

		switch {
		case nextNumericErr == nil && currentNumericErr == nil:
			if nextNumeric > currentNumeric {
				return 1
			}
			if nextNumeric < currentNumeric {
				return -1
			}
		case nextNumericErr == nil && currentNumericErr != nil:
			return -1
		case nextNumericErr != nil && currentNumericErr == nil:
			return 1
		default:
			if nextIdentifier > currentIdentifier {
				return 1
			}
			if nextIdentifier < currentIdentifier {
				return -1
			}
		}
	}

	return 0
}

func (pi *PluginInstaller) resolvePluginFile(pluginRoot string, relativePath string) (string, error) {
	resolvedPath, err := utils.ResolvePathWithinBase(pluginRoot, relativePath)
	if err != nil {
		return "", err
	}

	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return "", err
	}
	if fileInfo.IsDir() {
		return "", fmt.Errorf("path must point to a file")
	}

	return resolvedPath, nil
}

// copyDir recursively copies a directory
func (pi *PluginInstaller) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return pi.copyFile(path, targetPath)
	})
}

// copyFile copies a single file
func (pi *PluginInstaller) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// registerPlugin creates database record for the plugin
func (pi *PluginInstaller) registerPlugin(manifest *models.PluginMetadata, installPath string) error {
	dbInstance := db.GetWaDB()

	// 简化: 只存储必需的字段,安装路径可以通过 packageId 动态计算
	return dbInstance.InsertPlugin(pi.ctx, db.InsertPluginParams{
		PackageID: manifest.PackageID,
		Enabled:   true,
		Storage:   "{}",
	})
}
