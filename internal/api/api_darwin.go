package api

/*
   #cgo CFLAGS: -x objective-c
   #cgo LDFLAGS: -framework Cocoa -framework Foundation
   #import <Cocoa/Cocoa.h>

   int setClipboardPNGData(const void *bytes, int length) {
       @autoreleasepool {
           if (bytes == NULL || length <= 0) {
               return 0;
           }

           NSData *data = [NSData dataWithBytes:bytes length:length];
           NSImage *image = [[NSImage alloc] initWithData:data];
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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

func (a *WaApi) OpenFolderWithPath(path string) {
	if strings.HasPrefix(path, "~/") {
		path = strings.TrimPrefix(path, "~/")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}
		path = filepath.Join(homeDir, path)
	}
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if !stat.IsDir() || filepath.Ext(path) == ".app" {
		_ = exec.Command("open", "-R", path).Start()
	} else {
		_ = exec.Command("open", path).Start()
	}
}

func (a *WaApi) copyImageBytesToClipboard(imgBytes []byte) error {
	if len(imgBytes) == 0 {
		return fmt.Errorf("image data is empty")
	}

	if C.setClipboardPNGData(unsafe.Pointer(&imgBytes[0]), C.int(len(imgBytes))) == 0 {
		return fmt.Errorf("failed to write image to clipboard")
	}

	return nil
}
