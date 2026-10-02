//go:build darwin

package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUpdateScriptRollsBackFailedReplacement(t *testing.T) {
	for _, scenario := range []string{"missing prepared", "opener fails", "success"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			current := filepath.Join(dir, "App $(touch injected) `touch injected`.app")
			prepared := filepath.Join(dir, "next.app")
			backup := filepath.Join(dir, "backup.app")
			if err := os.Mkdir(current, 0700); err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(current, "version"), []byte("old"), 0600)
			if scenario != "missing prepared" {
				os.Mkdir(prepared, 0700)
				os.WriteFile(filepath.Join(prepared, "version"), []byte("new"), 0600)
			}
			opener := "/usr/bin/true"
			if scenario == "opener fails" {
				opener = "/usr/bin/false"
			}
			script := filepath.Join(dir, "apply.sh")
			os.WriteFile(script, []byte(applyUpdateScript), 0600)
			cmd := exec.Command("/bin/sh", script, "2147483647", current, prepared, backup, dir, opener)
			cmd.Dir = dir
			err := cmd.Run()
			if scenario == "success" && err != nil {
				t.Fatal(err)
			}
			if scenario != "success" && err == nil {
				t.Fatal("failure reported success")
			}
			data, err := os.ReadFile(filepath.Join(current, "version"))
			want := "old"
			if scenario == "success" {
				want = "new"
			}
			if err != nil || string(data) != want {
				t.Fatalf("current app not preserved: %s, %v", data, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "injected")); !os.IsNotExist(err) {
				t.Fatal("path was evaluated as shell source")
			}
		})
	}
}
