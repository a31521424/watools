package application

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setupApplicationTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("WATOOLS_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("WATOOLS_CACHE_DIR", filepath.Join(root, "cache"))
	return root
}

func writeTestFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseApplicationWindowsCategories(t *testing.T) {
	root := setupApplicationTest(t)
	plain := writeTestFile(t, filepath.Join(root, "document.txt"))
	directory := filepath.Join(root, "Directory.exe")
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path string
		kind error
	}{
		{filepath.Join(root, "Absent.exe"), ErrApplicationNotFound},
		{plain, ErrNotApplication},
		{directory, ErrNotApplication},
	} {
		command, err := ParseApplication(tc.path)
		if command != nil || !errors.Is(err, tc.kind) {
			t.Fatalf("path=%s command=%v error=%v; want %v", tc.path, command, err, tc.kind)
		}
	}
}

func TestShortcutMetadataFailureCategories(t *testing.T) {
	root := setupApplicationTest(t)
	path := writeTestFile(t, filepath.Join(root, "Shortcut.lnk"))
	_, err := getShortcutInfoWithRunner(path, func(string) (string, error) {
		return "", os.ErrPermission
	})
	if !errors.Is(err, ErrApplicationUnreadable) || !errors.Is(err, os.ErrPermission) || errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("wrong shortcut read classification: %v", err)
	}
	_, err = getShortcutInfoWithRunner(path, func(string) (string, error) {
		return "not JSON", nil
	})
	var syntaxErr *json.SyntaxError
	if !errors.Is(err, ErrInvalidMetadata) || !errors.As(err, &syntaxErr) || errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("wrong shortcut decode classification/cause: %v", err)
	}
	_, err = getShortcutInfoWithRunner(path, func(string) (string, error) { return "", nil })
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("empty shortcut metadata accepted: %v", err)
	}
	_, err = parseShortcutWithReader(path, func(string) (*shortcutInfo, error) {
		return nil, os.ErrPermission
	})
	if !errors.Is(err, ErrApplicationUnreadable) || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unclassified shortcut read failure lost category/cause: %v", err)
	}
	_, err = parseShortcutWithReader(path, func(string) (*shortcutInfo, error) {
		return nil, applicationError(ErrInvalidMetadata, "decode", path, &json.SyntaxError{})
	})
	if !errors.Is(err, ErrInvalidMetadata) || errors.Is(err, ErrApplicationUnreadable) || !errors.As(err, &syntaxErr) {
		t.Fatalf("typed shortcut decode failure changed category/cause: %v", err)
	}
	_, err = parseShortcutWithReader(path, func(string) (*shortcutInfo, error) { return nil, nil })
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("nil shortcut metadata accepted: %v", err)
	}
}

func TestShortcutPreservesShellIdentityWhenOptionalIconMissing(t *testing.T) {
	root := setupApplicationTest(t)
	path := writeTestFile(t, filepath.Join(root, "Shortcut.lnk"))
	command, err := parseShortcutWithReader(path, func(string) (*shortcutInfo, error) {
		return &shortcutInfo{
			TargetPath:   filepath.Join(root, "Target.exe"),
			IconLocation: filepath.Join(root, "MissingIcon.ico"),
			Description:  "Shortcut description",
		}, nil
	})
	if err != nil || command == nil || command.Path != path || command.Name != "Shortcut" || command.IconPath.IsSome() {
		t.Fatalf("command=%+v error=%v", command, err)
	}
}

func TestWindowsScanAndParseReturnPartialSuccess(t *testing.T) {
	root := setupApplicationTest(t)
	good := writeTestFile(t, filepath.Join(root, "Good.exe"))
	badRoot := writeTestFile(t, filepath.Join(root, "not-a-folder"))
	infos, scanErr := scanExecutablesInDirs([]string{root, badRoot, filepath.Join(root, "absent")}, make(map[string]struct{}))
	if len(infos) != 1 || infos[0].Path != good || !errors.Is(scanErr, ErrApplicationUnreadable) {
		t.Fatalf("paths=%v error=%v", infos, scanErr)
	}
	infos = append(infos, AppPathInfo{Path: filepath.Join(root, "Absent.exe")})
	commands, err := applicationsFromPathInfos(infos, scanErr)
	if len(commands) != 1 || commands[0].Path != good || !errors.Is(err, ErrApplicationUnreadable) || !errors.Is(err, ErrApplicationNotFound) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("commands=%v aggregate error=%v", commands, err)
	}
}

func TestRegisteredApplicationOptionalIconFailure(t *testing.T) {
	root := setupApplicationTest(t)
	path := writeTestFile(t, filepath.Join(root, "Registered.exe"))
	command, err := newApplicationFromInfo(AppPathInfo{Path: path, Name: "Registered Name", IconPath: filepath.Join(root, "Absent.ico")})
	if err != nil || command == nil || command.Name != "Registered Name" || command.IconPath.IsSome() {
		t.Fatalf("command=%v error=%v", command, err)
	}
	_, err = newApplicationFromInfo(AppPathInfo{Path: filepath.Join(root, "Absent.exe"), Name: "Absent"})
	if !errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("missing registered target was accepted: %v", err)
	}
}

func TestPathKeyWindows(t *testing.T) {
	if PathKey(`C:\Apps\..\Apps\TEST.exe`) != PathKey(`c:\apps\test.exe`) {
		t.Fatal("Windows identity should clean paths and fold case")
	}
}
