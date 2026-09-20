package layoutcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func fullscreenPanes() []zellijpane.Pane {
	off := false
	return []zellijpane.Pane{
		{ID: "0", IsPlugin: true, IsFocused: true},
		{ID: "1", Title: "agent", TerminalCommand: "pair wrap codex", IsFocused: true, IsFullscreen: &off, Columns: 80, Rows: 20},
		{ID: "2", Title: "draft", TerminalCommand: "nvim -u /pair/nvim/init.lua", IsFocused: true, IsFullscreen: &off, Columns: 80, Rows: 20},
		{ID: "3", Title: "terminal top", IsFullscreen: &off, Columns: 80, Rows: 20},
		{ID: "4", Title: "terminal bottom", IsFocused: true, IsFullscreen: &off, Columns: 80, Rows: 20},
	}
}

func TestPlanFullscreenUsesCallerAndPreservesHalf(t *testing.T) {
	for _, caller := range []string{"1", "2", "3", "4"} {
		for _, last := range []string{"", "3", "4", "99"} {
			panes := fullscreenPanes()
			plan, err := PlanFullscreen(panes, caller, last, nil, "stale")
			if err != nil {
				t.Fatal(err)
			}
			want := "4" // focused half is fallback
			if last == "3" || last == "4" {
				want = last
			}
			if caller == "3" || caller == "4" {
				want = caller
			}
			if plan.Operation != FullscreenExpand || plan.TerminalID != want || plan.ReturnID != caller {
				t.Fatalf("caller=%s last=%s: %+v", caller, last, plan)
			}
		}
	}
}

func TestPlanFullscreenCollapsesObservedHalf(t *testing.T) {
	panes := fullscreenPanes()
	on := true
	panes[4].IsFullscreen = &on
	for _, recorded := range []string{"1", "2", "4", "99", "", "plugin_0"} {
		plan, err := PlanFullscreen(panes, "4", "3", nil, recorded)
		if err != nil {
			t.Fatal(err)
		}
		want := recorded
		if recorded == "99" || recorded == "" || recorded == "plugin_0" {
			want = "2"
		}
		if plan.Operation != FullscreenCollapse || plan.TerminalID != "4" || plan.ReturnID != want {
			t.Fatalf("%s: %+v", recorded, plan)
		}
	}
	panes = append(panes[:2], panes[3:]...)
	plan, err := PlanFullscreen(panes, "4", "3", nil, "99")
	if err != nil || plan.Operation != FullscreenCollapse || plan.ReturnID != "" {
		t.Fatalf("no draft: %+v %v", plan, err)
	}
}

func TestPlanFullscreenRefusesUnknownIdentityAndDirection(t *testing.T) {
	for _, caller := range []string{"", "99", "plugin_0"} {
		if _, err := PlanFullscreen(fullscreenPanes(), caller, "", nil, ""); err == nil {
			t.Fatalf("accepted ambiguous/stale caller %q", caller)
		}
	}
	panes := fullscreenPanes()
	panes[4].IsFullscreen = nil
	if _, err := PlanFullscreen(panes, "2", "4", nil, ""); err == nil {
		t.Fatal("unknown fullscreen treated as tiled")
	}
	plan, err := PlanFullscreen(panes[:3], "2", "", nil, "")
	if err != nil || plan.Operation != FullscreenNoop {
		t.Fatalf("layout2: %+v %v", plan, err)
	}
}

func TestFullscreenTransitionOrderAndStoppedOutcomes(t *testing.T) {
	for _, tc := range []struct {
		op     FullscreenOperation
		target string
		want   []FullscreenEffect
	}{
		{FullscreenExpand, "2", []FullscreenEffect{FullscreenSave, FullscreenToggle}},
		{FullscreenCollapse, "2", []FullscreenEffect{FullscreenToggle, FullscreenFocus, FullscreenClear}},
		{FullscreenCollapse, "", []FullscreenEffect{FullscreenToggle, FullscreenClear}},
		{FullscreenCollapse, "4", []FullscreenEffect{FullscreenToggle, FullscreenClear}},
		{FullscreenNoop, "", nil},
	} {
		for failAt := -1; failAt < len(tc.want); failAt++ {
			for _, failure := range []FullscreenEvent{FullscreenFailed, FullscreenUnconfirmed} {
				s := NewFullscreenState(FullscreenPlan{Operation: tc.op, TerminalID: "4", ReturnID: tc.target})
				var got []FullscreenEffect
				s, effect := FullscreenTransition(s, FullscreenBegin)
				for effect != FullscreenNothing {
					got = append(got, effect)
					event := FullscreenConfirmed
					if len(got)-1 == failAt {
						event = failure
					}
					s, effect = FullscreenTransition(s, event)
					if len(got) > 4 {
						t.Fatal("unbounded effect loop")
					}
				}
				want := tc.want
				if failAt >= 0 {
					want = want[:failAt+1]
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("op=%v fail=%d got=%v want=%v", tc.op, failAt, got, want)
				}
				for _, late := range []FullscreenEvent{FullscreenBegin, FullscreenConfirmed, FullscreenFailed, FullscreenUnconfirmed} {
					next, eff := FullscreenTransition(s, late)
					if next != s || eff != FullscreenNothing {
						t.Fatal("terminal state accepted late outcome")
					}
				}
			}
		}
	}
}

