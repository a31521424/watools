package application

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"watools/pkg/logger"
	"watools/pkg/models"

	"github.com/samber/mo"
)

type shortcutInfo struct {
	TargetPath   string `json:"TargetPath"`
	IconLocation string `json:"IconLocation"`
	Description  string `json:"Description"`
}

type AppPathInfo struct {
	Path     string
	UpdateAt time.Time
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

func getWindowsApplicationPath() []AppPathInfo {
	var appPathInfos []AppPathInfo
	seen := make(map[string]struct{})

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
			if _, exists := seen[path]; exists {
				return nil
			}
			fi, err := os.Stat(path)
			if err != nil {
				return nil
			}
			appPathInfos = append(appPathInfos, AppPathInfo{Path: path, UpdateAt: fi.ModTime()})
			seen[path] = struct{}{}
			return nil
		})
	}

	logger.Info(fmt.Sprintf("Scanning start menu folders: %v", getStartMenuDirs()))
	return appPathInfos
}

func GetApplications() ([]*models.ApplicationCommand, error) {
	var commands []*models.ApplicationCommand

	for _, appPathInfo := range getWindowsApplicationPath() {
		if command, err := ParseApplication(appPathInfo.Path); err == nil {
			commands = append(commands, command)
		} else {
			logger.Error(err, fmt.Sprintf("Failed to parse shortcut for '%s'", appPathInfo.Path))
		}
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
