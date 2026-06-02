package main

import (
	"strings"
	"testing"
)

func TestNormalizeBranch(t *testing.T) {
	cases := []struct {
		branch, repoBase, want string
	}{
		{"main", "pair", "pair"},
		{"master", "pair", "pair"},
		{"HEAD", "pair", "pair"},
		{"", "pair", "pair"},
		{"winbar-recap", "pair", "winbar-recap"},
		{"feature/winbar-recap", "pair", "winbar-recap"},
		{"xx/winbar-recap", "pair", "winbar-recap"},
		{"42-winbar-recap", "pair", "#42 winbar-recap"},
		{"000027-auto-orientation-slug", "pair", "#000027 auto-orientation-slug"},
		// the edge the judge flagged: prefix AND embedded number
		{"xx/42-winbar-recap", "pair", "#42 winbar-recap"},
		{"feature/123_some_thing", "pair", "#123 some_thing"},
		// I-A: pipe is git-legal but would break the single-pipe channel → sanitized
		{"feat|wip", "pair", "feat/wip"},
		{"feature/a|b", "pair", "a/b"},
	}
	for _, c := range cases {
		if got := normalizeBranch(c.branch, c.repoBase); got != c.want {
			t.Errorf("normalizeBranch(%q,%q) = %q, want %q", c.branch, c.repoBase, got, c.want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	good := []string{
		"=== pair | doing tests ===",
		"=== #42 winbar-recap | testing slug gate ===",
	}
	bad := []string{
		"KEEP",
		"=== pair ===",           // no pipe
		"=== | right ===",        // empty left
		"=== left | ===",         // empty right
		"pair | doing tests",     // no fence
		"Sandbox restriction...", // hijack garbage
		"",
	}
	for _, s := range good {
		if !validateSlug(s) {
			t.Errorf("validateSlug(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if validateSlug(s) {
			t.Errorf("validateSlug(%q) = true, want false", s)
		}
	}
}

func TestExtractTurns(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"first prompt"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"reply one"}]}}`,
		// tool-only assistant turn: no text → skipped
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]}}`,
		`{"type":"summary","message":{"role":"assistant","content":"ignored non-user/assistant type"}}`,
		`{"type":"user","message":{"role":"user","content":"second prompt"}}`,
		``, // blank line tolerated
	}, "\n")

	turns := windowTurns(parseClaude([]byte(jsonl)), 10, 0, 40, 500)
	if len(turns) != 3 {
		t.Fatalf("got %d turns, want 3: %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "first prompt" {
		t.Errorf("turn0 = %+v", turns[0])
	}
	if turns[1].Text != "reply one" {
		t.Errorf("turn1 = %+v", turns[1])
	}
	if turns[2].Text != "second prompt" {
		t.Errorf("turn2 = %+v", turns[2])
	}
}

func TestExtractTurnsTrim(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, `{"type":"user","message":{"role":"user","content":"p"}}`)
	}
	turns := windowTurns(parseClaude([]byte(strings.Join(lines, "\n"))), 5, 0, 40, 500)
	if len(turns) != 5 {
		t.Fatalf("got %d, want last 5", len(turns))
	}
}

// TestSelectWindowUserBias is the C1 regression: a flat tail is all
// assistant; the window must extend back to include user turns.
func TestSelectWindowUserBias(t *testing.T) {
	all := []turn{{Role: "user", Text: "the real intent"}}
	for i := 0; i < 15; i++ {
		all = append(all, turn{Role: "assistant", Text: "narration"})
	}
	// recentTurns=10 → last 10 are all assistant; minUser=1 forces extension.
	win := selectWindow(all, 10, 1, 40)
	if countUser(win) < 1 {
		t.Fatalf("window has no user turn: %d turns, %d users", len(win), countUser(win))
	}
	if win[0].Role != "user" || win[0].Text != "the real intent" {
		t.Errorf("expected the user intent at window start, got %+v", win[0])
	}
}

func TestSelectWindowHardMax(t *testing.T) {
	// 50 assistant turns, no user → extension is capped at hardMax.
	var all []turn
	for i := 0; i < 50; i++ {
		all = append(all, turn{Role: "assistant", Text: "x"})
	}
	win := selectWindow(all, 10, 3, 20)
	if len(win) > 20 {
		t.Errorf("window %d exceeds hardMax 20", len(win))
	}
}

