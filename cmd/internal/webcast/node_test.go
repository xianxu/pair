package webcast

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

// TestNode runs the in-page control's and the viewer's node tests, as the
// broadcast does for its viewer.
func TestNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	repo, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	tests, err := filepath.Glob(filepath.Join(repo, "tests/webcast/*.test.mjs"))
	if err != nil || len(tests) == 0 {
		t.Fatalf("webcast node tests not found: %v %v", tests, err)
	}
	out, err := exec.Command(node, append([]string{"--test"}, tests...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
}
