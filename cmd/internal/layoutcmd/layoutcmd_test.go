package layoutcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestToggleFocusedFloatsRightTerminalOverlay(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("terminal", false)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := strings.Join([]string{
		"toggle-pane-embed-or-floating --pane-id 3",
		"change-floating-pane-coordinates --pane-id 3 --x 33% --y 0% --width 67% --height 100% --pinned true",
	}, "\n")
	if got := strings.Join(rt.ops, "\n"); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestToggleFocusedEmbedsFloatingRightTerminal(t *testing.T) {
	rt := &fakeRuntime{panesJSON: panesJSON("terminal", true)}
	var stderr bytes.Buffer

	code := RunToggleFocused(nil, rt, &stderr)

	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	got := strings.Join(rt.ops, "\n")
	if !strings.HasPrefix(got, "toggle-pane-embed-or-floating --pane-id 3\noverride-layout --apply-only-to-active-tab --layout-string ") {
		t.Fatalf("ops = %q, want embed then override-layout", got)
	}
	if strings.Contains(got, `size="67%"`) {
		t.Fatalf("ops = %q, restore layout must not keep 67%% width", got)
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