// I2: a user turn older than hardMax must still be anchored into the window —
// otherwise a long autonomous stretch yields a 100%-assistant window and the
// slug narrates the agent instead of the user's intent.
func TestSelectWindowAnchorsUserBeyondHardMax(t *testing.T) {
	all := []turn{{Role: "user", Text: "the original intent"}}
	for i := 0; i < 40; i++ {
		all = append(all, turn{Role: "assistant", Text: "narration"})
	}
	// recentTurns=10, minUser=3, hardMax=20 → extension caps at 20, all
	// assistant; the anchor must still pull in the lone user turn at index 0.
	win := selectWindow(all, 10, 3, 20)
	if countUser(win) != 1 || win[0].Text != "the original intent" {
		t.Fatalf("anchor failed: %d users, first=%q", countUser(win), win[0].Text)
	}
}

func TestSelectWindowNoUserAtAll(t *testing.T) {
	// Pathological: zero user turns anywhere → anchor loop finds none, no panic,
	// returns the capped window unchanged.
	var all []turn
	for i := 0; i < 30; i++ {
		all = append(all, turn{Role: "assistant", Text: "x"})
	}
	win := selectWindow(all, 10, 3, 20)
	if len(win) == 0 || len(win) > 20 {
		t.Fatalf("unexpected window len %d", len(win))
	}
}

// TestExtractTurnsKeepsUserIntent exercises the real-transcript shape: many
// recent assistant turns + tool_result-only user entries (no text, dropped),
// with the genuine user prompt further back. The window must still surface it.
func TestExtractTurnsKeepsUserIntent(t *testing.T) {
	var lines []string
	add := func(s string) { lines = append(lines, s) }
	add(`{"type":"user","message":{"role":"user","content":"fix the winbar padding bug"}}`)
	for i := 0; i < 12; i++ {
		add(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"working on it"}]}}`)
		// tool_result-only user turn — array content, no text block → dropped
		add(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}`)
	}
	turns := windowTurns(parseClaude([]byte(strings.Join(lines, "\n"))), 10, 1, 40, 500)
	found := false
	for _, tn := range turns {
		if tn.Role == "user" && tn.Text == "fix the winbar padding bug" {
			found = true
		}
	}
	if !found {
		t.Fatalf("user intent dropped from window of %d turns (%d users)", len(turns), countUser(turns))
	}
}

func TestExtractTurnsTruncate(t *testing.T) {
	long := strings.Repeat("x", 1000)
	line := `{"type":"user","message":{"role":"user","content":"` + long + `"}}`
	turns := windowTurns(parseClaude([]byte(line)), 10, 0, 40, 50)
	if len(turns) != 1 || len(turns[0].Text) != 50 {
		t.Fatalf("truncate failed: len=%d", len(turns[0].Text))
	}
}

