package app

/*
   #cgo CFLAGS: -x objective-c
   #cgo LDFLAGS: -framework Cocoa -framework Foundation
   #import <Cocoa/Cocoa.h>
   #include <dispatch/dispatch.h>

   static NSRunningApplication *previousActiveApp = nil;

   void storePreviousActiveApp() {
       NSArray *runningApps = [[NSWorkspace sharedWorkspace] runningApplications];
       for (NSRunningApplication *app in runningApps) {
           if ([app isActive] &&
               ![app isEqual:[NSRunningApplication currentApplication]]) {
               previousActiveApp = app;
               return;
           }
       }
   }

	void forceActivateAndFocusWindow() {
		dispatch_async(dispatch_get_main_queue(), ^{
		   NSApplication *app = [NSApplication sharedApplication];

		   [app unhide:nil];

		   // 系统进程级激活
		   NSRunningApplication *currentApp = [NSRunningApplication currentApplication];
		   [currentApp activateWithOptions:(NSApplicationActivateAllWindows | NSApplicationActivateIgnoringOtherApps)];

		   NSWindow *window = [app mainWindow];
		   if (window == nil && [[app windows] count] > 0) {
			  window = [[app windows] firstObject];
		   }

		   if (window != nil) {
			  // 保持悬浮层级
			  [window setLevel:NSFloatingWindowLevel];

			  // 【修正点】: 去掉了 NSWindowCollectionBehaviorCanJoinAllSpaces
			  // 只保留 MoveToActiveSpace 和 Transient
			  // MoveToActiveSpace: 确保窗口会跟随你切换到当前的桌面
			  // Transient: 辅助窗口行为，通常用于浮动面板
			  [window setCollectionBehavior: NSWindowCollectionBehaviorMoveToActiveSpace | NSWindowCollectionBehaviorTransient];

			  if ([window isMiniaturized]) {
				 [window deminiaturize:nil];
			  }

			  // 组合拳：显示 + 强制置前
			  [window makeKeyAndOrderFront:nil];
			  [window orderFrontRegardless];
			  [app activateIgnoringOtherApps:YES];

			  // 延时回马枪 (Double Tap)
			  dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.15 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
				 [app activateIgnoringOtherApps:YES];
				 [window makeKeyAndOrderFront:nil];
				 [window makeKeyWindow];
			  });
		   }
		});
	}

   void returnFocusToPreviousApp() {
       if (previousActiveApp != nil && ![previousActiveApp isTerminated]) {
           [previousActiveApp activateWithOptions:NSApplicationActivateIgnoringOtherApps];
           dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.1 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
               previousActiveApp = nil;
           });
       }
   }

   void hideCurrentWindow() {
       NSApplication *app = [NSApplication sharedApplication];
       [app hide:nil];
   }

   // ==================== Clipboard C Functions ====================

   // Returns JSON array of available clipboard types (caller must free)
   char* getClipboardTypes() {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           NSArray<NSPasteboardType> *types = [pasteboard types];

           if (types == nil || [types count] == 0) {
               return strdup("[]");
           }

           NSMutableArray *typeArray = [NSMutableArray array];
           for (NSPasteboardType type in types) {
               [typeArray addObject:type];
           }

           NSError *error = nil;
           NSData *jsonData = [NSJSONSerialization dataWithJSONObject:typeArray options:0 error:&error];
           if (error) {
               return strdup("[]");
           }

           NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
           return strdup([jsonString UTF8String]);
       }
   }

   int hasClipboardType(const char* type) {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           NSString *nsType = [NSString stringWithUTF8String:type];
           return [pasteboard availableTypeFromArray:@[nsType]] != nil ? 1 : 0;
       }
   }

   // Returns UTF-8 string from clipboard (caller must free)
   char* getClipboardText() {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           NSString *text = [pasteboard stringForType:NSPasteboardTypeString];
           if (text == nil) {
               return NULL;
           }
           return strdup([text UTF8String]);
       }
   }

   // Returns PNG image data from clipboard (caller must free)
   // Supports PNG, TIFF, JPEG and converts all to PNG format
   void* getClipboardImageData(size_t *outSize) {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];

           NSData *imageData = nil;
           NSArray *imageTypes = @[NSPasteboardTypePNG, NSPasteboardTypeTIFF, @"public.jpeg"];

           for (NSString *type in imageTypes) {
               imageData = [pasteboard dataForType:type];
               if (imageData != nil) {
                   break;
               }
           }

           if (imageData == nil) {
               *outSize = 0;
               return NULL;
           }

           NSImage *image = [[NSImage alloc] initWithData:imageData];
           if (image == nil) {
               *outSize = 0;
               return NULL;
           }

           CGImageRef cgImage = [image CGImageForProposedRect:NULL context:nil hints:nil];
           NSBitmapImageRep *bitmapRep = [[NSBitmapImageRep alloc] initWithCGImage:cgImage];
           NSData *pngData = [bitmapRep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];

           if (pngData == nil) {
               *outSize = 0;
               return NULL;
           }

           *outSize = [pngData length];
           void *buffer = malloc(*outSize);
           memcpy(buffer, [pngData bytes], *outSize);

           return buffer;
       }
   }

   // Returns JSON array of file paths from clipboard (caller must free)
   char* getClipboardFiles() {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           NSArray *classes = @[[NSURL class]];
           NSDictionary *options = @{};

           NSArray *fileURLs = [pasteboard readObjectsForClasses:classes options:options];

           if (fileURLs == nil || [fileURLs count] == 0) {
               return strdup("[]");
           }

           NSMutableArray *filePaths = [NSMutableArray array];
           for (NSURL *url in fileURLs) {
               if ([url isFileURL]) {
                   [filePaths addObject:[url path]];
               }
           }

           NSError *error = nil;
           NSData *jsonData = [NSJSONSerialization dataWithJSONObject:filePaths options:0 error:&error];
           if (error) {
               return strdup("[]");
           }

           NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
           return strdup([jsonString UTF8String]);
       }
   }
*/
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ClipboardContentType represents the primary type of content in clipboard
type ClipboardContentType string

