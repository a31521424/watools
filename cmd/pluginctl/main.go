package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"watools/config"
	"watools/internal/plugin"
	"watools/pkg/models"
	"watools/pkg/utils"
)

type officialPlugin struct {
	PackageID  string
	Name       string
	Version    string
	PluginDir  string
	Manifest   string
	OutputFile string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := initProjectConfig(); err != nil {
		fatalf("failed to initialize project config: %v", err)
	}

	switch os.Args[1] {
	case "list":
		if err := runList(); err != nil {
			fatalf("list failed: %v", err)
		}
	case "package":
		if err := runPackage(os.Args[2:]); err != nil {
			fatalf("package failed: %v", err)
		}
	case "install":
		if err := runInstall(os.Args[2:]); err != nil {
			fatalf("install failed: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `watools pluginctl

Usage:
  go run ./cmd/pluginctl list
  go run ./cmd/pluginctl package [plugin-package-id...]
  go run ./cmd/pluginctl install [plugin-package-id...]

Commands:
  list      List official plugins from plugins/official
  package   Build .wt archives into plugins/dist
  install   Package and install official plugins into the local WaTools cache
`)
}

func initProjectConfig() error {
	wailsJSONPath := filepath.Join(".", "wails.json")
	data, err := os.ReadFile(wailsJSONPath)
	if err != nil {
		return err
	}
	config.ParseProject(data)
	return nil
}

func runList() error {
	plugins, err := discoverOfficialPlugins()
	if err != nil {
		return err
	}
	for _, officialPlugin := range plugins {
		fmt.Printf("%s\t%s\t%s\n", officialPlugin.PackageID, officialPlugin.Version, officialPlugin.PluginDir)
	}
	return nil
}

func runPackage(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	outputDir := fs.String("output", filepath.Join("plugins", "dist"), "output directory for .wt archives")
	if err := fs.Parse(args); err != nil {
		return err
	}

	plugins, err := selectOfficialPlugins(fs.Args())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return err
	}

	for _, officialPlugin := range plugins {
		outputFile := filepath.Join(*outputDir, officialPlugin.PackageID+".wt")
		if err := packagePlugin(officialPlugin.PluginDir, outputFile); err != nil {
			return fmt.Errorf("package %s: %w", officialPlugin.PackageID, err)
		}
		fmt.Printf("packaged %s -> %s\n", officialPlugin.PackageID, outputFile)
	}
	return nil
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	outputDir := fs.String("output", filepath.Join("plugins", "dist"), "output directory for .wt archives")
	if err := fs.Parse(args); err != nil {
		return err
	}

	plugins, err := selectOfficialPlugins(fs.Args())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return err
	}

	installer := plugin.NewPluginInstaller(context.Background())
	for _, officialPlugin := range plugins {
		outputFile := filepath.Join(*outputDir, officialPlugin.PackageID+".wt")
		if err := packagePlugin(officialPlugin.PluginDir, outputFile); err != nil {
			return fmt.Errorf("package %s: %w", officialPlugin.PackageID, err)
		}
		if err := installer.InstallFromWtFile(outputFile); err != nil {
			return fmt.Errorf("install %s: %w", officialPlugin.PackageID, err)
		}
		fmt.Printf("installed %s from %s\n", officialPlugin.PackageID, outputFile)
	}
	return nil
}

func discoverOfficialPlugins() ([]officialPlugin, error) {
	rootDir := filepath.Join("plugins", "official")
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var plugins []officialPlugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(rootDir, entry.Name(), "plugin")
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		metadata, err := readPluginManifest(manifestPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", manifestPath, err)
		}
		if err := utils.ValidatePluginPackageID(metadata.PackageID); err != nil {
			return nil, fmt.Errorf("invalid packageId in %s: %w", manifestPath, err)
		}

		plugins = append(plugins, officialPlugin{
			PackageID: metadata.PackageID,
			Name:      metadata.Name,
			Version:   metadata.Version,
			PluginDir: pluginDir,
			Manifest:  manifestPath,
		})
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].PackageID < plugins[j].PackageID
	})
	return plugins, nil
}

func selectOfficialPlugins(requested []string) ([]officialPlugin, error) {
	plugins, err := discoverOfficialPlugins()
	if err != nil {
		return nil, err
	}
	if len(requested) == 0 {
		return plugins, nil
	}

	pluginMap := make(map[string]officialPlugin, len(plugins))
	for _, officialPlugin := range plugins {
		pluginMap[officialPlugin.PackageID] = officialPlugin
	}

	selected := make([]officialPlugin, 0, len(requested))
	for _, requestedID := range requested {
		officialPlugin, ok := pluginMap[requestedID]
		if !ok {
			return nil, fmt.Errorf("official plugin not found: %s", requestedID)
		}
		selected = append(selected, officialPlugin)
	}
	return selected, nil
}

func readPluginManifest(manifestPath string) (*models.PluginMetadata, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var metadata models.PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func packagePlugin(pluginDir string, outputFile string) error {
	if err := os.RemoveAll(outputFile); err != nil {
		return err
	}

	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	zipWriter := zip.NewWriter(output)
	defer zipWriter.Close()

	return filepath.Walk(pluginDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(pluginDir, path)
		if err != nil {
			return err
		}
		archivePath := filepath.ToSlash(relPath)
		if strings.HasPrefix(archivePath, ".") {
			return nil
		}

		fileHeader, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fileHeader.Name = archivePath
		fileHeader.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(fileHeader)
		if err != nil {
			return err
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, sourceFile)
		closeErr := sourceFile.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
