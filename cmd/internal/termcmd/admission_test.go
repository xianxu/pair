package termcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type nativeShortcutRuntime struct{ *fakeRuntime }

func (nativeShortcutRuntime) ListPanesJSON() ([]byte, error) {
	return layoutcmd.OSRuntime{}.ListPanesJSON()
}

type contextShortcutRuntime struct {
	*fakeRuntime
	read func(context.Context) ([]byte, error)
}

func (r contextShortcutRuntime) ListPanesJSONContext(ctx context.Context) ([]byte, error) {
	return r.read(ctx)
}
func (contextShortcutRuntime) ListPanesJSON() ([]byte, error) {
	panic("protected shortcut used legacy context-free read")
}

func TestTerminalShortcutRetainsOneAdmissionBudgetAndCancellation(t *testing.T) {
	for _, cancelAt := range []int{1, 2} {
		t.Run(fmt.Sprint(cancelAt), func(t *testing.T) {
			parent, stop := context.WithTimeout(context.Background(), 4*time.Second)
			defer stop()
			ctx, cancel := context.WithCancel(parent)
			defer cancel()
			deadline, _ := parent.Deadline()
			body, err := json.Marshal(shortcutAdmissionModel(2).Sessions["test"].Panes)
			if err != nil {
				t.Fatal(err)
			}
			var first context.Context
			reads := 0
			base := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}}
			rt := contextShortcutRuntime{fakeRuntime: base, read: func(observed context.Context) ([]byte, error) {
				reads++
				if first == nil {
					first = observed
				} else if observed != first {
					t.Error("confirmation received a fresh context")
				}
				if d, ok := observed.Deadline(); !ok || !d.Equal(deadline) {
					t.Errorf("caller deadline changed: %v,%v", d, ok)
				}
				if reads == cancelAt {
					cancel()
				}
				return body, nil
			}}
			if err := handleChordContext(ctx, workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation=%v", err)
			}
			if reads != cancelAt || len(base.ops) != 0 || len(base.recordedTerminal) != 0 {
				t.Fatalf("reads=%d actions=%v records=%v", reads, base.ops, base.recordedTerminal)
			}
		})
	}
}
func (nativeShortcutRuntime) ListPanesJSONContext(ctx context.Context) ([]byte, error) {
	return layoutcmd.OSRuntime{}.ListPanesJSONContext(ctx)
}
func (nativeShortcutRuntime) RunZellijAction(args ...string) error {
	return layoutcmd.OSRuntime{}.RunZellijAction(args...)
}

func shortcutAdmissionModel(draft int) zellijtest.Model {
	pane := func(id, tab int, command string) zellijtest.Pane {
		return zellijtest.Pane{"id": id, "is_plugin": false, "is_focused": id == 4, "is_floating": false, "is_fullscreen": false, "tab_id": tab, "tab_position": tab, "terminal_command": command, "pane_x": 0, "pane_y": 0, "pane_columns": 80, "pane_rows": 20}
	}
	return zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{pane(4, 1, "pair term"), pane(draft, 1, "nvim -u /pair/nvim/init.lua /data/draft.md"), pane(8, 2, "nvim -u /pair/nvim/init.lua /data/foreign.md")}}}}
}

func TestTerminalShortcutRequiresConfirmedCallerTab(t *testing.T) {
	for _, scenario := range []string{"healthy", "one-crossed-first", "one-crossed-confirmation", "stale-last-left", "wrong-kind-first", "wrong-kind-confirmation", "displaced-first", "changed-target", "missing-caller", "missing-draft", "ambiguous-draft"} {
		t.Run(scenario, func(t *testing.T) {
			f := zellijtest.New(t)
			for _, entry := range f.Env() {
				key, value, _ := strings.Cut(entry, "=")
				t.Setenv(key, value)
			}
			model := shortcutAdmissionModel(2)
			wrong := "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
			switch scenario {
			case "wrong-kind-first":
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong}, zellijtest.Fault{Ordinal: 2, Reply: &wrong}, zellijtest.Fault{Ordinal: 3, Reply: &wrong})
			case "wrong-kind-confirmation":
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &wrong}, zellijtest.Fault{Ordinal: 3, Reply: &wrong}, zellijtest.Fault{Ordinal: 4, Reply: &wrong})
			case "one-crossed-first":
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &wrong})
			case "one-crossed-confirmation":
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &wrong})
			case "displaced-first":
				body, err := json.Marshal(shortcutAdmissionModel(3).Sessions["test"].Panes)
				if err != nil {
					t.Fatal(err)
				}
				reply := string(body)
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &reply})
			case "changed-target":
				changed := shortcutAdmissionModel(3)
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &changed})
			case "missing-caller":
				session := model.Sessions["test"]
				session.Panes = session.Panes[1:]
				model.Sessions["test"] = session
			case "missing-draft":
				session := model.Sessions["test"]
				session.Panes = append(session.Panes[:1], session.Panes[2:]...)
				model.Sessions["test"] = session
			case "ambiguous-draft":
				session := model.Sessions["test"]
				session.Panes = append(session.Panes, shortcutAdmissionModel(3).Sessions["test"].Panes[1])
				model.Sessions["test"] = session
			}
			f.SetModel(model)
			rt := nativeShortcutRuntime{&fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}}}
			if scenario == "stale-last-left" {
				rt.lastLeft = "8"
			}
			err := handleChord(workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard)
			if scenario == "healthy" || scenario == "stale-last-left" || strings.HasPrefix(scenario, "one-crossed-") {
				if err != nil {
					t.Fatal(err)
				}
				if effects := f.Effects(); len(effects) != 1 || effects[0].Target != "terminal_2" {
					t.Fatalf("effects=%+v", effects)
				}
				wantReads := 2
				if strings.HasPrefix(scenario, "one-crossed-") {
					wantReads = 3
				}
				if len(f.Queries()) != wantReads+1 {
					t.Fatalf("queries=%+v; want %d reads and one focus", f.Queries(), wantReads)
				}
			} else {
				wantReads := 2
				switch scenario {
				case "wrong-kind-first":
					wantReads = 3
				case "wrong-kind-confirmation":
					wantReads = 4
				case "missing-caller", "missing-draft", "ambiguous-draft":
					wantReads = 1
				}
				if len(f.Queries()) != wantReads {
					t.Fatalf("refusal queries=%+v; want exactly %d reads", f.Queries(), wantReads)
				}
				if err == nil || len(f.Effects()) != 0 || len(rt.recordedTerminal) != 0 {
					t.Fatalf("uncertain shortcut error=%v effects=%+v records=%v", err, f.Effects(), rt.recordedTerminal)
				}
				for _, query := range f.Queries() {
					if !strings.Contains(strings.Join(query.Args, " "), "list-panes") {
						t.Fatalf("uncertain shortcut attempted mutation: %+v", query)
					}
				}
			}
		})
	}
}

