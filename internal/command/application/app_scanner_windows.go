package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/mo"
	"golang.org/x/sys/windows/registry"
)

type shortcutInfo struct {
	TargetPath   string `json:"TargetPath"`
	IconLocation string `json:"IconLocation"`
	Description  string `json:"Description"`
}

type AppPathInfo struct {
	Path     string
	UpdateAt time.Time
	Name     string
	IconPath string
	Desc     string
}

var (
	extraSearchDirs   []string
	extraSearchDirsMu sync.RWMutex
)

// AddSearchDir appends an extra directory to search for executables.
func AddSearchDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	dir = filepath.Clean(dir)
	extraSearchDirsMu.Lock()
	extraSearchDirs = append(extraSearchDirs, dir)
	extraSearchDirsMu.Unlock()
}

// SetSearchDirs replaces extra search directories.
func SetSearchDirs(dirs []string) {
	cleaned := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		cleaned = append(cleaned, filepath.Clean(dir))
	}
	extraSearchDirsMu.Lock()
	extraSearchDirs = cleaned
	extraSearchDirsMu.Unlock()
}

func getExtraSearchDirs() []string {
	extraSearchDirsMu.RLock()
	defer extraSearchDirsMu.RUnlock()
	dirs := make([]string, len(extraSearchDirs))
	copy(dirs, extraSearchDirs)
	return dirs
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("powershell failed: %w\n%s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func getShortcutInfo(lnkPath string) (*shortcutInfo, error) {
	return getShortcutInfoWithRunner(lnkPath, runPowerShell)
}

func getShortcutInfoWithRunner(lnkPath string, run func(string) (string, error)) (*shortcutInfo, error) {
	script := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut('" + escapePowerShellSingleQuoted(lnkPath) + "'); " +
		"$o=[pscustomobject]@{TargetPath=$s.TargetPath;IconLocation=$s.IconLocation;Description=$s.Description}; " +
		"$o | ConvertTo-Json -Compress"

	out, err := run(script)
	if err != nil {
		return nil, applicationError(ErrApplicationUnreadable, "read shortcut metadata", lnkPath, err)
	}
	if out == "" {
		return nil, applicationError(ErrInvalidMetadata, "decode shortcut metadata", lnkPath, fmt.Errorf("empty shortcut info"))
	}
	var info shortcutInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return nil, applicationError(ErrInvalidMetadata, "decode shortcut metadata", lnkPath, err)
	}
	return &info, nil
}

func expandWindowsEnv(value string) string {
	re := regexp.MustCompile("%[^%]+%")
	return re.ReplaceAllStringFunc(value, func(match string) string {
		key := strings.Trim(match, "%")
		if key == "" {
			return match
		}
		if val := os.Getenv(key); val != "" {
			return val
		}
		return match
	})
}

func normalizeIconPath(iconLocation string, fallback string) string {
	iconLocation = strings.TrimSpace(iconLocation)
	if iconLocation == "" {
		return fallback
	}
	parts := strings.Split(iconLocation, ",")
	iconPath := strings.TrimSpace(parts[0])
	if iconPath == "" {
		return fallback
	}
	iconPath = expandWindowsEnv(iconPath)
	if !filepath.IsAbs(iconPath) && fallback != "" {
		iconPath = filepath.Join(filepath.Dir(fallback), iconPath)
	}
	return iconPath
}

// PathKey gives the platform's application identity without touching the disk.
func PathKey(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

func isExecutablePath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".exe")
}

func extractExecutablePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "\"") {
		trimmed := strings.TrimPrefix(value, "\"")
		if idx := strings.Index(trimmed, "\""); idx >= 0 {
			value = trimmed[:idx]
		} else {
			value = strings.Trim(value, "\"")
		}
	} else {
		if idx := strings.Index(value, ","); idx >= 0 {
			value = value[:idx]
		}
		fields := strings.Fields(value)
		if len(fields) > 0 {
			value = fields[0]
		}
	}

	if idx := strings.Index(value, ","); idx >= 0 {
		value = value[:idx]
	}

	value = expandWindowsEnv(value)
	return strings.TrimSpace(value)
}

