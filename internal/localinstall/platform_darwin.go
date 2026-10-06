package localinstall

import (
	"fmt"
	"howett.net/plist"
	"os"
	"os/exec"
	"path/filepath"
)

func verifyBundle(bundle string) error {
	info, err := os.Lstat(bundle)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("application must be a real bundle directory")
	}
	// Locally produced bundles contain no symlinks; avoid following paths outside them.
	if err := filepath.Walk(bundle, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("application contains a symlink: %s", path)
		}
		return nil
	}); err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(bundle, "Contents/Info.plist"))
	if err != nil {
		return err
	}
	var identity struct {
		Identifier string `plist:"CFBundleIdentifier"`
		Executable string `plist:"CFBundleExecutable"`
	}
	if _, err := plist.Unmarshal(content, &identity); err != nil {
		return err
	}
	if identity.Identifier != "com.wails.watools" || identity.Executable != "watools" {
		return fmt.Errorf("application identity mismatch")
	}
	binary, err := os.Stat(filepath.Join(bundle, "Contents/MacOS/watools"))
	if err != nil {
		return err
	}
	if !binary.Mode().IsRegular() || binary.Mode()&0111 == 0 {
		return fmt.Errorf("application executable is invalid")
	}
	output, err := exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", bundle).CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature verification: %s: %w", output, err)
	}
	return nil
}
func copyBundle(source, target string) error {
	output, err := exec.Command("/usr/bin/ditto", source, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("bundle preparation: %s: %w", output, err)
	}
	return nil
}
