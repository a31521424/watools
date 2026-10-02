//go:build darwin

package update

import (
	"fmt"
	"howett.net/plist"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func installUpdateAsset(downloadedPath string) (InstallResult, error) {
	currentBundlePath, err := currentAppBundlePath()
	if err != nil {
		return InstallResult{
			Status:                "manual",
			Message:               "当前不是从 .app 安装包运行，已下载更新包，请手动替换应用",
			OpenPath:              downloadedPath,
			RequiresManualInstall: true,
		}, nil
	}

	targetDir := filepath.Dir(currentBundlePath)
	if err := ensureDirWritable(targetDir); err != nil {
		return InstallResult{
			Status:                "manual",
			Message:               "当前应用目录不可写，已下载更新包，请手动替换应用",
			OpenPath:              downloadedPath,
			RequiresManualInstall: true,
		}, nil
	}

	stagingDir, err := os.MkdirTemp(targetDir, ".watools-update-*")
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to create update staging directory: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(stagingDir)
		}
	}()
	extractDir := filepath.Join(stagingDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	if err := extractUpdateArchive(downloadedPath, extractDir); err != nil {
		return InstallResult{}, fmt.Errorf("failed to extract update: %w", err)
	}

	nextBundlePath, err := findExtractedAppBundle(extractDir)
	if err != nil {
		return InstallResult{}, err
	}

	scriptPath := filepath.Join(stagingDir, "apply-update.sh")
	if err := validateNextBundle(currentBundlePath, nextBundlePath); err != nil {
		return InstallResult{}, err
	}
	// Validate signatures when the running bundle is signed. Unsigned development
	// bundles retain the existing digest-based update policy.
	if exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", currentBundlePath).Run() == nil {
		if output, err := exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", nextBundlePath).CombinedOutput(); err != nil {
			return InstallResult{}, fmt.Errorf("invalid update signature: %w: %s", err, output)
		}
	}
	script := applyUpdateScript

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to write update script: %w", err)
	}

	if err := exec.Command("/bin/sh", scriptPath, strconv.Itoa(os.Getpid()), currentBundlePath, nextBundlePath, filepath.Join(stagingDir, "previous.app"), stagingDir, "/usr/bin/open").Start(); err != nil {
		return InstallResult{}, fmt.Errorf("failed to start update script: %w", err)
	}

	committed = true
	return InstallResult{
		Status:     "installing",
		Message:    "更新已准备完成，应用即将退出并替换自身",
		OpenPath:   currentBundlePath,
		ShouldQuit: true,
	}, nil
}

func currentAppBundlePath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	cleanPath := filepath.Clean(executablePath)
	marker := string(filepath.Separator) + "Contents" + string(filepath.Separator) + "MacOS" + string(filepath.Separator)
	index := strings.Index(cleanPath, marker)
	if index <= 0 {
		return "", fmt.Errorf("executable is not inside an .app bundle")
	}

	bundlePath := cleanPath[:index]
	if !strings.HasSuffix(strings.ToLower(bundlePath), ".app") {
		return "", fmt.Errorf("executable is not inside an .app bundle")
	}
	return bundlePath, nil
}

func ensureDirWritable(dir string) error {
	probeFile, err := os.CreateTemp(dir, ".watools-update-permission-*")
	if err != nil {
		return err
	}
	probePath := probeFile.Name()
	if err := probeFile.Close(); err != nil {
		_ = os.Remove(probePath)
		return err
	}
	return os.Remove(probePath)
}

func findExtractedAppBundle(root string) (string, error) {
	var bundlePath string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
			if bundlePath != "" {
				return fmt.Errorf("update archive contains multiple app bundles")
			}
			bundlePath = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to inspect extracted update bundle: %w", err)
	}
	if bundlePath == "" {
		return "", fmt.Errorf("update archive does not contain an .app bundle")
	}
	return bundlePath, nil
}

// Paths are positional parameters, never interpolated shell source. A failed
// replacement restores the previous bundle; successful updates retain a backup.
const applyUpdateScript = `#!/bin/sh
set -u
pid="$1"; current="$2"; prepared="$3"; backup="$4"; staging="$5"; opener="$6"
tries=0
while kill -0 "$pid" 2>/dev/null; do
 tries=$((tries + 1))
 if [ "$tries" -gt 60 ]; then exit 1; fi
 sleep 1
done
if ! mv "$current" "$backup"; then exit 1; fi
if ! mv "$prepared" "$current"; then
 mv "$backup" "$current" || exit 2
 exit 1
fi
if ! "$opener" "$current"; then
 mv "$current" "$staging/failed.app" || exit 2
 mv "$backup" "$current" || exit 2
 "$opener" "$current" || true
 exit 1
fi
`

func validateNextBundle(current, next string) error {
	read := func(root string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, "Contents", "Info.plist"))
		if err != nil {
			return "", err
		}
		var metadata struct {
			Identifier string `plist:"CFBundleIdentifier"`
			Executable string `plist:"CFBundleExecutable"`
		}
		if _, err := plist.Unmarshal(data, &metadata); err != nil {
			return "", err
		}
		if metadata.Identifier == "" || metadata.Executable == "" || filepath.Base(metadata.Executable) != metadata.Executable {
			return "", fmt.Errorf("invalid bundle metadata")
		}
		info, err := os.Stat(filepath.Join(root, "Contents", "MacOS", metadata.Executable))
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return "", fmt.Errorf("invalid bundle executable")
		}
		return metadata.Identifier, nil
	}
	oldID, err := read(current)
	if err != nil {
		return err
	}
	newID, err := read(next)
	if err != nil {
		return err
	}
	if newID != oldID {
		return fmt.Errorf("update bundle identifier does not match")
	}
	return nil
}
