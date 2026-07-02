package launcher

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractExplicitResume(t *testing.T) {
	cases := []struct {
		name  string
		agent string
		args  []string
		want  string
	}{
		{"claude space form", "claude", []string{"--model", "x", "--resume", "sid-1"}, "sid-1"},
		{"claude none", "claude", []string{"--model", "x"}, ""},
		{"agy space form", "agy", []string{"--conversation", "conv-9"}, "conv-9"},
		{"agy inline form", "agy", []string{"--conversation=conv-inline"}, "conv-inline"},
		{"agy legacy resume", "agy", []string{"--resume", "old-a"}, "old-a"},
		{"codex leading resume", "codex", []string{"resume", "cx-2", "--search"}, "cx-2"},
		{"codex resume not leading", "codex", []string{"--search", "resume", "cx-2"}, ""},
		{"codex resume no id", "codex", []string{"resume"}, ""},
		{"unknown agent", "gemini", []string{"--resume", "z"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractExplicitResume(tc.agent, tc.args); got != tc.want {
				t.Fatalf("extractExplicitResume(%q, %v) = %q, want %q", tc.agent, tc.args, got, tc.want)
			}
		})
	}
}

func TestBuildConfigJSON(t *testing.T) {
	got, err := buildConfigJSON("claude", []string{"--model", "opus"}, "sid-abc")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"agent":"claude","args":["--model","opus"],"session_id":"sid-abc"}` + "\n"
	if got != want {
		t.Fatalf("buildConfigJSON = %q, want %q", got, want)
	}
	// Round-trips through parseConfig.
	cfg, err := parseConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "claude" || cfg.SessionID != "sid-abc" || !reflect.DeepEqual(cfg.Args, []string{"--model", "opus"}) {
		t.Fatalf("parseConfig round-trip mismatch: %+v", cfg)
	}
}

func TestBuildConfigJSONNilArgsAndNoHTMLEscape(t *testing.T) {
	// nil args serialize as [] (jq's $ARGS.positional is never null).
	got, _ := buildConfigJSON("codex", nil, "")
	if !strings.Contains(got, `"args":[]`) {
		t.Fatalf("nil args should serialize as []: %q", got)
	}
	// A value with < > & stays LITERAL (jq / vim.json don't \u-escape it), so
	// a Go-written config is byte-compatible with the shell readers.
	got2, _ := buildConfigJSON("claude", []string{"a<b>&c"}, "")
	// Escaping OFF keeps the markup literal; the escaped form < must be absent.
	if !strings.Contains(got2, "a<b>&c") || strings.Contains(got2, "\\u003c") {
		t.Fatalf("expected literal (un-escaped) markup, got: %q", got2)
	}
}

func TestBuildConfigChoices(t *testing.T) {
	// Resumable + differing args → all four rows, numbered 1..4.
	choices := buildConfigChoices(true, []string{"--saved"}, []string{"--new"}, "sess-x")
	gotActions := actionsOf(choices)
	wantActions := []string{"saved+resume", "saved", "new+resume", "new"}
	if !reflect.DeepEqual(gotActions, wantActions) {
		t.Fatalf("actions = %v, want %v", gotActions, wantActions)
	}
	if !strings.HasPrefix(choices[0].Label, "1) use saved params + session") {
		t.Fatalf("choice 0 label = %q", choices[0].Label)
	}
	if !strings.Contains(choices[0].Label, "resume=sess-x") {
		t.Fatalf("choice 0 should carry the session id: %q", choices[0].Label)
	}

	// Args match → the "new" rows collapse away (byte-identical launches).
	matched := buildConfigChoices(true, []string{"--same"}, []string{"--same"}, "sess-x")
	if got := actionsOf(matched); !reflect.DeepEqual(got, []string{"saved+resume", "saved"}) {
		t.Fatalf("matched actions = %v", got)
	}

	// Not resumable → no resume rows; but empty new args still DIFFER from the
	// saved args, so the "drop the saved params" row (action "new") is offered.
	noresume := buildConfigChoices(false, []string{"--saved"}, nil, "sess-x")
	if got := actionsOf(noresume); !reflect.DeepEqual(got, []string{"saved", "new"}) {
		t.Fatalf("noresume actions = %v", got)
	}
	// Not resumable + matching args → the single "saved" row (nothing to choose).
	single := buildConfigChoices(false, []string{"--x"}, []string{"--x"}, "s")
	if got := actionsOf(single); !reflect.DeepEqual(got, []string{"saved"}) {
		t.Fatalf("single actions = %v", got)
	}
}

func TestSelectAction(t *testing.T) {
	choices := buildConfigChoices(true, []string{"--saved"}, []string{"--new"}, "s")
	if got := selectAction(choices, choices[2].Label); got != "new+resume" {
		t.Fatalf("selectAction = %q, want new+resume", got)
	}
	if got := selectAction(choices, "no such label"); got != "" {
		t.Fatalf("selectAction(unknown) = %q, want empty", got)
	}
}

func TestComposeTagRestartArgs(t *testing.T) {
	saved := []string{"--search"}
	newArgs := []string{"--model", "x"}

	// saved → cleaned saved args verbatim.
	if got := composeTagRestartArgs("saved", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, saved) {
		t.Fatalf("saved = %v", got)
	}
	// new → typed args verbatim.
	if got := composeTagRestartArgs("new", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, newArgs) {
		t.Fatalf("new = %v", got)
	}
	// saved+resume (claude) → saved args + --resume <sid> (flag trails).
	if got := composeTagRestartArgs("saved+resume", "claude", saved, newArgs, "sid"); !reflect.DeepEqual(got, []string{"--search", "--resume", "sid"}) {
		t.Fatalf("claude saved+resume = %v", got)
	}
	// saved+resume (codex) → resume subcommand LEADS args[0..1].
	if got := composeTagRestartArgs("saved+resume", "codex", []string{"--no-alt-screen"}, nil, "cx"); !reflect.DeepEqual(got, []string{"resume", "cx", "--no-alt-screen"}) {
		t.Fatalf("codex saved+resume = %v", got)
	}
	// new+resume strips a resume the user re-typed before appending the canonical one.
	if got := composeTagRestartArgs("new+resume", "claude", saved, []string{"--resume", "stale", "--flag"}, "sid"); !reflect.DeepEqual(got, []string{"--flag", "--resume", "sid"}) {
		t.Fatalf("new+resume = %v", got)
	}
	// agy resume binds via --conversation.
	if got := composeTagRestartArgs("saved+resume", "agy", nil, nil, "conv"); !reflect.DeepEqual(got, []string{"--conversation", "conv"}) {
		t.Fatalf("agy saved+resume = %v", got)
	}
}

func actionsOf(choices []configChoice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Action
	}
	return out
}
