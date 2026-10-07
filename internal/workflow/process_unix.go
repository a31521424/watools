//go:build darwin || linux

package workflow

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func stopProcess(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// TerminateTask and WaitTaskGroup share the ordinary task policy with the GUI
// bootstrap. Installation CLIs have their own graceful commit policy.
func TerminateTask(cmd *exec.Cmd) error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
func WaitTaskGroup(pid int) error       { return waitProcessGroup(pid) }
func TaskGroupAlive(pid int) (bool, error) {
	err := syscall.Kill(-pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return err == nil, err
}
func cancelledProcessExit(err error) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return false
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && (status.Signal() == syscall.SIGTERM || status.Signal() == syscall.SIGKILL)
}
func waitProcessGroup(pid int) error {
	// Wails gives its native host up to ten seconds to close. If a build task
	// ignores TERM, stop the entire group before releasing the source lock.
	deadline := time.Now().Add(10 * time.Second)
	forced := false
	for {
		err := syscall.Kill(-pid, 0)
		if err == syscall.ESRCH {
			return nil
		}
		if err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if forced {
				return fmt.Errorf("process group %d did not exit after KILL; temporary artifacts retained", pid)
			}
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				return err
			}
			forced = true
			deadline = time.Now().Add(5 * time.Second)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
