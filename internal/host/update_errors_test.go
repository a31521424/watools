package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"watools/internal/app"
	"watools/internal/localinstall"
)

type failingInstallation struct{ err error }

func (f failingInstallation) Status() (localinstall.Status, error) {
	return localinstall.Status{}, f.err
}
func (f failingInstallation) Configure(localinstall.Configuration) error        { return f.err }
func (f failingInstallation) Start(context.Context, localinstall.Request) error { return f.err }
func (f failingInstallation) Cancel() error                                     { return f.err }
func (f failingInstallation) Shutdown() error                                   { return f.err }

func TestLocalInstallationFailureMappingKeepsCauseAndSafeMessages(t *testing.T) {
	t.Setenv("WATOOLS_DATA_DIR", t.TempDir())
	t.Setenv("WATOOLS_CACHE_DIR", t.TempDir())
	secret := errors.New("synthetic credential in process or configuration path")
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{"validation", &localinstall.ValidationError{Message: "未知安装范围"}, "INVALID_ARGUMENT"},
		{"busy", fmt.Errorf("job: %w", localinstall.ErrBusy), "UNAVAILABLE"},
		{"read or process", fmt.Errorf("read configuration: %w", secret), "OPERATION_FAILED"},
		{"cancelled", fmt.Errorf("start: %w", context.Canceled), "CANCELLED"},
		{"timeout", fmt.Errorf("start: %w", context.DeadlineExceeded), "TIMEOUT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &UpdateService{local: failingInstallation{test.err}, ctx: context.Background()}
			for _, call := range []struct {
				operation string
				fn        func() error
			}{
				{"installation.configure", func() error { return service.ConfigureLocalInstallationApi(localinstall.Configuration{}) }},
				{"installation.start", func() error { return service.StartLocalInstallationApi(localinstall.Request{}) }},
				{"installation.cancel", service.CancelLocalInstallationApi},
				{"installation.shutdown", service.ServiceShutdown},
			} {
				err := call.fn()
				errorCode(t, err, test.code)
				var outer *Error
				if !errors.Is(err, test.err) || !errors.As(err, &outer) || outer.Operation != call.operation {
					t.Fatalf("lost cause or operation: %v", err)
				}
				if strings.Contains(string(MarshalError(err)), "credential") || strings.Contains(err.Error(), "credential") {
					t.Fatal("installation boundary leaked underlying diagnostics")
				}
			}
		})
	}
}

func TestLocalInstallationRealValidationAndIOClassification(t *testing.T) {
	manager := localinstall.NewManager(t.TempDir(), t.TempDir())
	service := &UpdateService{local: manager, ctx: context.Background()}
	errorCode(t, service.StartLocalInstallationApi(localinstall.Request{Scope: "unknown"}), "INVALID_ARGUMENT")
	errorCode(t, service.ConfigureLocalInstallationApi(localinstall.Configuration{SourceRoot: "relative"}), "INVALID_ARGUMENT")
	dataFile := filepath.Join(t.TempDir(), "synthetic-credential-data")
	if err := os.WriteFile(dataFile, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	service.local = localinstall.NewManager(dataFile, t.TempDir())
	err := service.StartLocalInstallationApi(localinstall.Request{Scope: "all"})
	errorCode(t, err, "OPERATION_FAILED")
	var pathError *os.PathError
	if !errors.As(err, &pathError) || strings.Contains(string(MarshalError(err)), "synthetic-credential") {
		t.Fatalf("configuration I/O cause lost or exposed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service.ctx = ctx
	errorCode(t, service.StartLocalInstallationApi(localinstall.Request{Scope: "all"}), "CANCELLED")
}

func TestExpiredClipboardSnapshotPreservesSentinel(t *testing.T) {
	service := NewAppService(&app.WaApp{}, nil, nil)
	id := strings.Repeat("a", 32)
	_, err := service.GetClipboardSnapshotTextApi(id)
	for _, err := range []error{err, service.RetainClipboardSnapshotApi(id)} {
		errorCode(t, err, "UNAVAILABLE")
		if !errors.Is(err, app.ErrClipboardSnapshotExpired) {
			t.Fatalf("clipboard snapshot sentinel lost: %v", err)
		}
	}
}
