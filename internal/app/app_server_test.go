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
	if err := a.SetClipboardText("synthetic\x00clipboard"); err == nil {
		t.Fatal("server mode must reject native clipboard writes")
	}
	content, err := a.GetClipboardContent()
	if err != nil || content.ContentType != ClipboardTypeEmpty || content.HasText || content.HasImage || content.HasFiles || len(content.Types) != 0 {
		t.Fatalf("server returned native clipboard: %+v, %v", content, err)
	}
	automatic, err := a.GetAutomaticClipboard()
	if err != nil || automatic.Length != 0 || automatic.Text != "" {
		t.Fatalf("server returned native automatic clipboard: %+v, %v", automatic, err)
	}
	a.clipboard.close()
}
