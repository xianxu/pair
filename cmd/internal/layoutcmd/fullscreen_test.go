package layoutcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		{ID: "3", Title: "terminal top", IsFullscreen: &off, Columns: 80, Rows: 20, Y: 0},
		{ID: "4", Title: "terminal bottom", IsFocused: true, IsFullscreen: &off, Columns: 80, Rows: 20, Y: 12},
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

// fullscreenWorld is a stateful zellij fake. Beyond fullscreen and focus it
// models what the #417 probes measured on 0.45.1: a CLI float leaves the
// floating layer hidden; show-floating-panes exits 2 when already shown; a
// re-embed lands the pane at the bottom of the column and dirties the swap
// layout; next-swap-layout re-tiles positionally (keeping half order) through a
// per-pane-count cycle; move-pane swaps a half with its neighbour.
type fullscreenWorld struct {
	panes                       []zellijpane.Pane
	current, last, record, fail string
	ops, logs                   []string
	busy, visible, dirty        bool
	swap                        string
	lists, nudges               int
}

var fakeSwapCycles = map[int][]string{
	3: {"BASE", "minimized", "third"},
	4: {"minimized-split", "third-split", "small-split"},
}

func newFullscreenWorld() *fullscreenWorld {
	return &fullscreenWorld{panes: fullscreenPanes(), current: "2", last: "4", swap: "small-split"}
}

type exitStatus int

func (e exitStatus) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitStatus) ExitCode() int { return int(e) }

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
func (w *fullscreenWorld) NudgeWrap() error {
	w.nudges++
	return nil
}
func (w *fullscreenWorld) CurrentTabJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"active_swap_layout_name": w.swap, "is_swap_layout_dirty": w.dirty})
}
func (w *fullscreenWorld) ListPanesJSON() ([]byte, error) {
	w.lists++
	var out []map[string]any
	for _, p := range w.panes {
		out = append(out, map[string]any{"id": p.ID, "title": p.Title, "terminal_command": p.TerminalCommand, "is_plugin": p.IsPlugin, "is_focused": p.IsFocused, "is_floating": p.IsFloating, "is_fullscreen": p.IsFullscreen, "pane_columns": p.Columns, "pane_rows": p.Rows, "pane_y": p.Y})
	}
	return json.Marshal(out)
}
func (w *fullscreenWorld) pane(id string) *zellijpane.Pane {
	for i := range w.panes {
		if w.panes[i].ID == id && !w.panes[i].IsPlugin {
			return &w.panes[i]
		}
	}
	return nil
}
func (w *fullscreenWorld) tiled() int {
	n := 0
	for _, p := range w.panes {
		if !p.IsPlugin && !p.IsFloating {
			n++
		}
	}
	return n
}
func (w *fullscreenWorld) RunZellijAction(args ...string) error {
	if err := w.step(strings.Join(args, " ")); err != nil {
		return err
	}
	switch args[0] {
	case "show-floating-panes":
		if w.visible {
			return exitStatus(2)
		}
		w.visible = true
		return nil
	case "next-swap-layout":
		cycle := fakeSwapCycles[w.tiled()]
		next := 0
		for i, name := range cycle {
			if name == w.swap {
				next = (i + 1) % len(cycle)
			}
		}
		w.swap, w.dirty = cycle[next], false
		return nil
	}
	id := args[len(args)-1]
	for i, arg := range args[:len(args)-1] {
		if arg == "--pane-id" {
			id = args[i+1]
		}
	}
	p := w.pane(id)
	if p == nil {
		return fmt.Errorf("missing pane in %v", args)
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
		w.current = p.ID
	case "focus-pane-id":
		w.current = p.ID
	case "toggle-pane-embed-or-floating":
		w.dirty = true
		if p.IsFloating {
			p.IsFloating, p.Y = false, 99
		} else {
			p.IsFloating, w.visible, w.current = true, false, p.ID
		}
	case "change-floating-pane-coordinates":
		if !p.IsFloating {
			return fmt.Errorf("placing tiled pane %s", p.ID)
		}
	case "move-pane":
		for i := range w.panes {
			if q := &w.panes[i]; q.ID != p.ID && (q.ID == "3" || q.ID == "4") && !q.IsFloating {
				p.Y, q.Y = q.Y, p.Y
			}
		}
		w.dirty = true
	default:
		return fmt.Errorf("unexpected action %v", args)
	}
	return nil
}

// order reads the halves top to bottom; re-tiling keeps order, not rows.
func (w *fullscreenWorld) order() string {
	if w.pane("3").Y < w.pane("4").Y {
		return "3,4"
	}
	return "4,3"
}

