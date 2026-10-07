package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"watools/internal/filelock"
)

const CommitMessage = "Artifacts ready. Applying application and plugin revisions."
const StopFailureMessage = "WATOOLS_TASK_STOP_FAILED"
const StopFailureFileEnv = "WATOOLS_INSTALL_STOP_FAILURE_FILE"

var ErrTaskStopFailed = errors.New("installation task could not be stopped")
var ErrInstallationCommitted = errors.New("installation committed with an output diagnostic failure")

const Help = `WaTools project CLI

  go run ./cmd/watoolsctl doctor
  go run ./cmd/watoolsctl setup
  go run ./cmd/watoolsctl dev [--port N]
  go run ./cmd/watoolsctl browser [--port N]
  go run ./cmd/watoolsctl bindings
  go run ./cmd/watoolsctl check
  go run ./cmd/watoolsctl check-race
  go run ./cmd/watoolsctl build [--debug]
  go run ./cmd/watoolsctl install [--scope all|application|plugins] [--plugin ID] [--app-dir DIR] [--no-launch]
  go run ./cmd/watoolsctl plugins list|package|install [package-id...]

Development tasks use .cache/dev; checks/builds use disposable data.
install applies to normal user data unless explicit WATOOLS_* directories are supplied.
Node/pnpm are used only for frontend tasks. No command resets user data.
`

