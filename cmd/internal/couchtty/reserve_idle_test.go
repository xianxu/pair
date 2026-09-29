package couchtty

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

var darkTrueColor = Palette{FG: white, BG: black, Known: true, TrueColor: true}

func idleChip(label string, idle IdleLevel) StatusActor {
	return StatusActor{Label: label, Glyph: couchcore.SlotGlyphDirty, Idle: idle,
		Thread: couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: couchcore.ThreadTag(label)}}
}

// An idle chip's label and its amber slot glyph both fade, each from its own
// base colour, and the text and click spans do not move.
func TestRenderStatusRowFadesAnIdleChip(t *testing.T) {
	for _, level := range []IdleLevel{IdleDay, IdleStale} {
		got := RenderStatusRow(80, StatusModel{Palette: darkTrueColor, Actors: []StatusActor{idleChip("pair", level)}})
		label, glyph := FadeStyle(darkTrueColor, level, baseDefault), FadeStyle(darkTrueColor, level, baseAmber)
		want := label + "pair\x1b[0m" + glyph + couchcore.SlotGlyphDirty + "\x1b[0m"
		if got.Body != want {
			t.Fatalf("level %d body = %q, want %q", level, got.Body, want)
		}
		fresh := RenderStatusRow(80, StatusModel{Palette: darkTrueColor, Actors: []StatusActor{idleChip("pair", IdleFresh)}})
		if !reflect.DeepEqual(got.Chips, fresh.Chips) {
			t.Fatalf("level %d chips = %+v, want the fresh chip's %+v", level, got.Chips, fresh.Chips)
		}
	}
}

// Level 0 draws exactly what the row drew before #247, in every palette.
func TestRenderStatusRowFreshChipIsUnchanged(t *testing.T) {
	actors := []StatusActor{idleChip("pair", IdleFresh), {Label: "brain", Active: true}, {Label: "ariadne", Bell: true}}
	before := RenderStatusRow(80, StatusModel{Actors: actors})
	for _, p := range []Palette{darkTrueColor, {}, {FG: black, BG: white, Known: true}} {
		if got := RenderStatusRow(80, StatusModel{Palette: p, Actors: actors}); got.Body != before.Body {
			t.Fatalf("palette %+v changed a fresh row: %q, want %q", p, got.Body, before.Body)
		}
	}
}

// Selection, a pending notification and a placeholder keep their own emphasis
// however idle the thread is.
func TestRenderStatusRowFadeYieldsToSelectionNotificationAndPlaceholder(t *testing.T) {
	for name, tc := range map[string]struct {
		actor StatusActor
		want  string
	}{
		"active":      {StatusActor{Label: "pair", Glyph: couchcore.SlotGlyphDirty, Active: true, Idle: IdleStale}, "[pair" + attentionSGR + "*\x1b[0m]"},
		"bell":        {StatusActor{Label: "pair", Bell: true, Idle: IdleStale}, attentionSGR + "pair\x1b[0m"},
		"placeholder": {StatusActor{Label: "pair", Placeholder: true, Idle: IdleStale}, placeholderSGR + "pair\x1b[0m"},
	} {
		got := RenderStatusRow(80, StatusModel{Palette: darkTrueColor, Actors: []StatusActor{tc.actor}}).Body
		if got != tc.want {
			t.Errorf("%s: body = %q, want %q", name, got, tc.want)
		}
		if strings.Contains(got, "38;2;") {
			t.Errorf("%s: faded bytes in %q", name, got)
		}
	}
}
