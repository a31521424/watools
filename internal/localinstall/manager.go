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
	mu                sync.Mutex
	dataDir, cacheDir string
	status            Status
	process           *exec.Cmd
	done              chan struct{}
	cancelled         bool
}

func NewManager(dataDir, cacheDir string) *Manager {
	return &Manager{dataDir: dataDir, cacheDir: cacheDir, status: Status{State: "idle", Output: []string{}, OfficialPlugins: []OfficialPlugin{}}}
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
	if m.status.State == "running" {
		return fmt.Errorf("安装任务执行中，不能修改源码目录")
	}
	if err := os.MkdirAll(m.dataDir, 0700); err != nil {
		return err
	}
	unlock, err := filelock.Acquire(filepath.Join(m.dataDir, ".local-install.lock"))
	if err != nil {
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
	if m.status.State == "running" {
		return fmt.Errorf("已有安装任务在执行")
	}
	if request.Scope != "all" && request.Scope != "application" && request.Scope != "plugins" {
		return fmt.Errorf("未知安装范围")
	}
	if request.Scope == "application" && len(request.PackageIDs) > 0 {
		return fmt.Errorf("应用安装不接受插件列表")
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
			return fmt.Errorf("无效的官方插件: %s", id)
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
	if err != nil {
		os.RemoveAll(bootstrapDir)
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
	m.done = make(chan struct{})
	go m.runJob(command, pipe, executable, bootstrapDir, value, request, runner.Env, m.done)
	return nil
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
	pipe, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	command.Stderr = command.Stdout
	if err := command.Start(); err != nil {
		pipe.Close()
		return nil, err
	}
	return pipe, nil
}
func (m *Manager) collect(command *exec.Cmd, pipe io.Reader) error {
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
		if line == workflow.CommitMessage {
			m.status.CanCancel = false
			m.status.Message = "正在切换应用和插件版本"
		}
		m.mu.Unlock()
	}
	if scanner.Err() != nil {
		_ = stopProcess(command)
		_, _ = io.Copy(io.Discard, pipe)
	}
	return errors.Join(command.Wait(), scanner.Err())
}
func (m *Manager) runJob(command *exec.Cmd, pipe io.Reader, executable, bootstrapDir string, value Configuration, request Request, environment []string, done chan struct{}) {
	defer close(done)
	defer os.RemoveAll(bootstrapDir)
	err := m.collect(command, pipe)
	m.mu.Lock()
	if err == nil && !m.cancelled {
		args := []string{"install", "--scope", request.Scope, "--app-dir", filepath.Dir(value.ApplicationPath), "--no-launch"}
		for _, id := range request.PackageIDs {
			args = append(args, "--plugin", id)
		}
		command = exec.Command(executable, args...)
		command.Dir = value.SourceRoot
		command.Env = environment
		nextPipe, startErr := startCommand(command)
		if startErr != nil {
			err = startErr
		} else {
			m.process = command
			m.status.Message = "正在准备依赖和构建"
			m.mu.Unlock()
			err = m.collect(command, nextPipe)
			m.mu.Lock()
		}
	}
	defer m.mu.Unlock()
	m.status.CanCancel = false
	m.process = nil
	if err == nil && !m.cancelled {
		m.status.State = "succeeded"
		m.status.Message = "安装完成"
		m.status.RestartRequired = m.status.RestartRequired || request.Scope != "plugins"
	} else if m.cancelled {
		// A cancellation racing the commit is deferred by the CLI. Its successful
		// completion is still an installed result and must show the restart prompt.
		if err == nil && m.status.Message == "正在切换应用和插件版本" {
			m.status.State = "succeeded"
			m.status.Message = "安装完成"
			m.status.RestartRequired = m.status.RestartRequired || request.Scope != "plugins"
		} else {
			m.status.State = "cancelled"
			m.status.Message = "任务已取消"
		}
	} else {
		m.status.State = "failed"
		m.status.Message = "安装失败: " + err.Error()
	}
}
func (m *Manager) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process == nil {
		return nil
	}
	if !m.status.CanCancel {
		return fmt.Errorf("正在提交安装，需等待完成或恢复")
	}
	m.cancelled = true
	m.status.CanCancel = false
	m.status.Message = "正在取消，等待子任务退出"
	return stopProcess(m.process)
}
func (m *Manager) Shutdown() {
	m.mu.Lock()
	process, done := m.process, m.done
	cancellable := m.status.CanCancel
	m.mu.Unlock()
	if process == nil {
		return
	}
	if cancellable {
		_ = m.Cancel()
	}
	// Native shutdown must not kill a commit midway. Build tasks forward cancellation
	// to their own children; commits finish before the application exits.
	if done != nil {
		<-done
	}
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