func findFirstExecutable(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", applicationError(ErrApplicationUnreadable, "scan installed executables", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", nil
}

func getRegistryAppInfos() ([]AppPathInfo, error) {
	var infos []AppPathInfo
	var scanErrors []error

	type regRoot struct {
		key  registry.Key
		path string
		flag uint32
	}

	roots := []regRoot{
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.WOW64_64KEY},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.WOW64_32KEY},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.WOW64_64KEY},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.WOW64_32KEY},
	}

	seen := make(map[string]struct{})

	for _, root := range roots {
		baseKey, err := registry.OpenKey(root.key, root.path, registry.READ|root.flag)
		if err != nil {
			if !errors.Is(err, registry.ErrNotExist) {
				scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "open application registry", root.path, err))
			}
			continue
		}
		subKeys, err := baseKey.ReadSubKeyNames(-1)
		baseKey.Close() // Read-only handle cleanup does not affect the scan result.
		if err != nil {
			scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "read application registry keys", root.path, err))
			continue
		}

		for _, sub := range subKeys {
			subKey, err := registry.OpenKey(root.key, root.path+`\`+sub, registry.READ|root.flag)
			if err != nil {
				scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "open application registry entry", root.path+`\`+sub, err))
				continue
			}

			displayName, _, nameErr := subKey.GetStringValue("DisplayName")
			if nameErr != nil && !errors.Is(nameErr, registry.ErrNotExist) {
				scanErrors = append(scanErrors, applicationError(ErrInvalidMetadata, "read application display name", root.path+`\`+sub, nameErr))
			}
			if strings.TrimSpace(displayName) == "" {
				subKey.Close()
				continue
			}

			displayIcon := optionalRegistryString(subKey, "DisplayIcon", sub)
			installLocation := optionalRegistryString(subKey, "InstallLocation", sub)
			description := optionalRegistryString(subKey, "Publisher", sub)

			subKey.Close()

			iconPath := extractExecutablePath(displayIcon)
			targetPath := iconPath

			if targetPath == "" || !isExecutablePath(targetPath) {
				installLocation = expandWindowsEnv(strings.TrimSpace(installLocation))
				if installLocation != "" && filepath.IsAbs(installLocation) {
					exePath, err := findFirstExecutable(installLocation)
					if err != nil {
						scanErrors = append(scanErrors, err)
					}
					if exePath != "" {
						targetPath = exePath
					}
				}
			}

			if targetPath == "" || !filepath.IsAbs(targetPath) || !isExecutablePath(targetPath) {
				continue
			}

			key := PathKey(targetPath)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			fi, err := statApplication(targetPath)
			if err != nil {
				scanErrors = append(scanErrors, err)
				continue
			}
			if fi.IsDir() {
				scanErrors = append(scanErrors, applicationError(ErrNotApplication, "inspect registered executable", targetPath, nil))
				continue
			}

			infos = append(infos, AppPathInfo{
				Path:     targetPath,
				UpdateAt: fi.ModTime(),
				Name:     strings.TrimSpace(displayName),
				IconPath: iconPath,
				Desc:     strings.TrimSpace(description),
			})
		}
	}

	return infos, errors.Join(scanErrors...)
}

func optionalRegistryString(key registry.Key, name, entry string) string {
	value, _, err := key.GetStringValue(name)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		logger.Warning(fmt.Sprintf("Application registry entry %q omits optional %s: %v", entry, name, err))
	}
	return value
}

func statApplication(path string) (os.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		kind := ErrApplicationUnreadable
		if errors.Is(err, os.ErrNotExist) {
			kind = ErrApplicationNotFound
		}
		return nil, applicationError(kind, "stat application", path, err)
	}
	return fi, nil
}

func parseShortcut(lnkPath string) (*models.ApplicationCommand, error) {
	return parseShortcutWithReader(lnkPath, getShortcutInfo)
}

func parseShortcutWithReader(lnkPath string, readInfo func(string) (*shortcutInfo, error)) (*models.ApplicationCommand, error) {
	fi, err := statApplication(lnkPath)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, applicationError(ErrNotApplication, "inspect shortcut", lnkPath, nil)
	}
	info, err := readInfo(lnkPath)
	if err != nil {
		kind := ErrApplicationUnreadable
		if errors.Is(err, ErrInvalidMetadata) {
			kind = ErrInvalidMetadata
		}
		return nil, applicationError(kind, "read shortcut metadata", lnkPath, err)
	}
	if info == nil {
		return nil, applicationError(ErrInvalidMetadata, "read shortcut metadata", lnkPath, fmt.Errorf("missing shortcut metadata"))
	}

	commandName := strings.TrimSuffix(filepath.Base(lnkPath), filepath.Ext(lnkPath))
	if commandName == "" {
		commandName = filepath.Base(lnkPath)
	}

	commandDescription := mo.TupleToOption(info.Description, info.Description != "")

	targetPath := strings.TrimSpace(info.TargetPath)
	targetPath = expandWindowsEnv(targetPath)
	if targetPath == "" {
		targetPath = lnkPath
	}

	iconPath := normalizeIconPath(info.IconLocation, targetPath)
	var commandIconPath mo.Option[string]
	if iconPath != "" {
		if fi, err := os.Stat(iconPath); err == nil && !fi.IsDir() {
			commandIconPath = mo.Some(iconPath)
		} else {
			logger.Warning(fmt.Sprintf("Shortcut %q uses default icon: %v", lnkPath, err))
		}
	}

	// Launch the shortcut through Windows Shell so Arguments and WorkingDirectory
	// retain their native semantics, and use its source path as the scan identity.
	return models.NewApplicationCommand(commandName, commandDescription, lnkPath, commandIconPath, mo.None[string](), fi.ModTime()), nil
}

func parseExecutable(exePath string) (*models.ApplicationCommand, error) {
	fi, err := statApplication(exePath)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, applicationError(ErrNotApplication, "inspect executable", exePath, nil)
	}
	commandName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	return models.NewApplicationCommand(commandName, mo.None[string](), exePath, mo.Some(exePath), mo.None[string](), fi.ModTime()), nil
}

func getStartMenuDirs() []string {
	var dirs []string

	if appData := os.Getenv("APPDATA"); appData != "" {
		dirs = append(dirs, filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		dirs = append(dirs, filepath.Join(programData, "Microsoft", "Windows", "Start Menu", "Programs"))
	}

	return dirs
}

func scanStartMenuShortcuts(seen map[string]struct{}) ([]AppPathInfo, error) {
	return scanWindowsPaths(getStartMenuDirs(), ".lnk", seen)
}

func scanExecutablesInDirs(dirs []string, seen map[string]struct{}) ([]AppPathInfo, error) {
	return scanWindowsPaths(dirs, ".exe", seen)
}

func scanWindowsPaths(dirs []string, extension string, seen map[string]struct{}) ([]AppPathInfo, error) {
	var infos []AppPathInfo
	var scanErrors []error
	for _, root := range dirs {
		if root == "" {
			continue
		}
		// An absent optional scan root is normal, but a failure below it is not.
		if fi, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "stat application scan folder", root, err))
			continue
		} else if !fi.IsDir() {
			scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "inspect application scan folder", root, fmt.Errorf("scan root is not a directory")))
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "walk application folder", path, err))
				return nil
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), extension) {
				return nil
			}
			key := PathKey(path)
			if _, exists := seen[key]; exists {
				return nil
			}
			fi, err := statApplication(path)
			if err != nil {
				scanErrors = append(scanErrors, err)
				return nil
			}
			infos = append(infos, AppPathInfo{Path: path, UpdateAt: fi.ModTime()})
			seen[key] = struct{}{}
			return nil
		})
		if walkErr != nil {
			scanErrors = append(scanErrors, applicationError(ErrApplicationUnreadable, "walk application folder", root, walkErr))
		}
	}
	return infos, errors.Join(scanErrors...)
}

func getWindowsApplicationPath() ([]AppPathInfo, error) {
	var scanErrors []error
	var appPathInfos []AppPathInfo
	seen := make(map[string]struct{})

	registryInfos, registryErr := getRegistryAppInfos()
	scanErrors = append(scanErrors, registryErr)
	for _, info := range registryInfos {
		key := PathKey(info.Path)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		appPathInfos = append(appPathInfos, info)
	}

	startMenuInfos, startMenuErr := scanStartMenuShortcuts(seen)
	scanErrors = append(scanErrors, startMenuErr)
	appPathInfos = append(appPathInfos, startMenuInfos...)

	extraDirs := getExtraSearchDirs()
	if len(extraDirs) > 0 {
		extraInfos, extraErr := scanExecutablesInDirs(extraDirs, seen)
		appPathInfos = append(appPathInfos, extraInfos...)
		scanErrors = append(scanErrors, extraErr)
	}

	logger.Info(fmt.Sprintf("Scanning registry, start menu, extra dirs: %v", extraDirs))
	return appPathInfos, errors.Join(scanErrors...)
}

func newApplicationFromInfo(info AppPathInfo) (*models.ApplicationCommand, error) {
	if info.Name == "" {
		return nil, applicationError(ErrInvalidMetadata, "read registered display name", info.Path, fmt.Errorf("missing display name"))
	}
	fi, err := statApplication(info.Path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() || !isExecutablePath(info.Path) {
		return nil, applicationError(ErrNotApplication, "inspect registered executable", info.Path, nil)
	}
	desc := mo.TupleToOption(info.Desc, info.Desc != "")
	var iconPath mo.Option[string]
	if info.IconPath != "" {
		if fi, err := os.Stat(info.IconPath); err == nil && !fi.IsDir() {
			iconPath = mo.Some(info.IconPath)
		} else {
			logger.Warning(fmt.Sprintf("Application %q uses default icon: %v", info.Path, err))
		}
	}
	return models.NewApplicationCommand(info.Name, desc, info.Path, iconPath, mo.None[string](), info.UpdateAt), nil
}

// GetApplications returns valid applications even when other paths fail to scan or parse.
func GetApplications() ([]*models.ApplicationCommand, error) {
	infos, err := GetAppPathInfosWithError()
	return applicationsFromPathInfos(infos, err)
}

func applicationsFromPathInfos(infos []AppPathInfo, scanErr error) ([]*models.ApplicationCommand, error) {
	var commands []*models.ApplicationCommand
	seen := make(map[string]struct{})
	errs := []error{scanErr}
	for _, info := range infos {
		var command *models.ApplicationCommand
		var err error
		if info.Name != "" {
			command, err = newApplicationFromInfo(info)
		} else {
			command, err = ParseApplication(info.Path)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		key := PathKey(command.Path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		commands = append(commands, command)
	}
	return commands, errors.Join(errs...)
}

func ParseApplication(appPath string) (*models.ApplicationCommand, error) {
	if _, err := statApplication(appPath); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(appPath))
	if ext == ".lnk" {
		return parseShortcut(appPath)
	}
	if ext == ".exe" {
		return parseExecutable(appPath)
	}
	return nil, applicationError(ErrNotApplication, "inspect application extension", appPath, nil)
}

func GetDefaultIconPath() string {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = "C:\\Windows"
	}
	return filepath.Join(systemRoot, "System32", "shell32.dll")
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
	return getWindowsApplicationPath()
}
