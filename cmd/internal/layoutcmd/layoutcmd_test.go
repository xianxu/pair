package layoutcmd

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
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
	listCalls       int
}

func (f *fakeRuntime) LastTerminalPaneID() (string, error) {
	return f.lastTerminal, nil
}

func (f *fakeRuntime) TerminalPaneIDs() ([]string, error) {
	return f.terminalPaneIDs, nil
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	f.listCalls++
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

// The adversarial class for resolveFromSidecars is THE REGISTRY DISAGREEING
// WITH THE PANE REPORT, and hand-picked cases are blind to it by construction —
// I would only write the disagreements I already thought of. So generate the
// space and assert the one property that matters (#220 PQ-2):
//
//	resolveFromSidecars answering  =>  its answer equals pickRightTerminal's
//
// That is what stops Alt+k and Alt+Shift+arrow from landing on different split
// halves (#216 BR-10). Where it declines, the caller falls back and correctness
// is pickRightTerminal's problem, unchanged.
func TestSidecarFastPathAgreesWithThePaneListWheneverItAnswers(t *testing.T) {
	// The adversarial class is THE REGISTRY DISAGREEING WITH THE PANE REPORT,
	// and hand-picked cases are blind to it by construction — I would only write
	// the disagreements I already thought of. So generate the space and assert
	// the one property that matters (#220 PQ-2):
	//
	//	resolveFromSidecars answering  =>  its answer equals pickRightTerminal's
	//
	// That is what stops Alt+k and Alt+Shift+arrow from landing on different
	// split halves (#216 BR-10).
	//
	// The axis that matters is COMPLETENESS, not registry size (BR-7). A
	// registry naming one of two right terminals is incomplete — the startup
	// race — and the generator found that the fast path diverges there. A
	// registry naming the only right terminal is complete, and is the state
	// that exercises the single-live-id branch. Generating one-pane worlds is
	// what reaches that branch at all; the first generator had none and left it
	// unexercised while claiming to cover it.
	pane := func(id string, kind string, focused bool) zellijpane.Pane {
		p := zellijpane.Pane{ID: id, IsFocused: focused, X: 75}
		switch kind {
		case "command":
			p.TerminalCommand = "sh -c exec pair term"
			p.Title = "[terminal 1]"
		case "title":
			p.Title = "terminal 1"
		case "registry": // neither command nor title: only the registry sees it
			p.Title = "[terminal 1]"
		}
		return p
	}
	kinds := []string{"command", "title", "registry"}
	records := []string{"none", "live-registered", "stale", "unregistered-present"}

	checked, answered, single := 0, 0, 0
	for _, size := range []int{1, 2} {
		for _, kindA := range kinds {
			for _, kindB := range kinds {
				for _, complete := range []bool{false, true} {
					for _, record := range records {
						for focus := -1; focus < size; focus++ {
							panes := []zellijpane.Pane{pane("3", kindA, focus == 0)}
							if size == 2 {
								panes = append(panes, pane("4", kindB, focus == 1))
							}
							// Complete: the registry names every right terminal
							// in the world. Otherwise: empty, which is the other
							// consistent state (it claims nothing).
							var registry []string
							if complete {
								for _, p := range panes {
									registry = append(registry, p.ID)
								}
							}
							lastTerminal := ""
							switch record {
							case "live-registered":
								if len(registry) > 0 {
									lastTerminal = registry[0]
								}
							case "stale":
								lastTerminal = "77"
							case "unregistered-present":
								lastTerminal = panes[len(panes)-1].ID
							}

							checked++
							fastID, ok := resolveFromSidecars(registry, lastTerminal)
							if !ok {
								continue
							}
							answered++
							// Count the BRANCH, not an input property (BR-12).
							// A one-entry registry also answers through the
							// record branch, so `len(registry) == 1` survives
							// deleting the single-live-id branch entirely.
							// Only "answered with no record" is that branch's
							// own signature.
							if lastTerminal == "" {
								single++
							}
							slow, found := pickRightTerminal(panes, lastTerminal, registry)
							if !found {
								t.Errorf("size=%d kinds=%s/%s complete=%v record=%s focus=%d: fast answered %q but the pane list found none",
									size, kindA, kindB, complete, record, focus, fastID)
								continue
							}
							if slow.ID != fastID {
								t.Errorf("size=%d kinds=%s/%s complete=%v record=%s focus=%d: fast=%q slow=%q — the two paths would land on different halves",
									size, kindA, kindB, complete, record, focus, fastID, slow.ID)
							}
						}
					}
				}
			}
		}
	}
	if answered == 0 {
		t.Fatalf("the fast path answered none of %d generated cases — the generator is broken, not the code", checked)
	}
	if single == 0 {
		t.Fatalf("no generated case was answered with NO recorded half (%d cases) — the single-live-id branch is unreached, so this space cannot prove what it claims", checked)
	}
	t.Logf("generated %d complete-or-empty-registry cases; answered %d, of which %d through the single-live-id branch (answered with no record)", checked, answered, single)
}

// The invariant that survives an INCONSISTENT registry: the fast path never
// invents an id. It answers with something the registry listed, or it declines
// — so the worst an unregistered half or a stale entry can cost is a fall back
// to the pane list, or the wrong half in a no-record case where
// pickRightTerminal's own answer is pane-order arbitrary anyway.
//
// This is deliberately weaker than the agreement property above, and the
// weakness is the honest one: with an incomplete registry and no recorded half,
// BOTH paths are guessing, and asserting they guess alike would assert a
// coincidence rather than a contract (#220).
func TestSidecarFastPathNeverInventsAnID(t *testing.T) {
	for _, registry := range [][]string{nil, {}, {"4"}, {"3", "4"}, {"9"}} {
		for _, record := range []string{"", "3", "4", "9", "77"} {
			id, ok := resolveFromSidecars(registry, record)
			if !ok {
				continue
			}
			found := false
			for _, candidate := range registry {
				if candidate == id {
					found = true
				}
			}
			if !found {
				t.Errorf("registry=%v record=%q: answered %q, which the registry never listed", registry, record, id)
			}
		}
	}
}

// The fast path must not eat the tie-break. When the registry cannot answer,
// the pane list has to be consulted — otherwise Alt+k and Alt+Shift+arrow stop
// agreeing about which split half they mean (#216 BR-10, the property #220's
// speed-up is not allowed to cost).
func TestSidecarFastPathFallsBackWhenItCannotAnswer(t *testing.T) {
	panes := []byte(`[
		{"id":3,"is_focused":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":true,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"}
	]`)
	for _, test := range []struct {
		name         string
		registry     []string
		lastTerminal string
		wantList     int
		wantID       string
	}{
		{"two halves, no record: only the pane list can choose", []string{"3", "4"}, "", 1, "4"},
		{"empty registry proves nothing", nil, "", 1, "4"},
		{"record names an unregistered half", []string{"4"}, "3", 1, "3"},
		{"record hits a registered half: no list call", []string{"3", "4"}, "3", 0, "3"},
		{"one live half, no record: nothing to tie-break", []string{"4"}, "", 0, "4"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rt := &fakeRuntime{panesJSON: panes, terminalPaneIDs: test.registry, lastTerminal: test.lastTerminal}
			id, ok, err := resolveRightTerminalID(rt)
			if err != nil || !ok {
				t.Fatalf("resolve = %q ok=%v err=%v", id, ok, err)
			}
			if id != test.wantID {
				t.Errorf("id = %q, want %q", id, test.wantID)
			}
			if rt.listCalls != test.wantList {
				t.Errorf("ListPanesJSON called %d times, want %d", rt.listCalls, test.wantList)
			}
		})
	}
}
