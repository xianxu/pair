package main

import (
	"reflect"
	"testing"
)

func TestScanTurnBoundaries(t *testing.T) {
	lines := []string{"hello", "❯ first prompt", "work", "❯ second prompt", "more"}
	got := scanTurnBoundaries(lines, "claude")
	want := []int{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// codex uses ›; claude glyph must not match it.
	if got := scanTurnBoundaries([]string{"› codex prompt"}, "codex"); !reflect.DeepEqual(got, []int{0}) {
		t.Fatalf("codex got %v want [0]", got)
	}
	if got := scanTurnBoundaries([]string{"› codex prompt"}, "claude"); len(got) != 0 {
		t.Fatalf("claude glyph wrongly matched codex prompt: %v", got)
	}
}

func TestLocateFoundWalksBackTwoTurns(t *testing.T) {
	lines := []string{"a", "❯ t1", "b", "❯ t2", "c", "anchorX", "new1", "new2"}
	anchor := []string{"anchorX"}
	res := locate(lines, anchor, scanTurnBoundaries(lines, "claude"), 2, 200)
	if res.Kind != Found {
		t.Fatalf("kind=%v want Found", res.Kind)
	}
	// anchor at index 5; walk back past 2 boundaries (3, then 1) → start at 1.
	if res.Start != 1 {
		t.Fatalf("start=%d want 1", res.Start)
	}
}

func TestLocateNoOpWhenAnchorFlushWithEnd(t *testing.T) {
	lines := []string{"a", "❯ t1", "b", "anchorX"}
	res := locate(lines, []string{"anchorX"}, scanTurnBoundaries(lines, "claude"), 2, 200)
	if res.Kind != NoOp {
		t.Fatalf("kind=%v want NoOp", res.Kind)
	}
}

func TestLocateFullRedistillWhenAnchorMissing(t *testing.T) {
	lines := []string{"a", "b", "c"}
	res := locate(lines, []string{"nope"}, nil, 2, 200)
	if res.Kind != FullRedistill || res.Start != 0 {
		t.Fatalf("got %+v want {FullRedistill 0}", res)
	}
}

func TestLocateEmptyAnchorIsFullRedistill(t *testing.T) {
	res := locate([]string{"a", "b"}, nil, nil, 2, 200)
	if res.Kind != FullRedistill || res.Start != 0 {
		t.Fatalf("got %+v want {FullRedistill 0}", res)
	}
}

// Newest occurrence of a recurring anchor wins. Boundaries are placed so that
// matching the newest vs the oldest "X" yields different slice starts: newest
// (index 4) walks back past the boundary at 2 → start 2; an (incorrect) oldest
// match (index 0) would have no boundary before it → start 0.
func TestLocateNewestOccurrenceWins(t *testing.T) {
	lines := []string{"X", "a", "❯ p", "b", "X", "c"}
	res := locate(lines, []string{"X"}, scanTurnBoundaries(lines, "claude"), 1, 200)
	if res.Kind != Found || res.Start != 2 {
		t.Fatalf("got %+v want Found start 2 (newest match)", res)
	}
}

// Fewer than lookbackTurns boundaries before the match → walk to start (0),
// then the line cap pulls it back to matchAt-cap.
func TestLocateLineCap(t *testing.T) {
	lines := make([]string, 500)
	for i := range lines {
		lines[i] = "x"
	}
	lines[450] = "anchorX"
	res := locate(lines, []string{"anchorX"}, nil, 2, 200)
	if res.Kind != Found || res.Start != 250 {
		t.Fatalf("got %+v want Found start 250", res)
	}
}

func TestLocateMultiLineAnchor(t *testing.T) {
	lines := []string{"a", "L1", "L2", "new"}
	res := locate(lines, []string{"L1", "L2"}, nil, 2, 200)
	// anchor matched at 1..2, end=3 < 4 → Found; no boundaries → start 0.
	if res.Kind != Found || res.Start != 0 {
		t.Fatalf("got %+v want Found start 0", res)
	}
}

func TestSplitFrozenTailByteExact(t *testing.T) {
	log := "## 2026-06-12\n\n- one\n\n- two\n\n- three\n"
	frozen, ek := splitFrozenTail(log)
	if frozen != "## 2026-06-12\n\n- one\n\n- two\n\n" {
		t.Fatalf("frozen=%q", frozen)
	}
	if ek != "- three\n" {
		t.Fatalf("ek=%q", ek)
	}
	if frozen+ek != log {
		t.Fatalf("frozen+ek must reconstruct the log byte-for-byte")
	}
}

func TestSplitFrozenTailMultiLineEntry(t *testing.T) {
	log := "## 2026-06-12\n\n- one\n\n- two line a\n  two line b\n"
	frozen, ek := splitFrozenTail(log)
	if frozen != "## 2026-06-12\n\n- one\n\n" || ek != "- two line a\n  two line b\n" {
		t.Fatalf("frozen=%q ek=%q", frozen, ek)
	}
	if frozen+ek != log {
		t.Fatalf("reconstruct mismatch")
	}
}

func TestSplitFrozenTailNoBullets(t *testing.T) {
	log := "## 2026-06-12\n\n"
	frozen, ek := splitFrozenTail(log)
	if frozen != log || ek != "" {
		t.Fatalf("frozen=%q ek=%q", frozen, ek)
	}
}

func TestSplitFirstEntry(t *testing.T) {
	first, rest := splitFirstEntry("- two-revised\n\n- new1\n\n- new2\n")
	if first != "- two-revised" {
		t.Fatalf("first=%q", first)
	}
	if rest != "- new1\n\n- new2\n" {
		t.Fatalf("rest=%q", rest)
	}
	// single entry → rest empty.
	first, rest = splitFirstEntry("- only\n")
	if first != "- only" || rest != "" {
		t.Fatalf("first=%q rest=%q", first, rest)
	}
}

func TestStripDateHeaders(t *testing.T) {
	// legacy multi-day log → flat header-free list, blank-line separated.
	if got := stripDateHeaders("## 2026-06-11\n\n- a\n\n## 2026-06-12\n\n- b\n"); got != "- a\n\n- b\n" {
		t.Fatalf("got %q want %q", got, "- a\n\n- b\n")
	}
	// already header-free → unchanged (idempotent on the new format).
	if got := stripDateHeaders("- a\n\n- b\n"); got != "- a\n\n- b\n" {
		t.Fatalf("header-free log altered: %q", got)
	}
}

func TestAssembleAppend(t *testing.T) {
	got := assemble("- one\n\n", "- two\n", "- three\n")
	want := "- one\n\n- two\n\n- three\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleReviseOnlyNoNew(t *testing.T) {
	got := assemble("- one\n\n", "- two-revised\n", "")
	want := "- one\n\n- two-revised\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleFirstEver(t *testing.T) {
	got := assemble("", "", "- a\n\n- b\n")
	want := "- a\n\n- b\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnchorSnippet(t *testing.T) {
	if got := anchorSnippet([]string{"a", "b", "c", "d"}, 2); !reflect.DeepEqual(got, []string{"c", "d"}) {
		t.Fatalf("got %v want [c d]", got)
	}
	if got := anchorSnippet([]string{"a"}, 3); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("got %v want [a]", got)
	}
}

