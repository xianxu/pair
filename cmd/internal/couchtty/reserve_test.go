package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// The reserved-row MECHANISM's tests moved with it to
// cmd/internal/hostty/reserve_test.go in pair#199. What remains here is the
// policy half: what couch draws on the row.

func TestRenderStatusRowMarksActiveAndPendingDistinctly(t *testing.T) {
	got := RenderStatusRow(80, StatusModel{Actors: []StatusActor{
		{Label: "brain", Active: true},
		{Label: "pair", Bell: true},
		{Label: "ariadne"},
	}}).Body

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
	got := RenderStatusRow(80, StatusModel{Actors: []StatusActor{{Label: "pair", Active: true, Bell: true}}}).Body
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
		got := RenderStatusRow(w, m).Body
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
	}).Body
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
	got := RenderStatusRow(40, StatusModel{Notice: "nothing running"}).Body
	if !strings.Contains(got, "nothing running") {
		t.Fatalf("the notice was dropped: %q", got)
	}
}
