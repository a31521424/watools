package application

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"watools/config"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/mo"
	"howett.net/plist"
)

type InfoPList struct {
	BundleName             string `plist:"CFBundleName"`
	BundleDisplayName      string `plist:"CFBundleDisplayName"`
	BundleIconFile         string `plist:"CFBundleIconFile"`
	BundleIconName         string `plist:"CFBundleIconName"`
	BundleVersion          string `plist:"CFBundleShortVersionString"`
	HumanReadableCopyright string `plist:"NSHumanReadableCopyright"`
}

func parseAppBundleInfoPlist(appPath string) (*models.ApplicationCommand, error) {
	return parseAppBundleWithReader(appPath, os.ReadFile)
}

func parseAppBundleWithReader(appPath string, readFile func(string) ([]byte, error)) (*models.ApplicationCommand, error) {
	var commandName string
	var commandDescription, commandIconPath mo.Option[string]

	fi, err := os.Stat(appPath)
	if err != nil {
		kind := ErrApplicationUnreadable
		if errors.Is(err, os.ErrNotExist) {
			kind = ErrApplicationNotFound
		}
		return nil, applicationError(kind, "stat application", appPath, err)
	}
	if !fi.IsDir() || !strings.HasSuffix(appPath, ".app") {
		return nil, applicationError(ErrNotApplication, "inspect application", appPath, nil)
	}

	dirUpdatedAt := fi.ModTime()
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	data, err := readFile(plistPath)
	if err != nil {
		kind := ErrApplicationUnreadable
		if errors.Is(err, os.ErrNotExist) {
			kind = ErrInvalidMetadata
		}
		return nil, applicationError(kind, "read application metadata", plistPath, err)
	}

	var infoPlist InfoPList
	decoder := plist.NewDecoder(bytes.NewReader(data))

	if err := decoder.Decode(&infoPlist); err != nil {
		return nil, applicationError(ErrInvalidMetadata, "decode application metadata", plistPath, err)
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
		commandIconPath = mo.Some(filepath.Join(appPath, "Contents", "Resources", iconName))
		if iconInfo, err := os.Stat(commandIconPath.MustGet()); err != nil || iconInfo.IsDir() {
			logger.Warning(fmt.Sprintf("Application %q uses default icon: icon unavailable (%v)", appPath, err))
			commandIconPath = mo.None[string]()
		}
	}
	if infoPlist.BundleIconName != "" && commandIconPath.IsNone() {
		iconPath, err := extractBundleIcon(appPath, commandName, infoPlist)
		if err != nil {
			logger.Warning(fmt.Sprintf("Application %q uses default icon: %v", appPath, err))
		} else {
			commandIconPath = mo.Some(iconPath)
		}
	}
	commandDescription = mo.TupleToOption(infoPlist.HumanReadableCopyright, infoPlist.HumanReadableCopyright != "")
	if _, err := exec.LookPath("mdls"); err == nil {
		cmd := exec.Command("mdls", "-name", "kMDItemDisplayName", "-raw", appPath)
		output, err := cmd.Output()
		if err == nil {
			displayName := strings.TrimSpace(string(output))
			if displayName != "" && displayName != "(null)" {
				commandName = strings.TrimSuffix(displayName, ".app")
			} else {
				logger.Warning(fmt.Sprintf("Application %q uses plist/file name: mdls display name unavailable", appPath))
			}
		} else {
			logger.Warning(fmt.Sprintf("Application %q uses plist/file name: mdls failed: %v", appPath, err))
		}
	} else {
		logger.Warning(fmt.Sprintf("Application %q uses plist/file name: mdls unavailable: %v", appPath, err))
	}
	return models.NewApplicationCommand(commandName, commandDescription, appPath, commandIconPath, mo.None[string](), dirUpdatedAt), nil
}

func extractBundleIcon(appPath, commandName string, info InfoPList) (string, error) {
	assetsPath := filepath.Join(appPath, "Contents", "Resources", "Assets.car")
	if _, err := os.Stat(assetsPath); err != nil {
		return "", fmt.Errorf("stat icon assets: %w", err)
	}
	cacheDir, err := optionalIconCacheDir()
	if err != nil {
		return "", err
	}
	folder := filepath.Join(cacheDir, "icns")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("create icon cache: %w", err)
	}
	outputPath := filepath.Join(folder, fmt.Sprintf("%s-%s.icns", commandName, info.BundleName))
	cmd := exec.Command("iconutil", "-c", "icns", assetsPath, info.BundleIconName, "-o", outputPath)
	if _, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("generate bundle icon: %w", err)
	}
	if fi, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("stat generated icon: %w", err)
	} else if fi.IsDir() {
		return "", fmt.Errorf("generated icon is a directory: %s", outputPath)
	}
	return outputPath, nil
}

