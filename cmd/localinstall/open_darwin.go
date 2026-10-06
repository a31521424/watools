package main

import "os/exec"

func openApplication(path string) error { return exec.Command("/usr/bin/open", path).Run() }