const (
	ClipboardTypeText  ClipboardContentType = "text"
	ClipboardTypeImage ClipboardContentType = "image"
	ClipboardTypeFiles ClipboardContentType = "files"
	ClipboardTypeEmpty ClipboardContentType = "empty"
)

// ClipboardContent contains comprehensive clipboard data with automatic type detection
type ClipboardContent struct {
	Types       []string             `json:"types"`       // All NSPasteboard types available
	ContentType ClipboardContentType `json:"contentType"` // Primary detected type
	HasText     bool                 `json:"hasText"`
	HasImage    bool                 `json:"hasImage"`
	HasFiles    bool                 `json:"hasFiles"`
	Text        string               `json:"text,omitempty"`
	ImageBase64 string               `json:"imageBase64,omitempty"` // PNG format, base64 encoded
	Files       []string             `json:"files,omitempty"`       // Absolute file paths
}

// ==================== Window Management ====================

func (a *WaApp) showAppAsPanel() {
	C.storePreviousActiveApp()
	C.forceActivateAndFocusWindow()
	time.Sleep(10 * time.Millisecond)
	a.checkAndRepositionIfNeeded()
	runtime.WindowShow(a.ctx)
	a.isHidden = false
}

func (a *WaApp) hideAppWithFocusReturn() {
	if !a.isHidden {
		C.returnFocusToPreviousApp()
		time.Sleep(100 * time.Millisecond)
		runtime.WindowHide(a.ctx)
		a.isHidden = true
	}
}

func (a *WaApp) HideApp() {
	a.hideAppWithFocusReturn()
}

func (a *WaApp) ShowApp() {
	a.showAppAsPanel()
}

func (a *WaApp) HideOrShowApp() {
	if a.isHidden {
		a.ShowApp()
	} else {
		a.HideApp()
	}
}

