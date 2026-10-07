package application

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupApplicationTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("WATOOLS_CACHE_DIR", filepath.Join(root, "cache"))
	// Keep optional native metadata helpers deterministic and isolated.
	t.Setenv("PATH", root)
	return root
}

func writeTestBundle(t *testing.T, root, name, metadata string) string {
	t.Helper()
	appPath := filepath.Join(root, name+".app")
	if err := os.MkdirAll(filepath.Join(appPath, "Contents", "Resources"), 0755); err != nil {
		t.Fatal(err)
	}
	if metadata != "" {
		if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), []byte(metadata), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return appPath
}

func TestParseApplicationFailureCategories(t *testing.T) {
	root := setupApplicationTest(t)
	missingMetadata := writeTestBundle(t, root, "MissingMetadata", "")
	broken := writeTestBundle(t, root, "Broken", "not a property list")
	plainDir := filepath.Join(root, "folder")
	if err := os.Mkdir(plainDir, 0755); err != nil {
		t.Fatal(err)
	}
	plainFile := filepath.Join(root, "file.app")
	if err := os.WriteFile(plainFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path  string
		kind, cause error
	}{
		{"absent root", filepath.Join(root, "Absent.app"), ErrApplicationNotFound, os.ErrNotExist},
		{"missing metadata", missingMetadata, ErrInvalidMetadata, os.ErrNotExist},
		{"corrupt metadata", broken, ErrInvalidMetadata, nil},
		{"non app directory", plainDir, ErrNotApplication, nil},
		{"non app file", plainFile, ErrNotApplication, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command, err := ParseApplication(tc.path)
			if command != nil || !errors.Is(err, tc.kind) {
				t.Fatalf("command=%v error=%v; want %v", command, err, tc.kind)
			}
			if tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("error lost cause %v: %v", tc.cause, err)
			}
			if tc.kind != ErrApplicationNotFound && errors.Is(err, ErrApplicationNotFound) {
				t.Fatalf("existing root classified as absent: %v", err)
			}
		})
	}
}

func TestParseApplicationReadFailurePreservesCause(t *testing.T) {
	root := setupApplicationTest(t)
	appPath := writeTestBundle(t, root, "ReadFailure", "<plist><dict/></plist>")
	for _, cause := range []error{os.ErrPermission, io.ErrUnexpectedEOF} {
		command, err := parseAppBundleWithReader(appPath, func(path string) ([]byte, error) {
			return nil, &os.PathError{Op: "read", Path: path, Err: cause}
		})
		if command != nil || !errors.Is(err, ErrApplicationUnreadable) || !errors.Is(err, cause) {
			t.Fatalf("command=%v error=%v; want unreadable and %v", command, err, cause)
		}
		var pathErr *os.PathError
		if !errors.As(err, &pathErr) || !strings.HasSuffix(pathErr.Path, "Contents/Info.plist") {
			t.Fatalf("lost metadata read stage: %v", err)
		}
	}
}

func TestParseApplicationUnreadableRoot(t *testing.T) {
	root := setupApplicationTest(t)
	appPath := filepath.Join(root, "Loop.app")
	if err := os.Symlink(appPath, appPath); err != nil {
		t.Fatal(err)
	}
	command, err := ParseApplication(appPath)
	var pathErr *os.PathError
	if command != nil || !errors.Is(err, ErrApplicationUnreadable) || !errors.As(err, &pathErr) || errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("root stat failure was not retained: command=%v error=%v", command, err)
	}
}

func TestParseApplicationOptionalMetadataFailures(t *testing.T) {
	root := setupApplicationTest(t)
	for _, tool := range []string{"mdls", "iconutil"} {
		if err := os.WriteFile(filepath.Join(root, tool), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	appPath := writeTestBundle(t, root, "Fallback", `<plist><dict>
<key>CFBundleDisplayName</key><string>Plist Name</string>
<key>CFBundleIconFile</key><string>Missing</string>
<key>CFBundleIconName</key><string>AppIcon</string>
</dict></plist>`)
	if err := os.WriteFile(filepath.Join(appPath, "Contents", "Resources", "Assets.car"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	command, err := ParseApplication(appPath)
	if err != nil || command == nil || command.Name != "Plist Name" || command.IconPath.IsSome() {
		t.Fatalf("optional failure did not degrade: command=%+v error=%v", command, err)
	}
	// A cache path that cannot be a directory must also remain optional.
	blockedCache := filepath.Join(root, "blocked-cache")
	if err := os.WriteFile(blockedCache, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WATOOLS_CACHE_DIR", blockedCache)
	command, err = ParseApplication(appPath)
	if err != nil || command == nil || command.IconPath.IsSome() {
		t.Fatalf("icon cache failure rejected application: command=%v error=%v", command, err)
	}
}

func TestScanReturnsPartialApplicationsAndErrors(t *testing.T) {
	root := setupApplicationTest(t)
	good := writeTestBundle(t, root, "Good", "<plist><dict/></plist>")
	missing := writeTestBundle(t, root, "Missing", "")
	broken := writeTestBundle(t, root, "Broken", "corrupt")
	unreadableRoot := filepath.Join(root, "unreadable")
	infos, scanErr := scanMacApplicationPaths([]string{root, unreadableRoot, root}, func(path string) ([]os.DirEntry, error) {
		if path == unreadableRoot {
			return nil, &os.PathError{Op: "readdir", Path: path, Err: os.ErrPermission}
		}
		return os.ReadDir(path)
	})
	if len(infos) != 3 || !errors.Is(scanErr, ErrApplicationUnreadable) || !errors.Is(scanErr, os.ErrPermission) {
		t.Fatalf("scan paths=%v error=%v", infos, scanErr)
	}
	paths := make(map[string]bool)
	for _, info := range infos {
		paths[info.Path] = true
	}
	if !paths[good] || !paths[missing] || !paths[broken] {
		t.Fatalf("scanner discarded invalid metadata candidates: %v", infos)
	}
	commands, err := applicationsFromPathInfos(infos, scanErr)
	if len(commands) != 1 || commands[0].Path != good || commands[0].Name != "Good" {
		t.Fatalf("lost successful application: %v", commands)
	}
	if !errors.Is(err, ErrInvalidMetadata) || !errors.Is(err, os.ErrPermission) || !errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("wrong aggregate categories/causes: %v", err)
	}
}

func TestScanNonApplicationAndAbsentOptionalFolder(t *testing.T) {
	root := setupApplicationTest(t)
	if err := os.WriteFile(filepath.Join(root, "File.app"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	infos, err := scanMacApplicationPaths([]string{root, filepath.Join(root, "absent")}, os.ReadDir)
	if len(infos) != 0 || !errors.Is(err, ErrNotApplication) || errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("paths=%v error=%v", infos, err)
	}
}

func TestPathKeyDarwin(t *testing.T) {
	if PathKey("/Applications/../Applications/Test.app") != "/Applications/Test.app" || PathKey("/Applications/TEST.app") == PathKey("/Applications/test.app") {
		t.Fatal("Darwin identity should clean paths and preserve case")
	}
}
