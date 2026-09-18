package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestShortcutHelpDerivesReservations(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	for _, binding := range couchkeys.Bindings() {
		if !strings.Contains(out.String(), binding.Key) || !strings.Contains(out.String(), binding.Help) {
			t.Errorf("navigation omitted: %+v", binding)
		}
	}
	for _, binding := range workbenchshortcut.GlobalBindings() {
		if binding.AgentReserved && !strings.Contains(out.String(), workbenchshortcut.ChordName(binding.Chord)) {
			t.Errorf("agent reservation omitted: %v", binding.Chord)
		}
	}
}

// `couch --help` renders Couch's chords through couchkeys.HelpSections -- the
// function Pair's Alt+h page uses -- and a binding added to the table reaches
// it (#282).
func TestCouchHelpRendersEveryDeclaredChord(t *testing.T) {
	extra := couchkeys.Binding{Action: couchkeys.ActionSwitch, Scope: couchkeys.ScopeSwitcher, Key: "Ctrl+Alt+z", Help: "probe row for #282", Encodings: [][]byte{[]byte("\x1bz")}}
	var with, without, def bytes.Buffer
	usageWith(&with, append(couchkeys.Bindings(), extra))
	usageWith(&without, couchkeys.Bindings())
	usage(&def)
	if !strings.Contains(with.String(), extra.Help) || strings.Contains(without.String(), extra.Help) {
		t.Fatal("couch --help does not render from the passed table")
	}
	if def.String() != without.String() {
		t.Fatal("usage does not render couchkeys.Bindings()")
	}
}
