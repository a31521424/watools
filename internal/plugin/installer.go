package plugin

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"watools/config"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"
	"watools/pkg/utils"
)

var installationMu sync.Mutex

type pluginRepository interface {
	GetPlugins(context.Context) ([]*models.PluginState, error)
	InsertPlugin(context.Context, db.InsertPluginParams) error
	DeletePlugin(context.Context, string) error
}

type PluginInstaller struct {
	ctx        context.Context
	pluginsDir string
	repository pluginRepository
}

func NewPluginInstaller(ctx context.Context) *PluginInstaller {
	pluginsDir := filepath.Join(config.ProjectDataDir(), "plugins")
	return &PluginInstaller{
		ctx:        ctx,
		pluginsDir: pluginsDir,
	}
}

func (pi *PluginInstaller) database() pluginRepository {
	if pi.repository != nil {
		return pi.repository
	}
	return db.GetWaDB()
}

// InstallFromWtFile installs a plugin from a .wt file (zip format)
func (pi *PluginInstaller) InstallFromWtFile(wtFilePath string) error {
	installationMu.Lock()
	defer installationMu.Unlock()
	logger.Info(fmt.Sprintf("Installing plugin from: %s", wtFilePath))

	// 1. 验证文件存在
	if info, err := os.Stat(wtFilePath); err != nil {
		return fmt.Errorf("plugin file unavailable: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("plugin package must be a regular file")
	}

	// 2. 创建临时解压目录
	tempDir, err := os.MkdirTemp("", "watools-plugin-*")
	if err != nil {
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
	_, installed, err := pi.findInstalledPlugin(manifest.PackageID)
	if err != nil {
		return err
	}
	if installed {
		installedRoot, _, rootErr := models.ResolveInstalledPlugin(pi.pluginsDir, manifest.PackageID)
		installedManifest, metadataErr := pi.readManifest(filepath.Join(installedRoot, "manifest.json"))
		// A damaged old manifest must not prevent a validated repair installation.
		if rootErr == nil && metadataErr == nil {
			versionComparison, err := comparePluginVersions(manifest.Version, installedManifest.Version)
			if err == nil && versionComparison < 0 {
				return fmt.Errorf(
					"plugin %s version %s is older than installed version %s",
					manifest.PackageID,
					manifest.Version,
					installedManifest.Version,
				)
			}

		}
	}

	// 7. 创建插件目录
	pluginDir, err := utils.ResolvePathWithinBase(pi.pluginsDir, manifest.PackageID)
	if err != nil {
		return fmt.Errorf("invalid package installation path: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "versions"), 0700); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// 8. 复制文件到插件目录
	revision := uuid.NewString()
	versionDir := filepath.Join(pluginDir, "versions", revision)
	if err := pi.copyDir(pluginRoot, versionDir); err != nil {
		os.RemoveAll(versionDir)
		return fmt.Errorf("failed to copy plugin files: %w", err)
	}

	// 9. 插入数据库记录
	if !installed {
		if err := pi.registerPlugin(manifest, versionDir); err != nil {
			os.RemoveAll(versionDir)
			return fmt.Errorf("failed to register plugin: %w", err)
		}
	}
	// The only commit for an upgrade is this pointer switch. User state is untouched.
	pointer, err := os.CreateTemp(pluginDir, ".current-*")
	if err == nil {
		_, err = fmt.Fprintf(pointer, `{"revision":%q}`, revision)
		if closeErr := pointer.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(pointer.Name(), filepath.Join(pluginDir, "current.json"))
		}
		os.Remove(pointer.Name())
	}
	if err != nil {
		os.RemoveAll(versionDir)
		if !installed {
			if rollbackErr := pi.database().DeletePlugin(pi.ctx, manifest.PackageID); rollbackErr != nil {
				return fmt.Errorf("commit failed: %v; rollback failed: %w", err, rollbackErr)
			}
		}
		return fmt.Errorf("failed to commit plugin revision: %w", err)
	}

	logger.Info(fmt.Sprintf("Plugin installed successfully: %s", manifest.PackageID))
	return nil
}

// UninstallPlugin uninstalls a plugin
func (pi *PluginInstaller) UninstallPlugin(packageID string) error {
	installationMu.Lock()
	defer installationMu.Unlock()
	logger.Info(fmt.Sprintf("Uninstalling plugin: %s", packageID))
	if err := utils.ValidatePluginPackageID(packageID); err != nil {
		return fmt.Errorf("invalid packageId: %w", err)
	}

	dbInstance := pi.database()

	// 1. 检查插件是否存在
	plugins, err := dbInstance.GetPlugins(pi.ctx)
	if err != nil {
		return err
	}
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
	trash := filepath.Join(pi.pluginsDir, ".removed-"+uuid.NewString())
	moved := false
	if err := os.Rename(pluginDir, trash); err == nil {
		moved = true
	} else if !os.IsNotExist(err) {
		return err
	}

	// 4. 从数据库删除
	if err := dbInstance.DeletePlugin(pi.ctx, packageID); err != nil {
		if moved {
			if restoreErr := os.Rename(trash, pluginDir); restoreErr != nil {
				return fmt.Errorf("delete failed: %v; restore failed: %w", err, restoreErr)
			}
		}
		return fmt.Errorf("failed to delete plugin from database: %w", err)
	}
	if moved {
		if err := os.RemoveAll(trash); err != nil {
			logger.Error(err, "Failed to clean removed plugin")
		}
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

	const maxFiles = 10000
	const maxFileSize = 64 << 20
	const maxTotalSize = 256 << 20
	if len(r.File) > maxFiles {
		return fmt.Errorf("plugin archive contains too many files")
	}
	var total uint64
	for _, f := range r.File {
		if f.Mode()&os.ModeType != 0 && !f.FileInfo().IsDir() {
			return fmt.Errorf("archive contains a special file: %s", f.Name)
		}
		if f.UncompressedSize64 > maxFileSize || total+f.UncompressedSize64 > maxTotalSize {
			return fmt.Errorf("plugin archive exceeds extraction limits")
		}
		total += f.UncompressedSize64
		fpath, err := utils.ResolvePathWithinBase(dest, f.Name)
		if err != nil {
			return fmt.Errorf("invalid file path %q: %w", f.Name, err)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0700); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0700); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		var written int64
		written, err = io.Copy(outFile, io.LimitReader(rc, maxFileSize+1))
		if closeErr := outFile.Close(); err == nil {
			err = closeErr
		}
		if closeErr := rc.Close(); err == nil {
			err = closeErr
		}
		if written > maxFileSize || uint64(written) != f.UncompressedSize64 {
			return fmt.Errorf("invalid expanded file size")
		}

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

func (pi *PluginInstaller) findInstalledPlugin(packageID string) (*models.PluginState, bool, error) {
	plugins, err := pi.database().GetPlugins(pi.ctx)
	if err != nil {
		return nil, false, err
	}
	for _, plugin := range plugins {
		if plugin.PackageID == packageID {
			return plugin, true, nil
		}
	}
	return nil, false, nil
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
		if !info.Mode().IsRegular() {
			return fmt.Errorf("plugin contains a special file")
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

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}
	if err := dstFile.Close(); err != nil {
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
	dbInstance := pi.database()

	// 简化: 只存储必需的字段,安装路径可以通过 packageId 动态计算
	return dbInstance.InsertPlugin(pi.ctx, db.InsertPluginParams{
		PackageID: manifest.PackageID,
		Enabled:   true,
		Storage:   "{}",
	})
}
