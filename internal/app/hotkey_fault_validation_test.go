package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.design/x/hotkey"
)

type faultValidationHotkey struct {
	registerErr        error
	unregisterErr      error
	registered         bool
	events             chan hotkey.Event
	resourcesOnFailure bool
	unregisterCalls    int
}

func (h *faultValidationHotkey) Register() error {
	if h.registerErr != nil {
		h.registered = h.resourcesOnFailure
		return h.registerErr
	}
	h.registered = true
	return nil
}
func (h *faultValidationHotkey) Unregister() error {
	h.unregisterCalls++
	if h.unregisterErr != nil {
		return h.unregisterErr
	}
	h.registered = false
	return nil
}
func (h *faultValidationHotkey) IsRegistered() bool           { return h.registered }
func (h *faultValidationHotkey) Keydown() <-chan hotkey.Event { return h.events }

type faultValidationConfigFile struct {
	*os.File
	writeErr, closeErr error
}

func (f *faultValidationConfigFile) Write([]byte) (int, error) { return 0, f.writeErr }
func (f *faultValidationConfigFile) Close() error              { return errors.Join(f.File.Close(), f.closeErr) }

func faultValidationManager(t *testing.T, attempts []*faultValidationHotkey) *HotkeyManager {
	t.Helper()
	hm := &HotkeyManager{
		configs:   map[string]HotkeyConfig{"show-hide-window": {ID: "show-hide-window", Name: "original", Hotkey: "ctrl+a"}},
		listeners: map[string]*HotkeyListener{},
		configDir: t.TempDir(),
	}
	i := 0
	hm.newHotkey = func([]hotkey.Modifier, hotkey.Key) registeredHotkey {
		if i >= len(attempts) {
			t.Fatalf("unexpected registration attempt %d", i+1)
		}
		h := attempts[i]
		i++
		h.events = make(chan hotkey.Event)
		return h
	}
	if err := hm.SaveConfigs(); err != nil {
		t.Fatal(err)
	}
	if err := hm.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, h := range attempts {
			h.unregisterErr = nil
		}
		hm.UnregisterAll()
	})
	return hm
}

func TestFaultValidationHotkeyDoubleRegistrationFailure(t *testing.T) {
	updateCause, recoveryCause := errors.New("new key conflict"), errors.New("old key conflict")
	hm := faultValidationManager(t, []*faultValidationHotkey{{}, {registerErr: updateCause}, {registerErr: recoveryCause}})
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Name: "replacement", Hotkey: "ctrl+b"}, hm)
	var outcome *HotkeyUpdateError
	if !errors.As(err, &outcome) || outcome.RecoveryErr == nil {
		t.Fatalf("missing incomplete recovery outcome: %v", err)
	}
	if !errors.Is(err, updateCause) || !errors.Is(err, recoveryCause) {
		t.Fatalf("lost registration cause: %v", err)
	}
	states := hm.GetHotkeyStates()
	if len(states) != 1 || states[0].Hotkey != "ctrl+a" || states[0].Registered || states[0].RegisteredHotkey != "" {
		t.Fatalf("desired configuration hid failed registration: %+v", states)
	}
}

func TestFaultValidationHotkeyWriteAndCloseFailureRestoresActualState(t *testing.T) {
	writeCause, closeCause := errors.New("write fault"), errors.New("close fault")
	hm := faultValidationManager(t, []*faultValidationHotkey{{}, {}, {}})
	hm.createConfigTemp = func(dir, pattern string) (hotkeyConfigFile, error) {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return &faultValidationConfigFile{File: file, writeErr: writeCause, closeErr: closeCause}, nil
	}
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Name: "replacement", Hotkey: "ctrl+b"}, hm)
	var outcome *HotkeyUpdateError
	if !errors.As(err, &outcome) || outcome.RecoveryErr != nil {
		t.Fatalf("complete restoration not reported: %v", err)
	}
	if !errors.Is(err, writeCause) || !errors.Is(err, closeCause) {
		t.Fatalf("lost write/close cause: %v", err)
	}
	states := hm.GetHotkeyStates()
	if len(states) != 1 || states[0].Hotkey != "ctrl+a" || !states[0].Registered || states[0].RegisteredHotkey != "ctrl+a" {
		t.Fatalf("actual state not restored: %+v", states)
	}
	data, err := os.ReadFile(filepath.Join(hm.configDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored []HotkeyConfig
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Hotkey != "ctrl+a" {
		t.Fatalf("failed save replaced persisted config: %+v", stored)
	}
}

func TestFaultValidationHotkeyFailedUnregisterRetainsActualOldRegistration(t *testing.T) {
	cause := errors.New("native unregister fault")
	hm := faultValidationManager(t, []*faultValidationHotkey{{unregisterErr: cause}})
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Name: "replacement", Hotkey: "ctrl+b"}, hm)
	if !errors.Is(err, cause) {
		t.Fatalf("lost unregister failure: %v", err)
	}
	states := hm.GetHotkeyStates()
	if len(states) != 1 || !states[0].Registered || states[0].RegisteredHotkey != "ctrl+a" || states[0].Hotkey != "ctrl+a" {
		t.Fatalf("failed unregister falsified state: %+v", states)
	}
}

