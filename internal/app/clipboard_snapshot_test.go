package app

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

func TestMatchingEdgesRespectUTF16Budget(t *testing.T) {
	for _, text := range []string{strings.Repeat("😀", 200), "  {" + strings.Repeat("中文😀", 100) + "}\n", "short\x00"} {
		head, tail := textPrefix(text), textSuffix(text)
		if !strings.HasPrefix(text, head) || !strings.HasSuffix(text, tail) || !utf8.ValidString(head) || !utf8.ValidString(tail) {
			t.Fatal("matching edges are not exact Unicode fragments")
		}
		if len(utf16.Encode([]rune(head))) > 128 || len(utf16.Encode([]rune(tail))) > 128 {
			t.Fatal("matching edge exceeds the UTF-16 budget")
		}
	}
}

func TestAutomaticClipboardPreservesOriginAndBoundsBridge(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	origin := "  \x00" + strings.Repeat("😀中文 sample \n", 100000) + "\n  "
	view, err := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "1", text: origin, length: 1300006}, nil
	})
	if err != nil || view.Text != "" || len(view.Preview) > 250 || !utf8.ValidString(view.Preview) {
		t.Fatalf("large origin escaped onto the preview bridge: %+v, %v", view, err)
	}
	if !strings.HasPrefix(origin, view.Prefix) || len(view.Prefix) > 512 || !utf8.ValidString(view.Prefix) {
		t.Fatal("prefix was derived from the abbreviated display")
	}
	if !strings.HasSuffix(origin, view.Suffix) || len(view.Suffix) > 512 || !utf8.ValidString(view.Suffix) {
		t.Fatal("suffix is not a bounded Unicode edge of the original")
	}
	text, err := snapshots.origin(view.SnapshotID)
	if err != nil || text != origin {
		t.Fatal("origin lost whitespace, NUL or Unicode")
	}
	// Copying another text never changes an already issued snapshot.
	next, err := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "2", text: " next\n", length: 6}, nil
	})
	if err != nil || next.Text != " next\n" || next.Preview != next.Text {
		t.Fatal("small text not preserved inline")
	}
	text, err = snapshots.origin(view.SnapshotID)
	if err != nil || text != origin {
		t.Fatal("launch snapshot followed the current clipboard")
	}
	snapshots.release(view.SnapshotID)
	if _, err = snapshots.origin(view.SnapshotID); !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("released origin is still accessible")
	}
}

func TestConcurrentAutomaticReadsReuseVersionAndReleasedCache(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	var materialized atomic.Int32
	read := func(known string) (clipboardTextRead, error) {
		if known == "clipboard-version" {
			return clipboardTextRead{version: known, unchanged: true}, nil
		}
		materialized.Add(1)
		return clipboardTextRead{version: "clipboard-version", text: "original", length: 8}, nil
	}
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			view, err := snapshots.get(read)
			if err != nil || view.Text != "original" {
				t.Error("concurrent read failed")
			}
		})
	}
	group.Wait()
	view, _ := snapshots.get(read)
	snapshots.release(view.SnapshotID)
	view, _ = snapshots.get(read)
	if text, err := snapshots.origin(view.SnapshotID); text != "original" || err != nil || materialized.Load() != 1 {
		t.Fatal("version cache reread clipboard body or failed to repin")
	}
}

func TestSnapshotLeaseExpiryBudgetAndShutdown(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	var ids []string
	for _, version := range []string{"1", "2", "3", "4", "5", "6"} {
		view, _ := snapshots.get(func(string) (clipboardTextRead, error) {
			return clipboardTextRead{version: version, text: "source", length: 6}, nil
		})
		ids = append(ids, view.SnapshotID)
	}
	if len(snapshots.items) != 4 {
		t.Fatal("snapshot count is unbounded")
	}
	if _, err := snapshots.origin(ids[0]); !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("oldest snapshot was not evicted")
	}
	latest := ids[len(ids)-1]
	before := snapshots.items[latest].expires
	if err := snapshots.retain(latest); err != nil || snapshots.items[latest].expires.Before(before) {
		t.Fatal("active lease was not extended")
	}
	snapshots.mu.Lock()
	snapshots.items[latest].expires = time.Now().Add(-time.Second)
	snapshots.mu.Unlock()
	if _, err := snapshots.origin(latest); !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("expired snapshot remained accessible")
	}
	snapshots.close()
	if _, err := snapshots.get(func(string) (clipboardTextRead, error) {
		t.Fatal("reader called after shutdown")
		return clipboardTextRead{}, nil
	}); err == nil {
		t.Fatal("closed clipboard cache reopened")
	}
}

func TestOneOversizedOriginIsPreservedAndEvictsOldSnapshots(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	view, _ := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "1", text: "small", length: 5}, nil
	})
	origin := strings.Repeat("a", clipboardSnapshotBudget+1)
	large, err := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "2", text: origin, length: len(origin)}, nil
	})
	if err != nil || len(snapshots.items) != 1 {
		t.Fatal("byte budget did not evict older snapshots")
	}
	if _, err := snapshots.origin(view.SnapshotID); err == nil {
		t.Fatal("old snapshot retained alongside oversized origin")
	}
	if text, err := snapshots.origin(large.SnapshotID); err != nil || len(text) != len(origin) {
		t.Fatal("oversized origin was rejected or truncated")
	}
}

func TestShutdownDoesNotWaitForNativeReaderOrReopenCache(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	view, err := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "1", text: "original", length: 8}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	started, resume := make(chan struct{}), make(chan struct{})
	var resumeOnce sync.Once
	unblock := func() { resumeOnce.Do(func() { close(resume) }) }
	defer unblock()
	readDone := make(chan error, 1)
	go func() {
		_, err := snapshots.get(func(known string) (clipboardTextRead, error) {
			close(started)
			<-resume // Represents a native read waiting for the application thread.
			return clipboardTextRead{version: known, unchanged: true}, nil
		})
		readDone <- err
	}()
	<-started
	closed := make(chan struct{})
	go func() { snapshots.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for a native reader")
	}
	unblock()
	if err := <-readDone; !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("in-flight reader repopulated a closed cache")
	}
	if _, err := snapshots.origin(view.SnapshotID); !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("closed cache returned a retained origin")
	}
	if _, err := snapshots.get(func(string) (clipboardTextRead, error) {
		t.Error("queued read reached native clipboard after shutdown")
		return clipboardTextRead{}, nil
	}); !errors.Is(err, ErrClipboardSnapshotExpired) {
		t.Fatal("closed cache accepted a new read")
	}
}

func TestUnchangedNativeReadKeepsSourceAcrossConcurrentExpiry(t *testing.T) {
	var snapshots clipboardSnapshots
	defer snapshots.close()
	view, err := snapshots.get(func(string) (clipboardTextRead, error) {
		return clipboardTextRead{version: "1", text: "original", length: 8}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := snapshots.get(func(known string) (clipboardTextRead, error) {
		// Cache access must remain possible during a native read. Simulate the
		// timer clearing the item while the OS checks its unchanged version.
		snapshots.mu.Lock()
		snapshots.items[view.SnapshotID].expires = time.Now().Add(-time.Second)
		snapshots.purge(time.Now())
		snapshots.mu.Unlock()
		return clipboardTextRead{version: known, unchanged: true}, nil
	})
	if err != nil || result.SnapshotID != view.SnapshotID || result.Text != "original" {
		t.Fatal("unchanged read lost its retained source during cache expiry")
	}
	if text, err := snapshots.origin(view.SnapshotID); err != nil || text != "original" {
		t.Fatal("unchanged source was not repinned")
	}
}
