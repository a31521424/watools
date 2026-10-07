package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func strconvQuote(value string) string { return strconv.Quote(value) }
func (r *Runner) wails(ctx context.Context, t Toolchain, args ...string) error {
	err := r.Run(ctx, r.Root, "go", append([]string{"run", "github.com/wailsapp/wails/v3/cmd/wails3@" + t.Wails}, args...)...)
	if r.cleanupSafe {
		err = errors.Join(err, restoreBindingPermissions(filepath.Join(r.Root, "frontend/bindings"), r.walkDir, r.chmod))
	}
	return err
}
func restoreBindingPermissions(root string, walk func(string, fs.WalkDirFunc) error, chmod func(string, fs.FileMode) error) error {
	var failures error
	err := walk(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("walk generated bindings at %s: %w", path, err))
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".json")) {
			if err := chmod(path, 0644); err != nil {
				failures = errors.Join(failures, fmt.Errorf("restore generated binding permission at %s: %w", path, err))
			}
		}
		return nil
	})
	return errors.Join(failures, wrapProcessError("walk generated bindings", err))
}
func (r *Runner) bindings(ctx context.Context, t Toolchain) error {
	dir := filepath.Join(r.Root, "frontend/dist")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		if err := os.WriteFile(index, []byte("<!doctype html><title>WaTools build placeholder</title>"), 0644); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("inspect frontend placeholder: %w", err)
	}
	return r.wails(ctx, t, "generate", "bindings", "-ts")
}
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}
func (r *Runner) bundle(ctx context.Context, t Toolchain, output string, development, debug, server bool) (string, error) {
	content, err := os.ReadFile(filepath.Join(r.Root, "wails.json"))
	if err != nil {
		return "", err
	}
	var project struct {
		Version string `json:"version"`
		Info    struct {
			ProductName string `json:"productName"`
			Comments    string `json:"comments"`
			Copyright   string `json:"copyright"`
		} `json:"info"`
	}
	if err := json.Unmarshal(content, &project); err != nil {
		return "", err
	}
	if output == "" {
		output = filepath.Join(r.Root, "build/bin")
		if development {
			output = filepath.Join(r.Root, ".cache/dev/bin")
		}
	}
	name := "watools.app"
	if development {
		name = "watools.dev.app"
	}
	if server {
		name = "watools.browser.app"
	}
	target := filepath.Join(output, name)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return "", err
	}
	bundle, err := os.MkdirTemp(output, ".watools-build-*.app")
	if err != nil {
		return "", err
	}
	defer func() {
		if r.cleanupSafe {
			r.cleanup(bundle)
		} else {
			fmt.Fprintf(r.Err, "Preserved unfinished bundle: %s\n", bundle)
		}
	}()
	contents := filepath.Join(bundle, "Contents")
	for _, dir := range []string{"MacOS", "Resources"} {
		if err := os.MkdirAll(filepath.Join(contents, dir), 0755); err != nil {
			return "", err
		}
	}
	args := []string{"build"}
	if debug {
		tag := "private_mac_apis"
		if server {
			tag = "server"
		}
		args = append(args, "-gcflags=all=-l", "-tags", tag)
	} else {
		args = append(args, "-tags", "production,private_mac_apis", "-trimpath", "-ldflags=-w -s")
	}
	args = append(args, "-buildvcs=false", "-o", filepath.Join(contents, "MacOS/watools"), ".")
	if err := r.Run(ctx, r.Root, "go", args...); err != nil {
		return "", err
	}
	icon := filepath.Join(output, "icons.icns")
	if err := r.wails(ctx, t, "generate", "icons", "-input", "build/appicon.png", "-macfilename", icon, "-windowsfilename", filepath.Join(output, "icon.ico")); err != nil {
		return "", err
	}
	if err := copyFile(icon, filepath.Join(contents, "Resources/icons.icns")); err != nil {
		return "", err
	}
	template := "Info.plist"
	if debug {
		template = "Info.dev.plist"
	}
	source, err := os.ReadFile(filepath.Join(r.Root, "build/darwin", template))
	if err != nil {
		return "", err
	}
	values := map[string]string{"ProductName": project.Info.ProductName, "BinaryName": "watools", "ProductIdentifier": "com.wails.watools", "ProductVersion": project.Version, "ProductComments": project.Info.Comments, "ProductCopyright": project.Info.Copyright}
	var missing string
	plist := regexp.MustCompile(`{{\.(\w+)}}`).ReplaceAllStringFunc(string(source), func(token string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{{."), "}}")
		value, ok := values[key]
		if !ok {
			missing = key
		}
		return html.EscapeString(value)
	})
	if missing != "" || strings.Contains(plist, "{{") {
		return "", fmt.Errorf("unsupported plist template field: %s", missing)
	}
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0644); err != nil {
		return "", err
	}
	assets := filepath.Join(r.Root, "build/darwin/Assets.car")
	if _, err := os.Stat(assets); err == nil {
		if err := copyFile(assets, filepath.Join(contents, "Resources/Assets.car")); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect optional bundle assets: %w", err)
	}
	if err := r.Run(ctx, r.Root, "codesign", "--force", "--deep", "--sign", "-", bundle); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := r.replaceBundle(bundle, target); err != nil {
		return "", err
	}
	fmt.Fprintf(r.Out, "Built %s\n", target)
	return target, nil
}

// Replace only a complete signed bundle; unsuccessful builds preserve the last
// usable output. The previous build is temporary, unlike installation backups.
func replaceBundle(staged, target string) error {
	return New("", nil, nil).replaceBundle(staged, target)
}
func (r *Runner) replaceBundle(staged, target string) error {
	previous, err := os.MkdirTemp(filepath.Dir(target), ".watools-previous-*")
	if err != nil {
		return err
	}
	if err := os.Remove(previous); err != nil {
		return err
	}
	moved := false
	defer func() {
		if !moved {
			r.cleanup(previous)
		}
	}()
	if err := os.Rename(target, previous); err == nil {
		moved = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		if moved {
			if restoreErr := os.Rename(previous, target); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restore previous build at %s: %w", previous, restoreErr))
			}
			moved = false
		}
		return err
	}
	moved = false
	return nil
}
