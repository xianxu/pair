package broadcast

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestViewerFit runs the viewer's pure fit tests in node, the way the
// terminal oracle tests drive node.
func TestViewerFit(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	_, file, _, _ := runtime.Caller(0)
	test := filepath.Join(filepath.Dir(file), "../../../tests/broadcast-viewer/fit.test.mjs")
	out, err := exec.Command(node, "--test", test).CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
}
