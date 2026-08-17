package procutil

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestAlive(t *testing.T) {
	if Alive("") {
		t.Error("empty pid must not be alive")
	}
	self := strconv.Itoa(os.Getpid())
	if !Alive(self) {
		t.Errorf("own pid %s should be alive", self)
	}
	// A pid that (almost certainly) doesn't exist.
	if Alive("2147483646") {
		t.Error("bogus high pid should not be alive")
	}
}

func TestCommand(t *testing.T) {
	if Command("") != "" {
		t.Error("empty pid must yield empty command")
	}
	self := strconv.Itoa(os.Getpid())
	got := Command(self)
	if got == "" {
		// A locked-down environment (sandbox / container) can block `ps`; skip
		// visibly there rather than hard-fail — `make test` runs on a normal
		// dev machine where ps works.
		if !psAvailable() {
			t.Skip("ps unavailable in this environment (sandbox); skipping command-line probe")
		}
		t.Errorf("own pid %s should have a non-empty command line", self)
	}
	// No trailing newline (ps -o command= is trimmed).
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("command line should be newline-trimmed, got %q", got)
	}
}

func TestDescendantPIDsIncludesNestedChildren(t *testing.T) {
	children := map[string][]string{
		"10": {"11", "12"},
		"11": {"13"},
		"13": {"14"},
	}
	got := DescendantPIDs("10", children)
	want := []string{"10", "11", "12", "13", "14"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("DescendantPIDs = %v, want %v", got, want)
	}
}

func psAvailable() bool {
	return exec.Command("ps", "-p", strconv.Itoa(os.Getpid()), "-o", "command=").Run() == nil
}
