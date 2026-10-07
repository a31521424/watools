package workflow

import (
	"errors"
	"os/exec"
)

func configureProcess(*exec.Cmd)          {}
func stopProcess(cmd *exec.Cmd) error     { return cmd.Process.Kill() }
func waitProcessGroup(int) error          { return nil }
func TerminateTask(cmd *exec.Cmd) error   { return stopProcess(cmd) }
func WaitTaskGroup(pid int) error         { return waitProcessGroup(pid) }
func TaskGroupAlive(int) (bool, error)    { return false, nil }
func cancelledProcessExit(err error) bool { var exit *exec.ExitError; return errors.As(err, &exit) }
