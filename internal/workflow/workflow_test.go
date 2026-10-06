package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"watools/internal/filelock"
)

type toolCall struct {
	Tool                        string
	Args                        []string
	Data, Cache, Cwd, Toolchain string
}

func TestMain(m *testing.M) {
	if os.Getenv("WATOOLS_TEST_TOOL") == "1" {
		args := os.Args[1:]
		tool := filepath.Base(os.Args[0])
		cwd, _ := os.Getwd()
		call := toolCall{tool, args, os.Getenv("WATOOLS_DATA_DIR"), os.Getenv("WATOOLS_CACHE_DIR"), cwd, os.Getenv("GOTOOLCHAIN")}
		content, _ := json.Marshal(call)
		file, _ := os.OpenFile(os.Getenv("WATOOLS_TEST_RECORD"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		file.Write(append(content, '\n'))
		file.Close()
		switch tool {
		case "node":
			fmt.Println("v24.11.0")
		case "pnpm":
			if len(args) > 0 && args[0] == "--version" {
				version := os.Getenv("WATOOLS_TEST_PNPM")
				if version == "" {
					version = "10.21.0"
				}
				fmt.Println(version)
			}
		case "xcrun":
			fmt.Println("/fake/macOS.sdk")
		case "go":
			if len(args) > 0 && args[0] == "version" {
				fmt.Println("go version " + os.Getenv("GOTOOLCHAIN") + " darwin/arm64")
			}
			if len(args) > 0 && args[0] == "build" {
				if os.Getenv("WATOOLS_TEST_FAIL_BUILD") != "" {
					os.Exit(8)
				}
				for i, arg := range args {
					if arg == "-o" {
						target := args[i+1]
						os.MkdirAll(filepath.Dir(target), 0755)
						if strings.HasSuffix(target, "localinstall") {
							source, _ := os.ReadFile(os.Args[0])
							os.WriteFile(target, source, 0755)
						} else {
							os.WriteFile(target, []byte("built"), 0755)
						}
					}
				}
			}
			for i, arg := range args {
				if arg == "-macfilename" {
					os.MkdirAll(filepath.Dir(args[i+1]), 0755)
					os.WriteFile(args[i+1], []byte("icon"), 0644)
				}
			}
			if len(args) > 0 && args[0] == "test" && os.Getenv("WATOOLS_TEST_FAIL_TEST") != "" {
				os.Exit(9)
			}
			if len(args) > 0 && args[0] == "mod" && os.Getenv("WATOOLS_TEST_BLOCK") != "" {
				os.WriteFile(os.Getenv("WATOOLS_TEST_READY"), []byte(fmt.Sprint(os.Getpid())), 0600)
				time.Sleep(30 * time.Second)
			}
		case "localinstall":
			time.Sleep(100 * time.Millisecond)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type fixture struct {
	r                           *Runner
	record, userData, userCache string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("macOS workflow")
	}
	temp := t.TempDir()
	root := filepath.Join(temp, "checkout with spaces")
	bin := filepath.Join(temp, "bin")
	for _, dir := range []string{bin, filepath.Join(root, "cmd/watoolsctl"), filepath.Join(root, "frontend/node_modules/typescript"), filepath.Join(root, "build/darwin")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("cmd/watoolsctl/main.go", "package main")
	write("go.mod", "module watools\ngo 1.26.0\ntoolchain go1.26.8\nrequire (\n github.com/wailsapp/wails/v3 v3.0.0-beta.27\n)\n")
	write("frontend/package.json", `{"packageManager":"pnpm@10.21.0"}`)
	write("wails.json", `{"name":"watools","version":"0.1.0","info":{"productName":"WaTools & QA","comments":"","copyright":""}}`)
	write("build/darwin/Info.plist", "<plist>{{.ProductName}} {{.ProductVersion}}</plist>")
	write("build/darwin/Info.dev.plist", "<plist>{{.ProductName}} {{.ProductVersion}}</plist>")
	write("build/config.yml", "dev_mode:\n  executes:\n    - command: [${WATOOLS_CTL}, dev-build]\n  port: ${WAILS_VITE_PORT}\n  binary: ${WATOOLS_DEV_BINARY}\n")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go", "node", "pnpm", "xcrun", "codesign"} {
		if err := os.Link(executable, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	f := fixture{r: New(root, &bytes.Buffer{}, &bytes.Buffer{}), record: filepath.Join(temp, "calls.jsonl"), userData: filepath.Join(temp, "user-data"), userCache: filepath.Join(temp, "user-cache")}
	f.r.SetEnv("PATH", bin)
	f.r.SetEnv("WATOOLS_TEST_TOOL", "1")
	f.r.SetEnv("GORACE", "atexit_sleep_ms=0")
	f.r.SetEnv("WATOOLS_TEST_RECORD", f.record)
	f.r.SetEnv("GOCACHE", "")
	f.r.SetEnv("GOMODCACHE", "")
	f.r.SetEnv("GOPATH", "")
	for _, dir := range []string{f.userData, f.userCache} {
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "sentinel"), []byte("preserved"), 0600)
	}
	f.r.SetEnv("WATOOLS_DATA_DIR", f.userData)
	f.r.SetEnv("WATOOLS_CACHE_DIR", f.userCache)
	t.Cleanup(func() {
		for _, dir := range []string{f.userData, f.userCache} {
			data, err := os.ReadFile(filepath.Join(dir, "sentinel"))
			if err != nil || string(data) != "preserved" {
				t.Error("caller data changed")
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Error("build leaked into caller data")
			}
		}
	})
	return f
}
func (f fixture) calls(t *testing.T) []toolCall {
	t.Helper()
	data, err := os.ReadFile(f.record)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var calls []toolCall
	if len(data) == 0 {
		return calls
	}
	last := bytes.LastIndexByte(data, '\n')
	if last < 0 {
		return calls
	}
	data = data[:last]
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var call toolCall
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, call)
	}
	return calls
}
func TestOptionValidationBeforeTools(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"plugins", "install", "--output", "/private"}, {"build", "-platform", "darwin/amd64"}, {"dev", "--port", "9246"}, {"dev", "--port", "0"}, {"dev", "--port", "65536"}, {"install", "--scope", "application", "--plugin", "watools.plugin.json"}, {"install", "--scope", "unknown"}, {"install", "--plugin", "../outside"}, {"bindings", "-save"}} {
		f := newFixture(t)
		if err := f.r.Execute(context.Background(), args); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if len(f.calls(t)) != 0 {
			t.Fatal("invalid command invoked tools")
		}
	}
}
func TestBuildUsesTemporaryDataAndPinnedVersions(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Execute(context.Background(), []string{"build"}); err != nil {
		t.Fatal(err)
	}
	calls := f.calls(t)
	var build toolCall
	for _, c := range calls {
		if c.Toolchain != "go1.26.8" {
			t.Fatal("toolchain not pinned")
		}
		if c.Data == f.userData || c.Cache == f.userCache {
			t.Fatal("build used caller data")
		}
		if c.Tool == "go" && len(c.Args) > 0 && c.Args[0] == "build" {
			build = c
		}
	}
	if !strings.Contains(strings.Join(build.Args, " "), "production,private_mac_apis") {
		t.Fatal("production tags missing")
	}
	if _, err := os.Stat(filepath.Dir(calls[len(calls)-1].Data)); !os.IsNotExist(err) {
		t.Fatal("temporary data leaked")
	}
	plist, _ := os.ReadFile(filepath.Join(f.r.Root, "build/bin/watools.app/Contents/Info.plist"))
	if !strings.Contains(string(plist), "WaTools &amp; QA") {
		t.Fatal("plist metadata not escaped")
	}
}
func TestInstallPreparesBeforeApplyAndRestoresFormalEnvironment(t *testing.T) {
	f := newFixture(t)
	os.RemoveAll(filepath.Join(f.r.Root, "frontend/node_modules"))
	if err := f.r.Execute(context.Background(), []string{"install", "--app-dir", filepath.Join(f.r.Root, "Apps with spaces"), "--no-launch"}); err != nil {
		t.Fatal(err)
	}
	calls := f.calls(t)
	apply := calls[len(calls)-1]
	if apply.Tool != "localinstall" || apply.Data != f.userData || apply.Cache != f.userCache {
		t.Fatal("apply did not restore formal environment")
	}
	if !strings.Contains(strings.Join(apply.Args, " "), "--go") || !strings.Contains(strings.Join(apply.Args, " "), "--no-launch") {
		t.Fatal("apply flags missing")
	}
	packageIndex, signIndex := -1, -1
	for i, c := range calls {
		if c.Tool == "go" && strings.Contains(strings.Join(c.Args, " "), " package ") {
			packageIndex = i
		}
		if c.Tool == "codesign" {
			signIndex = i
		}
		if i < len(calls)-1 && len(c.Args) > 0 && c.Args[0] != "version" && c.Args[0] != "--version" && c.Args[0] != "--show-sdk-path" {
			if c.Data == apply.Data {
				t.Fatal("prepare used formal data")
			}
		}
	}
	if packageIndex < 0 || signIndex < packageIndex {
		t.Fatal("packages were not prepared before application signing")
	}
	if _, err := os.Stat(filepath.Dir(calls[signIndex].Data)); !os.IsNotExist(err) {
		t.Fatal("installation artifacts leaked")
	}
}
func TestPluginOnlyNeedsNoNodeAndNoApplicationBuild(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Execute(context.Background(), []string{"install", "--scope", "plugins", "--plugin", "watools.plugin.json", "--no-launch"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls(t) {
		if c.Tool == "node" || c.Tool == "pnpm" || c.Tool == "codesign" {
			t.Fatal("plugin-only install invoked frontend or app tools")
		}
	}
}
func TestFailedBuildCannotApplyAndSourceLockIsReleased(t *testing.T) {
	f := newFixture(t)
	f.r.SetEnv("WATOOLS_TEST_FAIL_BUILD", "1")
	if err := f.r.Execute(context.Background(), []string{"install"}); err == nil {
		t.Fatal("failed build ignored")
	}
	for _, c := range f.calls(t) {
		if c.Tool == "localinstall" {
			t.Fatal("failed build applied")
		}
	}
	unlock, err := filelock.Acquire(filepath.Join(f.r.Root, ".cache/installation/source.lock"))
	if err != nil {
		t.Fatal("source lock leaked", err)
	}
	unlock()
}
func TestBindingsPrepareAssetsAndNormalizeModes(t *testing.T) {
	f := newFixture(t)
	dir := filepath.Join(f.r.Root, "frontend/bindings")
	os.MkdirAll(dir, 0755)
	file := filepath.Join(dir, "runtime.js")
	os.WriteFile(file, []byte("generated"), 0755)
	if err := f.r.Execute(context.Background(), []string{"bindings"}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(file)
	if info.Mode().Perm() != 0644 {
		t.Fatal("generated module remains executable")
	}
	if _, err := os.Stat(filepath.Join(f.r.Root, "frontend/dist/index.html")); err != nil {
		t.Fatal("embed placeholder missing")
	}
	calls := f.calls(t)
	if calls[len(calls)-2].Tool != "go" || calls[len(calls)-1].Tool != "pnpm" {
		t.Fatal("binding/build order changed")
	}
}
func TestDevConfigurationUsesNativeCLIAndSafePorts(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Execute(context.Background(), []string{"browser", "--port", "9255"}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(f.r.Root, ".cache/dev/wails-browser.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "watoolsctl\", dev-build]") || !strings.Contains(string(content), "9255") || !strings.Contains(string(content), "watools.browser.app") {
		t.Fatal("watcher configuration incorrect", string(content))
	}
}
func TestCancellationStopsChildAndSkipsLaterTasks(t *testing.T) {
	f := newFixture(t)
	ready := filepath.Join(t.TempDir(), "ready")
	f.r.SetEnv("WATOOLS_TEST_BLOCK", "1")
	f.r.SetEnv("WATOOLS_TEST_READY", ready)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- f.r.Execute(ctx, []string{"setup"}) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("task did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancellation reported success")
		}
	case <-time.After(7 * time.Second):
		t.Fatal("child did not stop")
	}
	for _, c := range f.calls(t) {
		if c.Tool == "pnpm" && len(c.Args) > 0 && c.Args[0] == "install" {
			t.Fatal("cancelled workflow continued")
		}
	}
}
func TestCommitFinishesDespiteCancellation(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// The tool's record signals the start of the atomic apply operation.
	done := make(chan error, 1)
	go func() { done <- f.r.Execute(ctx, []string{"install", "--scope", "plugins"}) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		calls := f.calls(t)
		if len(calls) > 0 && calls[len(calls)-1].Tool == "localinstall" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("apply did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal("commit interrupted", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("commit did not finish")
	}
}
func TestPNPMMismatchFailsBeforeMutation(t *testing.T) {
	f := newFixture(t)
	f.r.SetEnv("WATOOLS_TEST_PNPM", "11.2.2")
	if err := f.r.Execute(context.Background(), []string{"setup"}); err == nil {
		t.Fatal("pnpm mismatch accepted")
	}
	for _, c := range f.calls(t) {
		if len(c.Args) > 0 && (c.Args[0] == "mod" || c.Args[0] == "install") {
			t.Fatal("mismatch invoked mutating command")
		}
	}
}

func TestUnsuccessfulBuildPreservesPreviousOutput(t *testing.T) {
	f := newFixture(t)
	bundle := filepath.Join(f.r.Root, "build/bin/watools.app")
	if err := os.MkdirAll(bundle, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(bundle, "old-resource")
	os.WriteFile(sentinel, []byte("previous"), 0644)
	f.r.SetEnv("WATOOLS_TEST_FAIL_BUILD", "1")
	if err := f.r.Execute(context.Background(), []string{"build"}); err == nil {
		t.Fatal("build failure ignored")
	}
	data, _ := os.ReadFile(sentinel)
	if string(data) != "previous" {
		t.Fatal("unsuccessful build deleted prior output")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(bundle), ".watools-build-*"))
	if len(leftovers) != 0 {
		t.Fatal("failed build staging leaked")
	}
	f.r.SetEnv("WATOOLS_TEST_FAIL_BUILD", "")
	if err := f.r.Execute(context.Background(), []string{"build"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("successful build kept stale resource")
	}
}
