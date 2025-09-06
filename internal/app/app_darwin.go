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

// ShowAppAsPanel shows the app with panel behavior
func (a *WaApp) showAppAsPanel() {
	// Store the current active app BEFORE showing our window
	C.storePreviousActiveApp()

	// Small delay to ensure the previous app is stored
	time.Sleep(10 * time.Millisecond)

	// Check if screen configuration changed before showing
	a.checkAndRepositionIfNeeded()

	// Show our window
	runtime.WindowShow(a.ctx)
	a.isHidden = false
}

// HideAppWithFocusReturn hides the app and returns focus properly
func (a *WaApp) hideAppWithFocusReturn() {
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
