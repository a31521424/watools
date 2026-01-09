package models

import (
	"fmt"
	"os/exec"
)

func openApplication(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run application: %w\n%s", err, output)
	}
	return nil
}
