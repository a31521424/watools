package localinstall

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"watools/internal/filelock"
	"watools/internal/workflow"
	"watools/pkg/logger"
)

type Request struct {
	Scope      string   `json:"scope"`
	PackageIDs []string `json:"packageIds"`
}

type Status struct {
	Configuration   Configuration    `json:"configuration"`
	OfficialPlugins []OfficialPlugin `json:"officialPlugins"`
	JobID           uint64           `json:"jobId"`
	State           string           `json:"state"`
	Scope           string           `json:"scope"`
	Message         string           `json:"message"`
	Output          []string         `json:"output"`
	CanCancel       bool             `json:"canCancel"`
	RestartRequired bool             `json:"restartRequired"`
}

type Manager struct {
	mu                 sync.Mutex
	dataDir, cacheDir  string
	status             Status
	process            *exec.Cmd
	done               chan struct{}
	cancelled          bool
	phase              jobPhase
	jobErr             error
	stop               *processStop
	stopWake           chan struct{}
	collectSafe        bool
	processWait        *processWait
	outputExited       chan struct{}
	stalled            chan struct{}
	signalProcess      func(*exec.Cmd, bool) error
	waitProcessGroup   func(int) error
	outputDrainTimeout time.Duration
	removeAll          func(string) error
	stopFailureFile    string
}

type jobPhase uint8

const (
	phaseBootstrap jobPhase = iota
	phaseBuild
	phaseCommit
)

type processStop struct {
	done chan struct{}
	err  error
	safe bool
}
type processWait struct {
	done chan struct{}
	err  error
}

