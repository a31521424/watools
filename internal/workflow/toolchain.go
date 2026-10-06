package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Toolchain struct{ Go, Wails, PNPM string }

func ReadToolchain(root string) (Toolchain, error) {
	var t Toolchain
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return t, err
	}
	match := regexp.MustCompile(`(?m)^toolchain (go\S+)$`).FindSubmatch(module)
	if len(match) > 1 {
		t.Go = string(match[1])
	}
	match = regexp.MustCompile(`(?m)^\s*github\.com/wailsapp/wails/v3 (v\S+)`).FindSubmatch(module)
	if len(match) > 1 {
		t.Wails = string(match[1])
	}
	var front struct {
		PackageManager string `json:"packageManager"`
	}
	content, err := os.ReadFile(filepath.Join(root, "frontend/package.json"))
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(content, &front); err != nil {
		return t, err
	}
	t.PNPM = strings.TrimPrefix(front.PackageManager, "pnpm@")
	if t.Go == "" || t.Wails == "" || t.PNPM == "" || t.PNPM == front.PackageManager {
		return t, fmt.Errorf("missing pinned Go/Wails/pnpm versions")
	}
	return t, nil
}
func (r *Runner) prepare(t Toolchain) {
	tools := filepath.Join(r.Root, ".cache/tools")
	for key, value := range map[string]string{"GOTOOLCHAIN": t.Go, "GOWORK": "off", "CGO_ENABLED": "1", "MACOSX_DEPLOYMENT_TARGET": "12.0", "CGO_CFLAGS": "-mmacosx-version-min=12.0", "CGO_LDFLAGS": "-mmacosx-version-min=12.0", "WAILS_SERVER_HOST": "127.0.0.1", "WAILS_SERVER_PORT": "9246"} {
		r.SetEnv(key, value)
	}
	for key, value := range map[string]string{"GOCACHE": filepath.Join(tools, "go-build"), "GOMODCACHE": filepath.Join(tools, "go-mod"), "GOPATH": filepath.Join(tools, "gopath"), "WATOOLS_LOG_LEVEL": "debug"} {
		if EnvValue(r.Env, key) == "" {
			r.SetEnv(key, value)
		}
	}
	r.SetEnv("WATOOLS_DATA_DIR", filepath.Join(r.Root, ".cache/dev/data"))
	r.SetEnv("WATOOLS_CACHE_DIR", filepath.Join(r.Root, ".cache/dev/cache"))
}
func (r *Runner) preflight(ctx context.Context, t Toolchain, o Options) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("local native workflow currently supports macOS only")
	}
	goVersion, err := r.probe(ctx, r.Root, "go", "version")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(goVersion, "go version "+t.Go+" ") {
		return fmt.Errorf("expected %s; received %s", t.Go, goVersion)
	}
	needsFront := o.Task != "plugins" && !(o.Task == "install" && o.Scope == "plugins")
	if needsFront {
		node, err := r.probe(ctx, r.Root, "node", "--version")
		if err != nil {
			return err
		}
		parts := strings.Split(strings.TrimPrefix(node, "v"), ".")
		major, _ := strconv.Atoi(parts[0])
		minor := 0
		if len(parts) > 1 {
			minor, _ = strconv.Atoi(parts[1])
		}
		if major < 22 || major == 22 && minor < 12 {
			return fmt.Errorf("Node >=22.12 required; received %s", node)
		}
		pnpm, err := r.probe(ctx, filepath.Join(r.Root, "frontend"), "pnpm", "--version")
		if err != nil {
			return err
		}
		if pnpm != t.PNPM {
			return fmt.Errorf("expected pnpm %s; received %s", t.PNPM, pnpm)
		}
	}
	sdk, err := r.probe(ctx, r.Root, "xcrun", "--show-sdk-path")
	if err != nil {
		return err
	}
	if sdk == "" {
		return fmt.Errorf("macOS SDK unavailable")
	}
	if o.Task != "doctor" && o.Task != "setup" && o.Task != "plugins" && o.Task != "install" {
		if _, err := os.Stat(filepath.Join(r.Root, "frontend/node_modules/typescript")); err != nil {
			return fmt.Errorf("frontend dependencies missing; run go run ./cmd/watoolsctl setup")
		}
	}
	if o.Task == "doctor" {
		fmt.Fprintf(r.Out, "Go: %s\nWails: %s\npnpm: %s\nmacOS SDK: %s\nData: %s\nCache: %s\n", goVersion, t.Wails, t.PNPM, sdk, EnvValue(r.Env, "WATOOLS_DATA_DIR"), EnvValue(r.Env, "WATOOLS_CACHE_DIR"))
	}
	return nil
}
