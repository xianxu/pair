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
	for _, want := range []string{"agent", "context", "layout", "scrollback", "wrap", "term", "slug", "changelog", "continuation", "session-inventory", "session-watch", "session-log", "scribe", "review", "clip", "title", "keys"} {
		if !containsStr(names, want) {
			t.Fatalf("DispatchNames() = %v, missing implemented %q", names, want)
		}
	}
	// The handoff family (launch) is NOT a routable subcommand. wrap + scribe
	// became implemented streaming routes in #96 (PTY proxies on #92's seam).
	for _, absent := range []string{"launch"} {
		if containsStr(names, absent) {
			t.Fatalf("DispatchNames() = %v, must not contain non-implemented %q", names, absent)
		}
	}
}

func TestStreamingFlags(t *testing.T) {
	for _, s := range []string{"wrap", "term", "scribe", "changelog render", "continuation", "session-watch", "session-log append", "session-log commit", "title", "clip copy-on-select"} {
		if !IsStreaming(s) {
			t.Errorf("IsStreaming(%q) = false, want true (stdin/live-stderr/long-running)", s)
		}
	}
	for _, b := range []string{"slug", "context", "layout toggle-focused", "scrollback render", "scrollback open", "clip flash-pane"} {
		if IsStreaming(b) {
			t.Errorf("IsStreaming(%q) = true, want false (buffered)", b)
		}
	}
}

func TestResolveNestedFlatAndAlias(t *testing.T) {
	cases := []struct {
		args     []string
		wantName string
		wantRest []string
		wantOK   bool
	}{
		{[]string{"review", "open", "f"}, "review open", []string{"f"}, true},
		{[]string{"agent", "restart"}, "agent restart", []string{}, true},
		{[]string{"review", "definition", "req", "text"}, "review definition", []string{"req", "text"}, true},
		{[]string{"layout", "toggle-focused"}, "layout toggle-focused", []string{}, true},
		{[]string{"scrollback", "render"}, "scrollback render", []string{}, true},
		{[]string{"clip", "copy-on-select", "--orchestrate"}, "clip copy-on-select", []string{"--orchestrate"}, true},
		{[]string{"session-log", "append"}, "session-log append", []string{}, true},
		{[]string{"session-log", "commit"}, "session-log commit", []string{}, true},
		{[]string{"session-inventory", "--json"}, "session-inventory", []string{"--json"}, true},
		{[]string{"context", "T", "claude"}, "context", []string{"T", "claude"}, true},
		{[]string{"scrollback-render"}, "", nil, false}, // the M2 transitional alias is gone (#104 M3)
		{[]string{"changelog"}, "", nil, false},         // bare group token is not a family
		{[]string{"review"}, "", nil, false},            // group token alone is not a family
		{[]string{"frobnicate"}, "", nil, false},
	}
	for _, c := range cases {
		f, rest, ok := Resolve(c.args)
		if ok != c.wantOK || f.Name != c.wantName {
			t.Errorf("Resolve(%v) = (%q, %v), want (%q, %v)", c.args, f.Name, ok, c.wantName, c.wantOK)
			continue
		}
		if ok && len(rest) != len(c.wantRest) {
			t.Errorf("Resolve(%v) rest = %v, want %v", c.args, rest, c.wantRest)
		}
	}
}

func TestDispatchRoutesSessionInventoryUsage(t *testing.T) {
	res := Dispatch([]string{"session-inventory", "--agent", "unsupported"})
	if res.ExitCode != 1 || res.Stdout != "" || !strings.Contains(res.Stderr, "unsupported agent") {
		t.Fatalf("result=%#v", res)
	}
}

func TestDispatchNamesAreTopLevelTokens(t *testing.T) {
	names := DispatchNames()
	for _, want := range []string{"review", "layout", "scrollback", "changelog", "clip", "title"} {
		if !containsStr(names, want) {
			t.Errorf("DispatchNames() = %v, missing group/flat token %q", names, want)
		}
	}
	// Nested leaves are NOT peel-off tokens — the entrypoint only needs the first
	// token to decide dispatch-vs-launch.
	for _, absent := range []string{"review open", "clip copy-on-select"} {
		if containsStr(names, absent) {
			t.Errorf("DispatchNames() = %v, must not contain nested name %q", names, absent)
		}
	}
}

func TestDispatchNestedStreamingRefusesBufferedPath(t *testing.T) {
	// A nested streaming family reached on the buffered Dispatch path is a
	// programming error (it should go through cmd/pair-go's streaming seam).
	for _, args := range [][]string{{"clip", "copy-on-select"}, {"changelog", "render"}, {"session-log", "append"}, {"session-log", "commit"}} {
		res := Dispatch(args)
		if res.ExitCode != 2 || !strings.Contains(res.Stderr, "streaming subcommand") {
			t.Errorf("Dispatch(%v) = code %d stderr %q, want 2 + 'streaming subcommand'", args, res.ExitCode, res.Stderr)
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
				"scrollback render",
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

// wrap/term are implemented STREAMING subcommands: the buffered Dispatch path must
// refuse them (real stdio is required — they're routed via cmd/pair-go's streaming
// seam, not Dispatch). scribe behaves the same way.
func TestDispatchStreamingCommandRefusesBufferedPath(t *testing.T) {
	for _, name := range []string{"wrap", "term", "scribe"} {
		res := Dispatch([]string{name})
		if res.ExitCode != 2 {
			t.Fatalf("%s: ExitCode = %d, want 2", name, res.ExitCode)
		}
		if res.Stdout != "" {
			t.Fatalf("%s: Stdout = %q, want empty", name, res.Stdout)
		}
		for _, want := range []string{name, "streaming subcommand", "streaming seam"} {
			if !strings.Contains(res.Stderr, want) {
				t.Fatalf("%s: Stderr missing %q:\n%s", name, want, res.Stderr)
			}
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
	t.Setenv("PAIR_SCOPE_KEY", "scope")

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
	res := Dispatch([]string{"scrollback", "render"})
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
	const sessionID = "11111111-1111-4111-8111-111111111111"
	home = t.TempDir()
	data = filepath.Join(home, "data")
	cwd := filepath.Join(home, "repo")
	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
	proj := filepath.Join(home, ".claude", "projects", enc)
	mustMkdir(t, data)
	mustMkdir(t, cwd)
	mustMkdir(t, proj)
	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
	mustWrite(t, filepath.Join(data, "ledger-T.jsonl"),
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"T","agent":"claude","pair_log_offset":0,"native_watermarks":[]}`+"\n"+
			`{"v":1,"kind":"binding","scope_key":"scope","tag":"T","agent":"claude","launch_ordinal":1,"root_native_id":"`+sessionID+`"}`+"\n")
	mustWrite(t, filepath.Join(proj, sessionID+".jsonl"),
		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`+"\n")
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

// Alt+h → bin/pair-help → `pair keys`. If "keys" ever fell out of DispatchNames(),
// the peel-off would miss and `pair keys` would fall through to the launcher —
// starting a whole session inside the floating help pane (#132).
func TestDispatchKeysReturnsKeybindings(t *testing.T) {
	res := Dispatch([]string{"keys"})
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stderr:\n%s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "send buffer + clear") {
		t.Errorf("Stdout missing a real binding:\n%s", res.Stdout)
	}
	if strings.Contains(res.Stdout, "keybindings are on Alt+h") {
		t.Error("routed to the CLI synopsis instead of the keybindings")
	}
}
