//go:build !windows && !darwin

package update

import "fmt"

func installUpdateAsset(downloadedPath string) (InstallResult, error) {
	return InstallResult{}, fmt.Errorf("self-update is not supported on this platform")
}
