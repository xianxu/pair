package launcher

import (
	"reflect"
	"testing"
)

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"a", "--session-id", "b"}, "--session-id") {
		t.Error("should find --session-id")
	}
	if hasFlag([]string{"a", "b"}, "--session-id") {
		t.Error("should not find absent flag")
	}
}

func TestStripValuelessFlag(t *testing.T) {
	got := stripValuelessFlag([]string{"resume", "x", "--no-alt-screen", "--search"}, "--no-alt-screen")
	want := []string{"resume", "x", "--search"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStripFlagAllForms(t *testing.T) {
	// space form: removes the flag AND its following value.
	got := stripFlagAllForms([]string{"--resume", "abc", "--search", "--session-id", "uuid"}, "--session-id")
	if want := []string{"--resume", "abc", "--search"}; !reflect.DeepEqual(got, want) {
		t.Errorf("space form: got %v, want %v", got, want)
	}
	// inline form: removes flag=value.
	if got := stripFlagAllForms([]string{"--conversation=xyz", "--search"}, "--conversation"); !reflect.DeepEqual(got, []string{"--search"}) {
		t.Errorf("inline form: got %v", got)
	}
	// a trailing space-form flag with no value is dropped without panicking.
	if got := stripFlagAllForms([]string{"a", "--session-id"}, "--session-id"); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("trailing flag: got %v", got)
	}
}

func TestStripCodexResumeSubcommand(t *testing.T) {
	// leading `resume <id>` is dropped (position-sensitive).
	if got := stripCodexResumeSubcommand([]string{"resume", "id1", "--no-alt-screen"}); !reflect.DeepEqual(got, []string{"--no-alt-screen"}) {
		t.Errorf("leading resume: got %v", got)
	}
	// Codex accepts global options before the command: codex [OPTIONS] resume <id>.
	if got := stripCodexResumeSubcommand([]string{"--sandbox", "danger-full-access", "resume", "id1", "--no-alt-screen"}); !reflect.DeepEqual(got, []string{"--sandbox", "danger-full-access", "--no-alt-screen"}) {
		t.Errorf("global options before resume: got %v", got)
	}
	// A prompt containing "resume" is not a resume subcommand.
	if got := stripCodexResumeSubcommand([]string{"please", "resume", "id1"}); !reflect.DeepEqual(got, []string{"please", "resume", "id1"}) {
		t.Errorf("prompt resume must stay: got %v", got)
	}
}

func TestResumeTokenPerAgent(t *testing.T) {
	cases := []struct {
		agent, sid string
		want       []string
	}{
		{"claude", "s1", []string{"--resume", "s1"}},
		{"codex", "s1", []string{"resume", "s1"}},
		{"agy", "s1", []string{"--conversation", "s1"}},
		{"claude", "", nil},
		{"unknown", "s1", nil},
	}
	for _, c := range cases {
		if got := resumeToken(c.agent, c.sid); !reflect.DeepEqual(got, c.want) {
			t.Errorf("resumeToken(%q,%q) = %v, want %v", c.agent, c.sid, got, c.want)
		}
	}
}

func TestExtractExplicitResumeCodexAllowsGlobalOptionsBeforeCommand(t *testing.T) {
	got := extractExplicitResume("codex", []string{"--sandbox", "danger-full-access", "resume", "sid-1", "--no-alt-screen"})
	if got != "sid-1" {
		t.Fatalf("extractExplicitResume codex with globals = %q, want sid-1", got)
	}
	if got := extractExplicitResume("codex", []string{"please", "resume", "sid-1"}); got != "" {
		t.Fatalf("prompt text should not be treated as explicit resume, got %q", got)
	}
}

// Codex's `resume` subcommand must lead (args[0]); claude's --resume can trail.
func TestComposeResumeArgsOrdering(t *testing.T) {
	if got := composeResumeArgs("codex", []string{"--no-alt-screen"}, "sid"); !reflect.DeepEqual(got, []string{"resume", "sid", "--no-alt-screen"}) {
		t.Errorf("codex resume must lead: %v", got)
	}
	if got := composeResumeArgs("claude", []string{"--search"}, "sid"); !reflect.DeepEqual(got, []string{"--search", "--resume", "sid"}) {
		t.Errorf("claude resume trails: %v", got)
	}
	if got := composeResumeArgs("claude", []string{"--search"}, ""); !reflect.DeepEqual(got, []string{"--search"}) {
		t.Errorf("no sid → saved args unchanged: %v", got)
	}
}

// Named case for the idempotence behavior a port silently breaks (judge INFO #3).
func TestCodexAltScreenIdempotent(t *testing.T) {
	// Appends when absent.
	if got := codexAltScreenArgs([]string{"resume", "x"}, false); !reflect.DeepEqual(got, []string{"resume", "x", "--no-alt-screen"}) {
		t.Errorf("append when absent: %v", got)
	}
	// Idempotent: an existing --no-alt-screen is stripped before re-appending (no dup).
	if got := codexAltScreenArgs([]string{"resume", "x", "--no-alt-screen"}, false); !reflect.DeepEqual(got, []string{"resume", "x", "--no-alt-screen"}) {
		t.Errorf("no duplicate on re-apply: %v", got)
	}
	// Opt-out strips it and does not re-append.
	if got := codexAltScreenArgs([]string{"resume", "x", "--no-alt-screen"}, true); !reflect.DeepEqual(got, []string{"resume", "x"}) {
		t.Errorf("opt-out strips: %v", got)
	}
}

// Named case for the claude --session-id mint/skip decision (judge INFO #3).
func TestShouldMintClaudeSessionID(t *testing.T) {
	if !shouldMintClaudeSessionID("claude", "", nil) {
		t.Error("fresh claude with no resume/flags → mint")
	}
	if shouldMintClaudeSessionID("codex", "", nil) {
		t.Error("codex has no --session-id flag → never mint")
	}
	if shouldMintClaudeSessionID("claude", "resumed-sid", nil) {
		t.Error("explicit resume already pinned → skip")
	}
	if shouldMintClaudeSessionID("claude", "", []string{"--session-id", "u"}) {
		t.Error("user passed --session-id → their uuid wins, skip")
	}
	if shouldMintClaudeSessionID("claude", "", []string{"--fork-session"}) {
		t.Error("--fork-session → claude allocates internally, skip")
	}
}

// Every agent's resume binding is stripped before persisting — the claude subset
// AND agy --conversation (both forms) AND codex leading `resume <id>` — so a
// resume can't compound in the saved args on relaunch (review I1).
func TestPersistedConfigArgsStripsBinding(t *testing.T) {
	// claude subset.
	if got := persistedConfigArgs([]string{"--search", "--session-id", "u", "--resume", "r", "--no-alt-screen"}); !reflect.DeepEqual(got, []string{"--search", "--no-alt-screen"}) {
		t.Errorf("claude: got %v", got)
	}
	// agy --conversation, space + inline forms.
	if got := persistedConfigArgs([]string{"--conversation", "cid", "--search"}); !reflect.DeepEqual(got, []string{"--search"}) {
		t.Errorf("agy space form: got %v", got)
	}
	if got := persistedConfigArgs([]string{"--conversation=cid", "--search"}); !reflect.DeepEqual(got, []string{"--search"}) {
		t.Errorf("agy inline form: got %v", got)
	}
	// codex leading `resume <id>` (position-sensitive) + trailing saved flags kept.
	if got := persistedConfigArgs([]string{"resume", "sid", "--no-alt-screen"}); !reflect.DeepEqual(got, []string{"--no-alt-screen"}) {
		t.Errorf("codex resume subcommand: got %v", got)
	}
	// codex global options may precede the resume command.
	if got := persistedConfigArgs([]string{"--sandbox", "danger-full-access", "resume", "sid", "--no-alt-screen"}); !reflect.DeepEqual(got, []string{"--sandbox", "danger-full-access", "--no-alt-screen"}) {
		t.Errorf("codex global-options resume subcommand: got %v", got)
	}
}