func TestTrimLiveTail(t *testing.T) {
	content := []string{"❯ sent prompt", "agent work A", "stable last line"}
	// idle footer: empty box + rule + status bar → all stripped.
	idle := append(append([]string{}, content...), "❯ ", "────────", "⏵⏵ bypass · 3 shells")
	if got := trimLiveTail(idle, "claude"); !reflect.DeepEqual(got, content) {
		t.Fatalf("idle footer: got %v", got)
	}
	// thinking footer (multi-block: spinner + rule ABOVE the box, box + rule +
	// status below) → all stripped iteratively.
	thinking := append(append([]string{}, content...),
		"* Cerebrating… (3s · thinking with xhigh effort)", "", "────────", "❯ ", "────────", "  ⏵⏵ esc to interrupt")
	if got := trimLiveTail(thinking, "claude"); !reflect.DeepEqual(got, content) {
		t.Fatalf("thinking footer: got %v", got)
	}
	// codex empty box + status.
	cx := []string{"x", "stable", "› ", "────"}
	if got := trimLiveTail(cx, "codex"); !reflect.DeepEqual(got, []string{"x", "stable"}) {
		t.Fatalf("codex footer: got %v", got)
	}
	// committed markdown bullets ("* item", no spinner timer/ellipsis) are NOT
	// chrome → preserved.
	bullets := []string{"intro", "* item one", "* item two"}
	if got := trimLiveTail(bullets, "claude"); !reflect.DeepEqual(got, bullets) {
		t.Fatalf("markdown bullets over-trimmed: got %v", got)
	}
	// no footer at all → unchanged.
	plain := []string{"a", "b", "c"}
	if got := trimLiveTail(plain, "claude"); !reflect.DeepEqual(got, plain) {
		t.Fatalf("plain: got %v", got)
	}
	// context-meter footer: when the window fills, claude appends a right-aligned
	// "N% context used" line BELOW the status bar. As the last line it used to
	// stop trimLiveTail dead, leaking the whole footer into the anchor (#58).
	meter := append(append([]string{}, content...),
		"❯ ", "────────", "  ⏵⏵ bypass permissions on · esc to interrupt", "                  100% context used")
	if got := trimLiveTail(meter, "claude"); !reflect.DeepEqual(got, content) {
		t.Fatalf("context-meter footer: got %v", got)
	}
}