type faultValidationRestoreConfig struct {
	*HotkeyManager
	sets       int
	restoreErr error
}

func (m *faultValidationRestoreConfig) SetConfig(cfg HotkeyConfig) error {
	m.sets++
	if m.sets > 1 {
		return m.restoreErr
	}
	return m.HotkeyManager.SetConfig(cfg)
}

func TestFaultValidationHotkeyAllRecoveryCausesRemainInspectable(t *testing.T) {
	updateCause := errors.New("new registration fault")
	configCause := errors.New("configuration restoration fault")
	registrationCause := errors.New("registration restoration fault")
	hm := faultValidationManager(t, []*faultValidationHotkey{{}, {registerErr: updateCause}, {registerErr: registrationCause}})
	manager := &faultValidationRestoreConfig{HotkeyManager: hm, restoreErr: configCause}
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Hotkey: "ctrl+b"}, manager)
	var outcome *HotkeyUpdateError
	if !errors.As(err, &outcome) || outcome.RecoveryErr == nil {
		t.Fatalf("missing recovery outcome: %v", err)
	}
	for _, cause := range []error{updateCause, configCause, registrationCause} {
		if !errors.Is(err, cause) {
			t.Fatalf("cause %v missing from %v", cause, err)
		}
	}
	for _, cause := range []error{configCause, registrationCause} {
		if !errors.Is(outcome.RecoveryErr, cause) {
			t.Fatalf("recovery cause %v missing from %v", cause, outcome.RecoveryErr)
		}
	}
	states := hm.GetHotkeyStates()
	if len(states) != 1 || states[0].Hotkey != "ctrl+b" || states[0].Registered || states[0].RegisteredHotkey != "" {
		t.Fatalf("failed config recovery reported old desired state: %+v", states)
	}
}

func TestFaultValidationFailedRegistrationResidualResourcesAreRecoveredOrReported(t *testing.T) {
	for _, cleanupFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "recovered", true: "recovery_failed"}[cleanupFails], func(t *testing.T) {
			updateCause, recoveryCause := errors.New("native registration fault"), errors.New("residual native cleanup fault")
			residual := &faultValidationHotkey{registerErr: updateCause, resourcesOnFailure: true}
			attempts := []*faultValidationHotkey{{}, residual}
			if cleanupFails {
				residual.unregisterErr = recoveryCause
			} else {
				attempts = append(attempts, &faultValidationHotkey{})
			}
			hm := faultValidationManager(t, attempts)
			err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Hotkey: "ctrl+b"}, hm)
			var outcome *HotkeyUpdateError
			if !errors.As(err, &outcome) || !errors.Is(err, updateCause) {
				t.Fatalf("lost update outcome: %v", err)
			}
			if residual.unregisterCalls != 1 {
				t.Fatalf("residual registration cleanup calls = %d", residual.unregisterCalls)
			}
			states := hm.GetHotkeyStates()
			if len(states) != 1 || !states[0].Registered || states[0].Hotkey != "ctrl+a" {
				t.Fatalf("incorrect desired/actual state: %+v", states)
			}
			if cleanupFails {
				if !errors.Is(outcome.RecoveryErr, recoveryCause) || states[0].RegisteredHotkey != "ctrl+b" {
					t.Fatalf("residual new registration concealed: %+v, %v", states, err)
				}
			} else if outcome.RecoveryErr != nil || states[0].RegisteredHotkey != "ctrl+a" {
				t.Fatalf("restored registration not reported: %+v, %v", states, err)
			}
		})
	}
}

func TestFaultValidationConfigurationRecoveryFailureKeepsOldNativeRegistration(t *testing.T) {
	saveCause, configCause := errors.New("temporary configuration creation fault"), errors.New("configuration restoration fault")
	hm := faultValidationManager(t, []*faultValidationHotkey{{}, {}, {}})
	hm.createConfigTemp = func(string, string) (hotkeyConfigFile, error) { return nil, saveCause }
	manager := &faultValidationRestoreConfig{HotkeyManager: hm, restoreErr: configCause}
	err := (&WaApp{}).updateHotkeyWithManager(HotkeyConfigAPI{ID: "show-hide-window", Hotkey: "ctrl+b"}, manager)
	var outcome *HotkeyUpdateError
	if !errors.As(err, &outcome) || !errors.Is(outcome.UpdateErr, saveCause) || !errors.Is(outcome.RecoveryErr, configCause) {
		t.Fatalf("lost incomplete recovery causes: %v", err)
	}
	states := hm.GetHotkeyStates()
	if len(states) != 1 || states[0].Hotkey != "ctrl+b" || !states[0].Registered || states[0].RegisteredHotkey != "ctrl+a" {
		t.Fatalf("desired config replaced captured native recovery state: %+v", states)
	}
}
