package layoutcmd

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type terminalRecoveryRuntime struct {
	OSRuntime
	caller, last string
	registered   []string
}

func (r terminalRecoveryRuntime) CurrentPaneID() string               { return r.caller }
func (r terminalRecoveryRuntime) LastTerminalPaneID() (string, error) { return r.last, nil }
func (r terminalRecoveryRuntime) TerminalPaneIDs() ([]string, error)  { return r.registered, nil }

func terminalRecoveryFixture(t *testing.T, present bool) (*zellijtest.Fixture, terminalRecoveryRuntime) {
	t.Helper()
	f := zellijtest.New(t)
	for _, e := range f.Env() {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	t.Setenv("ZELLIJ_PANE_ID", "2")
	caller := zellijtest.DefaultPane(2, "nvim")
	caller["terminal_command"] = "nvim -u /pair/nvim/init.lua"
	caller["title"] = "draft"
	panes := []zellijtest.Pane{caller}
	if present {
		p := zellijtest.DefaultPane(3, "pair term")
		p["terminal_command"] = "pair term"
		p["is_focused"] = false
		panes = append(panes, p)
	}
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: panes, Tab: zellijtest.DefaultTab()}}})
	return f, terminalRecoveryRuntime{caller: "2", last: "3", registered: []string{"3"}}
}

func TestNewTerminalRecreatesConfirmedAbsentPane(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "other-tab"}[foreign], func(t *testing.T) {
			f, rt := terminalRecoveryFixture(t, false)
			if foreign {
				model := f.Model()
				s := model.Sessions["test"]
				p := zellijtest.DefaultPane(3, "pair term")
				p["terminal_command"] = "pair term"
				p["tab_id"] = 9
				p["tab_position"] = 1
				s.Panes = append(s.Panes, p)
				model.Sessions["test"] = s
				f.SetModel(model)
			}
			if err := SwitchRightTerminalTabContext(t.Context(), rt, workbenchshortcut.ChordAltShiftT); err != nil {
				t.Fatal(err)
			}
			effects := f.Effects()
			for _, p := range f.Model().Sessions["test"].Panes {
				if p["id"] == float64(2) && p["is_focused"] != true {
					t.Fatal("invoking pane lost focus")
				}
				if p["tab_id"] == float64(0) && p["title"] == "terminal" && p["is_focused"] != false {
					t.Fatal("recreated terminal stole client focus")
				}
			}
			if len(effects) != 1 || effects[0].Action != "new-pane" {
				t.Fatalf("lost terminal not recreated: %+v", effects)
			}
			args := effects[0].Args
			want := []string{"action", "new-pane", "--direction", "right", "--tab-id", "0", "--no-focus", "--name", "terminal", "--", "sh", "-c", `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`}
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("recovery is not bound to caller's tab: %q want %q", args, want)
			}
			if len(f.Model().Sessions["test"].Panes) != 2+map[bool]int{false: 0, true: 1}[foreign] {
				t.Fatal("fake did not retain created pane")
			}
		})
	}
}

func TestNewTerminalExistingPaneGetsOnlyOneChord(t *testing.T) {
	f, rt := terminalRecoveryFixture(t, true)
	if err := SwitchRightTerminalTabContext(t.Context(), rt, workbenchshortcut.ChordAltShiftT); err != nil {
		t.Fatal(err)
	}
	effects := f.Effects()
	if len(effects) != 1 || effects[0].Action != "write" || effects[0].Target != "terminal_3" {
		t.Fatalf("effects=%+v", effects)
	}
	if len(f.Queries()) != 3 {
		t.Fatalf("new action must confirm before effect even with cached pane id: %+v", f.Queries())
	}
}

func TestNewTerminalUnknownInventoryNeverCreates(t *testing.T) {
	for _, mode := range []string{"empty", "malformed", "missing-caller", "changed", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f, rt := terminalRecoveryFixture(t, false)
			ctx := t.Context()
			switch mode {
			case "empty", "malformed":
				raw := ""
				if mode == "malformed" {
					raw = `[{`
				}
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
			case "missing-caller":
				rt.caller = "999"
			case "changed":
				model := f.Model()
				s := model.Sessions["test"]
				p := zellijtest.DefaultPane(3, "pair term")
				p["terminal_command"] = "pair term"
				s.Panes = append(s.Panes, p)
				b, _ := json.Marshal(s.Panes)
				raw := string(b)
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &raw})
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err := SwitchRightTerminalTabContext(ctx, rt, workbenchshortcut.ChordAltShiftT); err == nil {
				t.Fatal("unknown inventory accepted")
			}
			if effects := f.Effects(); len(effects) != 0 {
				t.Fatalf("uncertain recovery caused effects: %+v", effects)
			}
		})
	}
}

func TestTerminalPreviousNextDoNotRecreateAbsentPane(t *testing.T) {
	for _, chord := range []workbenchshortcut.Chord{workbenchshortcut.ChordAltShiftLeft, workbenchshortcut.ChordAltShiftRight} {
		f, rt := terminalRecoveryFixture(t, false)
		rt.last = ""
		rt.registered = nil
		if err := SwitchRightTerminalTabContext(t.Context(), rt, chord); err != nil {
			t.Fatal(err)
		}
		if effects := f.Effects(); len(effects) != 0 {
			t.Fatalf("navigation created terminal: %+v", effects)
		}
	}
}
