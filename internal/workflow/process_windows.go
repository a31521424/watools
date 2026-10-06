package workflow

import "os/exec"

func configureProcess(*exec.Cmd)      {}
func stopProcess(cmd *exec.Cmd) error { return cmd.Process.Kill() }
func waitProcessGroup(int) error      { return nil }
