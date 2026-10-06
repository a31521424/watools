//go:build darwin || linux

package workflow

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd)  { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func stopProcess(cmd *exec.Cmd) error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
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
