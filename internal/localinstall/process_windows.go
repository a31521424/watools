package localinstall

import "os/exec"

func configureProcess(*exec.Cmd)      {}
func stopProcess(cmd *exec.Cmd) error { return cmd.Process.Kill() }
