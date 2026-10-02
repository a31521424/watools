package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHttpProxyLimitsAndCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/large" {
			_, _ = w.Write([]byte(strings.Repeat("x", maxProxyBody+1)))
			return
		}
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	a := &WaApi{httpClient: server.Client()}
	a.OnStartup(context.Background())
	defer a.Shutdown()
	if _, err := a.HttpProxy(HttpProxyRequest{URL: server.URL + "/large"}); err == nil {
		t.Fatal("oversized response accepted")
	}
	if _, err := a.HttpProxy(HttpProxyRequest{URL: server.URL, Timeout: 60001}); err == nil {
		t.Fatal("unbounded timeout accepted")
	}
	done := make(chan error, 1)
	go func() { _, err := a.HttpProxy(HttpProxyRequest{URL: server.URL, RequestID: "test"}); done <- err }()
	<-started
	a.CancelHttpProxy("test")
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("request was not cancelled")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not release request")
	}
	a.Shutdown()
	if _, err := a.HttpProxy(HttpProxyRequest{URL: server.URL}); err == nil {
		t.Fatal("request survived shutdown")
	}
}
func TestSavedImageDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	first, err := saveImageFile(dir, []byte("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := saveImageFile(dir, []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("filenames collide")
	}
	data, err := os.ReadFile(first)
	if err != nil || string(data) != "first" {
		t.Fatal("first image changed")
	}
}
func TestOpenFolderReturnsStatError(t *testing.T) {
	a := &WaApi{}
	if err := a.OpenFolderWithPath(filepath.Join(t.TempDir(), "missing", "nested")); err == nil {
		t.Fatal("missing path reported as success")
	}
}

func TestHttpProxyConcurrencyLimit(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 12)
	var active, maxSeen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := active.Add(1)
		defer active.Add(-1)
		for {
			old := maxSeen.Load()
			if count <= old || maxSeen.CompareAndSwap(old, count) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	a := &WaApi{httpClient: server.Client()}
	a.OnStartup(context.Background())
	defer a.Shutdown()
	var workers sync.WaitGroup
	errors := make(chan error, 12)
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); _, err := a.HttpProxy(HttpProxyRequest{URL: server.URL}); errors <- err }()
	}
	for i := 0; i < 8; i++ {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			close(release)
			t.Fatal("requests failed to start")
		}
	}
	select {
	case <-started:
		close(release)
		t.Fatal("more than 8 requests ran together")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	workers.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if maxSeen.Load() > 8 {
		t.Fatal("concurrency limit exceeded")
	}
}
