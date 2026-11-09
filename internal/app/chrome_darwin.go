//go:build darwin

package app

/*
#cgo CFLAGS: -x objective-c -mmacosx-version-min=10.13
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

static double lotusTitleBarHeight(void *nsWindow) {
	NSWindow *window = (NSWindow *)nsWindow;
	if (window == nil) {
		return -1;
	}
	if (([window styleMask] & NSWindowStyleMaskFullScreen) == NSWindowStyleMaskFullScreen) {
		return 0;
	}
	NSButton *closeButton = [window standardWindowButton:NSWindowCloseButton];
	NSView *contentView = [window contentView];
	if (closeButton == nil || contentView == nil) {
		return -1;
	}
	NSRect buttonFrame = [closeButton convertRect:[closeButton bounds] toView:contentView];
	if (NSIsEmptyRect(buttonFrame) || NSWidth(buttonFrame) < 1 || NSHeight(buttonFrame) < 1) {
		return -1;
	}
	double fromTop;
	if ([contentView isFlipped]) {
		fromTop = NSMidY(buttonFrame);
	} else {
		fromTop = NSHeight([contentView bounds]) - NSMidY(buttonFrame);
	}
	if (fromTop <= 0 || fromTop > 80) {
		return -1;
	}
	return fromTop * 2.0;
}
*/
import "C"

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func measureTitleBarHeight() float64 {
	appl := application.Get()
	if appl == nil {
		return fallbackTitleBarHeight
	}
	return application.InvokeSyncWithResult(func() float64 {
		win := currentWindow(appl)
		if win == nil {
			return fallbackTitleBarHeight
		}
		native := win.NativeWindow()
		if native == nil {
			return fallbackTitleBarHeight
		}
		return interpretTitleBarHeight(float64(C.lotusTitleBarHeight(native)))
	})
}

func currentWindow(appl *application.App) application.Window {
	if win := appl.Window.Current(); win != nil {
		return win
	}
	var fallback application.Window
	for _, win := range appl.Window.GetAll() {
		if win.IsFocused() {
			return win
		}
		fallback = win
	}
	return fallback
}
