package host

import (
	"errors"
	"strings"
	"testing"

	"watools/internal/app"
)

func TestHotkeyRecoveryFeedbackKeepsEveryCauseAndSafeMessages(t *testing.T) {
	primary := errors.New("synthetic-private-registration-detail")
	configRestore := errors.New("synthetic-private-config-detail")
	registerRestore := errors.New("synthetic-private-native-detail")
	for _, test := range []struct {
		name     string
		recovery error
		message  string
	}{
		{"restored", nil, "旧配置和注册已恢复"},
		{"config restore", configRestore, "恢复也失败"},
		{"native restore", registerRestore, "恢复也失败"},
		{"both restore", errors.Join(configRestore, registerRestore), "恢复也失败"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := &app.HotkeyUpdateError{UpdateErr: primary, RecoveryErr: test.recovery}
			err := hotkeyFailure(original)
			errorCode(t, err, "OPERATION_FAILED")
			if !errors.Is(err, primary) || !errors.Is(err, original) {
				t.Fatal("update failure cause was lost")
			}
			for _, cause := range []error{configRestore, registerRestore} {
				if errors.Is(test.recovery, cause) && !errors.Is(err, cause) {
					t.Fatal("recovery failure cause was lost")
				}
			}
			wire := string(MarshalError(err))
			if !strings.Contains(wire, test.message) || strings.Contains(wire, "synthetic-private") || strings.Contains(err.Error(), "synthetic-private") {
				t.Fatalf("unsafe or misleading hotkey feedback: %s", wire)
			}
		})
	}
	if hotkeyFailure(nil) != nil {
		t.Fatal("successful update returned an error")
	}
}

func TestHotkeyInputValidationDoesNotNeedNativeRegistration(t *testing.T) {
	service := NewAppService(nil, nil, nil)
	for _, request := range []HotkeyUpdateRequest{
		{ID: "unknown", Hotkey: "cmd+Space"},
		{ID: "show-hide-window", Hotkey: "no-such-key"},
		{ID: "show-hide-window", Hotkey: "cmd+Space", Name: strings.Repeat("a", 257)},
	} {
		errorCode(t, service.UpdateHotkeyApi(request), "INVALID_ARGUMENT")
	}
}