// The legacy cache getter panics on filesystem errors. Only adapt that call:
// icon caching is optional, while programming panics must still propagate.
func optionalIconCacheDir() (path string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if _, runtimeFailure := recovered.(runtime.Error); runtimeFailure {
				panic(recovered)
			}
			if cause, ok := recovered.(error); ok {
				err = fmt.Errorf("locate icon cache: %w", cause)
				return
			}
			panic(recovered)
		}
	}()
	return config.ProjectCacheDir(), nil
}

// PathKey gives the platform's application identity without touching the disk.
func PathKey(path string) string {
	return filepath.Clean(path)
}

type AppPathInfo struct {
	Path     string
	UpdateAt time.Time
}

func getMacApplicationPath() ([]AppPathInfo, error) {
	appFolderDirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		"/System/Library/CoreServices",
	}
	homeDir, homeErr := os.UserHomeDir()
	if homeErr == nil {
		appFolderDirs = append(appFolderDirs, filepath.Join(homeDir, "Applications"))
	}
	infos, scanErr := scanMacApplicationPaths(appFolderDirs, os.ReadDir)
	if homeErr != nil {
		scanErr = errors.Join(scanErr, applicationError(ErrApplicationUnreadable, "locate user application folder", "", homeErr))
	}
	return infos, scanErr
}

func scanMacApplicationPaths(appFolderDirs []string, readDir func(string) ([]os.DirEntry, error)) ([]AppPathInfo, error) {
	var appPathInfos []AppPathInfo
	var scanErrors []error
	logger.Info(fmt.Sprintf("Scanning app folders: %v", appFolderDirs))
	seen := make(map[string]struct{})
	for _, appFolderDir := range appFolderDirs {
		entries, err := readDir(appFolderDir)
		if err != nil {
			// Optional application folders may not exist on this machine.
			if !errors.Is(err, os.ErrNotExist) {
				scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "scan application folder", appFolderDir, err))
			}
			continue
		}
		for _, de := range entries {
			appPath := filepath.Join(appFolderDir, de.Name())
			if !strings.HasSuffix(appPath, ".app") {
				continue
			}
			if de.Type()&os.ModeSymlink != 0 {
				rp, err := filepath.EvalSymlinks(appPath)
				if err != nil {
					scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "resolve application symlink", appPath, err))
					continue
				}
				appPath = rp
			}
			if _, exists := seen[PathKey(appPath)]; exists {
				continue
			}
			fi, err := os.Stat(appPath)
			if err != nil {
				scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "stat scanned application", appPath, err))
				continue
			}
			if !fi.IsDir() {
				scanErrors = append(scanErrors, applicationError(ErrNotApplication, "inspect scanned application", appPath, nil))
				continue
			}
			appPathInfos = append(appPathInfos, AppPathInfo{Path: appPath, UpdateAt: fi.ModTime()})
			seen[PathKey(appPath)] = struct{}{}
		}
	}
	return appPathInfos, errors.Join(scanErrors...)
}

// GetApplications returns valid applications even when other paths fail to scan or parse.
func GetApplications() ([]*models.ApplicationCommand, error) {
	infos, err := GetAppPathInfosWithError()
	return applicationsFromPathInfos(infos, err)
}

func applicationsFromPathInfos(infos []AppPathInfo, scanErr error) ([]*models.ApplicationCommand, error) {
	var commands []*models.ApplicationCommand
	errs := []error{scanErr}
	for _, info := range infos {
		command, err := ParseApplication(info.Path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		commands = append(commands, command)
	}
	return commands, errors.Join(errs...)
}

func ParseApplication(appPath string) (*models.ApplicationCommand, error) {
	return parseAppBundleInfoPlist(appPath)
}

func GetDefaultIconPath() string {
	return "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/GenericApplicationIcon.icns"
}

// GetAppPathInfos is a compatibility wrapper that logs partial scan failures.
// Call GetAppPathInfosWithError when failures affect reconciliation decisions.
func GetAppPathInfos() []AppPathInfo {
	infos, err := GetAppPathInfosWithError()
	if err != nil {
		logger.Error(err, "Application scan returned partial paths")
	}
	return infos
}

// GetAppPathInfosWithError returns discovered paths and any scan failures.
func GetAppPathInfosWithError() ([]AppPathInfo, error) {
	return getMacApplicationPath()
}
