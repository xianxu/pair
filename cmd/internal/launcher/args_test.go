package launcher

import (
	"strings"
	"testing"
)

func TestParseLaunchArgsDefaultsToClaude(t *testing.T) {
	args, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", args.Agent)
	}
	if args.ForcedTag != "" {
		t.Fatalf("ForcedTag = %q, want empty", args.ForcedTag)
	}
	if len(args.AgentArgs) != 0 {
		t.Fatalf("AgentArgs = %#v, want empty", args.AgentArgs)
	}
}

func TestParseLaunchArgsAgentAndForwardedArgs(t *testing.T) {
	args, err := ParseArgs([]string{"codex", "--", "-p", "say hi"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "codex" {
		t.Fatalf("Agent = %q, want codex", args.Agent)
	}
	if got := strings.Join(args.AgentArgs, " "); got != "-p say hi" {
		t.Fatalf("AgentArgs = %q, want forwarded args", got)
	}
}

func TestParseLaunchArgsDefaultAgentWithForwardedArgs(t *testing.T) {
	args, err := ParseArgs([]string{"--", "--dangerously-skip-permissions"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", args.Agent)
	}
	if got := strings.Join(args.AgentArgs, " "); got != "--dangerously-skip-permissions" {
		t.Fatalf("AgentArgs = %q, want forwarded args", got)
	}
}

func TestParseLaunchArgsResumeNormalizesForcedTag(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "pair-demo"})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if args.Agent != "" {
		t.Fatalf("Agent = %q, want empty for resume inference", args.Agent)
	}
	if args.ForcedTag != "demo" {
		t.Fatalf("ForcedTag = %q, want demo", args.ForcedTag)
	}
}

func TestParseLaunchArgsUnexpectedPositionalGuidesAgentArgs(t *testing.T) {
	_, err := ParseArgs([]string{"codex", "extra"})
	if err == nil {
		t.Fatal("ParseArgs returned nil error")
	}
	msg := err.Error()
	for _, want := range []string{"unexpected positional arg", "use '--' to forward args to the agent"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error missing %q: %s", want, msg)
		}
	}
}

// A leading flag in the agent position (pair --help / -h) is not an agent — it
// errors so LaunchNative falls back to the shell (which owns help/flags). #99 M4.
func TestParseLaunchArgsLeadingFlagIsNotAnAgent(t *testing.T) {
	// -h/--help/help are native help (#99 M5c) — a distinct command, not an error.
	for _, flag := range []string{"-h", "--help", "help"} {
		if got, err := ParseArgs([]string{flag}); err != nil || got.Command != "help" {
			t.Fatalf("ParseArgs(%q) = (%+v, %v), want the help command", flag, got, err)
		}
	}
	// Any OTHER leading flag is still a usage error (not an agent name).
	for _, flag := range []string{"--version", "-x"} {
		t.Run(flag, func(t *testing.T) {
			_, err := ParseArgs([]string{flag})
			if err == nil {
				t.Fatalf("ParseArgs(%q) returned nil error; a leading flag must not be an agent", flag)
			}
			if !strings.Contains(err.Error(), "not an agent") {
				t.Fatalf("error = %q, want the leading-flag message", err)
			}
		})
	}
	// A flag AFTER the -- separator is a legitimate agent arg, not an error.
	if _, err := ParseArgs([]string{"claude", "--", "--help"}); err != nil {
		t.Fatalf("flag after -- should be an agent arg, got err %v", err)
	}
}

// As of #99 M5b all launcher subcommands (list/ls, rename, continue) parse
// natively — no ParseArgs verb falls back to the shell anymore; only a leading
// flag (--help) does (TestParseLaunchArgsLeadingFlagIsNotAnAgent). The parse
// contract for each native verb is pinned by TestParseRename / TestParseContinue
// / TestParseListIsNative.

// list/ls parse to the read-only list command marker (#99 M5a), no longer a
// shell-fallback error.
func TestParseLaunchArgsListIsNative(t *testing.T) {
	for _, verb := range []string{"list", "ls"} {
		t.Run(verb, func(t *testing.T) {
			got, err := ParseArgs([]string{verb})
			if err != nil {
				t.Fatalf("ParseArgs(%q) error = %v, want nil", verb, err)
			}
			if got.Command != "list" {
				t.Fatalf("Command = %q, want %q", got.Command, "list")
			}
		})
	}
}

// restart parses `restart [--new-session] [--rename-to <tag>]` (#94 M1, ported
// from bin/pair-restart.sh). Explicit case — without it, restart falls through
// the general loop and becomes an agent name (Agent="restart"), the silent trap.
func TestParseRestart(t *testing.T) {
	cases := []struct {
		name       string
		argv       []string
		newSession bool
		renameTo   string
		wantErr    bool
	}{
		{"bare", []string{"restart"}, false, "", false},
		{"new-session", []string{"restart", "--new-session"}, true, "", false},
		{"rename-to", []string{"restart", "--rename-to", "foo"}, false, "foo", false},
		{"new-session+rename", []string{"restart", "--new-session", "--rename-to", "foo"}, true, "foo", false},
		{"rename-to missing value", []string{"restart", "--rename-to"}, false, "", true},
		{"rename-to empty value", []string{"restart", "--rename-to", ""}, false, "", true},
		{"unknown arg", []string{"restart", "--bogus"}, false, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseArgs(tc.argv)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Command != "restart" {
				t.Fatalf("Command = %q, want restart", got.Command)
			}
			if got.NewSession != tc.newSession || got.RenameTo != tc.renameTo {
				t.Fatalf("got {NewSession:%v RenameTo:%q}, want {%v %q}", got.NewSession, got.RenameTo, tc.newSession, tc.renameTo)
			}
			if got.Agent != "" {
				t.Fatalf("Agent = %q, want empty (restart is not an agent)", got.Agent)
			}
		})
	}
}

// quit parses to the quit command marker (#94 M1, ported from bin/pair-quit.sh).
func TestParseQuit(t *testing.T) {
	got, err := ParseArgs([]string{"quit"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Command != "quit" {
		t.Fatalf("Command = %q, want quit", got.Command)
	}
	if got.Agent != "" {
		t.Fatalf("Agent = %q, want empty (quit is not an agent)", got.Agent)
	}
}
