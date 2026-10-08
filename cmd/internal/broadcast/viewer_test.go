package broadcast

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestViewerNode runs the viewer's node tests (font fit, connection state),
// the way the terminal oracle tests drive node.
func TestViewerNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	_, file, _, _ := runtime.Caller(0)
	tests, err := filepath.Glob(filepath.Join(filepath.Dir(file), "../../../tests/broadcast-viewer/*.test.mjs"))
	if err != nil || len(tests) < 2 {
		t.Fatalf("viewer node tests not found: %v %v", tests, err)
	}
	out, err := exec.Command(node, append([]string{"--test"}, tests...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
}
