package webcast

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNode runs the in-page control's and the viewer's node tests, as the
// broadcast does for its viewer.
func TestNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	_, file, _, _ := runtime.Caller(0)
	tests, err := filepath.Glob(filepath.Join(filepath.Dir(file), "../../../tests/webcast/*.test.mjs"))
	if err != nil || len(tests) == 0 {
		t.Fatalf("webcast node tests not found: %v %v", tests, err)
	}
	out, err := exec.Command(node, append([]string{"--test"}, tests...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
}
