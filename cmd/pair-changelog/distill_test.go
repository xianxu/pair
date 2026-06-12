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

func TestLastHeaderDate(t *testing.T) {
	if got := lastHeaderDate("## 2026-06-11\n\n- a\n\n## 2026-06-12\n\n- b\n"); got != "2026-06-12" {
		t.Fatalf("got %q want 2026-06-12", got)
	}
	if got := lastHeaderDate("- a\n"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestAssembleSameDayAppend(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two\n", "- three\n", "2026-06-12", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two\n\n- three\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleRolloverInsertsHeader(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two\n", "- three\n", "2026-06-13", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two\n\n## 2026-06-13\n\n- three\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleReviseOnlyNoNew(t *testing.T) {
	got := assemble("## 2026-06-12\n\n- one\n\n", "- two-revised\n", "", "2026-06-13", "2026-06-12")
	want := "## 2026-06-12\n\n- one\n\n- two-revised\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAssembleFirstEver(t *testing.T) {
	got := assemble("", "", "- a\n\n- b\n", "2026-06-12", "")
	want := "## 2026-06-12\n\n- a\n\n- b\n"
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
