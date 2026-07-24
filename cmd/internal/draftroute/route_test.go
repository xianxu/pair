package draftroute

import (
	"errors"
	"strings"
	"testing"
)

type fakeRuntime struct {
	panes []byte
	ops   []string
	errAt int
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	if f.errAt == -1 {
		return nil, errors.New("list failed")
	}
	return f.panes, nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	if f.errAt > 0 && len(f.ops) == f.errAt {
		return errors.New("action failed")
	}
	return nil
}

func TestRouteLuaAddressesEveryWriteToDraft(t *testing.T) {
	rt := &fakeRuntime{panes: []byte(`[
		{"id":1,"is_focused":false,"is_plugin":false,"terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_plugin":false,"terminal_command":"nvim -u /pair/nvim/init.lua /data/draft.md"},
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"terminal_command":"pair term"}
	]`)}

	if err := RouteLua(rt, "PairConfirmRestart"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"write --pane-id 2 28",
		"write --pane-id 2 14",
		"write-chars --pane-id 2 :lua PairConfirmRestart()",
		"write --pane-id 2 13",
	}
	if strings.Join(rt.ops, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ops:\n%s\nwant:\n%s", strings.Join(rt.ops, "\n"), strings.Join(want, "\n"))
	}
}

func TestRouteLuaReportsMissingDraft(t *testing.T) {
	rt := &fakeRuntime{panes: []byte(`[{"id":4,"is_focused":true,"terminal_command":"pair term"}]`)}
	err := RouteLua(rt, "PairConfirmRestart")
	if err == nil || !strings.Contains(err.Error(), "draft pane") {
		t.Fatalf("error = %v, want missing draft pane", err)
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want none", rt.ops)
	}
}

func TestRouteLuaStopsAndReturnsActionFailure(t *testing.T) {
	rt := &fakeRuntime{
		panes: []byte(`[{"id":2,"is_focused":false,"is_plugin":false,"terminal_command":"nvim -u /pair/nvim/init.lua /data/draft.md"}]`),
		errAt: 2,
	}
	err := RouteLua(rt, "PairConfirmRestart")
	if err == nil || !strings.Contains(err.Error(), "action failed") {
		t.Fatalf("error = %v, want action failure", err)
	}
	if len(rt.ops) != 2 {
		t.Fatalf("ops = %v, want stop after second action", rt.ops)
	}
}