func TestParseCodex(t *testing.T) {
	jsonl := strings.Join([]string{
		`{"type":"session_meta","payload":{"cwd":"/x"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"system init — skip"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"fix the bug"}]}}`,
		`{"type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{}"}}`,
		`{"type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"ok"}}`,
		`{"type":"response_item","payload":{"type":"reasoning"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"on it"}]}}`,
		`{"type":"event_msg","payload":{"type":"token_count"}}`,
	}, "\n")
	turns := parseCodex([]byte(jsonl))
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2 (user+assistant): %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "fix the bug" {
		t.Errorf("turn0 = %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Text != "on it" {
		t.Errorf("turn1 = %+v", turns[1])
	}
}

func TestParseGemini(t *testing.T) {
	doc := `{"sessionId":"abc","messages":[
		{"type":"user","content":[{"text":"why amd64 here?"}]},
		{"type":"gemini","content":"I'm in a linux sandbox","toolCalls":[{"name":"shell"}]},
		{"type":"info","content":"Request cancelled."},
		{"type":"user","content":[{"text":"part one"},{"text":"part two"}]}
	]}`
	turns := parseGemini([]byte(doc))
	if len(turns) != 3 {
		t.Fatalf("got %d turns, want 3 (user, gemini, user; info skipped): %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "why amd64 here?" {
		t.Errorf("turn0 = %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Text != "I'm in a linux sandbox" {
		t.Errorf("turn1 (gemini→assistant) = %+v", turns[1])
	}
	if turns[2].Text != "part one part two" {
		t.Errorf("turn2 (multi-part user) = %+v", turns[2])
	}
}

func TestParseAgy(t *testing.T) {
	doc := strings.Join([]string{
		`{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","content":"<USER_REQUEST>\nexplain the bug\n</USER_REQUEST>\n<METADATA>...</METADATA>"}`,
		`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","content":"I will look into it"}`,
		`{"step_index":2,"source":"SYSTEM","type":"CONVERSATION_HISTORY","status":"DONE"}`,
	}, "\n")
	turns := parseAgy([]byte(doc))
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2 (user+assistant): %+v", len(turns), turns)
	}
	if turns[0].Role != "user" || turns[0].Text != "explain the bug" {
		t.Errorf("turn0 = %+v", turns[0])
	}
	if turns[1].Role != "assistant" || turns[1].Text != "I will look into it" {
		t.Errorf("turn1 = %+v", turns[1])
	}
}


func TestParseTranscriptDispatch(t *testing.T) {
	// claude is the default/fallback parser
	claude := `{"type":"user","message":{"role":"user","content":"hi"}}`
	if got := parseTranscript("claude", []byte(claude)); len(got) != 1 || got[0].Text != "hi" {
		t.Errorf("claude dispatch: %+v", got)
	}
	agy := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","content":"<USER_REQUEST>\nhi\n</USER_REQUEST>"}`
	if got := parseTranscript("agy", []byte(agy)); len(got) != 1 || got[0].Text != "hi" {
		t.Errorf("agy dispatch: %+v", got)
	}
	if got := parseTranscript("unknown-agent", []byte(claude)); len(got) != 1 {
		t.Errorf("unknown agent should fall back to claude parser: %+v", got)
	}
}

func TestDecide(t *testing.T) {
	const left = "#42 winbar-recap"

	t.Run("valid new stomps left with branch", func(t *testing.T) {
		// model returns a different left; we must override it with branchLeft.
		write, val := decide(left, "=== old | prev focus ===", "=== wrong-left | testing gate ===")
		if !write {
			t.Fatal("want write")
		}
		if val != "=== #42 winbar-recap | testing gate ===" {
			t.Errorf("got %q", val)
		}
	})

	t.Run("KEEP, branch changed → write (left refreshes, focus kept)", func(t *testing.T) {
		w, v := decide(left, "=== oldbranch | y ===", "KEEP")
		if !w || v != "=== #42 winbar-recap | y ===" {
			t.Errorf("KEEP+branch-change: write=%v val=%q", w, v)
		}
	})

	t.Run("KEEP, same branch + focus → no write", func(t *testing.T) {
		if w, _ := decide(left, "=== #42 winbar-recap | y ===", "KEEP"); w {
			t.Error("KEEP with unchanged branch+focus must not write")
		}
	})

	t.Run("invalid → no write (keep last)", func(t *testing.T) {
		if w, _ := decide(left, "=== x | y ===", "Sandbox restriction. Let me ask..."); w {
			t.Error("invalid output must not write")
		}
	})

	t.Run("cold start: no prev, valid → write", func(t *testing.T) {
		write, val := decide(left, "", "=== ignored | first focus ===")
		if !write || val != "=== #42 winbar-recap | first focus ===" {
			t.Errorf("cold start write=%v val=%q", write, val)
		}
	})

	t.Run("cold start: no prev, KEEP → no write", func(t *testing.T) {
		if w, _ := decide(left, "", "KEEP"); w {
			t.Error("KEEP with no prev must not write")
		}
	})

	t.Run("preamble before slug → uses last line", func(t *testing.T) {
		write, val := decide(left, "", "here you go:\n=== ignored | the focus ===")
		if !write || val != "=== #42 winbar-recap | the focus ===" {
			t.Errorf("preamble handling: write=%v val=%q", write, val)
		}
	})

	t.Run("value == prev → no write (unchanged)", func(t *testing.T) {
		prev := "=== #42 winbar-recap | testing gate ==="
		if w, _ := decide(left, prev, "=== x | testing gate ==="); w {
			t.Error("identical computed value must not write")
		}
	})

	t.Run("focus containing | → no write", func(t *testing.T) {
		if w, _ := decide(left, "", "=== a | b | c ==="); w {
			t.Error("focus with a pipe must be rejected")
		}
	})

	t.Run("focus containing === → no write", func(t *testing.T) {
		if w, _ := decide(left, "", "=== a | b === c ==="); w {
			t.Error("focus with === must be rejected")
		}
	})
}

func TestRightOf(t *testing.T) {
	if got := rightOf("=== a | b c d ==="); got != "b c d" {
		t.Errorf("rightOf = %q", got)
	}
}
