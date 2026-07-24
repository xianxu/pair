package layoutcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestToggleFocusedExpandsFloatingTerminalInOnePreciseAction(t *testing.T) {
	rt := &fakeRuntime{panesJSON: layeredWorkbenchJSON(75, 75)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "change-floating-pane-coordinates --pane-id 4 --x 50 --y 0 --width 100 --height 51 --borderless false --pinned true"
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want one precise action %q", got, want)
	}
}

func TestToggleFocusedCollapsesFloatingTerminalInOnePreciseAction(t *testing.T) {
	rt := &fakeRuntime{panesJSON: layeredWorkbenchJSON(50, 100)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "change-floating-pane-coordinates --pane-id 4 --x 75 --y 0 --width 75 --height 51 --borderless false --pinned true"
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want one precise action %q", got, want)
	}
}

func TestToggleFocusedUsesFillerBoundaryRatherThanRoundedHalf(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":86,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":true,"is_floating":false,"pane_x":0,"pane_columns":86,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":86,"pane_columns":85,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"},
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":true,"pane_x":57,"pane_columns":114,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`)}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "change-floating-pane-coordinates --pane-id 4 --x 86 --y 0 --width 85 --height 51 --borderless false --pinned true"
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want filler-anchored collapse %q", got, want)
	}
}

func TestAlignFloatingTerminalUsesFillerBoundaryAtStartup(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":86,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":true,"is_floating":false,"pane_x":0,"pane_columns":86,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":86,"pane_columns":85,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"},
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":true,"pane_x":85,"pane_columns":85,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`)}

	if err := AlignFloatingTerminal(rt); err != nil {
		t.Fatal(err)
	}
	want := "change-floating-pane-coordinates --pane-id 4 --x 86 --y 0 --width 85 --height 51 --borderless false --pinned true"
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want startup alignment %q", got, want)
	}
}

func TestToggleFocusedUsesPercentageFallbackWithoutTiledGeometry(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":true,"title":"terminal","terminal_command":"pair term"}
	]`)}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := "change-floating-pane-coordinates --pane-id 4 --x 33% --y 0% --width 67% --height 100% --borderless false --pinned true"
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want percentage fallback %q", got, want)
	}
}

func TestToggleFocusedIgnoresLeftFocus(t *testing.T) {
	rt := &fakeRuntime{panesJSON: []byte(`[
		{"id":1,"is_plugin":false,"is_focused":true,"is_floating":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":4,"is_plugin":false,"is_focused":false,"is_floating":true,"title":"terminal","terminal_command":"pair term"}
	]`)}
	var stderr bytes.Buffer

	if code := RunToggleFocused(nil, rt, &stderr); code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want no-op for left focus", rt.ops)
	}
}

func layeredWorkbenchJSON(terminalX, terminalWidth int) []byte {
	return []byte(`[
		{"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_plugin":false,"is_focused":true,"is_floating":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_plugin":false,"is_focused":false,"is_floating":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"},
		{"id":4,"is_plugin":false,"is_focused":true,"is_floating":true,"pane_x":` + intString(terminalX) + `,"pane_columns":` + intString(terminalWidth) + `,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`)
}

func intString(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var reversed [20]byte
	i := len(reversed)
	for value > 0 {
		i--
		reversed[i] = digits[value%10]
		value /= 10
	}
	return string(reversed[i:])
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