func (r *Runner) Execute(ctx context.Context, args []string) error {
	o, err := Parse(args)
	if err != nil {
		return err
	}
	if o.Task == "help" {
		_, err := fmt.Fprint(r.Out, Help)
		return err
	}
	original := append([]string{}, r.Env...)
	t, err := ReadToolchain(r.Root)
	if err != nil {
		return err
	}
	r.prepare(t)
	if err := r.preflight(ctx, t, o); err != nil {
		return err
	}
	if o.Task == "doctor" {
		return nil
	}
	var temporary string
	switch o.Task {
	case "check", "check-race", "bindings", "build", "install":
		temporary, err = os.MkdirTemp("", "watools-workflow-*")
		if err != nil {
			return err
		}
		defer func() {
			if r.cleanupSafe {
				r.cleanup(temporary)
			} else {
				fmt.Fprintf(r.Err, "Preserved temporary artifacts: %s\n", temporary)
			}
		}()
		r.SetEnv("WATOOLS_DATA_DIR", filepath.Join(temporary, "data"))
		r.SetEnv("WATOOLS_CACHE_DIR", filepath.Join(temporary, "cache"))
	}
	for _, key := range []string{"WATOOLS_DATA_DIR", "WATOOLS_CACHE_DIR"} {
		if err := os.MkdirAll(EnvValue(r.Env, key), 0700); err != nil {
			return err
		}
	}
	// Mutating source operations share one kernel lock. Watcher child dev-build
	// acquires it for each build; the supervising dev session does not hold it.
	if o.Task != "dev" && o.Task != "browser" {
		dir := filepath.Join(r.Root, ".cache/installation")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		unlock, err := filelock.Acquire(filepath.Join(dir, "source.lock"))
		if err != nil {
			return err
		}
		defer unlock()
	}
	front := filepath.Join(r.Root, "frontend")
	run := func(name string, args ...string) error { return r.Run(ctx, r.Root, name, args...) }
	pnpm := func(args ...string) error { return r.Run(ctx, front, "pnpm", args...) }
	switch o.Task {
	case "setup":
		if err := run("go", "mod", "download"); err != nil {
			return err
		}
		return pnpm("install", "--frozen-lockfile")
	case "dev", "browser":
		if o.Task == "browser" {
			r.SetEnv("WATOOLS_BROWSER", "1")
		} else {
			r.SetEnv("WATOOLS_BROWSER", "0")
		}
		cli := filepath.Join(r.Root, ".cache/tools/watoolsctl")
		if err := run("go", "build", "-o", cli, "./cmd/watoolsctl"); err != nil {
			return err
		}
		template, err := os.ReadFile(filepath.Join(r.Root, "build/config.yml"))
		if err != nil {
			return err
		}
		quoted := strconvQuote(cli)
		binary := "watools.dev.app"
		if o.Task == "browser" {
			binary = "watools.browser.app"
		}
		content := strings.NewReplacer("${WAILS_VITE_PORT}", o.Port, "${WATOOLS_DEV_BINARY}", binary, "${WATOOLS_CTL}", quoted).Replace(string(template))
		config := filepath.Join(r.Root, ".cache/dev/wails-"+o.Task+".yml")
		if err := os.WriteFile(config, []byte(content), 0600); err != nil {
			return err
		}
		return r.wails(ctx, t, "dev", "-config", config, "-port", o.Port)
	case "bindings":
		if err := r.bindings(ctx, t); err != nil {
			return err
		}
		return pnpm("build")
	case "check":
		if err := pnpm("test"); err != nil {
			return err
		}
		if err := pnpm("build"); err != nil {
			return err
		}
		for _, args := range [][]string{{"test", "./..."}, {"test", "-tags", "server", "./internal/app", "./internal/api"}, {"vet", "./..."}, {"build", "-tags", "production,private_mac_apis", "./..."}} {
			if err := run("go", args...); err != nil {
				return err
			}
		}
		return nil
	case "check-race":
		if err := pnpm("build"); err != nil {
			return err
		}
		return run("go", "test", "-race", "./...")
	case "build", "dev-build":
		if err := r.bindings(ctx, t); err != nil {
			return err
		}
		if err := pnpm("build"); err != nil {
			return err
		}
		_, err := r.bundle(ctx, t, "", o.Task == "dev-build", o.Task == "dev-build" || o.Debug, o.Task == "dev-build" && EnvValue(r.Env, "WATOOLS_BROWSER") == "1")
		return err
	case "plugins":
		args := []string{"run", "./cmd/pluginctl", o.PluginAction}
		if o.PluginAction != "list" {
			args = append(args, "--output", filepath.Join(r.Root, ".cache/dev/packages"))
		}
		args = append(args, o.Plugins...)
		return run("go", args...)
	case "install":
		return r.install(ctx, t, o, temporary, original)
	}
	return fmt.Errorf("unhandled command: %s", o.Task)
}
func (r *Runner) install(ctx context.Context, t Toolchain, o Options, temp string, original []string) error {
	if _, err := fmt.Fprintln(r.Out, "Preparing dependencies and installation artifacts (user data untouched)."); err != nil {
		return fmt.Errorf("write installation preparation: %w", err)
	}
	if err := r.Run(ctx, r.Root, "go", "mod", "download"); err != nil {
		return err
	}
	if o.Scope != "plugins" {
		if err := r.Run(ctx, filepath.Join(r.Root, "frontend"), "pnpm", "install", "--frozen-lockfile"); err != nil {
			return err
		}
	}
	helper := filepath.Join(temp, "localinstall")
	if err := r.Run(ctx, r.Root, "go", "build", "-o", helper, "./cmd/localinstall"); err != nil {
		return err
	}
	packages := filepath.Join(temp, "packages")
	if o.Scope != "application" {
		args := append([]string{"run", "./cmd/pluginctl", "package", "--output", packages}, o.Plugins...)
		if err := r.Run(ctx, r.Root, "go", args...); err != nil {
			return err
		}
	}
	bundle := ""
	if o.Scope != "plugins" {
		if err := r.bindings(ctx, t); err != nil {
			return err
		}
		if err := r.Run(ctx, filepath.Join(r.Root, "frontend"), "pnpm", "build"); err != nil {
			return err
		}
		var err error
		bundle, err = r.bundle(ctx, t, filepath.Join(temp, "bundle"), false, false, false)
		if err != nil {
			return err
		}
	}
	goExecutable, err := FindExecutable("go", EnvValue(r.Env, "PATH"))
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(r.Out, CommitMessage); err != nil {
		return fmt.Errorf("write installation commit marker: %w", err)
	}
	for _, key := range []string{"WATOOLS_DATA_DIR", "WATOOLS_CACHE_DIR"} {
		r.SetEnv(key, EnvValue(original, key))
	}
	args := []string{"--source", r.Root, "--go", goExecutable, "--app-dir", o.AppDir}
	if bundle != "" {
		args = append(args, "--bundle", bundle)
	}
	if o.Scope != "application" {
		args = append(args, "--packages", packages)
	}
	if o.NoLaunch {
		args = append(args, "--no-launch")
	}
	return r.atomic(ctx, helper, args...)
}
