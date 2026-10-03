package host

import (
	"encoding/json"
	"watools/internal/plugin"
	"watools/internal/secrets"
)

type StorageService struct{ core *plugin.WaPlugin }

func NewStorageService(core *plugin.WaPlugin) *StorageService { return &StorageService{core: core} }
func (s *StorageService) GetPluginStorageKeyApi(r StorageKeyRequest) (json.RawMessage, error) {
	if e := s.secretAccess(r.PackageID, r.Key); e != nil {
		return nil, e
	}
	v, e := s.core.GetStorage(r.PackageID, r.Key)
	if e != nil {
		return nil, failure("storage.get", e)
	}
	// Returning a nil interface makes Wails send an empty response, conflating
	// a stored null with an empty string. RawMessage preserves the JSON value.
	data, e := json.Marshal(v)
	return data, failure("storage.get", e)
}
func (s *StorageService) SetPluginStorageKeyApi(r StorageSetRequest) error {
	if e := s.secretAccess(r.PackageID, r.Key); e != nil {
		return e
	}
	if len(r.Value) == 0 {
		return invalid("value 不能为空；清空值请显式传 null")
	}
	var value any
	if err := json.Unmarshal(r.Value, &value); err != nil {
		return invalid("value 必须是有效 JSON")
	}
	return failure("storage.set", s.core.SetStorage(r.PackageID, r.Key, value))
}
func (s *StorageService) DeletePluginStorageKeyApi(r StorageKeyRequest) error {
	if e := s.secretAccess(r.PackageID, r.Key); e != nil {
		return e
	}
	return failure("storage.remove", s.core.RemoveStorage(r.PackageID, r.Key))
}
func (s *StorageService) ClearPluginStorageApi(r PackageRequest) error {
	if e := s.packageAccess(r.PackageID); e != nil {
		return e
	}
	return failure("storage.clear", s.core.ClearStorage(r.PackageID))
}
func (s *StorageService) ListPluginStorageKeysApi(r PackageRequest) ([]string, error) {
	if e := s.packageAccess(r.PackageID); e != nil {
		return nil, e
	}
	v, e := s.core.ListStorageKeys(r.PackageID)
	return v, failure("storage.keys", e)
}
func (s *StorageService) packageAccess(id string) error {
	if e := validatePackage(id); e != nil {
		return e
	}
	if !s.core.IsEnabled(id) {
		return &Error{Code: "PLUGIN_DISABLED", Message: "插件未安装或已停用"}
	}
	return nil
}
func (s *StorageService) secretAccess(id, key string) error {
	if e := validateKey(id, key); e != nil {
		return e
	}
	if !s.core.IsEnabled(id) {
		return &Error{Code: "PLUGIN_DISABLED", Message: "插件未安装或已停用"}
	}
	return nil
}
func (s *StorageService) GetPluginSecretApi(id, key string) (string, error) {
	if e := s.secretAccess(id, key); e != nil {
		return "", e
	}
	v, e := secrets.Get(id, key)
	return v, failure("secret.get", e)
}
func (s *StorageService) SetPluginSecretApi(id, key string, value *string) error {
	if e := s.secretAccess(id, key); e != nil {
		return e
	}
	if value == nil {
		return invalid("value 必须是字符串")
	}
	return failure("secret.set", secrets.Set(id, key, *value))
}
func (s *StorageService) DeletePluginSecretApi(id, key string) error {
	if e := s.secretAccess(id, key); e != nil {
		return e
	}
	return failure("secret.remove", secrets.Remove(id, key))
}
