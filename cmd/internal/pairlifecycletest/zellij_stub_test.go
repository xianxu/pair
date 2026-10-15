package pairlifecycletest

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestEmptyZellijStubInventoryHasAnExplicitAbsenceReply(t *testing.T) {
	path, _ := StubZellij(t, map[string]string{})
	for _, format := range []string{"--short", "--no-formatting"} {
		cmd := exec.Command(path, "list-sessions", format)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if err == nil || cmd.ProcessState.ExitCode() != 1 || stdout.Len() != 0 || stderr.String() != "No active zellij sessions found.\n" {
			t.Fatalf("format=%s status=%v stdout=%q stderr=%q", format, err, stdout.String(), stderr.String())
		}
	}
}
