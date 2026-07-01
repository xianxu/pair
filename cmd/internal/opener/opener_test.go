package opener

import (
	"fmt"
	"strings"
	"testing"
)

func TestStripSGR(t *testing.T) {
	in := "\x1b[38;5;75mhello\x1b[0m \x1b[1mworld\x1b[22m"
	if got := stripSGR(in); got != "hello world" {
		t.Fatalf("stripSGR = %q", got)
	}
}

// ansiFixture builds 20 distinct, substantial (≥8-char) lines.
func ansiFixture() []string {
	a := make([]string, 20)
	for i := range a {
		a[i] = fmt.Sprintf("line-%02d-content", i)
	}
	return a
}

func TestMatchViewportHighConfidence(t *testing.T) {
	ansi := ansiFixture()
	dump := ansi[10:15] // a 5-line window starting at ansi index 10 (1-based line 11)
	got, ok := matchViewport(dump, ansi)
	if !ok || got != 11 {
		t.Fatalf("matchViewport = (%d,%v), want (11,true)", got, ok)
	}
}

func TestMatchViewportSubThresholdRejects(t *testing.T) {
	ansi := ansiFixture()
	dump := []string{ansi[10], "zzzzzzzzzz", "yyyyyyyyyy", "wwwwwwwwww", "vvvvvvvvvv"} // 1/5 match
	if got, ok := matchViewport(dump, ansi); ok {
		t.Fatalf("matchViewport = (%d,true), want reject (< 50%% match)", got)
	}
}

func TestMatchViewportTopClamp(t *testing.T) {
	ansi := ansiFixture()
	// Two blank leading lines then the top of the buffer → computed start < 1.
	dump := []string{"", "", ansi[0], ansi[1], ansi[2]}
	got, ok := matchViewport(dump, ansi)
	if !ok || got != 1 {
		t.Fatalf("matchViewport = (%d,%v), want (1,true) clamp", got, ok)
	}
}

func TestMatchViewportEmpty(t *testing.T) {
	if _, ok := matchViewport(nil, ansiFixture()); ok {
		t.Fatal("empty dump must not match")
	}
	if _, ok := matchViewport([]string{"", "  "}, ansiFixture()); ok {
		t.Fatal("all-blank dump must not match")
	}
}

func TestResolveSessionID(t *testing.T) {
	const A = "aaaa1111-2222-3333-4444-555566667777"
	const C = "cccc1111-2222-3333-4444-555566667777"
	// env wins
	if got := resolveSessionID(A, []byte(`{"session_id":"`+C+`"}`)); got != A {
		t.Fatalf("env should win: %q", got)
	}
	// config fallback
	if got := resolveSessionID("", []byte(`{"agent":"claude","session_id":"`+C+`"}`)); got != C {
		t.Fatalf("config fallback: %q", got)
	}
	// legacy (no env, no config sid)
	if got := resolveSessionID("", nil); got != "" {
		t.Fatalf("no id → empty: %q", got)
	}
	if got := resolveSessionID("", []byte(`{"agent":"claude"}`)); got != "" {
		t.Fatalf("config without session_id → empty: %q", got)
	}
	if got := resolveSessionID("", []byte(`not json`)); got != "" {
		t.Fatalf("bad json → empty: %q", got)
	}
}

func TestChangelogBase(t *testing.T) {
	if got := changelogBase("/dd", "t", "claude", "sid1"); got != "/dd/changelog-t-claude-sid1" {
		t.Fatalf("with sid: %q", got)
	}
	if got := changelogBase("/dd", "t", "claude", ""); got != "/dd/changelog-t-claude" {
		t.Fatalf("legacy: %q", got)
	}
}

func TestDistillerEnvAndInner(t *testing.T) {
	env := distillerEnv("/h/bin/pair", "/r.raw", "/e.jsonl", "/c.cleaned", "/l.md", "/a.anchor", "claude")
	want := map[string]bool{
		"PCL_BIN=/h/bin/pair": true, "PCL_RAW=/r.raw": true, "PCL_EVENTS=/e.jsonl": true,
		"PCL_CLEANED=/c.cleaned": true, "PCL_LOG=/l.md": true, "PCL_ANCHOR=/a.anchor": true, "PCL_AGENT=claude": true,
	}
	for _, kv := range env {
		if _, ok := want[kv]; !ok {
			t.Fatalf("unexpected env entry %q", kv)
		}
		delete(want, kv)
	}
	if len(want) != 0 {
		t.Fatalf("missing env entries: %v", want)
	}
	// The inner script runs render then changelog, gated on success (&&).
	for _, frag := range []string{"scrollback-render", "changelog", "&&", "$PCL_BIN", "$PCL_LOG"} {
		if !strings.Contains(distillerInner, frag) {
			t.Fatalf("distillerInner missing %q", frag)
		}
	}
}
