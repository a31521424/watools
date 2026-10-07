package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.design/x/hotkey"
)

type recoveryNativeHotkey struct {
	keydown    chan hotkey.Event
	registered bool
}

func (h *recoveryNativeHotkey) Register() error              { h.registered = true; return nil }
func (h *recoveryNativeHotkey) Unregister() error            { h.registered = false; return nil }
func (h *recoveryNativeHotkey) Keydown() <-chan hotkey.Event { return h.keydown }
func (h *recoveryNativeHotkey) IsRegistered() bool           { return h.registered }

func recoveryTestManager(t *testing.T) *HotkeyManager {
	t.Helper()
	manager := &HotkeyManager{
		listeners: make(map[string]*HotkeyListener),
		configs: map[string]HotkeyConfig{
			"show-hide-window": {ID: "show-hide-window", Name: "Launcher", Hotkey: "ctrl+a"},
		},
		configDir: t.TempDir(),
		newHotkey: func([]hotkey.Modifier, hotkey.Key) registeredHotkey {
			return &recoveryNativeHotkey{keydown: make(chan hotkey.Event)}
		},
	}
	t.Cleanup(manager.UnregisterAll)
	return manager
}

func TestHotkeyRecoveryPreservesUnregisteredBaseline(t *testing.T) {
	manager := recoveryTestManager(t)
	saveErr := errors.New("disk unavailable")
	manager.createConfigTemp = func(string, string) (hotkeyConfigFile, error) { return nil, saveErr }
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Hotkey: "ctrl+b"}, manager)
	var updateErr *HotkeyUpdateError
	if !errors.As(err, &updateErr) || updateErr.RecoveryErr != nil || !errors.Is(err, saveErr) {
		t.Fatalf("expected recovered save failure, got %v", err)
	}
	states := manager.GetHotkeyStates()
	if len(states) != 1 || states[0].Hotkey != "ctrl+a" || states[0].Registered || states[0].RegisteredHotkey != "" {
		t.Fatalf("unregistered baseline was changed: %+v", states)
	}
}

func TestHotkeyRecoveryPreservesPartialRegistrationBaseline(t *testing.T) {
	manager := recoveryTestManager(t)
	manager.configs["unknown"] = HotkeyConfig{ID: "unknown", Hotkey: "ctrl+c"}
	if err := manager.RegisterAll(); err == nil {
		t.Fatal("unknown hotkey should make the initial registration partial")
	}
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Hotkey: "ctrl+b"}, manager)
	var updateErr *HotkeyUpdateError
	if !errors.As(err, &updateErr) || updateErr.RecoveryErr != nil {
		t.Fatalf("expected partial baseline restored, got %v", err)
	}
	for _, state := range manager.GetHotkeyStates() {
		if state.ID == "show-hide-window" && (!state.Registered || state.RegisteredHotkey != "ctrl+a" || state.Hotkey != "ctrl+a") {
			t.Fatalf("old active hotkey was not restored: %+v", state)
		}
		if state.ID == "unknown" && (state.Registered || state.RegisteredHotkey != "") {
			t.Fatalf("previously inactive hotkey was registered: %+v", state)
		}
	}
}

func TestHotkeySnapshotUsesActualNativeState(t *testing.T) {
	manager := recoveryTestManager(t)
	if err := manager.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetConfig(HotkeyConfig{ID: "show-hide-window", Hotkey: "ctrl+b"}); err != nil {
		t.Fatal(err)
	}
	states := manager.GetHotkeyStates()
	if !states[0].Registered || states[0].Hotkey != "ctrl+b" || states[0].RegisteredHotkey != "ctrl+a" {
		t.Fatalf("snapshot inferred registration from new config: %+v", states)
	}
	// A native key can be released while its handler remains owned for cleanup.
	manager.listeners["show-hide-window"].hk.(*recoveryNativeHotkey).registered = false
	states = manager.GetHotkeyStates()
	if states[0].Registered || states[0].RegisteredHotkey != "" {
		t.Fatalf("residual resources were reported as a registered key: %+v", states)
	}
}

type recoveryFaultFile struct {
	*os.File
	writeErr error
	closeErr error
	short    bool
}

func (f *recoveryFaultFile) Write(data []byte) (int, error) {
	if f.short {
		return len(data) - 1, nil
	}
	return 0, f.writeErr
}

func (f *recoveryFaultFile) Close() error {
	return errors.Join(f.File.Close(), f.closeErr)
}

func TestHotkeySaveKeepsWriteAndCloseFailuresAndPreviousFile(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(map[bool]string{false: "write failure", true: "short write"}[short], func(t *testing.T) {
			manager := recoveryTestManager(t)
			if err := manager.SaveConfigs(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(manager.configDir, "config.json")
			old, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeErr, closeErr := errors.New("write failed"), errors.New("close failed")
			wantWriteErr := writeErr
			if short {
				wantWriteErr = io.ErrShortWrite
			}
			var temporary string
			manager.createConfigTemp = func(dir, pattern string) (hotkeyConfigFile, error) {
				file, err := os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				temporary = file.Name()
				return &recoveryFaultFile{File: file, writeErr: writeErr, closeErr: closeErr, short: short}, nil
			}
			if err := manager.SetConfig(HotkeyConfig{ID: "show-hide-window", Hotkey: "ctrl+b"}); err != nil {
				t.Fatal(err)
			}
			err = manager.SaveConfigs()
			if !errors.Is(err, wantWriteErr) || !errors.Is(err, closeErr) {
				t.Fatalf("required failure causes missing: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != string(old) {
				t.Fatalf("failed save changed previous file: %q, %v", data, err)
			}
			if _, err := os.Stat(temporary); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("temporary file was not cleaned: %v", err)
			}
		})
	}
}
