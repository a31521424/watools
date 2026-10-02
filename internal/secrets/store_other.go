//go:build (!darwin && !windows) || (darwin && !cgo)

package secrets

import "fmt"

func get(string) (string, error) {
	return "", fmt.Errorf("secure storage is unavailable on this build")
}
func set(string, string) error { return fmt.Errorf("secure storage is unavailable on this build") }
func remove(string) error      { return fmt.Errorf("secure storage is unavailable on this build") }
