package recorder

import (
	"os/exec"
	"testing"
)

func TestCaptureReadinessProbe(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the capture readiness fixture")
	}
	out, err := exec.Command(node, "capture_ready_test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("readiness probe: %v\n%s", err, out)
	}
}