func NewManager(dataDir, cacheDir string) *Manager {
	return &Manager{dataDir: dataDir, cacheDir: cacheDir, status: Status{State: "idle", Output: []string{}, OfficialPlugins: []OfficialPlugin{}}, signalProcess: signalProcess, waitProcessGroup: workflow.WaitTaskGroup, outputDrainTimeout: 5 * time.Second, removeAll: os.RemoveAll, stopWake: make(chan struct{}, 1), collectSafe: true, stalled: make(chan struct{})}
}
func (m *Manager) Status() (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, err := ReadConfiguration(m.dataDir)
	if err != nil {
		return Status{}, err
	}
	m.status.Configuration = value
	m.status.OfficialPlugins = []OfficialPlugin{}
	if value.SourceRoot != "" {
		plugins, err := ValidateSource(value.SourceRoot)
		if err == nil {
			m.status.OfficialPlugins = plugins
		} else if m.status.State != "running" {
			m.status.Message = "源码目录不可用，请重新设置: " + err.Error()
		}
	}
	result := m.status
	result.Output = append([]string{}, m.status.Output...)
	result.OfficialPlugins = append([]OfficialPlugin{}, m.status.OfficialPlugins...)
	return result, nil
}
func (m *Manager) Configure(value Configuration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.State == "running" || m.process != nil {
		return fmt.Errorf("安装任务执行中，不能修改源码目录: %w", ErrBusy)
	}
	if err := os.MkdirAll(m.dataDir, 0700); err != nil {
		return err
	}
	unlock, err := filelock.Acquire(filepath.Join(m.dataDir, ".local-install.lock"))
	if err != nil {
		if errors.Is(err, filelock.ErrBusy) {
			return fmt.Errorf("%w: %w", ErrBusy, err)
		}
		return err
	}
	defer unlock()
	old, err := ReadConfiguration(m.dataDir)
	if err != nil {
		return err
	}
	// Installation identity and successful receipt are owned by the installer.
	value.ApplicationPath = old.ApplicationPath
	value.InstalledAt = old.InstalledAt
	value.Version = old.Version
	value, err = ValidateConfiguration(value)
	if err != nil {
		return err
	}
	if err := SaveConfiguration(m.dataDir, value); err != nil {
		return err
	}
	m.status.Message = "源码目录已保存"
	return nil
}
func (m *Manager) Start(ctx context.Context, request Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.status.State == "running" || m.process != nil {
		return fmt.Errorf("已有安装任务在执行: %w", ErrBusy)
	}
	if request.Scope != "all" && request.Scope != "application" && request.Scope != "plugins" {
		return validation("未知安装范围", nil)
	}
	if request.Scope == "application" && len(request.PackageIDs) > 0 {
		return validation("应用安装不接受插件列表", nil)
	}
	value, err := ReadConfiguration(m.dataDir)
	if err != nil {
		return err
	}
	value, err = ValidateConfiguration(value)
	if err != nil {
		return err
	}
	plugins, err := ValidateSource(value.SourceRoot)
	if err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, p := range plugins {
		ids[p.PackageID] = true
	}
	seen := map[string]bool{}
	for _, id := range request.PackageIDs {
		if !ids[id] || seen[id] {
			return validation("官方插件选择无效或重复", nil)
		}
		seen[id] = true
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tools, err := workflow.ReadToolchain(value.SourceRoot)
	if err != nil {
		return err
	}
	bootstrapDir, err := os.MkdirTemp(m.cacheDir, "installer-cli-*")
	if err != nil {
		if err := os.MkdirAll(m.cacheDir, 0700); err != nil {
			return err
		}
		bootstrapDir, err = os.MkdirTemp(m.cacheDir, "installer-cli-*")
		if err != nil {
			return err
		}
	}
	executable := filepath.Join(bootstrapDir, "watoolsctl")
	environment := installationEnvironment(os.Environ(), value, m.dataDir, m.cacheDir)
	runner := workflow.New(value.SourceRoot, nil, nil)
	runner.Env = environment
	m.stopFailureFile = filepath.Join(bootstrapDir, "stop-failure")
	runner.SetEnv(workflow.StopFailureFileEnv, m.stopFailureFile)
	runner.SetEnv("GOTOOLCHAIN", tools.Go)
	runner.SetEnv("GOWORK", "off")
	for key, path := range map[string]string{"GOCACHE": "go-build", "GOMODCACHE": "go-mod", "GOPATH": "gopath"} {
		if workflow.EnvValue(runner.Env, key) == "" {
			runner.SetEnv(key, filepath.Join(value.SourceRoot, ".cache/tools", path))
		}
	}
	command := exec.Command(value.GoExecutable, "build", "-o", executable, "./cmd/watoolsctl")
	command.Dir = value.SourceRoot
	command.Env = runner.Env
	pipe, err := startCommand(command)
	if err != nil && pipe == nil {
		if cleanupErr := m.removeAll(bootstrapDir); cleanupErr != nil {
			logger.Error(cleanupErr, "Local installation cleanup retained "+bootstrapDir)
		}
		return err
	}
	m.status.JobID++
	m.status.State = "running"
	m.status.Scope = request.Scope
	m.status.Message = "正在准备 Go 安装工具"
	m.status.Output = []string{}
	m.status.CanCancel = true
	m.process = command
	m.cancelled = false
	m.phase = phaseBootstrap
	m.jobErr = err
	m.stop = nil
	m.collectSafe = true
	m.processWait = nil
	m.stalled = make(chan struct{})
	m.done = make(chan struct{})
	if err != nil {
		m.beginStopLocked(command, false)
	}
	go m.runJob(command, pipe, executable, bootstrapDir, value, request, runner.Env, m.done)
	return err
}
func installationEnvironment(original []string, value Configuration, dataDir, cacheDir string) []string {
	result := []string{}
	for _, item := range original {
		key, _, _ := strings.Cut(item, "=")
		if key != "PATH" && key != "WATOOLS_DATA_DIR" && key != "WATOOLS_CACHE_DIR" {
			result = append(result, item)
		}
	}
	return append(result, "PATH="+value.ToolPath, "WATOOLS_DATA_DIR="+dataDir, "WATOOLS_CACHE_DIR="+cacheDir)
}
func startCommand(command *exec.Cmd) (io.ReadCloser, error) {
	configureProcess(command)
	pipe, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	command.Stdout = writer
	command.Stderr = writer
	if err := command.Start(); err != nil {
		return nil, errors.Join(err, pipe.Close(), writer.Close())
	}
	if err := writer.Close(); err != nil {
		// The child already started. A nonnil reader transfers its ownership to
		// the caller even when closing the parent writer reports a failure.
		return pipe, fmt.Errorf("close parent output writer: %w", err)
	}
	return pipe, nil
}

// collect owns Wait and the output reader. Waiting and reading run together so
// inherited output pipes cannot keep a completed task alive indefinitely.
func (m *Manager) collect(command *exec.Cmd, pipe io.Reader) error {
	wait := &processWait{done: make(chan struct{})}
	outputExited := make(chan struct{})
	m.mu.Lock()
	m.processWait = wait
	m.outputExited = outputExited
	m.mu.Unlock()
	go func() { wait.err = command.Wait(); close(wait.done) }()
	readDone := make(chan error, 1)
	go func() {
		defer close(outputExited)
		scanner := bufio.NewScanner(pipe)
		scanner.Buffer(make([]byte, 4096), 64<<10)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 2000 {
				line = line[:2000] + "…"
			}
			m.mu.Lock()
			m.status.Output = append(m.status.Output, line)
			if len(m.status.Output) > 120 {
				m.status.Output = m.status.Output[len(m.status.Output)-120:]
			}
			if line == workflow.CommitMessage && m.phase != phaseBootstrap {
				m.phase = phaseCommit
				m.status.CanCancel = false
				m.status.Message = "正在切换应用和插件版本"
			}
			if line == workflow.StopFailureMessage && m.phase != phaseBootstrap {
				m.markStalledLocked(workflow.ErrTaskStopFailed)
			}
			m.mu.Unlock()
		}
		readErr := scanner.Err()
		if readErr != nil {
			m.mu.Lock()
			m.markStalledLocked(fmt.Errorf("read installation output: %w", readErr))
			// Never force the trusted install CLI: a read error may have hidden its
			// commit marker. Its TERM handler owns build stopping and atomic recovery.
			if m.phase != phaseCommit {
				m.beginStopLocked(command, false)
			}
			m.mu.Unlock()
			_, drainErr := io.Copy(io.Discard, pipe)
			readErr = errors.Join(fmt.Errorf("read installation output: %w", readErr), wrapError("drain installation output", drainErr))
		}
		readDone <- readErr
	}()
	var readErr, waitErr, stopErr error
	var handledStop *processStop
	closedPipe := false
	drainExpired := false
	closePipe := func() error {
		if closedPipe {
			return nil
		}
		if closer, ok := pipe.(io.Closer); ok {
			closedPipe = true
			return closer.Close()
		}
		return nil
	}
	readerDone, childDone := false, false
	var timer *time.Timer
	var deadline <-chan time.Time
	waitDone := wait.done
	notificationRead := false
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	for !readerDone || !childDone {
		m.mu.Lock()
		stop := m.stop
		m.mu.Unlock()
		var stopped <-chan struct{}
		if stop != nil && stop != handledStop {
			stopped = stop.done
		}
		select {
		case <-poll.C:
			m.mu.Lock()
			path, phase := m.stopFailureFile, m.phase
			m.mu.Unlock()
			if phase != phaseBootstrap && path != "" && !notificationRead {
				if content, err := os.ReadFile(path); err == nil {
					notificationRead = true
					m.markStalled(taskStopNotification(content))
				} else if !os.IsNotExist(err) {
					notificationRead = true
					m.markStalled(wrapError("read task stop notification", err))
				}
			}
		case result := <-readDone:
			readErr = errors.Join(readErr, result)
			readerDone = true
			readDone = nil
		case <-waitDone:
			childDone = true
			waitErr = wait.err
			waitDone = nil
			if !readerDone {
				timer = time.NewTimer(m.outputDrainTimeout)
				deadline = timer.C
			}
		case <-m.stopWake:
		case <-stopped:
			stopErr = stop.err
			m.mu.Lock()
			handledStop = stop
			m.collectSafe = stop.safe
			m.mu.Unlock()
			if !stop.safe {
				// Keep the process, job completion channel and resources until the real
				// process/group exits. Shutdown can report this bounded stop failure now.
				stopErr = errors.Join(stopErr, wrapError("close installation output", closePipe()))
				if timer != nil {
					timer.Stop()
				}
				return errors.Join(readErr, stopErr)
			}
		case <-deadline:
			deadline = nil
			if drainExpired {
				m.markStalled(errors.Join(readErr, waitErr, stopErr))
				return errors.Join(readErr, waitErr, stopErr)
			}
			drainExpired = true
			readErr = errors.Join(readErr, fmt.Errorf("installation output did not drain before deadline: %w", exec.ErrWaitDelay))
			if _, ok := pipe.(io.Closer); ok {
				readErr = errors.Join(readErr, wrapError("close installation output", closePipe()))
				timer = time.NewTimer(m.outputDrainTimeout)
				deadline = timer.C
			} else {
				// A non-closeable reader is only used by tests; do not claim it drained.
				m.markStalled(errors.Join(readErr, waitErr, stopErr))
				return errors.Join(readErr, waitErr, stopErr)
			}
		}
	}
	if timer != nil {
		timer.Stop()
	}
	<-outputExited
	readErr = errors.Join(readErr, wrapError("close installation output", closePipe()))
	m.mu.Lock()
	cancelled := m.cancelled
	phase := m.phase
	if phase != phaseBootstrap && m.stopFailureFile != "" && !notificationRead {
		if content, err := os.ReadFile(m.stopFailureFile); err == nil {
			m.markStalledLocked(taskStopNotification(content))
		} else if !os.IsNotExist(err) {
			m.markStalledLocked(wrapError("read task stop notification", err))
		}
	}
	if phase == phaseBootstrap && m.stop == nil {
		alive, probeErr := workflow.TaskGroupAlive(command.Process.Pid)
		if alive || probeErr != nil {
			m.beginStopLocked(command, false)
		}
	}
	stop := m.stop
	m.mu.Unlock()
	if stop != nil && stop != handledStop {
		<-stop.done
		stopErr = errors.Join(stopErr, stop.err)
		m.mu.Lock()
		m.collectSafe = stop.safe
		m.mu.Unlock()
	}
	// A signalled exit is expected after an accepted cancellation, but ordinary
	// nonzero exits and output/control failures remain real errors.
	if cancelled && phase != phaseCommit && processCancelled(waitErr, phase != phaseBootstrap) {
		waitErr = nil
	}
	return errors.Join(readErr, wrapError("wait installation process", waitErr), stopErr)
}
func taskStopNotification(content []byte) error {
	detail := strings.TrimPrefix(string(content), workflow.StopFailureMessage+"\n")
	if string(content) == workflow.StopFailureMessage {
		detail = ""
	}
	failure := workflow.ErrTaskStopFailed
	if detail != "" {
		failure = errors.Join(failure, fmt.Errorf("installation CLI task stop: %s", detail))
	}
	return failure
}
func (m *Manager) markStalled(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markStalledLocked(err)
}
func (m *Manager) markStalledLocked(err error) {
	m.collectSafe = false
	if !errors.Is(m.jobErr, err) {
		m.jobErr = errors.Join(m.jobErr, err)
	}
	select {
	case <-m.stalled:
	default:
		close(m.stalled)
	}
}
func wrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func (m *Manager) beginStopLocked(command *exec.Cmd, acceptedCancellation bool) error {
	signalErr := wrapError("stop local installation process", m.signalProcess(command, m.phase != phaseBootstrap))
	if signalErr != nil {
		m.jobErr = errors.Join(m.jobErr, signalErr)
		m.markStalledLocked(signalErr)
	}
	if signalErr == nil && acceptedCancellation {
		m.cancelled = true
		m.status.CanCancel = false
		m.status.Message = "正在取消，等待子任务退出"
	}
	if m.phase == phaseBootstrap && m.stop == nil {
		stop := &processStop{done: make(chan struct{})}
		m.stop = stop
		go func() {
			err := m.waitProcessGroup(command.Process.Pid)
			stop.err = errors.Join(signalErr, wrapError("wait installation task group", err))
			stop.safe = err == nil
			close(stop.done)
		}()
		select {
		case m.stopWake <- struct{}{}:
		default:
		}
	}
	return signalErr
}
func (m *Manager) cleanup(path string) {
	if err := m.removeAll(path); err != nil {
		m.mu.Lock()
		m.status.Output = append(m.status.Output, "临时安装工具清理失败，保留目录: "+path)
		m.mu.Unlock()
		// Harmless bootstrap leftovers do not reverse a completed installation.
		logger.Error(err, "Local installation cleanup retained "+path)
	}
}
func (m *Manager) awaitUnsafeProcessLocked(command *exec.Cmd, err error) {
	if !m.collectSafe {
		wait := m.processWait
		outputExited := m.outputExited
		m.status.CanCancel = false
		m.status.Message = "子任务停止失败，等待实际退出"
		m.jobErr = errors.Join(m.jobErr, err)
		m.mu.Unlock()
		if wait != nil {
			<-wait.done
		}
		for {
			alive, probeErr := workflow.TaskGroupAlive(command.Process.Pid)
			if probeErr == nil && !alive {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if outputExited != nil {
			<-outputExited
		}
		m.mu.Lock()
	}
}
func (m *Manager) runJob(command *exec.Cmd, pipe io.Reader, executable, bootstrapDir string, value Configuration, request Request, environment []string, done chan struct{}) {
	err := m.collect(command, pipe)
	m.mu.Lock()
	m.awaitUnsafeProcessLocked(command, err)
	if err == nil && m.jobErr == nil && !m.cancelled {
		args := []string{"install", "--scope", request.Scope, "--app-dir", filepath.Dir(value.ApplicationPath), "--no-launch"}
		for _, id := range request.PackageIDs {
			args = append(args, "--plugin", id)
		}
		command = exec.Command(executable, args...)
		command.Dir = value.SourceRoot
		command.Env = environment
		nextPipe, startErr := startCommand(command)
		if startErr != nil && nextPipe == nil {
			err = startErr
		} else {
			m.process = command
			m.phase = phaseBuild
			m.stop = nil
			m.processWait = nil
			m.collectSafe = true
			m.status.Message = "正在准备依赖和构建"
			if startErr != nil {
				m.jobErr = errors.Join(m.jobErr, startErr)
				m.beginStopLocked(command, false)
			}
			m.mu.Unlock()
			err = m.collect(command, nextPipe)
			m.mu.Lock()
			m.awaitUnsafeProcessLocked(command, err)
		}
	}
	m.jobErr = errors.Join(m.jobErr, err)
	if m.jobErr != nil {
		logger.Error(m.jobErr, "Local installation task reported a failure")
	}
	m.mu.Unlock()
	m.cleanup(bootstrapDir)
	m.mu.Lock()
	phase, cancelled := m.phase, m.cancelled
	// The trusted install command exits zero only after the commit completes.
	// Output/control failures remain diagnostic errors without reversing that fact.
	installed := phase != phaseBootstrap && command.ProcessState != nil && (command.ProcessState.Success() || command.ProcessState.ExitCode() == 131)
	m.status.CanCancel = false
	if installed {
		m.status.State = "succeeded"
		m.status.Message = "安装完成"
		if m.jobErr != nil {
			m.status.Message = "安装已完成，但停止或输出处理存在故障，请查看诊断日志"
		}
		m.status.RestartRequired = m.status.RestartRequired || request.Scope != "plugins"
	} else if cancelled && m.jobErr == nil {
		m.status.State = "cancelled"
		m.status.Message = "任务已取消"
	} else {
		m.status.State = "failed"
		m.status.Message = "安装失败，请查看诊断日志"
	}
	m.process = nil
	close(done)
	m.mu.Unlock()
}
func (m *Manager) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process == nil {
		return nil
	}
	if !m.status.CanCancel {
		return fmt.Errorf("正在提交安装，需等待完成或恢复: %w", ErrBusy)
	}
	return m.beginStopLocked(m.process, true)
}
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	process, done, stalled := m.process, m.done, m.stalled
	jobErr := m.jobErr
	var cancelErr error
	if process != nil && m.status.CanCancel {
		cancelErr = m.beginStopLocked(process, true)
	}
	stop := m.stop
	phase := m.phase
	m.mu.Unlock()
	if cancelErr != nil && phase != phaseBootstrap {
		return errors.Join(cancelErr, jobErr)
	}
	if stop != nil {
		<-stop.done
		if !stop.safe {
			m.mu.Lock()
			jobErr := m.jobErr
			m.mu.Unlock()
			return errors.Join(cancelErr, stop.err, jobErr)
		}
	}
	if done != nil {
		select {
		case <-done:
		case <-stalled:
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return errors.Join(cancelErr, m.jobErr)
}

// Wait is kept internal to tests and command-line integration.
func (m *Manager) wait(timeout time.Duration) bool {
	m.mu.Lock()
	done := m.done
	m.mu.Unlock()
	if done == nil {
		return true
	}
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
