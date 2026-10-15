package couchcore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistrationTerminalFixtureReturnsExplicitEmptyObservations(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.Symlink(binary, path); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_TEST_COUCH_TERMINAL_FIXTURE", "1")
	t.Setenv("PAIR_TEST_ZELLIJ_LOG", filepath.Join(dir, "calls"))
	t.Setenv("PAIR_TEST_ZELLIJ_READY", filepath.Join(dir, "not-ready"))
	for _, args := range [][]string{{"list-sessions", "--short"}, {"list-sessions", "--no-formatting"}} {
		cmd := exec.CommandContext(t.Context(), path, args...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		if err == nil || cmd.ProcessState.ExitCode() != 1 || len(stdout) != 0 || stderr.String() != "No active zellij sessions found.\n" {
			t.Errorf("%v = stdout %q stderr %q error %v; want explicit no-active-sessions exit1", args, stdout, stderr.String(), err)
		}
	}
	cmd := exec.CommandContext(t.Context(), path, "--session", "fixture", "action", "list-clients")
	out, err := cmd.Output()
	if err != nil || string(out) != "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" {
		t.Fatalf("clients = %q, %v; want header-only detached observation", out, err)
	}
}
