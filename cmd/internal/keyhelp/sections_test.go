package keyhelp

import (
	"errors"
	"strings"
	"testing"
)

// The composed document: wording comes from the sources, grouping/order from the
// catalog, and Alt+h must document ITSELF — it is the advertised discovery path that
// nvim/init.lua's PAIR_CHEATS keeps last-to-drop.
func TestSectionsDeriveWordingFromSources(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	find := func(key string) (Binding, bool) {
		for _, s := range secs {
			for _, b := range s.Bindings {
				if b.Key == key {
					return b, true
				}
			}
		}
		return Binding{}, false
	}
	send, ok := find("Alt+⏎")
	if !ok {
		t.Fatal("Alt+⏎ missing from help")
	}
	if send.Desc != "send buffer + clear" { // the wording authored in init.lua
		t.Errorf("Alt+⏎ desc = %q, want the init.lua wording", send.Desc)
	}
	if _, ok := find("Alt+h"); !ok {
		t.Error("Alt+h must document itself — it is the advertised discovery path")
	}
	if _, ok := find("Alt+x"); !ok {
		t.Error("Alt+x (quit) missing — GlobalBinding.Help not wired in")
	}
}

// Alt+t/w/r have nvim keymaps describing the draft NO-OP. Their user-facing wording
// must come from roleBindings. If the join ever falls back to "whichever source has
// prose", this fails — which is why rows name their source (#132 PQ-2).
func TestRoleLocalWordingComesFromRoleTableNotNvimNoOp(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	out := Render(secs)
	if strings.Contains(out, "disabled in draft") {
		t.Error("help shows the draft no-op desc as a feature description")
	}
	if !strings.Contains(out, "new terminal tab") { // roleBindings' Help for Alt+t
		t.Error("Alt+t wording did not come from roleBindings")
	}
}

// A dual-meaning key renders TWICE, in different sections, each with its own
// wording: Alt+k focuses the terminal from the draft (init.lua) and jumps back to the
// left pane from the terminal (shortcut.go).
func TestDualMeaningKeyRendersInBothContexts(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, s := range secs {
		for _, b := range s.Bindings {
			if b.Key == "Alt+k" {
				seen = append(seen, s.Title+": "+b.Desc)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("Alt+k should appear once per context, got %v", seen)
	}
	if seen[0] == seen[1] {
		t.Errorf("both Alt+k rows carry the same wording: %v", seen)
	}
}

// Regression pin for the original bug in its exact form.
func TestHelpNeverTellsYouToPressAltH(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	out := Render(secs)
	if strings.Contains(out, "keybindings are on Alt+h") {
		t.Error("the help must not refer the reader back to the help key (#132)")
	}
	if !strings.Contains(out, "send buffer") {
		t.Error("the help must contain real bindings, not a CLI synopsis (#132)")
	}
}

type failingSources struct{}

func (failingSources) Read(string) ([]byte, error) { return nil, errors.New("boom") }

func TestSectionsSurfacesSourceErrors(t *testing.T) {
	if _, err := Sections(failingSources{}); err == nil {
		t.Fatal("a source read failure must surface, not render empty help")
	}
}