func TestTerminalDecisionUsesAdmittedDraftWithoutAnotherLookup(t *testing.T) {
	rt := &fakeRuntime{currentPaneID: "4", failList: true}
	decision := workbenchshortcut.ShortcutDecision{Disposition: workbenchshortcut.DispositionHandle, DraftLuaFunction: "PairConfirmQuit", FocusDraft: true}
	if err := runDecision(decision, workbenchPanes{draft: zellijpane.Pane{ID: "2"}}, rt, strings.NewReader(""), io.Discard); err != nil {
		t.Fatal(err)
	}
	if rt.listCalls != 0 || len(rt.ops) != 5 {
		t.Fatalf("admitted draft looked up again: reads=%d actions=%v", rt.listCalls, rt.ops)
	}
}

func TestTerminalInertShortcutAllowsConfirmedMissingDraft(t *testing.T) {
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
	model := shortcutAdmissionModel(2)
	session := model.Sessions["test"]
	session.Panes = append(session.Panes[:1], session.Panes[2:]...)
	model.Sessions["test"] = session
	f.SetModel(model)
	rt := nativeShortcutRuntime{&fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}}}
	if err := handleChord(workbenchshortcut.ChordAltJ, rt, strings.NewReader(""), io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(f.Queries()) != 2 || len(f.Effects()) != 0 || len(rt.recordedTerminal) != 0 {
		t.Fatalf("inert chord queries=%+v effects=%+v records=%v", f.Queries(), f.Effects(), rt.recordedTerminal)
	}
}

func TestTerminalRegistrationRemovalDeniesStalePaneMetadata(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, metadata := range []string{"startup-command", "terminal-title", "both"} {
			t.Run(fmt.Sprintf("cached=%v/%s", cached, metadata), func(t *testing.T) {
				model := shortcutAdmissionModel(2)
				own := model.Sessions["test"].Panes[0]
				if metadata != "startup-command" {
					own["title"] = "terminal"
				}
				if metadata == "terminal-title" {
					own["terminal_command"] = ""
				}
				body, err := json.Marshal(model.Sessions["test"].Panes)
				if err != nil {
					t.Fatal(err)
				}
				rt := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}, panesJSON: string(body)}
				if cached {
					rt.cachedDraft = "2"
				}
				if err := handleChord(workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard); err != nil {
					t.Fatal(err)
				}
				wantReads := 2
				if cached {
					wantReads = 0
				}
				if strings.Join(rt.ops, ",") != "focus-pane-id 2" || strings.Join(rt.recordedTerminal, ",") != "4" || rt.listCalls != wantReads {
					t.Fatalf("registered: effects=%v records=%v reads=%d", rt.ops, rt.recordedTerminal, rt.listCalls)
				}

				// Model the live registry filter dropping the exited wrapper. Its
				// startup command, title and cached draft have not changed.
				rt.terminalPaneIDs = nil
				rt.ops, rt.recordedTerminal, rt.listCalls = nil, nil, 0
				if err := handleChord(workbenchshortcut.ChordAltK, rt, strings.NewReader(""), io.Discard); err != nil {
					t.Fatal(err)
				}
				if len(rt.ops) != 0 || len(rt.recordedTerminal) != 0 || rt.listCalls != 2 {
					t.Fatalf("removed registration: effects=%v records=%v reads=%d", rt.ops, rt.recordedTerminal, rt.listCalls)
				}
				if err := splitTerminalDown(rt); err == nil || len(rt.ops) != 0 {
					t.Fatalf("removed registration admitted split: error=%v effects=%v", err, rt.ops)
				}
			})
		}
	}
}
