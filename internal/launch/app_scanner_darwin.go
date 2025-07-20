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
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/models"
)

type InfoPList struct {
	BundleName             string `plist:"CFBundleName"`
	BundleDisplayName      string `plist:"CFBundleDisplayName"`
	BundleIconFile         string `plist:"CFBundleIconFile"`
	BundleIconName         string `plist:"CFBundleIconName"`
	BundleVersion          string `plist:"CFBundleShortVersionString"`
	HumanReadableCopyright string `plist:"NSHumanReadableCopyright"`
}

func parseAppBundleInfoPlist(appPath string) *models.ApplicationCommand {
	var commandName, commandDescription, commandPath, commandIconPath string
	var commandID int64
	var commandCategory models.CommandCategory

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
	if infoPlist.BundleDisplayName != "" {
		commandName = infoPlist.BundleDisplayName
	} else {
		commandName = infoPlist.BundleName
	}
	if commandName == "" {
		commandName = strings.TrimSuffix(filepath.Base(appPath), ".app")
	}
	if infoPlist.BundleIconFile != "" {
		iconName := infoPlist.BundleIconFile
		if !strings.HasSuffix(iconName, ".icns") {
			iconName += ".icns"
		}
		commandIconPath = filepath.Join(appPath, "Contents", "Resources", iconName)
		if _, err := os.Stat(commandIconPath); os.IsNotExist(err) {
			commandIconPath = ""
		}
	}
	if infoPlist.BundleIconName != "" && commandIconPath == "" {
		assetsCarPath := filepath.Join(appPath, "Contents", "Resources", "Assets.car")
		outputIcnsFolder := filepath.Join(config.ProjectCacheDir(), "icns")
		if err := os.MkdirAll(outputIcnsFolder, 0755); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create icns folder: %s", outputIcnsFolder))
		}
		outputIcnsPath := filepath.Join(outputIcnsFolder, fmt.Sprintf("%s-%s.icns", commandName, infoPlist.BundleName))
		if _, err := os.Stat(assetsCarPath); err == nil {
			cmd := exec.Command("iconutil", "-c", "icns", assetsCarPath, infoPlist.BundleIconName, "-o", outputIcnsPath)
			if _, err := cmd.CombinedOutput(); err != nil {
				logger.Error(err, fmt.Sprintf("Failed to generate icns for app: %s", commandName))
			} else {
				commandIconPath = outputIcnsPath
			}
		}
	}
	commandDescription = infoPlist.HumanReadableCopyright
	if _, err := exec.LookPath("mdls"); err == nil {
		cmd := exec.Command("mdls", "-name", "kMDItemDisplayName", "-raw", appPath)
		output, err := cmd.Output()
		if err == nil {
			displayName := strings.TrimSpace(string(output))
			if displayName != "" && displayName != "(null)" {
				commandName = strings.TrimSuffix(displayName, ".app")
			}
		}
	}
	return models.NewApplicationCommand(commandName, commandDescription, commandCategory, commandPath, commandIconPath, commandID)
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

func (*macAppScanner) GetApplications() ([]*models.ApplicationCommand, error) {
	var commands []*models.ApplicationCommand

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

func (*macAppScanner) ParseApplication(appPath string) (*models.ApplicationCommand, error) {
	if command := parseAppBundleInfoPlist(appPath); command != nil {
		return command, nil
	}
	return nil, fmt.Errorf("failed to parse Info.plist for '%s'", appPath)
}

func (*macAppScanner) GetDefaultIconPath() string {
	return "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/GenericApplicationIcon.icns"
}
