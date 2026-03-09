package utils

import "testing"

func TestValidatePluginPackageID(t *testing.T) {
	t.Parallel()

	validIDs := []string{
		"watools.plugin.demo",
		"watools.plugin.demo-1",
		"watools.plugin.demo_alpha.beta",
	}
	for _, packageID := range validIDs {
		if err := ValidatePluginPackageID(packageID); err != nil {
			t.Fatalf("expected packageID %q to be valid: %v", packageID, err)
		}
	}

	invalidIDs := []string{
		"",
		"watools.plugin",
		"watools.plugin../demo",
		"watools.plugin./demo",
		"watools.plugin.demo/app",
		"../watools.plugin.demo",
		"watools.plugin.Demo",
	}
	for _, packageID := range invalidIDs {
		if err := ValidatePluginPackageID(packageID); err == nil {
			t.Fatalf("expected packageID %q to be invalid", packageID)
		}
	}
}

func TestResolvePathWithinBase(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	resolvedPath, err := ResolvePathWithinBase(baseDir, "plugin/index.html")
	if err != nil {
		t.Fatalf("expected valid path, got error: %v", err)
	}
	if resolvedPath == "" {
		t.Fatal("expected resolved path to be non-empty")
	}

	invalidPaths := []string{
		"",
		".",
		"..",
		"../plugin/index.html",
		"/etc/passwd",
		`..\plugin\index.html`,
	}
	for _, invalidPath := range invalidPaths {
		if _, err := ResolvePathWithinBase(baseDir, invalidPath); err == nil {
			t.Fatalf("expected path %q to be rejected", invalidPath)
		}
	}
}
