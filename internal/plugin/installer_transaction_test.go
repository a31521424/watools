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
func (r *testPluginRepository) DeletePlugin(_ context.Context, id string) error {
	r.deleteCalls++
	if r.deleteErr != nil {
		return r.deleteErr
	}
	for i, state := range r.states {
		if state.PackageID == id {
			r.states = append(r.states[:i], r.states[i+1:]...)
			break
		}
	}
	return nil
}
func makeTestPackage(t *testing.T, version, body string) string {
	t.Helper()
	return makeTestPackageID(t, "watools.plugin.test", version, body)
}
func makeTestPackageID(t *testing.T, id, version, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example.wt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range map[string]string{"manifest.json": fmt.Sprintf(`{"packageId":%q,"name":"Test","version":%q,"entry":"app.js"}`, id, version), "app.js": body, "nested/helper.js": body} {
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

func TestBatchFailureRestoresEarlierUpgradeAndKeepsUserState(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "old")); err != nil {
		t.Fatal(err)
	}
	state := repo.states[0]
	state.Enabled = false
	before, revision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, state.PackageID)
	err := pi.InstallBatch([]string{makeTestPackage(t, "1.1.0", "new")}, func() error { return fmt.Errorf("receipt unavailable") })
	if err == nil {
		t.Fatal("commit error ignored")
	}
	after, afterRevision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, state.PackageID)
	if before != after || revision != afterRevision || repo.states[0] != state || state.Enabled || state.Storage["preserved"] != "yes" {
		t.Fatal("rollback lost old plugin state")
	}
}
func TestBatchNewInstallRollsBackRegistration(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	err := pi.InstallBatch([]string{makeTestPackage(t, "1.0.0", "new")}, func() error { return fmt.Errorf("receipt unavailable") })
	if err == nil || len(repo.states) != 0 {
		t.Fatal("failed batch left registration")
	}
	if _, err := os.Stat(filepath.Join(pi.pluginsDir, "watools.plugin.test/current.json")); !os.IsNotExist(err) {
		t.Fatal("failed batch left active pointer")
	}
}
func TestBatchRejectsDuplicatePackagesBeforeMutation(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	if err := pi.InstallBatch([]string{makeTestPackage(t, "1.0.0", "one"), makeTestPackage(t, "1.1.0", "two")}, nil); err == nil {
		t.Fatal("duplicate package accepted")
	}
	if repo.insertCalls != 0 {
		t.Fatal("invalid batch mutated database")
	}
}
func TestInstallationLockProtectsIndependentInstallerProcesses(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	unlock, err := pi.lockDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "new")); err == nil {
		t.Fatal("locked installation accepted")
	}
	unlock()
	if err := pi.InstallFromWtFile(makeTestPackage(t, "1.0.0", "new")); err != nil {
		t.Fatal(err)
	}
}

func TestLaterPackageFailureRestoresEarlierPackages(t *testing.T) {
	repo := &testPluginRepository{}
	pi := &PluginInstaller{ctx: context.Background(), pluginsDir: t.TempDir(), repository: repo}
	for _, id := range []string{"watools.plugin.first", "watools.plugin.last"} {
		if err := pi.InstallFromWtFile(makeTestPackageID(t, id, "1.0.0", "old")); err != nil {
			t.Fatal(err)
		}
	}
	firstRoot, firstRevision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.first")
	lastRoot, lastRevision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.last")
	if err := pi.InstallBatch([]string{makeTestPackageID(t, "watools.plugin.first", "1.1.0", "new"), makeTestPackageID(t, "watools.plugin.last", "0.9.0", "downgrade")}, nil); err == nil {
		t.Fatal("invalid later package accepted")
	}
	root, revision, _ := models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.first")
	if root != firstRoot || revision != firstRevision {
		t.Fatal("earlier upgrade was not rolled back")
	}
	root, revision, _ = models.ResolveInstalledPlugin(pi.pluginsDir, "watools.plugin.last")
	if root != lastRoot || revision != lastRevision || len(repo.states) != 2 {
		t.Fatal("later failure lost state")
	}
}
