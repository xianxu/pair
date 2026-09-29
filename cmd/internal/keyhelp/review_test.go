package keyhelp

import (
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"strings"
	"testing"
)

func TestReviewHelpDerivesEveryMappingAndSurvivesHostLayer(t *testing.T) {
	src := mustReadTreeSource(t, "nvim/review.lua")
	scan := ParseReviewKeymaps(src)
	if n := len(scan.Resolved) + len(scan.Dynamic) + len(scan.Unresolved); n != strings.Count(src, keymapCall) {
		t.Fatalf("review keymaps without review: desc: documented %d calls of %d", n, strings.Count(src, keymapCall))
	}
	for _, km := range append(append(scan.Resolved, scan.Dynamic...), scan.Unresolved...) {
		if !reviewClassified(km) {
			t.Errorf("review mapping %q (%s) has no help classification", km.Raw, km.Desc)
		}
	}
	sections, err := reviewSections(src)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(Layer(nil, []workbenchshortcut.Chord{workbenchshortcut.ChordAltN}, sections))
	for _, want := range []string{"Alt+a", "Alt+r", "Alt+Shift+A", "Alt+Shift+R", "Alt+q", "Alt+⏎", "Alt+n", "Alt+Shift+N", "]m", "[m", "quote selection", "Esc"} {
		if !strings.Contains(out, want) {
			t.Errorf("review help missing %q:\n%s", want, out)
		}
	}
	if reviewClassified(NvimKeymap{Key: "<M-z>", Raw: "'<M-z>'", Desc: "new mapping"}) {
		t.Fatal("new review binding silently classified")
	}
	// Descriptions are consumed from the live mapping, not copied into Go.
	changed, err := reviewSections(strings.ReplaceAll(src, "review: accept 🤖 suggestion (§5)", "review: changed accept behavior"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Render(changed), "changed accept behavior") {
		t.Fatal("help wording did not follow mapping desc")
	}
}
