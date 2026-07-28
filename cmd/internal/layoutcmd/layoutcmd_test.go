package layoutcmd

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// The toggle tests run against a STATEFUL fake: each resize action mutates the
// geometry the next ListPanesJSON read reports, because zellij's per-step
// resize amount is a runtime detail pair never hardcodes — the executor loops
// read → step → act until the width converges. Live smoke item 4 is the
// conformance check for this modeled behavior.

func TestToggleFocusedExpandsTiledTerminalToTwoThirds(t *testing.T) {
	rt := &fakeRuntime{screenCols: 150, terminalCols: 75, resizeDelta: 8}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	// 75 → 83 → 91 → 99, |99-100| within tolerance.
	want := "resize increase left,resize increase left,resize increase left"
	if got := strings.Join(rt.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
	if rt.terminalCols != 99 {
		t.Fatalf("terminalCols = %d, want 99 (~2/3 of 150)", rt.terminalCols)
	}
}

func TestToggleFocusedCollapsesTiledTerminalToHalf(t *testing.T) {
	rt := &fakeRuntime{screenCols: 150, terminalCols: 100, resizeDelta: 8}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	// 100 ≥ 60% of 150 reads as expanded; 100 → 92 → 84 → 76, |76-75| within tolerance.
	want := "resize decrease left,resize decrease left,resize decrease left"
	if got := strings.Join(rt.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestToggleFocusedStopsWhenResizeHasNoEffect(t *testing.T) {
	// zellij refusing the resize (FIXED pane, minimum sizes) must not spin
	// the loop: no progress → stop after the first attempt.
	rt := &fakeRuntime{screenCols: 150, terminalCols: 75, resizeDelta: 0}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 1 {
		t.Fatalf("ops = %v, want exactly one attempted resize", rt.ops)
	}
}

func TestToggleFocusedRespectsStepCap(t *testing.T) {
	rt := &fakeRuntime{screenCols: 150, terminalCols: 75, resizeDelta: 1}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != terminalResizeMaxSteps {
		t.Fatalf("len(ops) = %d, want cap %d", len(rt.ops), terminalResizeMaxSteps)
	}
}

func TestToggleFocusedIgnoresLeftFocus(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":1,"is_plugin":false,"is_focused":true,"is_floating":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":4,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":75,"title":"terminal","terminal_command":"pair term"}
	]`)}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want no-op for left focus", rt.ops)
	}
}

func TestToggleFocusedRefusesWithoutGeometry(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":false,"title":"terminal","terminal_command":"pair term"}
	]`)}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want no-op without tiled geometry", rt.ops)
	}
}

func TestFocusRightTerminalFocusesTiledTerminalByID(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":2,"is_focused":true,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"},
		{"id":4,"is_focused":false,"is_floating":false,"pane_x":75,"title":"terminal","terminal_command":"pair term"}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "focus-pane-id 4" {
		t.Fatalf("ops = %v, want [focus-pane-id 4]", rt.ops)
	}
}

func TestFocusRightTerminalPrefersRecordedSplitHalf(t *testing.T) {
	// Focus sits in the left stack, so neither tiled right terminal reports
	// is_focused; the recorded last-terminal pane id picks the half.
	rt := &fakeRuntime{lastTerminal: "4", panesJSON: []byte(`[
		{"id":2,"is_focused":true,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"},
		{"id":3,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "focus-pane-id 4" {
		t.Fatalf("ops = %v, want [focus-pane-id 4]", rt.ops)
	}
}

func TestFocusRightTerminalPrefersRecordedOverZellijFocus(t *testing.T) {
	// zellij's is_focused on right-side panes is stale memory while the user
	// sits in the left stack (live smoke: it pointed at the top half right
	// after the user left the bottom one). The pair-authored record wins.
	rt := &fakeRuntime{lastTerminal: "4", panesJSON: []byte(`[
		{"id":3,"is_focused":true,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "focus-pane-id 4" {
		t.Fatalf("ops = %v, want [focus-pane-id 4] (recorded half)", rt.ops)
	}
}

func TestFocusRightTerminalIgnoresStaleRecordedID(t *testing.T) {
	rt := &fakeRuntime{lastTerminal: "9", panesJSON: []byte(`[
		{"id":2,"is_focused":true,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"},
		{"id":3,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "focus-pane-id 3" {
		t.Fatalf("ops = %v, want [focus-pane-id 3]", rt.ops)
	}
}

func TestFocusRightTerminalSeesRegistryOnlySplitHalf(t *testing.T) {
	// A split half as zellij 0.44.3 actually reports it: terminal_command
	// null, #118 tab-strip title. Only the registry identifies it; the
	// recorded last-used half must still be reachable.
	rt := &fakeRuntime{lastTerminal: "4", terminalPaneIDs: []string{"1", "4"}, panesJSON: []byte(`[
		{"id":2,"is_focused":true,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"},
		{"id":1,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":null}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "focus-pane-id 4" {
		t.Fatalf("ops = %v, want [focus-pane-id 4]", rt.ops)
	}
}

func TestFocusRightTerminalFallsBackToRelativeMoveWithoutTerminal(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":0,"is_focused":true,"is_floating":false,"title":"agent","terminal_command":"pair wrap claude"},
		{"id":2,"is_focused":false,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"}
	]`)}
	if err := FocusRightTerminal(rt); err != nil {
		t.Fatal(err)
	}
	if len(rt.ops) != 1 || rt.ops[0] != "move-focus right" {
		t.Fatalf("ops = %v, want [move-focus right]", rt.ops)
	}
}

type fakeRuntime struct {
	panesJSON       []byte
	ops             []string
	lastTerminal    string
	terminalPaneIDs []string
	// Stateful tiled geometry, active when screenCols > 0: ListPanesJSON
	// renders a workbench from terminalCols, and each resize action mutates
	// terminalCols by ±resizeDelta — modeling that zellij applies some
	// runtime-defined amount per resize step.
	screenCols   int
	terminalCols int
	resizeDelta  int
}

func (f *fakeRuntime) LastTerminalPaneID() (string, error) {
	return f.lastTerminal, nil
}

func (f *fakeRuntime) TerminalPaneIDs() ([]string, error) {
	return f.terminalPaneIDs, nil
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	if f.screenCols > 0 {
		leftCols := f.screenCols - f.terminalCols
		return []byte(`[
			{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + strconv.Itoa(leftCols) + `,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
			{"id":2,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + strconv.Itoa(leftCols) + `,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
			{"id":4,"is_plugin":false,"is_focused":true,"is_floating":false,"pane_x":` + strconv.Itoa(leftCols) + `,"pane_columns":` + strconv.Itoa(f.terminalCols) + `,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
		]`), nil
	}
	return f.panesJSON, nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	op := strings.Join(args, " ")
	f.ops = append(f.ops, op)
	switch op {
	case "resize increase left":
		f.terminalCols += f.resizeDelta
	case "resize decrease left":
		f.terminalCols -= f.resizeDelta
	}
	return nil
}
