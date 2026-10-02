package plugin

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"watools/pkg/db"
	"watools/pkg/models"
)

type testPluginRepository struct {
	states                        []*models.PluginState
	insertErr, deleteErr, listErr error
	insertCalls, deleteCalls      int
}

func (r *testPluginRepository) GetPlugins(context.Context) ([]*models.PluginState, error) {
	return r.states, r.listErr
}
func (r *testPluginRepository) InsertPlugin(_ context.Context, p db.InsertPluginParams) error {
	r.insertCalls++
	if r.insertErr != nil {
		return r.insertErr
	}
	r.states = append(r.states, &models.PluginState{PackageID: p.PackageID, Enabled: p.Enabled, Storage: map[string]interface{}{"preserved": "yes"}, UsedCount: 42})
	return nil
}
func (r *testPluginRepository) DeletePlugin(context.Context, string) error {
	r.deleteCalls++
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.states = nil
	return nil
}
func makeTestPackage(t *testing.T, version, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example.wt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range map[string]string{"manifest.json": fmt.Sprintf(`{"packageId":"watools.plugin.test","name":"Test","version":%q,"entry":"app.js"}`, version), "app.js": body, "nested/helper.js": body} {
		w, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestUpgradePreservesStateAndChangesAllAssetURLs(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "old")); err != nil {
		t.Fatal(err)
	}
	oldRoot, oldRevision, err := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	if err != nil {
		t.Fatal(err)
	}
	state := repo.states[0]
	state.Enabled = false
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.1.0", "new")); err != nil {
		t.Fatal(err)
	}
	root, revision, err := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	if err != nil {
		t.Fatal(err)
	}
	if revision == oldRevision || root == oldRoot {
		t.Fatal("upgrade reused cached revision URL")
	}
	if repo.insertCalls != 1 || repo.deleteCalls != 0 || repo.states[0] != state || state.Enabled || state.UsedCount != 42 || state.Storage["preserved"] != "yes" {
		t.Fatal("upgrade overwrote user state")
	}
	for _, asset := range []string{"app.js", "nested/helper.js"} {
		data, err := os.ReadFile(filepath.Join(root, asset))
		if err != nil || string(data) != "new" {
			t.Fatal("new revision incomplete")
		}
		data, err = os.ReadFile(filepath.Join(oldRoot, asset))
		if err != nil || string(data) != "old" {
			t.Fatal("old running revision removed")
		}
	}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "0.9.0", "bad")); err == nil {
		t.Fatal("downgrade accepted")
	}
	current, _, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	if current != root {
		t.Fatal("rejected upgrade changed active revision")
	}
}
func TestFailedUninstallRestoresPlugin(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "old")); err != nil {
		t.Fatal(err)
	}
	root, _, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	repo.deleteErr = fmt.Errorf("database unavailable")
	if err := pi.UninstallPlugin("watools.plugin.test"); err == nil {
		t.Fatal("database error ignored")
	}
	if _, err := os.Stat(filepath.Join(root, "app.js")); err != nil {
		t.Fatal("failed uninstall deleted plugin")
	}
}
func TestFailedRegistrationNeverPublishesRevision(t *testing.T) {
	repo := &testPluginRepository{insertErr: fmt.Errorf("database unavailable")}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "new")); err == nil {
		t.Fatal("registration failure ignored")
	}
	if _, err := os.Stat(filepath.Join(pi.pluginsDir, "watools.plugin.test", "current.json")); !os.IsNotExist(err) {
		t.Fatal("failed installation published revision")
	}
}

func TestDatabaseLookupFailureCannotReplaceInstalledPlugin(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "old")); err != nil {
		t.Fatal(err)
	}
	root, revision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	repo.listErr = fmt.Errorf("read failed")
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.1.0", "new")); err == nil {
		t.Fatal("database read failure ignored")
	}
	after, afterRevision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.test")
	if root != after || revision != afterRevision || repo.insertCalls != 1 || repo.deleteCalls != 0 {
		t.Fatal("failed lookup changed installation")
	}
}
