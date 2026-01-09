package application

import (
	"encoding/json"
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
	script := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut('" + escapePowerShellSingleQuoted(lnkPath) + "'); " +
		"$o=[pscustomobject]@{TargetPath=$s.TargetPath;IconLocation=$s.IconLocation;Description=$s.Description}; " +
		"$o | ConvertTo-Json -Compress"

	out, err := runPowerShell(script)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, fmt.Errorf("empty shortcut info")
	}
	var info shortcutInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return nil, fmt.Errorf("failed to parse shortcut info: %w", err)
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

func normalizePathKey(path string) string {
	if path == "" {
		return ""
	}
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

func findFirstExecutable(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

func getRegistryAppInfos() []AppPathInfo {
	var infos []AppPathInfo

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
			continue
		}
		subKeys, err := baseKey.ReadSubKeyNames(-1)
		baseKey.Close()
		if err != nil {
			continue
		}

		for _, sub := range subKeys {
			subKey, err := registry.OpenKey(root.key, root.path+`\`+sub, registry.READ|root.flag)
			if err != nil {
				continue
			}

			displayName, _, _ := subKey.GetStringValue("DisplayName")
			if strings.TrimSpace(displayName) == "" {
				subKey.Close()
				continue
			}

			displayIcon, _, _ := subKey.GetStringValue("DisplayIcon")
			installLocation, _, _ := subKey.GetStringValue("InstallLocation")
			description, _, _ := subKey.GetStringValue("Publisher")

			subKey.Close()

			iconPath := extractExecutablePath(displayIcon)
			targetPath := iconPath

			if targetPath == "" || !isExecutablePath(targetPath) {
				installLocation = expandWindowsEnv(strings.TrimSpace(installLocation))
				if installLocation != "" && filepath.IsAbs(installLocation) {
					if exePath := findFirstExecutable(installLocation); exePath != "" {
						targetPath = exePath
					}
				}
			}

			if targetPath == "" || !filepath.IsAbs(targetPath) || !isExecutablePath(targetPath) {
				continue
			}

			if _, err := os.Stat(targetPath); err != nil {
				continue
			}

			key := normalizePathKey(targetPath)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			fi, err := os.Stat(targetPath)
			if err != nil {
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

	return infos
}

func parseShortcut(lnkPath string) (*models.ApplicationCommand, error) {
	fi, err := os.Stat(lnkPath)
	if err != nil {
		return nil, err
	}
	info, err := getShortcutInfo(lnkPath)
	if err != nil {
		return nil, err
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
		if _, err := os.Stat(iconPath); err == nil {
			commandIconPath = mo.Some(iconPath)
		}
	}

	return models.NewApplicationCommand(commandName, commandDescription, targetPath, commandIconPath, mo.None[string](), fi.ModTime()), nil
}

func parseExecutable(exePath string) (*models.ApplicationCommand, error) {
	fi, err := os.Stat(exePath)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("path is directory: %s", exePath)
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

func scanStartMenuShortcuts(seen map[string]struct{}) []AppPathInfo {
	var appPathInfos []AppPathInfo

	for _, root := range getStartMenuDirs() {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".lnk") {
				return nil
			}
			key := normalizePathKey(path)
			if _, exists := seen[key]; exists {
				return nil
			}
			fi, err := os.Stat(path)
			if err != nil {
				return nil
			}
			appPathInfos = append(appPathInfos, AppPathInfo{Path: path, UpdateAt: fi.ModTime()})
			seen[key] = struct{}{}
			return nil
		})
	}

	return appPathInfos
}

func scanExecutablesInDirs(dirs []string, seen map[string]struct{}) []AppPathInfo {
	var infos []AppPathInfo
	for _, root := range dirs {
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".exe") {
				return nil
			}
			key := normalizePathKey(path)
			if _, exists := seen[key]; exists {
				return nil
			}
			fi, err := os.Stat(path)
			if err != nil {
				return nil
			}
			infos = append(infos, AppPathInfo{Path: path, UpdateAt: fi.ModTime()})
			seen[key] = struct{}{}
			return nil
		})
	}
	return infos
}

func getWindowsApplicationPath() []AppPathInfo {
	var appPathInfos []AppPathInfo
	seen := make(map[string]struct{})

	registryInfos := getRegistryAppInfos()
	for _, info := range registryInfos {
		key := normalizePathKey(info.Path)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		appPathInfos = append(appPathInfos, info)
	}

	startMenuInfos := scanStartMenuShortcuts(seen)
	appPathInfos = append(appPathInfos, startMenuInfos...)

	extraDirs := getExtraSearchDirs()
	if len(extraDirs) > 0 {
		appPathInfos = append(appPathInfos, scanExecutablesInDirs(extraDirs, seen)...)
	}

	logger.Info(fmt.Sprintf("Scanning registry, start menu, extra dirs: %v", extraDirs))
	return appPathInfos
}

func newApplicationFromInfo(info AppPathInfo) (*models.ApplicationCommand, error) {
	if info.Name == "" {
		return nil, fmt.Errorf("missing display name")
	}
	desc := mo.TupleToOption(info.Desc, info.Desc != "")
	var iconPath mo.Option[string]
	if info.IconPath != "" {
		if _, err := os.Stat(info.IconPath); err == nil {
			iconPath = mo.Some(info.IconPath)
		}
	}
	return models.NewApplicationCommand(info.Name, desc, info.Path, iconPath, mo.None[string](), info.UpdateAt), nil
}

func GetApplications() ([]*models.ApplicationCommand, error) {
	var commands []*models.ApplicationCommand
	seen := make(map[string]struct{})

	for _, appPathInfo := range getWindowsApplicationPath() {
		if appPathInfo.Name != "" {
			command, err := newApplicationFromInfo(appPathInfo)
			if err == nil {
				key := normalizePathKey(command.Path)
				if key != "" {
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						commands = append(commands, command)
					}
				}
				continue
			}
		}
		command, err := ParseApplication(appPathInfo.Path)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to parse application for '%s'", appPathInfo.Path))
			continue
		}
		key := normalizePathKey(command.Path)
		if key != "" {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}
		commands = append(commands, command)
	}
	return commands, nil
}

func ParseApplication(appPath string) (*models.ApplicationCommand, error) {
	ext := strings.ToLower(filepath.Ext(appPath))
	if ext == ".lnk" {
		return parseShortcut(appPath)
	}
	if ext == ".exe" {
		return parseExecutable(appPath)
	}
	return nil, fmt.Errorf("unsupported application path: %s", appPath)
}

func GetDefaultIconPath() string {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = "C:\\Windows"
	}
	return filepath.Join(systemRoot, "System32", "shell32.dll")
}

func GetAppPathInfos() []AppPathInfo {
	return getWindowsApplicationPath()
}
