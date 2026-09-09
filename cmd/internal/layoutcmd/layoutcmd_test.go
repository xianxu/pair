package layoutcmd

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"strconv"
	"strings"
	"testing"
)

// The toggle fires a fixed three-action burst (zellij's tiled resize step is
// a stable 5% of the screen, so 1/2 ↔ ~2/3 is always exactly three steps) —
// no settle pauses, no re-reads. #124.

func TestToggleFocusedExpandsInOneBurst(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(tiledWorkbenchJSON(75, 150))}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "resize increase left,resize increase left,resize increase left"
	if got := strings.Join(rt.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestToggleFocusedCollapsesInOneBurst(t *testing.T) {
	// 105/150 = 70% ≥ 60% reads as expanded.
	rt := &fakeRuntime{panesJSON: []byte(tiledWorkbenchJSON(105, 150))}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "resize decrease left,resize decrease left,resize decrease left"
	if got := strings.Join(rt.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func tiledWorkbenchJSON(terminalCols, screenCols int) string {
	left := strconv.Itoa(screenCols - terminalCols)
	return `[
		{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + left + `,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + left + `,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":false,"pane_x":` + left + `,"pane_columns":` + strconv.Itoa(terminalCols) + `,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`
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
}

func (f *fakeRuntime) LastTerminalPaneID() (string, error) {
	return f.lastTerminal, nil
}

func (f *fakeRuntime) TerminalPaneIDs() ([]string, error) {
	return f.terminalPaneIDs, nil
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	return f.panesJSON, nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	return nil
}

// Alt+Left is \x1b[1;3D — 27 91 49 59 51 68 as decimal bytes. Asserted as the
// COMPLETE argv rather than a prefix: the bytes are the whole payload, and a
// prefix check would pass with the wrong arrow.
func TestSwitchRightTerminalTabWritesTheChordToTheRecordedHalf(t *testing.T) {
	for _, test := range []struct {
		name  string
		chord workbenchshortcut.Chord
		want  string
	}{
		{"previous", workbenchshortcut.ChordAltLeft, "write --pane-id 4 27 91 49 59 51 68"},
		{"next", workbenchshortcut.ChordAltRight, "write --pane-id 4 27 91 49 59 51 67"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Two split halves; the recorded one must win, exactly as the focus
			// jump picks it — otherwise Alt+k and Alt+Shift+arrow disagree.
			rt := &fakeRuntime{lastTerminal: "4", panesJSON: []byte(`[
				{"id":3,"is_focused":true,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
				{"id":4,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}
			]`)}
			if err := SwitchRightTerminalTab(rt, test.chord); err != nil {
				t.Fatal(err)
			}
			if len(rt.ops) != 1 || rt.ops[0] != test.want {
				t.Fatalf("ops = %v, want [%s]", rt.ops, test.want)
			}
		})
	}
}

func TestSwitchRightTerminalTabIsInertWithoutATerminalPane(t *testing.T) {
	// layout2, or a layout3 whose right pane exited: nothing to switch, and the
	// chord must not report or move focus.
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":0,"is_focused":true,"is_floating":false,"title":"agent","terminal_command":"pair wrap claude"},
		{"id":2,"is_focused":false,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"}
	]`)}
	if err := SwitchRightTerminalTab(rt, workbenchshortcut.ChordAltLeft); err != nil {
		t.Fatalf("inert case returned %v, want nil", err)
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want none", rt.ops)
	}
}

func TestRunSwitchTerminalTabParsesItsDirection(t *testing.T) {
	panes := []byte(`[{"id":3,"is_focused":true,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}]`)
	for _, test := range []struct {
		name string
		args []string
		code int
		ops  int
	}{
		{"prev", []string{"prev"}, 0, 1},
		{"next", []string{"next"}, 0, 1},
		{"missing", nil, 2, 0},
		{"unknown", []string{"sideways"}, 2, 0},
		{"too many", []string{"prev", "next"}, 2, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			rt := &fakeRuntime{panesJSON: panes}
			var stderr bytes.Buffer
			if code := RunSwitchTerminalTab(test.args, rt, &stderr); code != test.code {
				t.Errorf("exit = %d, want %d (stderr %q)", code, test.code, stderr.String())
			}
			if len(rt.ops) != test.ops {
				t.Errorf("ops = %v, want %d", rt.ops, test.ops)
			}
		})
	}
}
