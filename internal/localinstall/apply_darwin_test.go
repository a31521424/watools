package localinstall

import (
	"archive/zip"
	"context"
	"fmt"
	"howett.net/plist"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"watools/config"
	"watools/pkg/db"
	"watools/pkg/models"
)

func signedTestBundle(t *testing.T, version string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "watools.app")
	if err := os.MkdirAll(filepath.Join(root, "Contents/MacOS"), 0700); err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile("/bin/echo")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Contents/MacOS/watools"), binary, 0700); err != nil {
		t.Fatal(err)
	}
	info, err := plist.Marshal(map[string]string{"CFBundleIdentifier": "com.wails.watools", "CFBundleExecutable": "watools", "CFBundleVersion": version}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Contents/Info.plist"), info, 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", root).CombinedOutput()
	if err != nil {
		t.Fatalf("sign: %s %v", output, err)
	}
	return root
}
func preparedTestPlugin(t *testing.T, version string) string {
	t.Helper()
	file, err := os.Create(filepath.Join(t.TempDir(), "plugin.wt"))
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, body := range map[string]string{"manifest.json": fmt.Sprintf(`{"packageId":"watools.plugin.test","name":"Test","version":%q,"entry":"app.js"}`, version), "app.js": "export default {}"} {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}
func TestApplicationAndPluginRollbackOnReceiptFailure(t *testing.T) {
	temp := t.TempDir()
	dataDir := filepath.Join(temp, "data")
	cacheDir := filepath.Join(temp, "cache")
	t.Setenv("WATOOLS_DATA_DIR", dataDir)
	t.Setenv("WATOOLS_CACHE_DIR", cacheDir)
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project, err := os.ReadFile(filepath.Join(source, "wails.json"))
	if err != nil {
		t.Fatal(err)
	}
	config.ParseProject(project)
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	value := Configuration{SourceRoot: source, GoExecutable: goExecutable, ToolPath: os.Getenv("PATH"), ApplicationPath: filepath.Join(temp, "Applications/watools.app"), Version: "old"}
	first := signedTestBundle(t, "1")
	if _, err := Apply(context.Background(), dataDir, ApplyRequest{Configuration: value, Bundle: first, Packages: []string{preparedTestPlugin(t, "1.0.0")}}); err != nil {
		t.Fatal(err)
	}
	defer db.GetWaDB().Close()
	before, err := os.ReadFile(filepath.Join(value.ApplicationPath, "Contents/Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	beforeRoot, beforeRevision, err := models.ResolveInstalledPlugin(filepath.Join(dataDir, "plugins"), "watools.plugin.test")
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(dataDir, "installation.json")
	if err := os.Rename(receipt, receipt+".saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(receipt, 0700); err != nil {
		t.Fatal(err)
	}
	value.Version = "new"
	if _, err := Apply(context.Background(), dataDir, ApplyRequest{Configuration: value, Bundle: signedTestBundle(t, "2"), Packages: []string{preparedTestPlugin(t, "1.1.0")}}); err == nil {
		t.Fatal("receipt failure ignored")
	}
	after, err := os.ReadFile(filepath.Join(value.ApplicationPath, "Contents/Info.plist"))
	if err != nil || string(before) != string(after) {
		t.Fatal("application rollback did not restore old bundle")
	}
	root, revision, err := models.ResolveInstalledPlugin(filepath.Join(dataDir, "plugins"), "watools.plugin.test")
	if err != nil || root != beforeRoot || revision != beforeRevision {
		t.Fatal("plugin rollback did not restore old revision")
	}
	if err := verifyBundle(value.ApplicationPath); err != nil {
		t.Fatal("restored bundle signature invalid", err)
	}
	// Existing unrelated app directories must never be overwritten.
	foreign := filepath.Join(temp, "foreign/watools.app")
	if err := os.MkdirAll(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "sentinel"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	value.ApplicationPath = foreign
	if _, err := Apply(context.Background(), dataDir, ApplyRequest{Configuration: value, Bundle: first}); err == nil {
		t.Fatal("foreign application overwritten")
	}
	content, _ := os.ReadFile(filepath.Join(foreign, "sentinel"))
	if string(content) != "keep" {
		t.Fatal("foreign application damaged")
	}
}
