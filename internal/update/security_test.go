package update

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validTestAsset() ManifestAsset {
	osName, arch, kind, name := "darwin", "universal", "app-zip", "watools.zip"
	if runtime.GOOS == "windows" {
		osName, arch, kind, name = "windows", "amd64", "nsis", "watools.exe"
	}
	return ManifestAsset{OS: osName, Arch: arch, InstallerType: kind, AssetName: name, Size: 3, SHA256: strings.Repeat("a", 64), URL: "https://github.com/" + repositorySlug + "/releases/download/v1/" + name}
}
func TestManifestAssetValidation(t *testing.T) {
	base := validTestAsset()
	if err := validateAsset(base); err != nil {
		t.Fatal(err)
	}
	for _, modify := range []func(*ManifestAsset){func(a *ManifestAsset) { a.AssetName = "../escape.zip" }, func(a *ManifestAsset) { a.SHA256 = "" }, func(a *ManifestAsset) { a.URL = "https://example.com/file.zip" }, func(a *ManifestAsset) { a.Size = 513 << 20 }, func(a *ManifestAsset) { a.InstallerType = "unexpected" }} {
		a := base
		modify(&a)
		if err := validateAsset(a); err == nil {
			t.Fatalf("unsafe asset accepted: %+v", a)
		}
	}
}
func TestCachedAssetRequiresDigestAndRejectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.zip")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyExistingAsset(path, ""); err == nil {
		t.Fatal("missing checksum accepted")
	}
	sum := sha256.Sum256([]byte("old"))
	digest := hex.EncodeToString(sum[:])
	if _, err := verifyExistingAsset(path, digest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyExistingAsset(path, digest); err == nil {
		t.Fatal("modified package accepted")
	}
}
func TestUpdateArchiveRejectsTraversalAndSymlinks(t *testing.T) {
	for _, name := range []string{"../escape", "link"} {
		dir := t.TempDir()
		path := filepath.Join(dir, "update.zip")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		z := zip.NewWriter(f)
		header := &zip.FileHeader{Name: name}
		if name == "link" {
			header.SetMode(os.ModeSymlink | 0777)
		}
		w, err := z.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("outside"))
		z.Close()
		f.Close()
		if err := extractUpdateArchive(path, filepath.Join(dir, "extract")); err == nil {
			t.Fatal("unsafe archive accepted")
		}
	}
}

func TestInstallRechecksRememberedPackage(t *testing.T) {
	dir := t.TempDir()
	asset := validTestAsset()
	path := filepath.Join(dir, asset.AssetName)
	os.WriteFile(path, []byte("old"), 0600)
	sum := sha256.Sum256([]byte("old"))
	asset.SHA256 = hex.EncodeToString(sum[:])
	s := &Service{}
	s.remember(path, asset)
	os.WriteFile(path, []byte("new"), 0600)
	if _, err := s.Install(path); err == nil {
		t.Fatal("remembered package installed after tampering")
	}
	os.WriteFile(path, []byte("old"), 0600)
	asset.Size = 4
	if _, err := verifyReleaseAsset(path, asset); err == nil {
		t.Fatal("cached size mismatch ignored")
	}
}
