package dispatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchNamesDeriveFromImplementedStatus(t *testing.T) {
	names := DispatchNames()
	// The full implemented set MUST be present: the public `pair <sub>` peel-off
	// keys off DispatchNames(), so if one of these were accidentally left
	// `planned`, `pair changelog` would fall through to the launcher (start a
	// session) with no other test catching it.
	for _, want := range []string{"context", "scrollback-render", "slug", "changelog", "continuation", "session-watch"} {
		if !containsStr(names, want) {
			t.Fatalf("DispatchNames() = %v, missing implemented %q", names, want)
		}
	}
	// Planned + handoff families are NOT routable subcommands.
	for _, absent := range []string{"launch", "wrap", "scribe"} {
		if containsStr(names, absent) {
			t.Fatalf("DispatchNames() = %v, must not contain non-implemented %q", names, absent)
		}
	}
}

func TestStreamingFlags(t *testing.T) {
	for _, s := range []string{"changelog", "continuation", "session-watch"} {
		if !IsStreaming(s) {
			t.Errorf("IsStreaming(%q) = false, want true (stdin/live-stderr/long-running)", s)
		}
	}
	for _, b := range []string{"slug", "context", "scrollback-render"} {
		if IsStreaming(b) {
			t.Errorf("IsStreaming(%q) = true, want false (buffered)", b)
		}
	}
}

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

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
				"Implemented commands:",
				"launch",
				"compatibility handoff",
				"context",
				"scrollback-render",
				"wrap",
				"slug",
				"not implemented in this skeleton",
			} {
				if !strings.Contains(res.Stdout, want) {
					t.Fatalf("Stdout missing %q:\n%s", want, res.Stdout)
				}
			}
			if strings.Contains(res.Stdout, "launch             session lifecycle and public pair launcher flow (planned; not implemented") {
				t.Fatalf("Stdout still labels launch unimplemented:\n%s", res.Stdout)
			}
			if strings.Contains(res.Stdout, "decision-phase only") {
				t.Fatalf("Stdout still labels launch decision-phase only:\n%s", res.Stdout)
			}
			for _, stale := range []string{
				"context           agent pane context meter (planned; not implemented",
				"scrollback-render raw PTY capture to ANSI scrollback (planned; not implemented",
			} {
				if strings.Contains(res.Stdout, stale) {
					t.Fatalf("Stdout still labels helper unimplemented (%q):\n%s", stale, res.Stdout)
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
	for _, want := range []string{"pair-go", "dispatcher skeleton", "launch handoff: bin/pair"} {
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

func TestDispatchLaunchReportsProcessHandoff(t *testing.T) {
	res := Dispatch([]string{"launch", "--help"})
	if res.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
	}
	if res.Stdout != "" {
		t.Fatalf("Stdout = %q, want empty", res.Stdout)
	}
	for _, want := range []string{"pair-go launch", "process handoff", "cmd/pair-go"} {
		if !strings.Contains(res.Stderr, want) {
			t.Fatalf("Stderr missing %q:\n%s", want, res.Stderr)
		}
	}
}

func TestDispatchContextReturnsHelperOutput(t *testing.T) {
	home, data := writeContextFixture(t)
	t.Setenv("HOME", home)
	t.Setenv("PAIR_DATA_DIR", data)

	res := Dispatch([]string{"context", "T", "claude"})
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stderr:\n%s", res.ExitCode, res.Stderr)
	}
	if res.Stderr != "" {
		t.Fatalf("Stderr = %q, want empty", res.Stderr)
	}
	if strings.TrimSpace(res.Stdout) != "398k" {
		t.Fatalf("Stdout = %q, want 398k", res.Stdout)
	}
}

func TestDispatchScrollbackRenderUsage(t *testing.T) {
	res := Dispatch([]string{"scrollback-render"})
	if res.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
	}
	if res.Stdout != "" {
		t.Fatalf("Stdout = %q, want empty", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "usage: pair-scrollback-render") {
		t.Fatalf("Stderr missing usage:\n%s", res.Stderr)
	}
}

func TestDispatchSlugRoutesToRunner(t *testing.T) {
	// No PAIR_TAG/PAIR_DATA_DIR → slug no-ops and returns 0; it writes only to
	// files, so the buffered Result carries no stdout/stderr.
	t.Setenv("PAIR_TAG", "")
	t.Setenv("PAIR_DATA_DIR", "")
	res := Dispatch([]string{"slug"})
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.Stdout != "" || res.Stderr != "" {
		t.Fatalf("slug route should produce no buffered output; got stdout=%q stderr=%q", res.Stdout, res.Stderr)
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

func writeContextFixture(t *testing.T) (home, data string) {
	t.Helper()
	home = t.TempDir()
	data = filepath.Join(home, "data")
	cwd := filepath.Join(home, "repo")
	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
	proj := filepath.Join(home, ".claude", "projects", enc)
	mustMkdir(t, data)
	mustMkdir(t, cwd)
	mustMkdir(t, proj)
	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
	return home, data
}

func mustMkdir(t *testing.T, d string) {
	t.Helper()
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
