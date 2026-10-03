//go:build server

package api

import (
	"strings"
	"testing"
)

func TestServerRejectsNativeImageClipboard(t *testing.T) {
	err := (&WaApi{}).CopyBase64ImageToClipboard("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Z2yEAAAAASUVORK5CYII=")
	if err == nil || !strings.Contains(err.Error(), "unavailable in browser/server mode") {
		t.Fatalf("expected unsupported native clipboard, got %v", err)
	}
}
