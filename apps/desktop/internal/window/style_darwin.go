package window

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void unifiedToolbar(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (@available(macOS 11.0, *)) {
			for (NSWindow *w in [NSApp windows]) {
				w.toolbarStyle = NSWindowToolbarStyleUnified;
			}
		}
	});
}
*/
import "C"

func Style() {
	C.unifiedToolbar()
}
