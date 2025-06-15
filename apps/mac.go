package apps

import (
	"bytes"
	"howett.net/plist"
	"io"
	"os"
	"path/filepath"
	"strings"
	"watools/schemas"
)

type InfoPList struct {
	BundleName             string `plist:"CFBundleName"`
	BundleDisplayName      string `plist:"CFBundleDisplayName"`
	BundleIconFile         string `plist:"CFBundleIconFile"`
	BundleVersion          string `plist:"CFBundleShortVersionString"`
	HumanReadableCopyright string `plist:"NSHumanReadableCopyright"`
}

func parseAppBundleInfoPlist(appPath string) *schemas.Command {
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
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
	command := schemas.Command{
		Category: schemas.CategoryApplication,
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
		command.IconPath = filepath.Join(appPath, "Contents", "Resources", iconName)
	}
	command.Description = infoPlist.HumanReadableCopyright
	return &command
}

func getMacApplicationPath() []string {
	var appPaths []string
	appFolderDirs := []string{"/Applications", "/System/Applications"}
	if homeDir, err := os.UserHomeDir(); err == nil {
		appFolderDirs = append(appFolderDirs, filepath.Join(homeDir, "Applications"))
	}
	for _, appFolderDir := range appFolderDirs {
		apps, err := os.ReadDir(appFolderDir)
		if err != nil {
			continue
		}
		for _, app := range apps {
			if app.IsDir() {
				appPath := filepath.Join(appFolderDir, app.Name())
				if _, err := os.Stat(filepath.Join(appPath, "Contents", "Info.plist")); err == nil {
					appPaths = append(appPaths, appPath)
				}
			}
			//	TODO: Safari is a special case
		}
	}
	return appPaths
}

func GetMacApplication() []schemas.Command {
	var commands []schemas.Command

	for _, appPath := range getMacApplicationPath() {
		if command := parseAppBundleInfoPlist(appPath); command != nil {
			commands = append(commands, *command)
		}
	}
	return commands
}
