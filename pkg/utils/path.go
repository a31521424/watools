package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var pluginPackageIDPattern = regexp.MustCompile(`^watools\.plugin\.[a-z0-9][a-z0-9._-]*$`)

func ValidatePluginPackageID(packageID string) error {
	if !pluginPackageIDPattern.MatchString(packageID) {
		return fmt.Errorf("packageId must match %q", pluginPackageIDPattern.String())
	}
	return nil
}

func ResolvePathWithinBase(baseDir string, relativePath string) (string, error) {
	if relativePath == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if strings.Contains(relativePath, "\x00") {
		return "", fmt.Errorf("path cannot contain NUL byte")
	}
	normalizedInput := strings.ReplaceAll(relativePath, "\\", "/")
	if strings.HasPrefix(normalizedInput, "/") {
		return "", fmt.Errorf("path cannot be absolute")
	}
	if strings.Contains(normalizedInput, ":") {
		return "", fmt.Errorf("path cannot contain drive specifier")
	}
	for _, segment := range strings.Split(normalizedInput, "/") {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("path cannot contain relative traversal segments")
		}
	}

	cleanedRelative := path.Clean("/" + normalizedInput)
	cleanedRelative = strings.TrimPrefix(cleanedRelative, "/")
	if cleanedRelative == "" || cleanedRelative == "." {
		return "", fmt.Errorf("path cannot resolve to base directory")
	}

	targetPath := filepath.Join(baseDir, filepath.FromSlash(cleanedRelative))
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}

	if absTarget != absBase && !strings.HasPrefix(absTarget, absBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base directory")
	}

	return absTarget, nil
}
