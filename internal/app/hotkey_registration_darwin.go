package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#import <Carbon/Carbon.h>
#include <stdint.h>
extern void waHotkeyPressed(uintptr_t handle);
typedef struct { EventHotKeyRef key; EventHandlerRef handler; } WaHotkeyRegistration;
static OSStatus waHotkeyHandler(EventHandlerCallRef next, EventRef event, void *userdata) {
    EventHotKeyID id;
    OSStatus status=GetEventParameter(event,kEventParamDirectObject,typeEventHotKeyID,NULL,sizeof(id),NULL,&id);
    if(status!=noErr || id.signature!=0x5741544f || id.id!=(UInt32)(uintptr_t)userdata) return eventNotHandledErr;
    waHotkeyPressed((uintptr_t)userdata);
    return noErr;
}
static OSStatus waRegisterHotkey(UInt32 modifiers, UInt32 key, uintptr_t handle, WaHotkeyRegistration *registration) {
    EventHotKeyID id={.signature=0x5741544f,.id=(UInt32)handle};
    OSStatus status=RegisterEventHotKey(key,modifiers,id,GetApplicationEventTarget(),kEventHotKeyExclusive,&registration->key);
    if(status!=noErr) return status;
    EventTypeSpec type={kEventClassKeyboard,kEventHotKeyPressed};
    status=InstallApplicationEventHandler(&waHotkeyHandler,1,&type,(void *)handle,&registration->handler);
    if(status!=noErr) {UnregisterEventHotKey(registration->key);registration->key=NULL;}
    return status;
}
static OSStatus waUnregisterHotkey(WaHotkeyRegistration *registration) {
    if(registration->key) {
        OSStatus status=UnregisterEventHotKey(registration->key);
        if(status!=noErr) return status;
        registration->key=NULL;
    }
    if(registration->handler) {
        OSStatus status=RemoveEventHandler(registration->handler);
        if(status!=noErr) return status;
        registration->handler=NULL;
    }
    return noErr;
}
*/
import "C"

import (
	"fmt"
	"runtime/cgo"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.design/x/hotkey"
)

// Carbon permits shared registrations unless exclusivity is requested. Keep
// the native registration and handler together so failures reach UpdateHotkey,
// and release both on the same AppKit thread before deleting the Go callback.
type macHotkey struct {
	mu         sync.Mutex
	modifiers  []hotkey.Modifier
	key        hotkey.Key
	native     C.WaHotkeyRegistration
	handle     cgo.Handle
	keydown    chan hotkey.Event
	registered bool
}

// Dispatch before acquiring any lifecycle lock: Wails invokes ServiceShutdown
// on the main thread, which must never wait for a lock held by a dispatcher.
func runHotkeyLifecycle(operation func() error) error {
	if application.Get() == nil || application.System.IsServer() {
		return operation()
	}
	return application.InvokeSyncWithResult(operation)
}

func newPlatformHotkey(modifiers []hotkey.Modifier, key hotkey.Key) registeredHotkey {
	return &macHotkey{modifiers: modifiers, key: key}
}
func (hk *macHotkey) Register() error {
	return runHotkeyLifecycle(hk.register)
}

func (hk *macHotkey) register() error {
	hk.mu.Lock()
	defer hk.mu.Unlock()
	if hk.registered {
		return fmt.Errorf("hotkey already registered")
	}
	if application.System.IsServer() {
		return fmt.Errorf("native hotkeys are unavailable in browser/server mode")
	}
	if application.Get() == nil {
		return fmt.Errorf("native hotkeys require a running desktop application")
	}
	var modifiers uint32
	for _, modifier := range hk.modifiers {
		modifiers |= uint32(modifier)
	}
	// Each registration owns its event stream; an unregistered stream is closed.
	hk.keydown = make(chan hotkey.Event, 1)
	handle := cgo.NewHandle(hk)
	status := C.waRegisterHotkey(C.UInt32(modifiers), C.UInt32(hk.key), C.uintptr_t(handle), &hk.native)
	if status != 0 {
		handle.Delete()
		return fmt.Errorf("failed to register hotkey (macOS status %d; the key may be in use)", int(status))
	}
	hk.handle = handle
	hk.registered = true
	return nil
}
func (hk *macHotkey) Unregister() error {
	return runHotkeyLifecycle(hk.unregister)
}

func (hk *macHotkey) unregister() error {
	hk.mu.Lock()
	defer hk.mu.Unlock()
	if !hk.registered {
		return nil
	}
	status := C.waUnregisterHotkey(&hk.native)
	if status != 0 {
		return fmt.Errorf("failed to unregister hotkey (macOS status %d)", int(status))
	}
	hk.handle.Delete()
	hk.registered = false
	close(hk.keydown)
	return nil
}
func (hk *macHotkey) Keydown() <-chan hotkey.Event { return hk.keydown }

//export waHotkeyPressed
func waHotkeyPressed(handle C.uintptr_t) {
	hk := cgo.Handle(handle).Value().(*macHotkey)
	// A native event handler must not block the AppKit thread on an application
	// action; the listener processes notifications on its own goroutine.
	select {
	case hk.keydown <- hotkey.Event{}:
	default:
	}
}
