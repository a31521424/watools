package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDamagedPluginRemainsManageableAndStorageIsPrivate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", root)
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	state := &PluginState{PackageID: "watools.plugin.test", Enabled: true, UsedCount: 42, Storage: map[string]interface{}{"apiKey": "private"}}
	info := state.GetFullInfo()
	if info == nil || info["packageId"] != state.PackageID || info["loadError"] == nil {
		t.Fatal("damaged plugin disappeared")
	}
	if _, exists := info["storage"]; exists {
		t.Fatal("plugin list exposed storage")
	}
	dir := filepath.Join(root, "plugins", state.PackageID)
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"packageId":"watools.plugin.test","name":"Test","version":"1.0.0","entry":"app.js"}`), 0600)
	info = state.GetFullInfo()
	if info["name"] != "Test" || info["loadError"] != nil || info["usedCount"] != float64(42) {
		t.Fatalf("valid plugin not recovered: %+v", info)
	}
	if _, exists := info["storage"]; exists {
		t.Fatal("valid plugin exposed storage")
	}
}
