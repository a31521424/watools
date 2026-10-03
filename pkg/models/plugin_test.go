package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDamagedPluginRemainsManageableAndStorageIsPrivate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", root)
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	state := &PluginState{PackageID: "watools.plugin.test", Enabled: true, UsedCount: 42, Storage: map[string]interface{}{"apiKey": "private"}}
	info := state.GetFullInfo()
	if info.PackageID != state.PackageID || info.LoadError == "" {
		t.Fatal("damaged plugin disappeared")
	}
	if data, _ := json.Marshal(info); strings.Contains(string(data), "private") || strings.Contains(string(data), "storage") {
		t.Fatal("plugin list exposed storage")
	}
	dir := filepath.Join(root, "plugins", state.PackageID)
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"packageId":"watools.plugin.test","name":"Test","version":"1.0.0","entry":"app.js"}`), 0600)
	info = state.GetFullInfo()
	if info.Name != "Test" || info.LoadError != "" || info.UsedCount != 42 {
		t.Fatalf("valid plugin not recovered: %+v", info)
	}
	if data, _ := json.Marshal(info); strings.Contains(string(data), "private") || strings.Contains(string(data), "storage") {
		t.Fatal("valid plugin exposed storage")
	}
}
