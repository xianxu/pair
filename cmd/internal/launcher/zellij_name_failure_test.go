package launcher

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
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
	for _, name := range []string{
		"failed-connect-stderr", "nonzero-partial-stdout", "timeout-partial-stdout", "timeout-partial-stderr", "timeout-partial-validator",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			reply := zellijtest.Case(t, "launcher-name-failure-"+name)
			body := zellijtest.ShellCase(t, "launcher-name-failure-"+name)
			if strings.HasPrefix(name, "timeout-") {
				// Retain the original partial-output-then-block control. A
				// generator format change must not turn it into a quick exit.
				suffix := fmt.Sprintf("; exit %d", reply.Code)
				if !strings.HasSuffix(body, suffix) {
					t.Fatal("fixture shell reply has no final status")
				}
				body = strings.TrimSuffix(body, suffix) + "\nexec /bin/sleep 30\n"
			}
			script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$PAIR562_NAME_FAILURE_CALLS\"\n" + body
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
	for _, tc := range []struct {
		name     string
		accepted bool
	}{
		{"clients", true}, {"no-active", true}, {"one-other-active", true},
		{"wrong-status", false}, {"partial-stdout", false}, {"other-request", false},
		{"extra-diagnostic", false}, {"discovery-failed", false}, {"discovery-empty", false},
		{"unrelated-stderr", false}, {"empty-success", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			reply := zellijtest.Case(t, "launcher-name-probe-"+tc.name)

			script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' \"$PAIR562_NAME_STDOUT\"\nprintf '%%s' \"$PAIR562_NAME_STDERR\" >&2\nexit %d\n", reply.Code)
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PAIR562_NAME_STDOUT", reply.Stdout)
			t.Setenv("PAIR562_NAME_STDERR", reply.Stderr)
			err := (OSRuntime{}).ProbeSessionName("pair-new")
			if (err == nil) != tc.accepted {
				t.Fatalf("accepted=%v err=%v; want %v", err == nil, err, tc.accepted)
			}
		})
	}
}
