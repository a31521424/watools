//go:build !darwin

package main

import "fmt"

func openApplication(string) error { return fmt.Errorf("unsupported platform") }
