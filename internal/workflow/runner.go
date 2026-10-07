// Package workflow implements the project CLI without loading desktop services.
package workflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	Root             string
	Env              []string
	Out, Err         io.Writer
	cleanupSafe      bool
	walkDir          func(string, fs.WalkDirFunc) error
	chmod            func(string, fs.FileMode) error
	removeAll        func(string) error
	stopProcess      func(*exec.Cmd) error
	waitProcessGroup func(int) error
	outputMu         sync.Mutex
}

// exec owns asynchronous output copies while lifecycle diagnostics can write
// before the child exits. Serialize both paths without exposing ReaderFrom.
type commandWriter struct {
	mu       *sync.Mutex
	writer   io.Writer
	failures *error
}

func (w commandWriter) Write(content []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.writer.Write(content)
	if err == nil && n != len(content) {
		err = io.ErrShortWrite
	}
	if err != nil && w.failures != nil {
		*w.failures = errors.Join(*w.failures, fmt.Errorf("write task output: %w", err))
	}
	return n, err
}
func (r *Runner) output(writer io.Writer) io.Writer {
	return commandWriter{mu: &r.outputMu, writer: writer}
}

func New(root string, out, err io.Writer) *Runner {
	if out == nil {
		out = io.Discard
	}
	if err == nil {
		err = io.Discard
	}
	return &Runner{Root: root, Env: os.Environ(), Out: out, Err: err, cleanupSafe: true, walkDir: filepath.WalkDir, chmod: os.Chmod, removeAll: os.RemoveAll, stopProcess: stopProcess, waitProcessGroup: waitProcessGroup}
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
	if err := ctx.Err(); err != nil && !atomic {
		return err
	}
	executable, err := FindExecutable(name, EnvValue(r.Env, "PATH"))
	if err != nil {
		return err
	}
	command := exec.Command(executable, args...)
	command.Dir = cwd
	command.Env = r.Env
	var outputErr error
	command.Stdout = commandWriter{mu: &r.outputMu, writer: out, failures: &outputErr}
	command.Stderr = commandWriter{mu: &r.outputMu, writer: stderr, failures: &outputErr}
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
		safe    bool
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
			stopErr := r.stopProcess(command)
			groupErr := r.waitProcessGroup(command.Process.Pid)
			result.safe = groupErr == nil
			result.err = errors.Join(wrapProcessError("terminate task", stopErr), wrapProcessError("wait task group", groupErr))
		}
		stopped <- result
	}()
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	var result cancellation
	reported := false
	select {
	case err = <-waited:
		close(finished)
		result = <-stopped
	case result = <-stopped:
		if result.err != nil {
			result.err = r.reportStopFailure(result.err)
			reported = true
		}
		// Reporting a bounded stop failure is separate from releasing resources.
		// Keep this CLI, its lock and its workspace until the actual owner exits.
		err = <-waited
		close(finished)
	}
	if ctx.Err() != nil && !atomic {
		// Cancellation may arrive just after the watcher observed normal exit.
		if !result.handled {
			stopErr := r.stopProcess(command)
			groupErr := r.waitProcessGroup(command.Process.Pid)
			result.safe = groupErr == nil
			result.err = errors.Join(wrapProcessError("terminate task", stopErr), wrapProcessError("wait task group", groupErr))
		}
		if result.err != nil {
			if !reported {
				result.err = r.reportStopFailure(result.err)
			}
			if !result.safe {
				r.cleanupSafe = false
				for {
					alive, probeErr := TaskGroupAlive(command.Process.Pid)
					if probeErr == nil && !alive {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				r.cleanupSafe = true
			}
			return errors.Join(result.err, wrapProcessError("wait task", err), outputErr)
		}
		if err != nil && !cancelledProcessExit(err) {
			return errors.Join(wrapProcessError("wait cancelled task", err), outputErr)
		}
		if outputErr != nil {
			return outputErr
		}
		return ctx.Err()
	}
	if !atomic {
		// Direct-child exit is insufficient when descendants still own inherited
		// output or temporary files. Finish the ordinary task group first.
		alive, probeErr := TaskGroupAlive(command.Process.Pid)
		if alive || probeErr != nil {
			stopErr := r.stopProcess(command)
			groupErr := r.waitProcessGroup(command.Process.Pid)
			controlErr := errors.Join(wrapProcessError("inspect completed task group", probeErr), wrapProcessError("terminate remaining task group", stopErr), wrapProcessError("wait remaining task group", groupErr))
			if controlErr != nil {
				controlErr = r.reportStopFailure(controlErr)
			}
			if groupErr != nil {
				r.cleanupSafe = false
				for {
					alive, probeErr := TaskGroupAlive(command.Process.Pid)
					if probeErr == nil && !alive {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				r.cleanupSafe = true
			}
			err = errors.Join(err, controlErr)
		}
	}
	err = errors.Join(err, outputErr)
	if err != nil {
		if atomic && command.ProcessState != nil && command.ProcessState.Success() {
			err = errors.Join(ErrInstallationCommitted, err)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}
func (r *Runner) reportStopFailure(cause error) error {
	err := errors.Join(ErrTaskStopFailed, cause)
	if path := EnvValue(r.Env, StopFailureFileEnv); path != "" {
		if writeErr := os.WriteFile(path+".tmp", []byte(StopFailureMessage+"\n"+cause.Error()), 0600); writeErr != nil {
			err = errors.Join(err, fmt.Errorf("write task stop notification: %w", writeErr))
		} else if renameErr := os.Rename(path+".tmp", path); renameErr != nil {
			err = errors.Join(err, fmt.Errorf("publish task stop notification: %w", renameErr))
		}
	}
	if _, writeErr := fmt.Fprintf(r.output(r.Err), "Task stop failed; retaining its resources until actual exit: %v\n", cause); writeErr != nil {
		err = errors.Join(err, fmt.Errorf("write task stop diagnostic: %w", writeErr))
	}
	if _, writeErr := fmt.Fprintln(r.output(r.Out), StopFailureMessage); writeErr != nil {
		err = errors.Join(err, fmt.Errorf("write task stop marker: %w", writeErr))
	}
	return err
}
func (r *Runner) Run(ctx context.Context, cwd string, name string, args ...string) error {
	if _, err := fmt.Fprintf(r.output(r.Out), "\n> %s %s\n", name, strings.Join(args, " ")); err != nil {
		return fmt.Errorf("write task announcement: %w", err)
	}
	return r.command(ctx, cwd, false, r.Out, r.Err, name, args...)
}
func (r *Runner) atomic(ctx context.Context, name string, args ...string) error {
	if _, err := fmt.Fprintf(r.output(r.Out), "\n> %s %s\n", name, strings.Join(args, " ")); err != nil {
		return fmt.Errorf("write commit announcement: %w", err)
	}
	return r.command(ctx, r.Root, true, r.Out, r.Err, name, args...)
}
func wrapProcessError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func (r *Runner) cleanup(path string) {
	if err := r.removeAll(path); err != nil {
		fmt.Fprintf(r.Err, "Temporary cleanup failed; retained %s: %v\n", path, err)
	}
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
