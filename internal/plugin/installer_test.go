package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindManifestPathRejectsMultipleManifests(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	paths := []string{
		filepath.Join(rootDir, "manifest.json"),
		filepath.Join(rootDir, "nested", "manifest.json"),
	}
	for _, manifestPath := range paths {
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
			t.Fatalf("failed to create manifest directory: %v", err)
		}
		if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create manifest file: %v", err)
		}
	}

	installer := &PluginInstaller{}
	if _, err := installer.findManifestPath(rootDir); err == nil {
		t.Fatal("expected multiple manifests to be rejected")
	}
}

func TestResolvePluginFileRequiresExistingFile(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "app.js"), []byte("export default [];"), 0644); err != nil {
		t.Fatalf("failed to write app.js: %v", err)
	}

	installer := &PluginInstaller{}
	if _, err := installer.resolvePluginFile(rootDir, "app.js"); err != nil {
		t.Fatalf("expected app.js to resolve successfully: %v", err)
	}
	if _, err := installer.resolvePluginFile(rootDir, "../outside.js"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := installer.resolvePluginFile(rootDir, "missing.js"); err == nil {
		t.Fatal("expected missing file to be rejected")
	}
}
