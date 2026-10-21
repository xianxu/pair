package termcmd

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"io"
	"strings"
	"testing"
)

func TestTerminalNativeFallbackIgnoresCosmeticTitle(t *testing.T) {
	for _, omitCommand := range []bool{false, true} {
		name := "registered-template"
		if omitCommand {
			name = "registered-without-command"
		}
		t.Run(name, func(t *testing.T) {
			f := zellijtest.New(t)
			for _, entry := range f.Env() {
				k, v, _ := strings.Cut(entry, "=")
				t.Setenv(k, v)
			}
			model := shortcutAdmissionModel(2)
			own := model.Sessions["test"].Panes[0]
			own["title"] = "terminal"
			if omitCommand {
				own["terminal_command"] = ""
			}
			f.SetModel(model)
			own["title"] = "shell"
			body, err := json.Marshal(model.Sessions["test"].Panes)
			if err != nil {
				t.Fatal(err)
			}
			reply := string(body)
			f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &reply})
			base := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}} // No cached draft: native admission is required.
			rt := nativeShortcutRuntime{base}
			if err := handleChord(workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard); err != nil {
				t.Fatalf("cosmetic title refused: %v", err)
			}
			if effects := f.Effects(); len(effects) != 1 || effects[0].Target != "terminal_2" {
				t.Fatalf("effects=%+v", effects)
			}
			if len(f.Queries()) != 3 {
				t.Fatalf("want exactly two reads and admitted focus: %+v", f.Queries())
			}
			base.terminalPaneIDs = nil
			if err := handleChord(workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard); err != nil {
				t.Fatalf("removed registration: %v", err)
			}
			if len(f.Effects()) != 1 || len(f.Queries()) != 5 || len(base.recordedTerminal) != 1 {
				t.Fatalf("stale native metadata authorized another effect: effects=%+v queries=%+v records=%v", f.Effects(), f.Queries(), base.recordedTerminal)
			}
		})
	}
}
