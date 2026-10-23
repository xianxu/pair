package couchkeys

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestEveryBindingIsComplete(t *testing.T) {
	for _, b := range Bindings() {
		if b.Action == 0 || b.Key == "" || b.Help == "" || len(b.Encodings) == 0 || (b.Scope != ScopeEveryPane && b.Scope != ScopeSwitcher) {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

// One byte sequence, one chord: the interceptor's first match would silently
// shadow a second.
func TestEncodingsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, b := range Bindings() {
		for _, e := range b.Encodings {
			if prior, ok := seen[string(e)]; ok {
				t.Errorf("%q is both %s and %s", e, prior, b.Key)
			}
			seen[string(e)] = b.Key
		}
	}
}

// A switcher chord's bytes are Pair's outside the switcher, so they are exactly
// Pair's encodings for the declared chord -- derived, not restated.
func TestSwitcherChordsCarryPairsEncodings(t *testing.T) {
	for _, b := range Bindings() {
		if b.Scope == ScopeSwitcher && (b.Chord == 0 || !reflect.DeepEqual(b.Encodings, workbenchshortcut.ChordEncodings(b.Chord))) {
			t.Errorf("%s encodings are not Pair's %v", b.Key, b.Chord)
		}
	}
}

// An every-pane chord whose bytes are a Pair chord takes that chord from Pair
// and must declare it, or Layer cannot drop Pair's row (#282).
func TestEveryPaneChordsDeclareThePairChordTheyTake(t *testing.T) {
	for _, b := range Bindings() {
		if b.Scope != ScopeEveryPane {
			continue
		}
		for _, e := range b.Encodings {
			if chord, ok := workbenchshortcut.DecodeChord(e); ok && chord != b.Chord {
				t.Errorf("%s takes Pair's %s without declaring it", b.Key, workbenchshortcut.ChordName(chord))
			}
		}
	}
}

// Couch leaves Alt+h to Pair: it is not a Couch chord in any scope (#282).
func TestCouchDoesNotClaimAltH(t *testing.T) {
	for _, b := range Bindings() {
		if b.Chord == workbenchshortcut.ChordAltH {
			t.Fatalf("%s claims Alt+h", b.Key)
		}
		for _, e := range b.Encodings {
			if chord, ok := workbenchshortcut.DecodeChord(e); ok && chord == workbenchshortcut.ChordAltH {
				t.Fatalf("%s frames Alt+h", b.Key)
			}
		}
	}
}

// `couch --help` renders Help verbatim and keeps internal operation names off
// its public surface (couchcmd TestPublicHelpListsOnlyPublicSurface).
func TestHelpAvoidsInternalOperationNames(t *testing.T) {
	for _, b := range Bindings() {
		for _, word := range []string{"start", "park", "resume"} {
			if strings.Contains(b.Help, word) {
				t.Errorf("%s help %q uses %q", b.Key, b.Help, word)
			}
		}
	}
}

func TestBindingsReturnsCopies(t *testing.T) {
	first := Bindings()
	first[0].Help = "mutated"
	first[0].Encodings[0][0] ^= 0xff
	second := Bindings()
	if second[0].Help == "mutated" || bytes.Equal(second[0].Encodings[0], first[0].Encodings[0]) {
		t.Fatal("Bindings leaked its table")
	}
}

func TestHelpSectionsRenderEveryChordByScope(t *testing.T) {
	secs := HelpSections(Bindings())
	if len(secs) != 2 {
		t.Fatalf("sections=%d, want every-pane + switcher", len(secs))
	}
	out := keyhelp.Render(secs)
	for _, b := range Bindings() {
		if !strings.Contains(out, b.Key) || !strings.Contains(out, b.Help) {
			t.Errorf("missing %s: %s", b.Key, b.Help)
		}
	}
	for _, r := range secs[0].Bindings {
		if r.Context != keyhelp.ContextHost {
			t.Errorf("%s in every-pane section: %v", r.Key, r.Context)
		}
	}
	for _, r := range secs[1].Bindings {
		if r.Context != keyhelp.ContextHostMenu {
			t.Errorf("%s in switcher section: %v", r.Key, r.Context)
		}
	}
	if !strings.Contains(secs[1].Title, "Ctrl+Space") {
		t.Errorf("switcher title %q does not name its opener", secs[1].Title)
	}
}

// Only every-pane chords that are Pair chords are claimed; switcher chords share
// Pair's bytes but never take them from a Pair pane.
func TestClaimedIsEveryPanePairChordsOnly(t *testing.T) {
	// Today's claims are the relaunch pair (#284); Alt+d and Alt+x are switcher
	// chords that share Pair's bytes and are not claimed.
	today := []workbenchshortcut.Chord{workbenchshortcut.ChordAltN, workbenchshortcut.ChordCtrlAltN}
	if got := Claimed(Bindings()); !reflect.DeepEqual(got, today) {
		t.Fatalf("today's table claims %v, want %v", got, today)
	}
	extra := Binding{Action: ActionSwitch, Scope: ScopeEveryPane, Chord: workbenchshortcut.ChordAltL, Key: "Alt+l", Help: "probe",
		Encodings: workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltL)}
	if got := Claimed(append(Bindings(), extra)); !reflect.DeepEqual(got, append(today, workbenchshortcut.ChordAltL)) {
		t.Fatalf("claimed = %v", got)
	}
}

// The override the operator asked for: a chord Couch claims replaces Pair's
// row for it with no edit outside the table (#282). Alt+l stands in for any
// Pair chord Couch might take later.
func TestClaimingAPairChordReplacesItsRowOnThePage(t *testing.T) {
	extra := Binding{Action: ActionSwitch, Scope: ScopeEveryPane, Chord: workbenchshortcut.ChordAltL, Key: "Alt+l", Help: "probe row for #282",
		Encodings: workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltL)}
	bs := append(Bindings(), extra)
	pair, err := keyhelp.HostedSections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	page := keyhelp.Layer(HelpSections(bs), Claimed(bs), pair)
	found := 0
	for _, s := range page {
		for _, b := range s.Bindings {
			if b.Chord == workbenchshortcut.ChordAltL {
				found++
				if b.Desc != extra.Help {
					t.Errorf("Pair's Alt+l row survived: %q", b.Desc)
				}
			}
		}
	}
	if found != 1 {
		t.Fatalf("Alt+l rows = %d, want exactly Couch's", found)
	}
}

// No key has two meanings in one context: an every-pane Couch key never reaches
// Pair, so no Pair row left on the page Couch presents may carry its label. Read
// off the layered page, because a claimed chord's Pair rows are Layer's to drop
// (#284 claims Alt+n); what this adds is the draft-local rows, which have no
// chord for Layer to match.
func TestNoPairRowSharesAnEveryPaneCouchKey(t *testing.T) {
	pair, err := keyhelp.HostedSections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	bs := Bindings()
	taken := map[string]bool{}
	for _, b := range bs {
		if b.Scope == ScopeEveryPane {
			taken[b.Key] = true
		}
	}
	couch := len(HelpSections(bs))
	for _, s := range keyhelp.Layer(HelpSections(bs), Claimed(bs), pair)[couch:] {
		for _, b := range s.Bindings {
			if taken[b.Key] {
				t.Errorf("%s also has a Pair meaning: %q", b.Key, b.Desc)
			}
		}
	}
}
