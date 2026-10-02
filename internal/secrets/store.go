package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"watools/pkg/utils"
)

// Secrets are separate from ordinary plugin storage and are never returned in
// plugin listings. Namespace validation is not a sandbox for trusted plugin code.
func account(packageID, key string) (string, error) {
	if err := utils.ValidatePluginPackageID(packageID); err != nil {
		return "", err
	}
	if key == "" || len(key) > 256 {
		return "", fmt.Errorf("invalid secret key")
	}
	sum := sha256.Sum256([]byte(packageID + "\x00" + key))
	return hex.EncodeToString(sum[:]), nil
}
func Get(packageID, key string) (string, error) {
	name, err := account(packageID, key)
	if err != nil {
		return "", err
	}
	return get(name)
}
func Set(packageID, key, value string) error {
	name, err := account(packageID, key)
	if err != nil {
		return err
	}
	if len(value) > 64<<10 {
		return fmt.Errorf("secret exceeds size limit")
	}
	return set(name, value)
}
func Remove(packageID, key string) error {
	name, err := account(packageID, key)
	if err != nil {
		return err
	}
	return remove(name)
}
