package layoutcmd

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestToggleFocusedWidensLeftSide(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("agent", 50, 50)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	got := strings.Join(rt.ops, "\n")
	if !strings.Contains(got, "override-layout --apply-only-to-active-tab --layout-string") {
		t.Fatalf("ops = %v, want override-layout", rt.ops)
	}
	if !strings.Contains(got, `pane size="67%" split_direction="horizontal"`) {
		t.Fatalf("layout = %q, want left side 67%%", got)
	}
	if strings.Contains(got, `pane size="67%" name="terminal"`) {
		t.Fatalf("layout = %q, terminal must not be wide", got)
	}
}

func TestToggleFocusedWidensRightSide(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("terminal", 50, 50)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	got := strings.Join(rt.ops, "\n")
	if !strings.Contains(got, `pane size="67%" name="terminal"`) {
		t.Fatalf("layout = %q, want terminal side 67%%", got)
	}
	if strings.Contains(got, `pane size="67%" split_direction="horizontal"`) {
		t.Fatalf("layout = %q, left side must not be wide", got)
	}
}

func TestToggleFocusedWideSideReturnsToBalanced(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("terminal", 33, 67)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	got := strings.Join(rt.ops, "\n")
	if strings.Contains(got, `size="67%"`) {
		t.Fatalf("layout = %q, want balanced layout with no horizontal 67%% size", got)
	}
}

func TestToggleFocusedPreservesDraftRung(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSONWithDraftRows("agent", 50, 50, 20)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	got := strings.Join(rt.ops, "\n")
	if !strings.Contains(got, `pane size="33%" name="draft" borderless=true`) {
		t.Fatalf("layout = %q, want third-height draft rung preserved", got)
	}
}

func panesJSON(focused string, leftCols, rightCols int) []byte {
	return panesJSONWithDraftRows(focused, leftCols, rightCols, 12)
}

func panesJSONWithDraftRows(focused string, leftCols, rightCols, draftRows int) []byte {
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
	return []byte(`[
		{"id":1,"is_plugin":false,"is_focused":` + focusAgent + `,"is_floating":false,"title":"codex","terminal_command":"pair wrap codex","pane_x":0,"pane_columns":` + itoa(leftCols) + `,"pane_rows":28},
		{"id":2,"is_plugin":false,"is_focused":` + focusDraft + `,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md","pane_x":0,"pane_columns":` + itoa(leftCols) + `,"pane_rows":` + itoa(draftRows) + `},
		{"id":3,"is_plugin":false,"is_focused":` + focusTerminal + `,"is_floating":false,"title":"terminal","terminal_command":"pair term","pane_x":` + itoa(leftCols) + `,"pane_columns":` + itoa(rightCols) + `,"pane_rows":40}
	]`)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

type fakeRuntime struct {
	panesJSON []byte
	ops       []string
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	return f.panesJSON, nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	return nil
}