// ==================== Clipboard API ====================

// GetClipboardTypes returns all available NSPasteboard type identifiers
// Common types: public.utf8-plain-text, public.png, public.file-url
func (a *WaApp) GetClipboardTypes() ([]string, error) {
	cTypes := C.getClipboardTypes()
	if cTypes == nil {
		return []string{}, nil
	}
	defer C.free(unsafe.Pointer(cTypes))

	jsonStr := C.GoString(cTypes)
	var types []string
	if err := json.Unmarshal([]byte(jsonStr), &types); err != nil {
		return nil, fmt.Errorf("failed to parse clipboard types: %w", err)
	}

	return types, nil
}

// HasClipboardType checks if clipboard contains a specific type identifier
func (a *WaApp) HasClipboardType(typeStr string) bool {
	cType := C.CString(typeStr)
	defer C.free(unsafe.Pointer(cType))
	return C.hasClipboardType(cType) == 1
}

// GetClipboardText returns plain text from clipboard
func (a *WaApp) GetClipboardText() (string, error) {
	cText := C.getClipboardText()
	if cText == nil {
		return "", fmt.Errorf("no text in clipboard")
	}
	defer C.free(unsafe.Pointer(cText))
	return C.GoString(cText), nil
}

// GetClipboardImage returns clipboard image as base64-encoded PNG
// Automatically converts TIFF/JPEG to PNG format for consistency
func (a *WaApp) GetClipboardImage() (string, error) {
	var size C.size_t
	cData := C.getClipboardImageData(&size)

	if cData == nil || size == 0 {
		return "", fmt.Errorf("no image in clipboard")
	}
	defer C.free(cData)

	goData := C.GoBytes(cData, C.int(size))
	return base64.StdEncoding.EncodeToString(goData), nil
}

// GetClipboardFiles returns absolute file paths from clipboard
func (a *WaApp) GetClipboardFiles() ([]string, error) {
	cFiles := C.getClipboardFiles()
	if cFiles == nil {
		return []string{}, nil
	}
	defer C.free(unsafe.Pointer(cFiles))

	jsonStr := C.GoString(cFiles)
	var files []string
	if err := json.Unmarshal([]byte(jsonStr), &files); err != nil {
		return nil, fmt.Errorf("failed to parse clipboard files: %w", err)
	}

	return files, nil
}

// GetClipboardContent performs automatic type detection and returns all available content
// Priority order for ContentType: Files > Image > Text > Empty
func (a *WaApp) GetClipboardContent() (*ClipboardContent, error) {
	content := &ClipboardContent{}

	types, err := a.GetClipboardTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get clipboard types: %w", err)
	}
	content.Types = types

	if len(types) == 0 {
		content.ContentType = ClipboardTypeEmpty
		return content, nil
	}

	// Detect available content types
	content.HasText = a.HasClipboardType("public.utf8-plain-text") || a.HasClipboardType("NSStringPboardType")
	content.HasImage = a.HasClipboardType("public.png") || a.HasClipboardType("public.tiff") || a.HasClipboardType("public.jpeg")
	content.HasFiles = a.HasClipboardType("public.file-url")

	// Determine primary type by priority
	if content.HasFiles {
		content.ContentType = ClipboardTypeFiles
	} else if content.HasImage {
		content.ContentType = ClipboardTypeImage
	} else if content.HasText {
		content.ContentType = ClipboardTypeText
	} else {
		content.ContentType = ClipboardTypeEmpty
	}

	// Populate available content
	if content.HasText {
		if text, err := a.GetClipboardText(); err == nil {
			content.Text = text
		}
	}

	if content.HasImage {
		if imageBase64, err := a.GetClipboardImage(); err == nil {
			content.ImageBase64 = imageBase64
		}
	}

	if content.HasFiles {
		if files, err := a.GetClipboardFiles(); err == nil {
			content.Files = files
		}
	}

	return content, nil
}
