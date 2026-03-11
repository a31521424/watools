package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"watools/pkg/models"
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

func TestComparePluginVersions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		next      string
		current   string
		want      int
		expectErr bool
	}{
		{name: "equal version", next: "1.0.0", current: "1.0.0", want: 0},
		{name: "equal version with missing patch", next: "1.0", current: "1.0.0", want: 0},
		{name: "higher patch", next: "1.0.1", current: "1.0.0", want: 1},
		{name: "lower minor", next: "1.1.0", current: "1.2.0", want: -1},
		{name: "release higher than prerelease", next: "1.0.0", current: "1.0.0-beta.1", want: 1},
		{name: "prerelease ordering", next: "1.0.0-beta.2", current: "1.0.0-beta.1", want: 1},
		{name: "invalid version", next: "abc", current: "1.0.0", expectErr: true},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := comparePluginVersions(testCase.next, testCase.current)
			if testCase.expectErr {
				if err == nil {
					t.Fatal("expected comparePluginVersions to return an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("comparePluginVersions returned error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("comparePluginVersions(%q, %q) = %d, want %d", testCase.next, testCase.current, got, testCase.want)
			}
		})
	}
}

func TestValidateManifestRequiresParsableVersion(t *testing.T) {
	t.Parallel()

	installer := &PluginInstaller{}
	manifest := &models.PluginMetadata{
		PackageID: "watools.plugin.example",
		Name:      "Example",
		Version:   "invalid-version",
		Entry:     "app.js",
	}

	if err := installer.validateManifest(manifest); err == nil {
		t.Fatal("expected validateManifest to reject an invalid version")
	}
}
