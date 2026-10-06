// Package workflow implements the project CLI without loading desktop services.
package workflow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	Root        string
	Env         []string
	Out, Err    io.Writer
	cleanupSafe bool
}

func New(root string, out, err io.Writer) *Runner {
	return &Runner{Root: root, Env: os.Environ(), Out: out, Err: err, cleanupSafe: true}
}
func (r *Runner) SetEnv(key, value string) {
	next := make([]string, 0, len(r.Env)+1)
	for _, entry := range r.Env {
		if !strings.HasPrefix(entry, key+"=") {
			next = append(next, entry)
		}
	}
	if value != "" {
		next = append(next, key+"="+value)
	}
	r.Env = next
}
func EnvValue(env []string, key string) string {
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, key+"="); ok {
			return value
		}
	}
	return ""
}
func FindExecutable(name, searchPath string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	for _, dir := range filepath.SplitList(searchPath) {
		if dir == "" {
			continue
		}
		file := filepath.Join(dir, name)
		if info, err := os.Stat(file); err == nil && info.Mode().IsRegular() && (info.Mode()&0111 != 0 || filepath.Ext(file) == ".exe") {
			return filepath.Abs(file)
		}
	}
	return "", fmt.Errorf("cannot find %s in tool PATH", name)
}
func (r *Runner) command(ctx context.Context, cwd string, atomic bool, out, stderr io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	executable, err := FindExecutable(name, EnvValue(r.Env, "PATH"))
	if err != nil {
		return err
	}
	command := exec.Command(executable, args...)
	command.Dir = cwd
	command.Env = r.Env
	command.Stdout = out
	command.Stderr = stderr
	// Bound inherited output pipes after the direct child exits. Cancellation
	// separately stops the full task group before its workspace can be released.
	command.WaitDelay = 5 * time.Second
	configureProcess(command)
	if err := command.Start(); err != nil {
		return err
	}
	finished := make(chan struct{})
	type cancellation struct {
		handled bool
		err     error
	}
	stopped := make(chan cancellation, 1)
	go func() {
		select {
		case <-ctx.Done():
		case <-finished:
		}
		result := cancellation{}
		if ctx.Err() != nil && !atomic {
			result.handled = true
			_ = stopProcess(command)
			result.err = waitProcessGroup(command.Process.Pid)
		}
		stopped <- result
	}()
	err = command.Wait()
	close(finished)
	result := <-stopped
	if ctx.Err() != nil && !atomic {
		// Cancellation may arrive just after the watcher observed normal exit.
		if !result.handled {
			_ = stopProcess(command)
			result.err = waitProcessGroup(command.Process.Pid)
		}
		if result.err != nil {
			r.cleanupSafe = false
			return result.err
		}
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}
func (r *Runner) Run(ctx context.Context, cwd string, name string, args ...string) error {
	fmt.Fprintf(r.Out, "\n> %s %s\n", name, strings.Join(args, " "))
	return r.command(ctx, cwd, false, r.Out, r.Err, name, args...)
}
func (r *Runner) atomic(ctx context.Context, name string, args ...string) error {
	fmt.Fprintf(r.Out, "\n> %s %s\n", name, strings.Join(args, " "))
	return r.command(ctx, r.Root, true, r.Out, r.Err, name, args...)
}
func (r *Runner) probe(ctx context.Context, cwd, name string, args ...string) (string, error) {
	var out, stderr bytes.Buffer
	if err := r.command(ctx, cwd, false, &out, &stderr, name, args...); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(out.String()), nil
}
func FindRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	executable, _ := os.Executable()
	for _, start := range []string{current, filepath.Dir(executable)} {
		for dir := start; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "cmd/watoolsctl/main.go")); err == nil {
				if _, err := os.Stat(filepath.Join(dir, "wails.json")); err == nil {
					return dir, nil
				}
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	return "", fmt.Errorf("run watoolsctl from the WaTools checkout")
}
