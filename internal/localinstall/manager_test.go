package localinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func managerFixture(t *testing.T, program string) (*Manager, Configuration) {
	t.Helper()
	if os.PathSeparator != '/' {
		t.Skip("POSIX process fixture")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	goExecutable := filepath.Join(temp, "tools with spaces", "go")
	if err := os.MkdirAll(filepath.Dir(goExecutable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goExecutable, []byte("#!/bin/sh\nif [ \"$1\" = build ]; then cp \"$0\" \"$3\"; chmod 700 \"$3\"; exit 0; fi\n"+program), 0700); err != nil {
		t.Fatal(err)
	}
	value := Configuration{SourceRoot: root, GoExecutable: goExecutable, ToolPath: "/usr/bin:/bin", ApplicationPath: filepath.Join(temp, "Applications/watools.app")}
	m := NewManager(filepath.Join(temp, "data"), filepath.Join(temp, "cache"))
	if err := SaveConfiguration(m.dataDir, value); err != nil {
		t.Fatal(err)
	}
	return m, value
}
func TestRunnerArgumentsEnvironmentAndBoundedOutput(t *testing.T) {
	m, _ := managerFixture(t, "printf '%s\\n' \"$@\" \"$WATOOLS_DATA_DIR\" \"$WATOOLS_CACHE_DIR\"\ni=0; while [ $i -lt 200 ]; do echo line-$i; i=$((i+1)); done\n")
	if err := m.Start(context.Background(), Request{Scope: "plugins", PackageIDs: []string{"watools.plugin.json"}}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("job did not exit")
	}
	entries, readErr := os.ReadDir(m.cacheDir)
	if readErr != nil || len(entries) != 0 {
		t.Fatal("job left compiled installer behind")
	}
	status, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "succeeded" || status.RestartRequired || len(status.Output) != 120 {
		t.Fatalf("unexpected status: %+v", status)
	}
	status.Output[0] = "changed"
	next, _ := m.Status()
	if next.Output[0] == "changed" {
		t.Fatal("status exposed internal output")
	}
	// A separate short output verifies exact argv and formal data overrides.
	m, _ = managerFixture(t, "printf '%s\\n' \"$@\" \"$PATH\" \"$WATOOLS_DATA_DIR\" \"$WATOOLS_CACHE_DIR\"\n")
	if err := m.Start(context.Background(), Request{Scope: "plugins", PackageIDs: []string{"watools.plugin.json"}}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("job did not exit")
	}
	status, _ = m.Status()
	output := strings.Join(status.Output, "\n")
	for _, expected := range []string{"install\n--scope\nplugins", "--scope\nplugins", "--plugin\nwatools.plugin.json", m.dataDir, m.cacheDir} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing argv/environment %q in %q", expected, output)
		}
	}
}
func TestRunnerCancellationWaitsAndRejectsConcurrentWork(t *testing.T) {
	m, value := managerFixture(t, "echo started\nexec sleep 30\n")
	if err := m.Start(context.Background(), Request{Scope: "all"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(context.Background(), Request{Scope: "all"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("concurrent job error = %v", err)
	}
	if err := m.Configure(value); !errors.Is(err, ErrBusy) {
		t.Fatalf("configuration while building error = %v", err)
	}
	if err := m.Cancel(); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("cancellation did not terminate process group")
	}
	status, _ := m.Status()
	if status.State != "cancelled" || status.CanCancel || status.RestartRequired {
		t.Fatalf("unexpected cancellation: %+v", status)
	}
	m.Shutdown()
}
func TestCommitPhaseCannotBeCancelled(t *testing.T) {
	m, _ := managerFixture(t, "echo 'Artifacts ready. Applying application and plugin revisions.'\nsleep 0.2\n")
	if err := m.Start(context.Background(), Request{Scope: "application"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, _ := m.Status()
		if !status.CanCancel {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("commit phase not detected")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := m.Cancel(); !errors.Is(err, ErrBusy) {
		t.Fatalf("cancelling commit error = %v", err)
	}
	m.Shutdown()
	status, _ := m.Status()
	if status.State != "succeeded" || !status.RestartRequired {
		t.Fatalf("unexpected result: %+v", status)
	}
}
func TestInvalidSelectionDoesNotStartProcess(t *testing.T) {
	m, _ := managerFixture(t, "exit 0\n")
	for _, request := range []Request{{Scope: "all|application"}, {Scope: "plugins", PackageIDs: []string{"foreign.plugin"}}, {Scope: "application", PackageIDs: []string{"watools.plugin.json"}}, {Scope: "plugins", PackageIDs: []string{"watools.plugin.json", "watools.plugin.json"}}} {
		if err := m.Start(context.Background(), request); err == nil {
			t.Fatalf("accepted invalid request: %+v", request)
		}
	}
	status, _ := m.Status()
	if status.JobID != 0 {
		t.Fatal("invalid request spawned job")
	}
}
func TestConfigurationPreservesReceiptAndDoesNotExecuteSource(t *testing.T) {
	m, value := managerFixture(t, "exit 99\n")
	value.InstalledAt = "2026-01-01T00:00:00Z"
	value.Version = "old"
	if err := SaveConfiguration(m.dataDir, value); err != nil {
		t.Fatal(err)
	}
	incoming := value
	incoming.ApplicationPath = "/tmp/foreign.app"
	incoming.InstalledAt = "forged"
	incoming.Version = "forged"
	if err := m.Configure(incoming); err != nil {
		t.Fatal(err)
	}
	saved, err := ReadConfiguration(m.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ApplicationPath != value.ApplicationPath || saved.Version != "old" || saved.InstalledAt != value.InstalledAt {
		t.Fatal("configuration overwrote installation identity")
	}
	incoming.SourceRoot = t.TempDir()
	if err := m.Configure(incoming); err == nil {
		t.Fatal("foreign source accepted")
	}
	saved, _ = ReadConfiguration(m.dataDir)
	if saved.SourceRoot != value.SourceRoot {
		t.Fatal("invalid source replaced configuration")
	}
}

func TestLegacyReceiptResolvesGoFromRecordedToolPath(t *testing.T) {
	m, value := managerFixture(t, "exit 0\n")
	legacy := `{"sourceRoot":` + strconv.Quote(value.SourceRoot) + `,"nodeExecutable":"/obsolete/node","toolPath":` + strconv.Quote(filepath.Dir(value.GoExecutable)+":/usr/bin:/bin") + `,"applicationPath":` + strconv.Quote(value.ApplicationPath) + `,"installedAt":"preserved","version":"old"}`
	if err := os.WriteFile(filepath.Join(m.dataDir, "installation.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	saved, err := ReadConfiguration(m.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	saved, err = ValidateConfiguration(saved)
	if err != nil {
		t.Fatal(err)
	}
	if saved.GoExecutable != value.GoExecutable || saved.InstalledAt != "preserved" || saved.Version != "old" {
		t.Fatal("legacy receipt did not preserve identity and resolve Go")
	}
	if err := m.Configure(saved); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(m.dataDir, "installation.json"))
	if strings.Contains(string(content), "nodeExecutable") {
		t.Fatal("legacy executable field survived save")
	}
}

func TestOverlongOutputFailsWithoutHanging(t *testing.T) {
	m, _ := managerFixture(t, "i=0; while [ $i -lt 70000 ]; do printf x; i=$((i+1)); done; echo; exec sleep 30\n")
	if err := m.Start(context.Background(), Request{Scope: "plugins"}); err != nil {
		t.Fatal(err)
	}
	if !m.wait(5 * time.Second) {
		t.Fatal("overlong output hung the installer")
	}
	status, _ := m.Status()
	if status.State != "failed" {
		t.Fatal("output failure reported success")
	}
	entries, _ := os.ReadDir(m.cacheDir)
	if len(entries) != 0 {
		t.Fatal("failed job left compiled installer")
	}
}
