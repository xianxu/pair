package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// The whole reserved-row design is this off-by-one: the child gets rows-1 and
// the region stops at rows-1, so a child scrolling at ITS bottom line scrolls
// inside the region and cannot reach the row below.
func TestReserveStopsOneRowShortOfTheScreen(t *testing.T) {
	got := Reserve(24)
	if !strings.Contains(got, "\x1b[1;23r") {
		t.Fatalf("Reserve(24) = %q, want a region ending at row 23", got)
	}
	if strings.Contains(got, "\x1b[1;24r") {
		t.Fatalf("Reserve(24) claimed the whole screen: %q", got)
	}
}

func TestChildRowsIsOneLessThanTheHost(t *testing.T) {
	if got := ChildRows(24); got != 23 {
		t.Fatalf("ChildRows(24) = %d, want 23", got)
	}
	// A terminal too short to reserve from must not produce a zero-row child.
	if got := ChildRows(1); got != 1 {
		t.Fatalf("ChildRows(1) = %d, want 1 — a zero-row pty is not a thing", got)
	}
	if got := ChildRows(0); got != 1 {
		t.Fatalf("ChildRows(0) = %d, want 1 — a zero-row pty is not a thing", got)
	}
}

// Without save/restore the child's cursor lands wherever the row was painted,
// which shows up as the caret jumping to the bottom line on every notification.
func TestPaintRowBracketsThePaintWithSaveAndRestore(t *testing.T) {
	got := PaintRow(24, "hello")

	save := strings.Index(got, "\x1b7")
	restore := strings.Index(got, "\x1b8")
	text := strings.Index(got, "hello")
	if save < 0 || restore < 0 {
		t.Fatalf("PaintRow lacks save/restore: %q", got)
	}
	if !(save < text && text < restore) {
		t.Fatalf("the paint is not bracketed by save/restore: %q", got)
	}
	if !strings.Contains(got, "\x1b[24;1H") {
		t.Fatalf("PaintRow did not move to the reserved row: %q", got)
	}
}

func TestReleaseResetsTheRegion(t *testing.T) {
	if !strings.Contains(Release(), "\x1b[r") {
		t.Fatalf("Release() = %q, want a region reset", Release())
	}
}

func TestRenderStatusRowMarksActiveAndPendingDistinctly(t *testing.T) {
	got := RenderStatusRow(80, StatusModel{Actors: []StatusActor{
		{Label: "brain", Active: true},
		{Label: "pair", Bell: true},
		{Label: "ariadne"},
	}})

	if !strings.Contains(got, "[brain]") {
		t.Fatalf("the active actor is not marked: %q", got)
	}
	if strings.Contains(got, "*") || !strings.Contains(got, "\x1b[") {
		t.Fatalf("pending attention should color the existing label without adding cells: %q", got)
	}
	plain := string(ansi.Strip([]byte(got)))
	if plain != "[brain]  pair  ariadne" {
		t.Fatalf("attention changed status text or order: %q", plain)
	}
	if strings.Contains(got, "[pair]") {
		t.Fatalf("active and pending are not distinct: %q", got)
	}
}

func TestRenderStatusAttentionDoesNotRestyleFocusedActor(t *testing.T) {
	got := RenderStatusRow(80, StatusModel{Actors: []StatusActor{{Label: "pair", Active: true, Bell: true}}})
	if got != "[pair]" {
		t.Fatalf("focused attention treatment = %q, want ordinary focused chip", got)
	}
}

func TestRenderStatusRowFitsTheWidth(t *testing.T) {
	m := StatusModel{
		Actors: []StatusActor{{Label: "one", Active: true}, {Label: "two"}, {Label: "three"}},
		Notice: "a notice long enough to need cutting off somewhere sensible",
	}
	for _, w := range []int{10, 20, 40, 80} {
		got := RenderStatusRow(w, m)
		if textwidth.Width(got) > w {
			t.Fatalf("width %d: rendered %d columns: %q", w, textwidth.Width(got), got)
		}
	}
}

// Labels and notices carry AGENT-PUBLISHED text (couchcore.Describe reads a
// sidecar the session writes). A description containing \x1b[2J would clear the
// operator's screen from the status row.
func TestRenderStatusRowStripsControlBytesFromUntrustedText(t *testing.T) {
	got := RenderStatusRow(80, StatusModel{
		Actors: []StatusActor{{Label: "ev\x1b[2Jil", Active: true}},
		Notice: "also \x07 bad \x1b[31m",
	})
	for _, bad := range []string{"\x1b", "\x07"} {
		if strings.Contains(got, bad) {
			t.Fatalf("control byte %q survived into the status row: %q", bad, got)
		}
	}
	if !strings.Contains(got, "evil") {
		t.Fatalf("stripping mangled the visible text: %q", got)
	}
}

func TestRenderStatusRowWithNoActors(t *testing.T) {
	got := RenderStatusRow(40, StatusModel{Notice: "nothing running"})
	if !strings.Contains(got, "nothing running") {
		t.Fatalf("the notice was dropped: %q", got)
	}
}
