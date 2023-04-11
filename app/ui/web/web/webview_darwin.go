//go:build darwin

package web

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int WEBVIEW_SetActivationPolicy(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	[NSApp activateIgnoringOtherApps:YES];
	[NSApp.mainWindow setLevel:CGShieldingWindowLevel()];
    return 0;
}
*/
import "C"
import "time"

func setActivationPolicy() {
	go func() {
		time.Sleep(time.Millisecond * 200)
		C.WEBVIEW_SetActivationPolicy()
	}()

}
