package launcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keep the production IO method and assignment-local monotone acceptor in the
// same regression: even partial output from a failed CLI must not teach bounds.
type failingCLINameProbeRuntime struct{ *fakeRuntime }

func (*failingCLINameProbeRuntime) ProbeSessionName(name string) error {
	return (OSRuntime{}).ProbeSessionName(name)
}

func TestFailedZellijNameProbeCannotTeachMonotoneBounds(t *testing.T) {
	for _, tc := range []struct{ name, script string }{
		{"failed-connect-stderr", "printf 'failed to connect to Zellij session\\n' >&2\nexit 2\n"},
		{"nonzero-partial-stdout", "printf 'CLIENT_ID\\n'\nexit 2\n"},
		{"timeout-partial-stdout", "printf 'CLIENT_ID\\n'\nexec /bin/sleep 30\n"},
		{"timeout-partial-stderr", "printf 'failed to connect to Zellij session\\n' >&2\nexec /bin/sleep 30\n"},
		// A canceled operation cannot establish even a negative bound from a
		// fragment of validator-looking text before termination.
		{"timeout-partial-validator", "printf 'session name must be less than 108 characters\\n' >&2\nexec /bin/sleep 30\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$PAIR562_NAME_FAILURE_CALLS\"\n" + tc.script
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PAIR562_NAME_FAILURE_CALLS", log)
			accepts, refusedAt, queryError := sessionNameAcceptor(&failingCLINameProbeRuntime{newFakeRuntime()})
			if accepts("pair-probe-name") {
				t.Error("failed subprocess taught a positive name bound")
			}
			if err := queryError(); err == nil || errors.Is(err, errSessionNameTooLong) {
				t.Errorf("failed subprocess query error = %v; want uncertain, not validator refusal", err)
			}
			if bound, known := refusedAt(); known {
				t.Errorf("failed subprocess taught negative bound %d", bound)
			}
			// Both sides of the first name's length remain unanswered. The assignment
			// stops at this observation instead of making more probes or using bounds.
			if accepts("x") || accepts("pair-an-even-longer-unobserved-name") {
				t.Error("unknown query admitted a later name")
			}
			calls, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Split(strings.TrimSpace(string(calls)), "\n"); len(got) != 1 || got[0] != "--session pair-probe-name action list-clients" {
				t.Errorf("unknown query continued probing: %q", calls)
			}
		})
	}
}

// Post-parse absence is evidence only of name validation, never ownership.
// Pinned v0.45.1 parses --session with validate_session before action dispatch.
func TestZellijNameProbeRecognizesOnlyEstablishedValidatorReplies(t *testing.T) {
	const missing = "Session 'pair-new' not found. The following sessions are active:\nother\n"
	for _, tc := range []struct {
		name, stdout, stderr string
		exit                 int
		accepted             bool
	}{
		{"clients", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", "", 0, true},
		{"no-active", "", "There is no active session!\n", 1, true},
		{"one-other-active", "", missing, 1, true},
		{"wrong-status", "", "There is no active session!\n", 2, false},
		{"partial-stdout", "partial", "There is no active session!\n", 1, false},
		{"other-request", "", "Session 'foreign' not found. The following sessions are active:\nother\n", 1, false},
		{"extra-diagnostic", "", missing + "failed to connect\n", 1, false},
		{"discovery-failed", "", "Session 'pair-new' not found. The following sessions are active:\nError occurred: unavailable\n", 1, false},
		{"discovery-empty", "", "Session 'pair-new' not found. The following sessions are active:\nNo active zellij sessions found.\n", 1, false},
		{"unrelated-stderr", "", "failed to connect\n", 1, false},
		{"empty-success", "", "", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' \"$PAIR562_NAME_STDOUT\"\nprintf '%%s' \"$PAIR562_NAME_STDERR\" >&2\nexit %d\n", tc.exit)
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PAIR562_NAME_STDOUT", tc.stdout)
			t.Setenv("PAIR562_NAME_STDERR", tc.stderr)
			err := (OSRuntime{}).ProbeSessionName("pair-new")
			if (err == nil) != tc.accepted {
				t.Fatalf("accepted=%v err=%v; want %v", err == nil, err, tc.accepted)
			}
		})
	}
}
