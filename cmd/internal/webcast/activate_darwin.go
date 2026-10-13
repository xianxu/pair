package webcast

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// activateProcess brings the app with this pid to the front. By pid, not
// bundle: the cast Chrome shares Google Chrome's bundle id with the
// operator's own Chrome, so `open -a` could raise the wrong one. JXA reaches
// AppKit without an Automation permission; 2 is
// NSApplicationActivateIgnoringOtherApps.
func activateProcess(pid int) func() error {
	script := fmt.Sprintf(`ObjC.import("AppKit"); `+
		`const app = $.NSRunningApplication.runningApplicationWithProcessIdentifier(%d); `+
		`if (!app.isNil()) app.activateWithOptions(2);`, pid)
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script).CombinedOutput()
		if err != nil {
			return fmt.Errorf("webcast: bring Chrome to the front: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
}