type fullscreenWorld struct {
	panes                       []zellijpane.Pane
	current, last, record, fail string
	ops, logs                   []string
	busy                        bool
	lists                       int
}

func newFullscreenWorld() *fullscreenWorld {
	return &fullscreenWorld{panes: fullscreenPanes(), current: "2", last: "4"}
}
func (w *fullscreenWorld) CurrentPaneID() string                              { return w.current }
func (w *fullscreenWorld) LastTerminalPaneID() (string, error)                { return w.last, nil }
func (w *fullscreenWorld) TerminalPaneIDs() ([]string, error)                 { return []string{"3", "4"}, nil }
func (w *fullscreenWorld) FullscreenStore() workbenchshortcut.FullscreenStore { return w }
func (w *fullscreenWorld) TryLock() (func(), bool, error) {
	if w.busy {
		return nil, false, nil
	}
	w.busy = true
	return func() { w.busy = false }, true, nil
}
func (w *fullscreenWorld) Read() (string, error) { return w.record, nil }
func (w *fullscreenWorld) step(op string) error {
	w.ops = append(w.ops, op)
	if strings.HasPrefix(op, w.fail) && w.fail != "" {
		return errors.New("injected " + op)
	}
	return nil
}
func (w *fullscreenWorld) Write(id string) error {
	if err := w.step("save " + id); err != nil {
		return err
	}
	w.record = id
	return nil
}
func (w *fullscreenWorld) Clear() error {
	if err := w.step("clear"); err != nil {
		return err
	}
	w.record = ""
	return nil
}
func (w *fullscreenWorld) LogFailure(err error) { w.logs = append(w.logs, err.Error()) }
func (w *fullscreenWorld) ListPanesJSON() ([]byte, error) {
	w.lists++
	var out []map[string]any
	for _, p := range w.panes {
		out = append(out, map[string]any{"id": p.ID, "title": p.Title, "terminal_command": p.TerminalCommand, "is_plugin": p.IsPlugin, "is_focused": p.IsFocused, "is_fullscreen": p.IsFullscreen, "pane_columns": p.Columns, "pane_rows": p.Rows})
	}
	return json.Marshal(out)
}
func (w *fullscreenWorld) RunZellijAction(args ...string) error {
	if err := w.step(strings.Join(args, " ")); err != nil {
		return err
	}
	id := args[len(args)-1]
	for i := range w.panes {
		p := &w.panes[i]
		if p.ID != id || p.IsPlugin {
			continue
		}
		switch args[0] {
		case "toggle-fullscreen":
			on := !(*p.IsFullscreen)
			p.IsFullscreen = &on
			if on {
				p.Columns, p.Rows = 160, 40
			} else {
				p.Columns, p.Rows = 80, 20
			}
			w.current = id
		case "focus-pane-id":
			w.current = id
		default:
			return fmt.Errorf("unexpected action %v", args)
		}
		return nil
	}
	return fmt.Errorf("missing pane %s", id)
}

func TestFullscreenRuntimeRoundTripAndFailures(t *testing.T) {
	w := newFullscreenWorld()
	for i := 0; i < 2; i++ {
		var errout bytes.Buffer
		if code := RunToggleFocused(nil, w, &errout); code != 0 {
			t.Fatalf("code=%d %s", code, &errout)
		}
	}
	if w.current != "2" || w.last != "4" || w.record != "" || w.lists != 2 || w.panes[4].Columns != 80 || w.panes[4].Rows != 20 {
		t.Fatalf("bad round trip: %+v", w)
	}
	want := []string{"save 2", "toggle-fullscreen --pane-id 4", "toggle-fullscreen --pane-id 4", "focus-pane-id 2", "clear"}
	if !reflect.DeepEqual(w.ops, want) {
		t.Fatalf("ops %v", w.ops)
	}
	for _, failure := range []string{"save", "toggle-fullscreen", "focus-pane-id", "clear"} {
		w := newFullscreenWorld()
		if failure == "focus-pane-id" || failure == "clear" {
			if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 0 {
				t.Fatal(code)
			}
			w.ops = nil
		}
		w.fail = failure
		if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 1 {
			t.Fatalf("%s: code %d", failure, code)
		}
		if len(w.logs) != 1 || !strings.Contains(w.logs[0], failure) || w.busy {
			t.Fatalf("%s: %+v", failure, w)
		}
		if !strings.HasPrefix(w.ops[len(w.ops)-1], failure) {
			t.Fatalf("continued after failure: %v", w.ops)
		}
	}
	w = newFullscreenWorld()
	w.busy = true
	if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 0 || len(w.ops) != 0 || w.lists != 0 {
		t.Fatal("busy toggle did work")
	}
}
