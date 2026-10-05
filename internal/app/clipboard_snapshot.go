package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
	"unicode/utf8"
)

const clipboardInlineLimit = 2000
const clipboardSnapshotTTL = 2 * time.Minute
const clipboardSnapshotBudget = 64 << 20

var ErrClipboardSnapshotExpired = errors.New("clipboard snapshot expired")

// AutomaticClipboard contains only a bounded preview for large text. Length is
// measured in UTF-16 code units, matching JavaScript string.length.
type AutomaticClipboard struct {
	Version    string `json:"version"`
	SnapshotID string `json:"snapshotId"`
	Text       string `json:"text"`
	Preview    string `json:"preview"`
	Prefix     string `json:"prefix"`
	Suffix     string `json:"suffix"`
	Length     int    `json:"length"`
}

type clipboardTextRead struct {
	version   string
	text      string
	length    int
	unchanged bool
}

type clipboardTextSnapshot struct {
	view    AutomaticClipboard
	origin  string
	expires time.Time
}

// The latest snapshot is also the version cache. Older snapshots remain pinned
// briefly for in-flight launches, within a count and byte budget. One oversized
// current snapshot is allowed: text is never silently truncated or rejected.
type clipboardSnapshots struct {
	readMu sync.Mutex // Serialise native reads without blocking cache lifecycle.
	mu     sync.Mutex
	latest *clipboardTextSnapshot
	items  map[string]*clipboardTextSnapshot
	timer  *time.Timer
	closed bool
}

func textPreview(text string) string {
	start, end := 0, len(text)
	for i := 0; i < 30 && start < len(text); i++ {
		_, size := utf8.DecodeRuneInString(text[start:])
		start += size
	}
	for i := 0; i < 30 && end > start; i++ {
		_, size := utf8.DecodeLastRuneInString(text[:end])
		end -= size
	}
	return text[:start] + "  ......  " + text[end:]
}

func textPrefix(text string) string {
	end, units := 0, 0
	for end < len(text) && units < 128 {
		r, size := utf8.DecodeRuneInString(text[end:])
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > 128 {
			break
		}
		end, units = end+size, units+width
	}
	return text[:end]
}

func textSuffix(text string) string {
	start, units := len(text), 0
	for start > 0 && units < 128 {
		r, size := utf8.DecodeLastRuneInString(text[:start])
		width := 1
		if r > 0xffff {
			width = 2
		}
		if units+width > 128 {
			break
		}
		start, units = start-size, units+width
	}
	return text[start:]
}

func (c *clipboardSnapshots) purge(now time.Time) {
	for id, item := range c.items {
		if !now.Before(item.expires) {
			delete(c.items, id)
		}
	}
	if c.latest != nil && !now.Before(c.latest.expires) {
		c.latest = nil
	}
}

func (c *clipboardSnapshots) schedule() {
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = time.AfterFunc(clipboardSnapshotTTL, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.purge(time.Now())
		if !c.closed && (c.latest != nil || len(c.items) > 0) {
			c.schedule()
		}
	})
}

func (c *clipboardSnapshots) get(read func(string) (clipboardTextRead, error)) (*AutomaticClipboard, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrClipboardSnapshotExpired
	}
	now := time.Now()
	c.purge(now)
	cached := c.latest
	version := ""
	if cached != nil {
		version = cached.view.Version
	}
	c.mu.Unlock()
	// AppKit reads synchronously dispatch to the main thread. Shutdown also
	// runs there, so it must be able to close the cache while a read is queued.
	value, err := read(version)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrClipboardSnapshotExpired
	}
	if err != nil {
		return nil, err
	}
	now = time.Now()
	c.purge(now)
	if c.items == nil {
		c.items = make(map[string]*clipboardTextSnapshot)
	}
	if value.unchanged && cached != nil {
		// The timer may have expired this item while the native version check
		// was in flight. The retained source still makes an unchanged read valid.
		cached.expires = now.Add(clipboardSnapshotTTL)
		c.latest = cached
		c.items[cached.view.SnapshotID] = cached
		c.schedule()
		view := cached.view
		return &view, nil
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return nil, err
	}
	view := AutomaticClipboard{Version: value.version, SnapshotID: hex.EncodeToString(random[:]), Length: value.length, Prefix: textPrefix(value.text), Suffix: textSuffix(value.text)}
	if value.length <= clipboardInlineLimit {
		view.Text, view.Preview = value.text, value.text
	} else {
		view.Preview = textPreview(value.text)
	}
	item := &clipboardTextSnapshot{view: view, origin: value.text, expires: now.Add(clipboardSnapshotTTL)}
	c.latest = item
	c.items[view.SnapshotID] = item
	for {
		bytes, oldestID := 0, ""
		var oldest time.Time
		for id, snapshot := range c.items {
			bytes += len(snapshot.origin)
			if id != view.SnapshotID && (oldestID == "" || snapshot.expires.Before(oldest)) {
				oldestID, oldest = id, snapshot.expires
			}
		}
		if oldestID == "" || (len(c.items) <= 4 && bytes <= clipboardSnapshotBudget) {
			break
		}
		delete(c.items, oldestID)
	}
	c.schedule()
	return &view, nil
}

func (c *clipboardSnapshots) origin(id string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.purge(time.Now())
	if item := c.items[id]; item != nil {
		return item.origin, nil
	}
	return "", ErrClipboardSnapshotExpired
}

func (c *clipboardSnapshots) release(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, id)
}

func (c *clipboardSnapshots) retain(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.purge(time.Now())
	item := c.items[id]
	if item == nil {
		return ErrClipboardSnapshotExpired
	}
	item.expires = time.Now().Add(clipboardSnapshotTTL)
	c.schedule()
	return nil
}

func (c *clipboardSnapshots) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.timer != nil {
		c.timer.Stop()
	}
	c.items, c.latest = nil, nil
}

func (a *WaApp) GetAutomaticClipboard() (*AutomaticClipboard, error) {
	return a.clipboard.get(a.readAutomaticClipboardText)
}

func (a *WaApp) GetClipboardSnapshotText(id string) (string, error) {
	return a.clipboard.origin(id)
}

func (a *WaApp) ReleaseClipboardSnapshot(id string)      { a.clipboard.release(id) }
func (a *WaApp) RetainClipboardSnapshot(id string) error { return a.clipboard.retain(id) }