func TestLooksLikeChangelog(t *testing.T) {
	if !looksLikeChangelog("## 2026-06-12\n\n- an entry\n  wrapped\n\n- another\n") {
		t.Fatal("valid change log rejected")
	}
	// a conversation continuation with bullets but bare prose lines → rejected.
	hijack := "Can you either:\n- Grant me read permission on the file\n- Or paste the logic\n\nAnd from changelog.lua, the reload logic?"
	if looksLikeChangelog(hijack) {
		t.Fatal("bulleted conversation continuation wrongly accepted")
	}
	if looksLikeChangelog("just some prose, no bullets at all") {
		t.Fatal("prose accepted")
	}
	if looksLikeChangelog("") {
		t.Fatal("empty accepted")
	}
}

func TestChunkLines(t *testing.T) {
	got := chunkLines([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, 4)
	if len(got) != 3 || len(got[0]) != 4 || len(got[1]) != 4 || len(got[2]) != 2 {
		t.Fatalf("got %v", got)
	}
	if got := chunkLines([]string{"a", "b"}, 4); len(got) != 1 {
		t.Fatalf("under-size should be one chunk, got %d", len(got))
	}
	if got := chunkLines(nil, 4); len(got) != 1 {
		t.Fatalf("empty → one (empty) chunk, got %d", len(got))
	}
}

func TestParseAnchor(t *testing.T) {
	// with header
	turns, snip := parseAnchor("turns:3\nL1\nL2\nL3\n")
	if turns != 3 || !reflect.DeepEqual(snip, []string{"L1", "L2", "L3"}) {
		t.Fatalf("with-header: turns=%d snip=%v", turns, snip)
	}
	// legacy: no header → turns 0, whole content is the snippet
	turns, snip = parseAnchor("L1\nL2\n")
	if turns != 0 || !reflect.DeepEqual(snip, []string{"L1", "L2"}) {
		t.Fatalf("legacy: turns=%d snip=%v", turns, snip)
	}
	// malformed count → turns 0, the "turns:x" line stays in the snippet
	turns, snip = parseAnchor("turns:notanumber\nL1\n")
	if turns != 0 || !reflect.DeepEqual(snip, []string{"turns:notanumber", "L1"}) {
		t.Fatalf("malformed: turns=%d snip=%v", turns, snip)
	}
	// empty
	turns, snip = parseAnchor("")
	if turns != 0 || snip != nil {
		t.Fatalf("empty: turns=%d snip=%v", turns, snip)
	}
}
