package recorder

import (
	"os/exec"
	"testing"
)

func TestBoxTextCollector(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the DOM collector fixture")
	}
	out, err := exec.Command(node, "box_text_test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("collector: %v\n%s", err, out)
	}
}
