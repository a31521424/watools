package app

/*
   #cgo CFLAGS: -x objective-c
   #cgo LDFLAGS: -framework Cocoa -framework Foundation -framework Carbon -framework UniformTypeIdentifiers
   #import <Cocoa/Cocoa.h>
   #import <Carbon/Carbon.h>
   #import <dispatch/dispatch.h>

   static NSRunningApplication *previousActiveApp = nil;

   static void clearPreviousActiveApp() {
       if (previousActiveApp != nil) {
           [previousActiveApp release];
           previousActiveApp = nil;
       }
   }

   static void storePreviousActiveApp() {
       @autoreleasepool {
           NSRunningApplication *frontmostApp = [[NSWorkspace sharedWorkspace] frontmostApplication];
           NSRunningApplication *currentApp = [NSRunningApplication currentApplication];
           if (frontmostApp == nil || [frontmostApp isEqual:currentApp]) {
               return;
           }

           if (previousActiveApp != nil && [previousActiveApp isEqual:frontmostApp]) {
               return;
           }

           clearPreviousActiveApp();
           previousActiveApp = [frontmostApp retain];
       }
   }

   static void activateCurrentAppWindowOnMainThread(void *handle);
   static void returnFocusToPreviousAppOnMainThread();

   static void activateCurrentAppWindow(void *handle) {
       @autoreleasepool {
           if ([NSThread isMainThread]) {
               activateCurrentAppWindowOnMainThread(handle);
               return;
           }

           dispatch_sync(dispatch_get_main_queue(), ^{
               activateCurrentAppWindowOnMainThread(handle);
           });
       }
   }

   static void returnFocusToPreviousApp() {
       @autoreleasepool {
           if ([NSThread isMainThread]) {
               returnFocusToPreviousAppOnMainThread();
               return;
           }

           dispatch_sync(dispatch_get_main_queue(), ^{
               returnFocusToPreviousAppOnMainThread();
           });
       }
   }

   static void hideCurrentWindow() {
       NSApplication *app = [NSApplication sharedApplication];
       [app hide:nil];
   }

   typedef struct {
       int ok;
       int screenWidth;
       int visibleWidth;
       int visibleOriginX;
       int windowWidth;
       int leftMargin;
       int rightMargin;
   } WaWindowPositionMetrics;

   static void activateCurrentAppWindowOnMainThread(void *handle) {
       NSApplication *app = [NSApplication sharedApplication];
       NSWindow *window = (NSWindow *)handle;

       [app unhide:nil];
       if (@available(macOS 14.0, *)) {
           [app activate];
       } else {
           [app activateIgnoringOtherApps:YES];
       }

       if (window != nil) {
           [window orderFront:nil];
           if ([window canBecomeKeyWindow]) {
               [window makeKeyWindow];
           }
       }
   }

   static void returnFocusToPreviousAppOnMainThread() {
       if (previousActiveApp == nil || [previousActiveApp isTerminated]) {
           clearPreviousActiveApp();
           return;
       }

       NSRunningApplication *currentApp = [NSRunningApplication currentApplication];
       if (currentApp == nil || ![currentApp isActive]) {
           clearPreviousActiveApp();
           return;
       }

       if (@available(macOS 14.0, *)) {
           [[NSApplication sharedApplication] yieldActivationToApplication:previousActiveApp];
           [previousActiveApp activateFromApplication:currentApp options:0];
       } else {
           [previousActiveApp activateWithOptions:0];
       }

       dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.1 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
           clearPreviousActiveApp();
       });
   }

   static int isSecureEventInputEnabled() {
       return IsSecureEventInputEnabled() ? 1 : 0;
   }

   static WaWindowPositionMetrics positionCurrentWindowOnMainThread(void *handle, int windowWidth, int windowHeight, int topOffset) {
       WaWindowPositionMetrics metrics = {0};
       NSWindow *window = (NSWindow *)handle;
       if (window == nil) {
           return metrics;
       }

       NSScreen *screen = [window screen];
       if (screen == nil) {
           screen = [NSScreen mainScreen];
       }

       if (screen == nil) {
           return metrics;
       }

       NSRect frame = [screen frame];
       NSRect visibleFrame = [screen visibleFrame];
       NSRect windowFrame = [window frame];
       CGFloat actualWindowWidth = windowFrame.size.width;
       CGFloat actualWindowHeight = windowFrame.size.height;

       windowFrame.origin.x = frame.origin.x + (frame.size.width - actualWindowWidth) / 2.0;
       windowFrame.origin.y = visibleFrame.origin.y + visibleFrame.size.height - actualWindowHeight - topOffset;

       [window setFrame:windowFrame display:YES animate:NO];

       metrics.ok = 1;
       metrics.screenWidth = (int)frame.size.width;
       metrics.visibleWidth = (int)visibleFrame.size.width;
       metrics.visibleOriginX = (int)visibleFrame.origin.x;
       metrics.windowWidth = (int)actualWindowWidth;
       metrics.leftMargin = (int)(windowFrame.origin.x - frame.origin.x);
       metrics.rightMargin = (int)((frame.origin.x + frame.size.width) - (windowFrame.origin.x + actualWindowWidth));
       return metrics;
   }

   static WaWindowPositionMetrics positionCurrentWindow(void *handle, int windowWidth, int windowHeight, int topOffset) {
       @autoreleasepool {
           if ([NSThread isMainThread]) {
               return positionCurrentWindowOnMainThread(handle, windowWidth, windowHeight, topOffset);
           }

           __block WaWindowPositionMetrics metrics = {0};
           dispatch_sync(dispatch_get_main_queue(), ^{
               metrics = positionCurrentWindowOnMainThread(handle, windowWidth, windowHeight, topOffset);
           });
           return metrics;
       }
   }

   // ==================== Clipboard C Functions ====================

   // Returns JSON array of available clipboard types (caller must free)
   static char* getClipboardTypes() {
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

           NSString *jsonString = [[[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding] autorelease];
           return strdup([jsonString UTF8String]);
       }
   }

   static int hasClipboardType(const char* type) {
       @autoreleasepool {
           NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
           NSString *nsType = [NSString stringWithUTF8String:type];
           return [pasteboard availableTypeFromArray:@[nsType]] != nil ? 1 : 0;
       }
   }

   // Returns UTF-8 string from clipboard (caller must free)
   static char* getClipboardText() {
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
   static void* getClipboardImageData(size_t *outSize) {
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

           NSImage *image = [[[NSImage alloc] initWithData:imageData] autorelease];
           if (image == nil) {
               *outSize = 0;
               return NULL;
           }

           CGImageRef cgImage = [image CGImageForProposedRect:NULL context:nil hints:nil];
           NSBitmapImageRep *bitmapRep = [[[NSBitmapImageRep alloc] initWithCGImage:cgImage] autorelease];
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
   static char* getClipboardFiles() {
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

           NSString *jsonString = [[[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding] autorelease];
           return strdup([jsonString UTF8String]);
       }
   }
*/
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"time"
	"unsafe"
	"watools/pkg/logger"
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
	application.InvokeSync(func() { C.storePreviousActiveApp() })
	time.Sleep(10 * time.Millisecond)
	a.checkAndRepositionIfNeeded()
	a.window.Show()
	a.positionWindow()
	C.activateCurrentAppWindow(a.window.NativeWindow())
	a.isHidden.Store(false)
}

