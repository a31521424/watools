package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLocalCredentials(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("WATOOLS_CACHE_DIR", filepath.Join(root, "cache"))
	const packageID = "watools.plugin.test"
	if value, err := Get(packageID, "apiKey"); err != nil || value != "" {
		t.Fatalf("missing key: value=%q error=%v", value, err)
	}
	for _, value := range []string{"first", "替换后的 key", ""} {
		if err := Set(packageID, "apiKey", value); err != nil {
			t.Fatal(err)
		}
		if got, err := Get(packageID, "apiKey"); err != nil || got != value {
			t.Fatalf("round trip: value=%q error=%v", got, err)
		}
	}
	name, _ := account(packageID, "apiKey")
	info, err := os.Stat(secretPath(name))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("expected owner-only file: info=%v error=%v", info, err)
	}
	dir, err := os.Stat(filepath.Dir(secretPath(name)))
	if err != nil || dir.Mode().Perm() != 0700 {
		t.Fatalf("expected owner-only directory: info=%v error=%v", dir, err)
	}
	if err := Set(packageID, "apiKey", "private-value"); err != nil {
		t.Fatal(err)
	}
	if got, err := Get("watools.plugin.other", "apiKey"); err != nil || got != "" {
		t.Fatalf("namespaces should stay separate: value=%q error=%v", got, err)
	}
	if err := Set(packageID, "apiKey", strings.Repeat("x", (64<<10)+1)); err == nil {
		t.Fatal("oversized value accepted")
	}
	if got, err := Get(packageID, "apiKey"); err != nil || got != "private-value" {
		t.Fatal("rejected update changed the stored value")
	}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := Set(packageID, "apiKey", "concurrent-value"); err != nil {
				t.Error(err)
			}
			if _, err := Get(packageID, "apiKey"); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	files, err := os.ReadDir(filepath.Dir(secretPath(name)))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files left behind: files=%v error=%v", files, err)
	}
	for i := 0; i < 2; i++ {
		if err := Remove(packageID, "apiKey"); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := Get(packageID, "apiKey"); err != nil || got != "" {
		t.Fatal("removed value still present")
	}
	if err := os.WriteFile(secretPath(name), make([]byte, (64<<10)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(packageID, "apiKey"); err == nil {
		t.Fatal("oversized local file accepted")
	}
}
