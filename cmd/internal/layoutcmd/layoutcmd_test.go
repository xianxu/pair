package layoutcmd

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestToggleFocusedExpandsTiledRightTerminalWithoutChangingTopology(t *testing.T) {
	rt := &fakeRuntime{
		panesJSON: workbenchJSON(75, 75, false),
		panesJSONAfterActions: [][]byte{
			workbenchJSON(50, 100, false),
		},
	}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if got, want := strings.Join(rt.ops, "\n"), "resize increase left --pane-id 3"; got != want {
		t.Fatalf("ops = %q, want topology-preserving resize %q", got, want)
	}
}

func TestToggleFocusedBalancesExpandedTiledRightTerminal(t *testing.T) {
	rt := &fakeRuntime{
		panesJSON: workbenchJSON(50, 100, false),
		panesJSONAfterActions: [][]byte{
			workbenchJSON(75, 75, false),
		},
	}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if got, want := strings.Join(rt.ops, "\n"), "resize decrease left --pane-id 3"; got != want {
		t.Fatalf("ops = %q, want topology-preserving resize %q", got, want)
	}
}

func TestToggleFocusedDoesNotAcceptVisiblyOffCenterCollapse(t *testing.T) {
	rt := &fakeRuntime{
		panesJSON: workbenchJSON(50, 100, false),
		panesJSONAfterActions: [][]byte{
			workbenchJSON(60, 90, false),
			workbenchJSON(68, 82, false),
			workbenchJSON(75, 75, false),
		},
	}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := strings.Join([]string{
		"resize decrease left --pane-id 3",
		"resize decrease left --pane-id 3",
		"resize decrease left --pane-id 3",
	}, "\n")
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want exact balanced reconciliation %q", got, want)
	}
}

func TestToggleFocusedReversesOneStepWhenExpansionOvershoots(t *testing.T) {
	rt := &fakeRuntime{
		panesJSON: workbenchJSON(75, 75, false),
		panesJSONAfterActions: [][]byte{
			workbenchJSON(55, 95, false),
			workbenchJSON(45, 105, false),
		},
	}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := strings.Join([]string{
		"resize increase left --pane-id 3",
		"resize increase left --pane-id 3",
		"resize decrease left --pane-id 3",
	}, "\n")
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want closest-width rollback %q", got, want)
	}
}

func TestToggleFocusedNeverFloatsOrOverridesLayout(t *testing.T) {
	rt := &fakeRuntime{
		panesJSON: workbenchJSON(75, 75, false),
		panesJSONAfterActions: [][]byte{
			workbenchJSON(50, 100, false),
		},
	}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	for _, op := range rt.ops {
		if strings.Contains(op, "floating") || strings.HasPrefix(op, "override-layout ") {
			t.Fatalf("toggle must preserve the tiled split tree, got %q", op)
		}
	}
}

func TestToggleFocusedIgnoresLeftFocus(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("agent", false)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want no-op for left focus", rt.ops)
	}
}

func workbenchJSON(leftWidth, terminalWidth int, terminalFloating bool) []byte {
	floating := "false"
	if terminalFloating {
		floating = "true"
	}
	return []byte(`[
		{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + itoa(leftWidth) + `,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":` + itoa(leftWidth) + `,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_plugin":false,"is_focused":true,"is_floating":` + floating + `,"pane_x":` + itoa(leftWidth) + `,"pane_columns":` + itoa(terminalWidth) + `,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func panesJSON(focused string, terminalFloating bool) []byte {
	focusAgent := "false"
	focusDraft := "false"
	focusTerminal := "false"
	switch focused {
	case "agent":
		focusAgent = "true"
	case "draft":
		focusDraft = "true"
	case "terminal":
		focusTerminal = "true"
	}
	floating := "false"
	if terminalFloating {
		floating = "true"
	}
	return []byte(`[
		{"id":1,"is_plugin":false,"is_focused":` + focusAgent + `,"is_floating":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":` + focusDraft + `,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_plugin":false,"is_focused":` + focusTerminal + `,"is_floating":` + floating + `,"title":"terminal","terminal_command":"pair term"}
	]`)
}

type fakeRuntime struct {
	panesJSON             []byte
	panesJSONAfterActions [][]byte
	ops                   []string
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	if len(f.ops) > 0 && len(f.panesJSONAfterActions) > 0 {
		next := f.panesJSONAfterActions[0]
		f.panesJSONAfterActions = f.panesJSONAfterActions[1:]
		return next, nil
	}
	return f.panesJSON, nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	return nil
}
