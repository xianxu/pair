package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestShortcutHelpDerivesReservations(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	for _, binding := range couchtty.CouchNavigationBindings() {
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
