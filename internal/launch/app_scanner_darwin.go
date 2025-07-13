package launch

import (
	"bytes"
	"fmt"
	"howett.net/plist"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"watools/pkg/logger"
	"watools/pkg/models"
)

type InfoPList struct {
	BundleName             string `plist:"CFBundleName"`
	BundleDisplayName      string `plist:"CFBundleDisplayName"`
	BundleIconFile         string `plist:"CFBundleIconFile"`
	BundleVersion          string `plist:"CFBundleShortVersionString"`
	HumanReadableCopyright string `plist:"NSHumanReadableCopyright"`
}

func parseAppBundleInfoPlist(appPath string) *models.Command {
	plistPath := filepath.Join(strings.TrimSpace(appPath), "Contents", "Info.plist")
	plistFile, err := os.Open(plistPath)
	if err != nil {
		return nil
	}
	defer plistFile.Close()

	data, err := io.ReadAll(plistFile)
	if err != nil {
		return nil
	}

	var infoPlist InfoPList
	decoder := plist.NewDecoder(bytes.NewReader(data))

	if err := decoder.Decode(&infoPlist); err != nil {
		return nil
	}
	command := models.Command{
		Category: models.CategoryApplication,
		Path:     appPath,
	}
	if infoPlist.BundleDisplayName != "" {
		command.Name = infoPlist.BundleDisplayName
	} else {
		command.Name = infoPlist.BundleName
	}
	if command.Name == "" {
		command.Name = strings.TrimSuffix(filepath.Base(appPath), ".app")
	}
	if infoPlist.BundleIconFile != "" {
		iconName := infoPlist.BundleIconFile
		if !strings.HasSuffix(iconName, ".icns") {
			iconName += ".icns"
		}
		command.IconPath = filepath.Join(appPath, "Contents", "Resources", iconName)
		if _, err := os.Stat(command.IconPath); os.IsNotExist(err) {
			command.IconPath = ""
		}
	}
	command.Description = infoPlist.HumanReadableCopyright
	if _, err := exec.LookPath("mdls"); err == nil {
		cmd := exec.Command("mdls", "-name", "kMDItemDisplayName", "-raw", appPath)
		output, err := cmd.Output()
		if err == nil {
			displayName := strings.TrimSpace(string(output))
			if displayName != "" && displayName != "(null)" {
				command.Name = strings.TrimSuffix(displayName, ".app")
			}
		}
	}
	return &command
}

func getMacApplicationPath() []string {
	var appPaths []string
	appFolderDirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		"/System/Library/CoreServices",
		"/Developer/Applications",
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		appFolderDirs = append(appFolderDirs, filepath.Join(homeDir, "Applications"))
	}
	logger.Info(fmt.Sprintf("Scanning app folders: %v", appFolderDirs))
	seen := make(map[string]bool)
	for _, appFolderDir := range appFolderDirs {
		apps, err := os.ReadDir(appFolderDir)
		if err != nil {
			logger.Error(err, "Failed to read app folder dir")
			continue
		}
		for _, app := range apps {
			appPath := filepath.Join(appFolderDir, app.Name())
			if app.IsDir() && !seen[appPath] {
				if _, err := os.Stat(filepath.Join(appPath, "Contents", "Info.plist")); err == nil {
					appPaths = append(appPaths, appPath)
					seen[appPath] = true
				}
			}
			//	TODO: Safari is a special case
		}
	}
	return appPaths
}

type macAppScanner struct {
	AppScanner
}

func NewAppScanner() AppScanner {
	return &macAppScanner{}
}

func (*macAppScanner) GetApplications() ([]*models.Command, error) {
	var commands []*models.Command

	for _, appPath := range getMacApplicationPath() {
		if command := parseAppBundleInfoPlist(appPath); command != nil {
			commands = append(commands, command)
		} else {
			logger.Info(fmt.Sprintf("Failed to parse Info.plist for '%s'", appPath))
		}
	}
	return commands, nil
}

func (*macAppScanner) RunApplication(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("failed to find application file '%s': %w", path, err)
	}
	cmd := exec.Command("open", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run application: %w\n%s", err, output)
	}
	return nil
}

func (*macAppScanner) ParseApplication(appPath string) (*models.Command, error) {
	if command := parseAppBundleInfoPlist(appPath); command != nil {
		return command, nil
	}
	return nil, fmt.Errorf("failed to parse Info.plist for '%s'", appPath)
}
