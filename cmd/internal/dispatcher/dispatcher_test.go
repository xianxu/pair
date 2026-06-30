package dispatcher

import (
	"strings"
	"testing"
)

func TestDispatchHelpListsPlannedFamiliesWithoutClaimingSupport(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			res := Dispatch(args)
			if res.ExitCode != 0 {
				t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
			}
			if res.Stderr != "" {
				t.Fatalf("Stderr = %q, want empty", res.Stderr)
			}
			for _, want := range []string{
				"Usage: pair-go <command> [args]",
				"launch",
				"wrap",
				"slug",
				"not implemented in this skeleton",
			} {
				if !strings.Contains(res.Stdout, want) {
					t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
				}
			}
		})
	}
}

func TestDispatchVersionIsDevelopmentSkeletonMetadata(t *testing.T) {
	res := Dispatch([]string{"version"})
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.Stderr != "" {
		t.Fatalf("Stderr = %q, want empty", res.Stderr)
	}
	for _, want := range []string{"pair-go", "dispatcher skeleton", "public launcher: bin/pair"} {
		if !strings.Contains(res.Stdout, want) {
			t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
		}
	}
}

func TestDispatchPlannedCommandReturnsUnsupported(t *testing.T) {
	res := Dispatch([]string{"wrap"})
	if res.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
	}
	if res.Stdout != "" {
		t.Fatalf("Stdout = %q, want empty", res.Stdout)
	}
	for _, want := range []string{"wrap", "planned", "not implemented", "pair-go help"} {
		if !strings.Contains(res.Stderr, want) {
			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
		}
	}
}

func TestDispatchUnknownCommandReturnsUsageHint(t *testing.T) {
	res := Dispatch([]string{"frobnicate"})
	if res.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
	}
	if res.Stdout != "" {
		t.Fatalf("Stdout = %q, want empty", res.Stdout)
	}
	for _, want := range []string{"unknown command", "frobnicate", "pair-go help"} {
		if !strings.Contains(res.Stderr, want) {
			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
		}
	}
}