func (a *WaApp) hideAppWithFocusReturn() {
	if !a.isHidden.Load() {
		C.returnFocusToPreviousApp()
		time.Sleep(100 * time.Millisecond)
		a.window.Hide()
		a.isHidden.Store(true)
	}
}

func (a *WaApp) HideApp() {
	if application.System.IsServer() {
		return
	}
	a.hideAppWithFocusReturn()
}

func (a *WaApp) ShowApp() {
	if application.System.IsServer() {
		return
	}
	a.showAppAsPanel()
}

func (a *WaApp) HideOrShowApp() {
	if a.isHidden.Load() {
		a.ShowApp()
	} else {
		a.HideApp()
	}
}

func (a *WaApp) IsSecureEventInputEnabled() bool {
	return C.isSecureEventInputEnabled() == 1
}

func (a *WaApp) positionWindowOnScreen(windowWidth int, windowHeight int) bool {
	screen, ok := a.getPrimaryScreen()
	if !ok {
		return false
	}

	topOffset := clampWindowTopOffset(screen.Size.Height, windowHeight)
	metrics := C.positionCurrentWindow(a.window.NativeWindow(), C.int(windowWidth), C.int(windowHeight), C.int(topOffset))
	if metrics.ok == 0 {
		return false
	}

	logger.Info(fmt.Sprintf(
		"Window positioning metrics: screenWidth=%d visibleWidth=%d visibleOriginX=%d leftMargin=%d windowWidth=%d rightMargin=%d",
		int(metrics.screenWidth),
		int(metrics.visibleWidth),
		int(metrics.visibleOriginX),
		int(metrics.leftMargin),
		int(metrics.windowWidth),
		int(metrics.rightMargin),
	))

	return true
}

// ==================== Clipboard API ====================

// v3 bindings run concurrently. NSPasteboard and its cached Cocoa objects
// must be accessed on the application main thread, just like window operations.

// GetClipboardTypes returns all available NSPasteboard type identifiers
// Common types: public.utf8-plain-text, public.png, public.file-url
func (a *WaApp) GetClipboardTypes() ([]string, error) {
	var cTypes *C.char
	application.InvokeSync(func() { cTypes = C.getClipboardTypes() })
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
	var available C.int
	application.InvokeSync(func() { available = C.hasClipboardType(cType) })
	return available == 1
}

// GetClipboardText returns plain text from clipboard
func (a *WaApp) GetClipboardText() (string, error) {
	var cText *C.char
	application.InvokeSync(func() { cText = C.getClipboardText() })
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
	var cData unsafe.Pointer
	application.InvokeSync(func() { cData = C.getClipboardImageData(&size) })

	if cData == nil || size == 0 {
		return "", fmt.Errorf("no image in clipboard")
	}
	defer C.free(cData)

	goData := C.GoBytes(cData, C.int(size))
	return base64.StdEncoding.EncodeToString(goData), nil
}

// GetClipboardFiles returns absolute file paths from clipboard
func (a *WaApp) GetClipboardFiles() ([]string, error) {
	var cFiles *C.char
	application.InvokeSync(func() { cFiles = C.getClipboardFiles() })
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
	// Headless development has no native event loop or desktop clipboard.
	if application.System.IsServer() {
		return &ClipboardContent{Types: []string{}, ContentType: ClipboardTypeEmpty}, nil
	}
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
