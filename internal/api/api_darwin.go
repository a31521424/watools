package api

/*
   #cgo CFLAGS: -x objective-c
   #cgo LDFLAGS: -framework Cocoa -framework Foundation
   #import <Cocoa/Cocoa.h>

   static int setClipboardPNGData(const void *bytes, int length) {
       @autoreleasepool {
           if (bytes == NULL || length <= 0) {
               return 0;
           }

           NSData *data = [NSData dataWithBytes:bytes length:length];
           NSImage *image = [[[NSImage alloc] initWithData:data] autorelease];
           if (image == nil) {
               return 0;
           }

           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           [pasteboard clearContents];
           BOOL ok = [pasteboard writeObjects:@[image]];
           return ok ? 1 : 0;
       }
   }
*/
import "C"

import (
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

func (a *WaApi) OpenFolderWithPath(path string) error {
	if strings.HasPrefix(path, "~/") {
		path = strings.TrimPrefix(path, "~/")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(homeDir, path)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path unavailable: %w", err)
	}
	if !stat.IsDir() || filepath.Ext(path) == ".app" {
		return exec.Command("open", "-R", path).Start()
	} else {
		return exec.Command("open", path).Start()
	}
}

func (a *WaApi) copyImageBytesToClipboard(imgBytes []byte) error {
	if len(imgBytes) == 0 {
		return fmt.Errorf("image data is empty")
	}

	var written C.int
	// v3 dispatches bindings concurrently; serialize all NSPasteboard access
	// on the native main thread, including writes alongside clipboard reads.
	application.InvokeSync(func() {
		written = C.setClipboardPNGData(unsafe.Pointer(&imgBytes[0]), C.int(len(imgBytes)))
	})
	if written == 0 {
		return fmt.Errorf("failed to write image to clipboard")
	}

	return nil
}
