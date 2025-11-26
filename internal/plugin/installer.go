package plugin

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"watools/config"
	"watools/pkg/db"
	"watools/pkg/logger"
	"watools/pkg/models"
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
	manifestPath := filepath.Join(tempDir, "manifest.json")
	manifest, err := pi.readManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// 5. 验证必需字段
	if err := pi.validateManifest(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// 6. 检查插件是否已安装
	dbInstance := db.GetWaDB()
	plugins := dbInstance.GetPlugins(pi.ctx)
	for _, p := range plugins {
		if p.PackageID == manifest.PackageID {
			return fmt.Errorf("plugin already installed: %s", manifest.PackageID)
		}
	}

	// 7. 创建插件目录
	pluginDir := filepath.Join(pi.pluginsDir, manifest.PackageID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// 8. 复制文件到插件目录
	if err := pi.copyDir(tempDir, pluginDir); err != nil {
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
	pluginDir := filepath.Join(pi.pluginsDir, packageID)
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
		fpath := filepath.Join(dest, f.Name)

		// 安全检查: 防止路径遍历攻击
		if !filepath.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
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
	if manifest.Name == "" {
		return fmt.Errorf("name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("version is required")
	}
	if manifest.Entry == "" {
		return fmt.Errorf("entry is required")
	}
	return nil
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
