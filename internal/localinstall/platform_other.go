//go:build !darwin

package localinstall

import "fmt"

func verifyBundle(string) error {
	return fmt.Errorf("local application installation currently supports macOS only")
}
func copyBundle(string, string) error {
	return fmt.Errorf("local application installation currently supports macOS only")
}
