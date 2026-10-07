package localinstall

import (
	"errors"
	"fmt"
	"os/exec"
)

func configureProcess(*exec.Cmd)      {}
func stopProcess(cmd *exec.Cmd) error { return cmd.Process.Kill() }
func signalProcess(cmd *exec.Cmd, installer bool) error {
	if installer {
		return fmt.Errorf("graceful installation cancellation is unavailable on Windows; waiting preserves the installation commit")
	}
	return stopProcess(cmd)
}
func processCancelled(err error, installer bool) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && (!installer || exit.ExitCode() == 130)
}
