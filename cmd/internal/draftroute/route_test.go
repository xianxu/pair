package draftroute

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCachedDraftPane(t *testing.T) {
	body, err := json.Marshal(CachedPaneRecord{
		Session: "pair-work",
		PaneID:  "42",
		PID:     ProcessID("9001"),
	})
	if err != nil {
		t.Fatal(err)
	}
	alive := func(pid string) bool { return pid == "9001" }

	if got, ok := ValidateCachedDraftPane(body, "pair-work", alive); !ok || got != "42" {
		t.Fatalf("valid record = %q, %v; want 42, true", got, ok)
	}
	if _, ok := ValidateCachedDraftPane(body, "pair-other", alive); ok {
		t.Fatal("stale session accepted")
	}
	if _, ok := ValidateCachedDraftPane(body, "pair-work", func(string) bool { return false }); ok {
		t.Fatal("dead draft process accepted")
	}
	if _, ok := ValidateCachedDraftPane([]byte("bad json"), "pair-work", alive); ok {
		t.Fatal("invalid record accepted")
	}
}

func TestValidateCachedDraftPaneAcceptsNumericPIDFromNeovim(t *testing.T) {
	body := []byte(`{"session":"pair-work","pane_id":"42","pid":9001}`)
	alive := func(pid string) bool { return pid == "9001" }

	if got, ok := ValidateCachedDraftPane(body, "pair-work", alive); !ok || got != "42" {
		t.Fatalf("numeric-pid record = %q, %v; want 42, true", got, ok)
	}
}

func TestCachedDraftPaneIDFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("ZELLIJ_SESSION_NAME", "pair-work")
	body, err := json.Marshal(CachedPaneRecord{
		Session: "pair-work",
		PaneID:  "42",
		PID:     ProcessID(fmt.Sprint(os.Getpid())),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "draft-pane-work.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok := CachedDraftPaneIDFromEnv(); !ok || got != "42" {
		t.Fatalf("cached pane = %q, %v; want 42, true", got, ok)
	}
}

type fakeRuntime struct {
	panes     []byte
	cached    string
	listCalls int
	ops       []string
	errAt     int
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	f.listCalls++
	if f.errAt == -1 {
		return nil, errors.New("list failed")
	}
	return f.panes, nil
}

func (f *fakeRuntime) CachedDraftPaneID() (string, bool) {
	return f.cached, f.cached != ""
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	if f.errAt > 0 && len(f.ops) == f.errAt {
		return errors.New("action failed")
	}
	return nil
}

func TestRouteLuaFocusesBeforeAddressingEveryWriteToDraft(t *testing.T) {
	rt := &fakeRuntime{panes: []byte(`[
		{"id":1,"is_focused":false,"is_plugin":false,"terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_plugin":false,"terminal_command":"nvim -u /pair/nvim/init.lua /data/draft.md"},
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"terminal_command":"pair term"}
	]`)}

	if err := RouteLua(rt, "PairConfirmRestart", true); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"focus-pane-id 2",
		"write --pane-id 2 28",
		"write --pane-id 2 14",
		"write-chars --pane-id 2 :lua PairConfirmRestart()",
		"write --pane-id 2 13",
	}
	if strings.Join(rt.ops, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ops:\n%s\nwant:\n%s", strings.Join(rt.ops, "\n"), strings.Join(want, "\n"))
	}
}

func TestRouteLuaUsesValidatedCachedDraftWithoutListingPanes(t *testing.T) {
	rt := &fakeRuntime{cached: "42"}

	if err := RouteLua(rt, "PairConfirmRestart", false); err != nil {
		t.Fatal(err)
	}
	if rt.listCalls != 0 {
		t.Fatalf("list calls = %d, want 0 on cached fast path", rt.listCalls)
	}
	if got := rt.ops[0]; got != "write --pane-id 42 28" {
		t.Fatalf("first op = %q, want cached pane 42", got)
	}
}

func TestRouteLuaFocusFailurePerformsNoWrites(t *testing.T) {
	rt := &fakeRuntime{cached: "42", errAt: 1}
	err := RouteLua(rt, "PairConfirmRestart", true)
	if err == nil || !strings.Contains(err.Error(), "focus") {
		t.Fatalf("error = %v, want focus failure", err)
	}
	if got := strings.Join(rt.ops, "\n"); got != "focus-pane-id 42" {
		t.Fatalf("ops = %q, want focus only", got)
	}
}

func TestRouteLuaNonConfirmationDoesNotFocus(t *testing.T) {
	rt := &fakeRuntime{cached: "42"}
	if err := RouteLua(rt, "PairLayoutBigger", false); err != nil {
		t.Fatal(err)
	}
	if got := rt.ops[0]; got != "write --pane-id 42 28" {
		t.Fatalf("first op = %q, want write without focus", got)
	}
}

func TestRouteLuaReportsMissingDraft(t *testing.T) {
	rt := &fakeRuntime{panes: []byte(`[{"id":4,"is_focused":true,"terminal_command":"pair term"}]`)}
	err := RouteLua(rt, "PairConfirmRestart", true)
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
	err := RouteLua(rt, "PairConfirmRestart", false)
	if err == nil || !strings.Contains(err.Error(), "action failed") {
		t.Fatalf("error = %v, want action failure", err)
	}
	if len(rt.ops) != 2 {
		t.Fatalf("ops = %v, want stop after second action", rt.ops)
	}
}
