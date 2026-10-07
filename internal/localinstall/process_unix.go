//go:build darwin || linux

package localinstall

import (
	"errors"
	"os/exec"
	"syscall"
	"watools/internal/workflow"
)

func configureProcess(cmd *exec.Cmd)  { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func stopProcess(cmd *exec.Cmd) error { return workflow.TerminateTask(cmd) }
func signalProcess(cmd *exec.Cmd, installer bool) error {
	if !installer {
		return stopProcess(cmd)
	}
	err := syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
	return err
}
func processCancelled(err error, installer bool) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return false
	}
	if installer {
		return exit.ExitCode() == 130
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && (status.Signal() == syscall.SIGTERM || status.Signal() == syscall.SIGKILL)
}
