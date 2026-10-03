package host

import (
	"encoding/json"
	"time"
	"watools/pkg/utils"
)

type ApplicationUsage struct {
	ID         string     `json:"id"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	UsedCount  *int64     `json:"usedCount"`
}
type PluginUsage struct {
	PackageID  string     `json:"packageId"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	UsedCount  *int64     `json:"usedCount"`
}

func validateUsage(id string, at *time.Time, count *int64) error {
	if id == "" {
		return invalid("统计对象 ID 不能为空")
	}
	if at == nil || at.IsZero() {
		return invalid("lastUsedAt 必须是有效时间")
	}
	if count == nil || *count < 0 || *count > 1e12 {
		return invalid("usedCount 必须是 0 到 1000000000000 的整数")
	}
	return nil
}

type PackageRequest struct {
	PackageID string `json:"packageId"`
}
type StorageKeyRequest struct {
	PackageID string `json:"packageId"`
	Key       string `json:"key"`
}
type StorageSetRequest struct {
	PackageID string          `json:"packageId"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
}

func validatePackage(id string) error {
	if err := utils.ValidatePluginPackageID(id); err != nil {
		return invalid("packageId 格式不正确")
	}
	return nil
}
func validateKey(id, key string) error {
	if err := validatePackage(id); err != nil {
		return err
	}
	if key == "" {
		return invalid("key 不能为空")
	}
	return nil
}
