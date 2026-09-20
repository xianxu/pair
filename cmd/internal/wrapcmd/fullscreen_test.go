package wrapcmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestAgentExecutesFullscreenWithoutUserError(t *testing.T) {
	original := toggleFocusedLayout
	t.Cleanup(func() { toggleFocusedLayout = original })
	called := 0
	toggleFocusedLayout = func(_ []string, _ layoutcmd.FullscreenRuntime, out io.Writer) int {
		called++
		_, _ = io.WriteString(out, "diagnostic already logged")
		return 1
	}
	var stderr bytes.Buffer
	p := &proxy{stderr: &stderr, shortcutErrorReporter: func(error) { t.Fatal("user notification") }}
	if !p.executeWorkbenchDecision(workbenchshortcut.ShortcutDecision{Disposition: workbenchshortcut.DispositionHandle, Action: workbenchshortcut.ActionToggleFocusedLayout}) || called != 1 {
		t.Fatal("fullscreen did not reach layout executor")
	}
	if stderr.Len() != 0 {
		t.Fatalf("user error: %q", stderr.String())
	}
}
