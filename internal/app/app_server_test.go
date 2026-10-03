//go:build server && (darwin || windows)

package app

import "testing"

func TestServerDoesNotAccessNativeClipboardOrWindow(t *testing.T) {
	// No application/window is initialized in this test. Desktop calls must
	// remain safe in the real headless build, including automatic focus reads.
	a := &WaApp{}
	a.HideApp()
	a.ShowApp()
	a.HideOrShowApp()
	content, err := a.GetClipboardContent()
	if err != nil || content.ContentType != ClipboardTypeEmpty || content.HasText || content.HasImage || content.HasFiles || len(content.Types) != 0 {
		t.Fatalf("server returned native clipboard: %+v, %v", content, err)
	}
}
