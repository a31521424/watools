//go:build darwin || windows

package watcher

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestStopWithPendingEvent(t *testing.T) {
	for i := 0; i < 20; i++ {
		fw, err := NewFSWatcher()
		if err != nil {
			t.Fatal(err)
		}
		fw.watchDirs = []string{t.TempDir()}
		if err := fw.Start(); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(fw.watchDirs[0], "Example.app")
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := fw.Stop(); err != nil {
			t.Fatal(err)
		}
		if err := fw.Stop(); err != nil {
			t.Fatal(err)
		}
		for range fw.EventChannel() {
		} // completion, including all delayed work
		if err := fw.Start(); err == nil {
			t.Fatal("a closed watcher must reject restart")
		}
	}
}
func TestFinalStateWinsOverStaleEvent(t *testing.T) {
	fw, err := NewFSWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()
	path := filepath.Join(t.TempDir(), "Example.app")
	if !fw.processEvent(fsnotify.Event{Name: path, Op: fsnotify.Create}) {
		t.Fatal("unexpected cancellation")
	}
	if event := <-fw.EventChannel(); event.Type != AppRemoved {
		t.Fatal("stale create resurrected removed application")
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	fw.processEvent(fsnotify.Event{Name: path, Op: fsnotify.Remove})
	if event := <-fw.EventChannel(); event.Type == AppRemoved {
		t.Fatal("stale remove deleted replacement")
	}
}
func TestFilesystemBurstProducesFinalRemoval(t *testing.T) {
	fw, err := NewFSWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()
	dir := t.TempDir()
	fw.watchDirs = []string{dir}
	if err := fw.Start(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "Example.app")
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "Example.lnk")
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	} else if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}

	select {
	case <-fw.EventChannel():
	case <-time.After(3 * time.Second):
		t.Fatal("create event was lost")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-fw.EventChannel():
		if event.Type != AppRemoved {
			t.Fatalf("got %+v", event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("final event was lost")
	}
}

func TestRetryCancellationInterruptsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	op := NewRetryableOperation(func() error { attempts++; cancel(); return fmt.Errorf("retry") }, NewDefaultErrorHandler(&RetryConfig{MaxRetries: 3}), &RetryConfig{MaxRetries: 3, RetryInterval: 60000})
	started := time.Now()
	if err := op.ExecuteContext(ctx); err == nil {
		t.Fatal("cancellation accepted as success")
	}
	if time.Since(started) > time.Second || attempts != 1 {
		t.Fatal("shutdown waited for backoff")
	}
}