func TestRightPaneRuntimeCycleRestoresWorkbench(t *testing.T) {
	for _, tc := range []struct {
		name, caller, swap string
		target             string
	}{
		{"draft-to-recorded-bottom", "2", "small-split", "4"},
		{"agent-third", "1", "third-split", "4"},
		{"top-half-minimized", "3", "minimized-split", "3"},
		{"dirty-split-after-alt-shift-d", "2", "third", "4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newFullscreenWorld()
			w.current, w.swap = tc.caller, tc.swap
			w.dirty = !strings.HasSuffix(tc.swap, "-split")
			wantSwap := restoreTarget(tc.swap, 2)
			press := func() {
				t.Helper()
				var errout bytes.Buffer
				if code := RunToggleFocused(nil, w, &errout); code != 0 {
					t.Fatalf("code=%d %s logs=%v", code, &errout, w.logs)
				}
			}
			press() // normal → focus
			if p := w.pane(tc.target); !p.IsFloating || !w.visible || w.current != tc.target || w.nudges != 1 {
				t.Fatalf("focus: %+v visible=%v current=%s nudges=%d", p, w.visible, w.current, w.nudges)
			}
			if got := DecodeExpandRecord(w.record); got.Return != tc.caller || got.Swap != tc.swap || strings.Join(got.Order, ",") != "3,4" {
				t.Fatalf("record %q", w.record)
			}
			press() // focus → maximize
			p := w.pane(tc.target)
			// Re-embedding the top half scrambles the order; the move-pane that
			// fixes it leaves the swap layout dirty (as live zellij does).
			if p.IsFloating || !*p.IsFullscreen || w.swap != wantSwap || w.dirty != (tc.target == "3") || w.order() != "3,4" || w.nudges != 2 {
				t.Fatalf("maximize: %+v swap=%s dirty=%v order=%s logs=%v", p, w.swap, w.dirty, w.order(), w.logs)
			}
			press() // maximize → normal
			if *p.IsFullscreen || w.current != tc.caller || w.record != "" || w.order() != "3,4" || len(w.logs) != 0 {
				t.Fatalf("normal: %+v current=%s record=%q order=%s logs=%v", p, w.current, w.record, w.order(), w.logs)
			}
		})
	}
}

func TestRightPaneRuntimeFailures(t *testing.T) {
	// A failing step stops the press at that step and is logged; the lock is released.
	for _, tc := range []struct {
		presses int
		failure string
	}{
		{0, "save"}, {0, "toggle-pane-embed-or-floating"}, {0, "change-floating-pane-coordinates"},
		{1, "toggle-pane-embed-or-floating"}, {1, "toggle-fullscreen"},
		{2, "toggle-fullscreen"}, {2, "focus-pane-id"}, {2, "clear"},
	} {
		w := newFullscreenWorld()
		for i := 0; i < tc.presses; i++ {
			if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 0 {
				t.Fatalf("setup press %d: %d %v", i, code, w.logs)
			}
		}
		w.ops, w.fail = nil, tc.failure
		if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 1 {
			t.Fatalf("%d/%s: code %d", tc.presses, tc.failure, code)
		}
		if len(w.logs) != 1 || !strings.Contains(w.logs[0], tc.failure) || w.busy {
			t.Fatalf("%d/%s: %+v", tc.presses, tc.failure, w)
		}
		if !strings.HasPrefix(w.ops[len(w.ops)-1], tc.failure) {
			t.Fatalf("%d/%s continued after failure: %v", tc.presses, tc.failure, w.ops)
		}
	}
	// A restore that cannot reach its layout degrades, never blocks maximize.
	w := newFullscreenWorld()
	if RunToggleFocused(nil, w, &bytes.Buffer{}) != 0 {
		t.Fatal("focus")
	}
	w.record = ExpandRecord{Return: "2", Swap: "nonexistent"}.Encode()
	if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 0 || !*w.pane("4").IsFullscreen {
		t.Fatalf("degraded restore blocked maximize: %d %v", code, w.logs)
	}
	if len(w.logs) != 1 || !strings.Contains(w.logs[0], "restore tiling") {
		t.Fatalf("logs %v", w.logs)
	}
	// Busy lock: the press is dropped, not queued.
	w = newFullscreenWorld()
	w.busy = true
	if code := RunToggleFocused(nil, w, &bytes.Buffer{}); code != 0 || len(w.ops) != 0 || w.lists != 0 {
		t.Fatal("busy toggle did work")
	}
}

func TestRightPaneRuntimeShowAlreadyVisibleIsSuccess(t *testing.T) {
	w := newFullscreenWorld()
	// A pre-existing visible floating layer makes show-floating-panes exit 2.
	w.visible = true
	w.panes[4].IsFloating = false
	if code := RunToggleFocused(nil, &keepVisible{w}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit-2 show treated as failure: %v", w.logs)
	}
}

// keepVisible models a floating layer that a float does not hide.
type keepVisible struct{ *fullscreenWorld }

func (k *keepVisible) RunZellijAction(args ...string) error {
	err := k.fullscreenWorld.RunZellijAction(args...)
	if args[0] == "toggle-pane-embed-or-floating" {
		k.visible = true
	}
	return err
}
