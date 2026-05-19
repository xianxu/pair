package main

import (
	"container/list"
	"strings"
	"testing"
)

// newAgentProxy builds a proxy wired just enough for updateAgentOutput
// to capture spans into the LRU. The agent-output file path is a stub
// — non-empty so the function doesn't early-return, but the LRU is what
// we inspect (we never call flushAgentFile in tests).
func newAgentProxy() *proxy {
	return &proxy{
		agentOutputFile: "/dev/null",
		spans:           map[string]*spanEntry{},
		spanOrder:       list.New(),
	}
}

// spanTexts returns the span texts in LRU order (front = oldest). The
// LRU keys by "<color>\t<text>"; tests don't usually care about the
// color prefix, so this returns just the text bytes as strings.
func spanTexts(p *proxy) []string {
	var out []string
	for el := p.spanOrder.Front(); el != nil; el = el.Next() {
		key := el.Value.(string)
		// key = "<color>\t<text>"
		if i := strings.IndexByte(key, '\t'); i >= 0 {
			out = append(out, key[i+1:])
		} else {
			out = append(out, key)
		}
	}
	return out
}

// sgr builds an `\x1b[<params>m` SGR sequence with literal-byte escapes.
func sgr(params string) string { return "\x1b[" + params + "m" }

func TestUpdateAgentOutput_BasicSingleColorSpan(t *testing.T) {
	p := newAgentProxy()
	p.updateAgentOutput([]byte(sgr("31") + "hello" + sgr("0")))
	got := spanTexts(p)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %q, want [\"hello\"]", got)
	}
}

func TestUpdateAgentOutput_FinalizesOnFGChange(t *testing.T) {
	// Red "foo" then blue "bar" — FG change between them must finalize
	// the red span before the new color starts collecting "bar".
	p := newAgentProxy()
	p.updateAgentOutput([]byte(sgr("31") + "foo" + sgr("34") + "bar" + sgr("0")))
	got := spanTexts(p)
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Errorf("got %q, want [\"foo\", \"bar\"]", got)
	}
}

func TestUpdateAgentOutput_FinalizesOnLineBreak(t *testing.T) {
	// A newline must close out the in-progress span — colored runs don't
	// span lines for purposes of suggestion text.
	p := newAgentProxy()
	p.updateAgentOutput([]byte(sgr("31") + "foo\nbar" + sgr("0")))
	got := spanTexts(p)
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Errorf("got %q, want [\"foo\", \"bar\"]", got)
	}
}

func TestUpdateAgentOutput_DefaultFGNotCaptured(t *testing.T) {
	// Plain text outside any SGR shouldn't enter the span buffer.
	p := newAgentProxy()
	p.updateAgentOutput([]byte("plain text\n"))
	got := spanTexts(p)
	if len(got) != 0 {
		t.Errorf("got %q, want empty", got)
	}
}

func TestUpdateAgentOutput_CursorEscapeMidSpanPreservesWordBoundary(t *testing.T) {
	// The 949aeec bug fix in bin/pair-wrap.py, now ported to the Go
	// path. Claude's TUI paints inline `code` spans cell-by-cell, using
	// CUF / similar cursor moves to skip over the spaces that already
	// live in the cells. Without inserting a placeholder space on
	// non-SGR escape match, "make nous-install" merges into the
	// unusable autocomplete candidate "makenous-install".
	p := newAgentProxy()
	// "\x1b[1C" is CUF (cursor forward 1). The visible cells between
	// "make" and "nous-install" are spaces in the agent pane; the
	// emulator just jumps the cursor over them.
	p.updateAgentOutput([]byte(sgr("31") + "make" + "\x1b[1C" + "nous-install" + sgr("0")))
	got := spanTexts(p)
	if len(got) != 1 || got[0] != "make nous-install" {
		t.Errorf("got %q, want [\"make nous-install\"]", got)
	}
}

func TestUpdateAgentOutput_MultipleCursorEscapesCollapseToSingleSpace(t *testing.T) {
	// A burst of cursor-positioning escapes between two painted runs
	// should not balloon the placeholder into multiple spaces. One
	// gap between words is enough for word-boundary purposes.
	p := newAgentProxy()
	p.updateAgentOutput([]byte(
		sgr("31") + "a" + "\x1b[1C" + "\x1b[1C" + "\x1b[1C" + "b" + sgr("0"),
	))
	got := spanTexts(p)
	if len(got) != 1 || got[0] != "a b" {
		t.Errorf("got %q, want [\"a b\"]", got)
	}
}

func TestUpdateAgentOutput_PendingCarryoverAcrossChunks(t *testing.T) {
	// SGR sequences arrive split across pty read boundaries. The
	// trailing partial bytes must be held in agentPending and prepended
	// to the next chunk.
	p := newAgentProxy()
	full := sgr("31") + "hello" + sgr("0")
	// Split right in the middle of the closing SGR: "...hello\x1b[0" | "m"
	splitAt := len(full) - 1
	p.updateAgentOutput([]byte(full[:splitAt]))
	p.updateAgentOutput([]byte(full[splitAt:]))
	got := spanTexts(p)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %q, want [\"hello\"]", got)
	}
}

func TestUpdateAgentOutput_LRUDedupMoveToBack(t *testing.T) {
	// Re-emitting the same (color, text) tuple bumps count and moves
	// the entry to the back of the order, but does not duplicate.
	p := newAgentProxy()
	in := []byte(sgr("31") + "foo" + sgr("0"))
	p.updateAgentOutput(in)
	p.updateAgentOutput(in)
	got := spanTexts(p)
	if len(got) != 1 || got[0] != "foo" {
		t.Errorf("got %q, want exactly [\"foo\"] (dedup'd)", got)
	}
	// Inspect the LRU entry directly to confirm count.
	for _, e := range p.spans {
		if string(e.text) == "foo" && e.count != 2 {
			t.Errorf("foo: count = %d, want 2", e.count)
		}
	}
}

func TestUpdateAgentOutput_LRUMoveToBackOrdersByRecency(t *testing.T) {
	// foo, bar, foo (again) → order should end up [bar, foo] because
	// foo's re-emission moves it to the back.
	p := newAgentProxy()
	p.updateAgentOutput([]byte(sgr("31") + "foo" + sgr("0")))
	p.updateAgentOutput([]byte(sgr("31") + "bar" + sgr("0")))
	p.updateAgentOutput([]byte(sgr("31") + "foo" + sgr("0")))
	got := spanTexts(p)
	want := []string{"bar", "foo"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %q, want %q", got, want)
	}
}

