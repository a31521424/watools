//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation
#import <Cocoa/Cocoa.h>

// Global variable to store the previous active application
static NSRunningApplication *previousActiveApp = nil;

// Store the currently active application before showing our window
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

// Return focus to the previously active application and wait for activation
void returnFocusToPreviousApp() {
    if (previousActiveApp != nil && ![previousActiveApp isTerminated]) {
        // Force activate the previous application
        [previousActiveApp activateWithOptions:NSApplicationActivateIgnoringOtherApps];

        // Give time for the application to fully activate
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.1 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
            // Clear reference only after successful activation
            previousActiveApp = nil;
        });
    }
}

// Hide current application window properly
void hideCurrentWindow() {
    NSApplication *app = [NSApplication sharedApplication];
    [app hide:nil];
}
*/
import "C"

import (
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// EnablePanelBehavior configures the window to behave like a macOS panel
func (a *WaApp) EnablePanelBehavior() {
	// Listen for window blur events to implement auto-hide
	runtime.EventsOn(a.ctx, "window-blur", func(optionalData ...interface{}) {
		if !a.isHidden {
			// First return focus to previous app
			C.returnFocusToPreviousApp()

			// Small delay to ensure focus is returned properly
			time.Sleep(50 * time.Millisecond)

			// Then hide our app
			a.HideApp()
		}
	})
}

// ShowAppAsPanel shows the app with panel behavior
func (a *WaApp) ShowAppAsPanel() {
	// Store the current active app BEFORE showing our window
	C.storePreviousActiveApp()

	// Small delay to ensure the previous app is stored
	time.Sleep(10 * time.Millisecond)

	// Show our window
	runtime.WindowShow(a.ctx)
	a.isHidden = false
}

// HideAppWithFocusReturn hides the app and returns focus properly
func (a *WaApp) HideAppWithFocusReturn() {
	if !a.isHidden {
		// Return focus first
		C.returnFocusToPreviousApp()

		// Give time for focus to return
		time.Sleep(100 * time.Millisecond)

		// Hide our window
		runtime.WindowHide(a.ctx)
		a.isHidden = true
	}
}
