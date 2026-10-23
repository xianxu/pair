package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"image/color"
	"strings"
	"testing"
)

func TestInactiveOwnerChipsAndCountsFadeInBothStyles(t *testing.T) {
	for _, style := range []TabStyle{TabStyleCollapse, TabStyleMultiline} {
		for _, palette := range []Palette{darkTrueColor, {FG: black, BG: white, Known: true, TrueColor: true}, {}} {
			for _, inactive := range []bool{false, true} {
				a := hubActor("repo:1", "repo", 1)
				a.OtherHub = inactive
				b, cells, err := renderChrome(ptychild.Size{Rows: 24, Cols: 80}, StatusModel{TabStyle: style, Palette: palette, Actors: []StatusActor{a}})
				if err != nil {
					t.Fatal(err)
				}
				if palette.Known && inactive {
					want := color.RGBAModel.Convert(blend(palette.FG, palette.BG, 0.5))
					if cells[0].Style.Fg == nil || color.RGBAModel.Convert(cells[0].Style.Fg) != want {
						t.Fatalf("style%d inactive chip color%v want%v: %q", style, cells[0].Style.Fg, want, b.Body())
					}
				}
				if !inactive && (strings.Contains(b.Body(), "\x1b[2m") || strings.Contains(b.Body(), "38;2;")) {
					t.Fatalf("active owner faded: %q", b.Body())
				}
			}
		}
	}
	// Three quiet workers of another owner fold. Every drawn count cell must
	// inherit the same ownership shade, while a current-owner count stays plain.
	for _, inactive := range []bool{false, true} {
		var actors []StatusActor
		for i := 1; i <= 3; i++ {
			a := hubActor("repo:1", "repo", i)
			a.OtherHub = inactive
			actors = append(actors, a)
		}
		bar, cells, err := renderChrome(ptychild.Size{Rows: 24, Cols: 80}, StatusModel{TabStyle: TabStyleCollapse, Palette: darkTrueColor, Actors: actors})
		if err != nil {
			t.Fatal(err)
		}
		if len(bar.GroupCounts) != 1 {
			t.Fatalf("missing count%q", bar.Body())
		}
		for x := bar.GroupCounts[0].Start; x < bar.GroupCounts[0].End; x++ {
			fg := cells[x].Style.Fg
			if inactive && (fg == nil || color.RGBAModel.Convert(fg) != (color.RGBA{128, 128, 128, 255})) {
				t.Fatalf("count cell%d color%v: %q", x, fg, bar.Body())
			}
			if !inactive && fg != nil {
				t.Fatalf("current owner count faded: %q", bar.Body())
			}
		}
	}
}

func TestInactiveOwnerStylingKeepsCuesFallbacksAndHitMaps(t *testing.T) {
	for _, style := range []TabStyle{TabStyleCollapse, TabStyleMultiline} {
		for _, palette := range []Palette{darkTrueColor, {}, {NoColor: true}} {
			for _, cue := range []string{"hub", "bell", "blocked", "placeholder", "glyph"} {
				a := hubActor("repo", "repo", 0)
				switch cue {
				case "hub":
					a.Hub = true
				case "bell":
					a.Bell = true
				case "blocked":
					a.Blocked = true
				case "placeholder":
					a.Placeholder = true
				case "glyph":
					a.Glyph = "*"
				}
				model := StatusModel{TabStyle: style, Palette: palette, Actors: []StatusActor{a}}
				before := RenderStatusBar(80, 2, model)
				model.Actors[0].OtherHub = true
				after := RenderStatusBar(80, 2, model)
				if rowPlain(before.Rows[0]) != rowPlain(after.Rows[0]) || len(before.Chips) != len(after.Chips) {
					t.Fatalf("dimming changed geometry: %q -> %q", before.Body(), after.Body())
				}
				for i := range before.Chips {
					if before.Chips[i] != after.Chips[i] {
						t.Fatal("dimming moved click span")
					}
				}
				if palette.NoColor && before.Body() != after.Body() {
					t.Fatalf("NO_COLOR changed rendition: %q -> %q", before.Body(), after.Body())
				}
				if cue == "blocked" && !strings.Contains(after.Body(), blockedSGR) {
					t.Fatal("lost themed red")
				}
				if cue == "bell" && !strings.Contains(after.Body(), attentionSGR) {
					t.Fatal("lost attention amber")
				}
				if cue == "hub" && !strings.Contains(after.Body(), reverseSGR) {
					t.Fatal("lost hub reverse")
				}
			}
		}
	}
	for _, palette := range []Palette{{}, {NoColor: true}, {FG: white, BG: black, Known: true}} {
		actors := []StatusActor{hubActorOwned("repo:1", "repo", 1, "other"), hubActorOwned("repo:2", "repo", 2, "other"), hubActorOwned("repo:3", "repo", 3, "other")}
		bar := RenderStatusBar(80, 2, StatusModel{TabStyle: TabStyleCollapse, Palette: palette, Actors: actors})
		if palette.NoColor && strings.Contains(bar.Body(), "\x1b[") {
			t.Fatalf("NO_COLOR faded count: %q", bar.Body())
		}
		if !palette.Known && !palette.NoColor && !strings.Contains(bar.Body(), "\x1b[2m") {
			t.Fatalf("unknown palette count lost dim fallback: %q", bar.Body())
		}
		if palette.Known && !strings.Contains(bar.Body(), "\x1b[38;5;244m") {
			t.Fatalf("256-color count not half blended: %q", bar.Body())
		}
	}
}
