package zellijpane

import "testing"

// realShape mirrors an actual `zellij action list-panes --json --command` body:
// a tab-position-keyed map whose values are arrays of pane manifests. The pane
// objects are what Parse must surface; the "0"/"1" tab keys and any wrapper are
// not panes.
const realShape = `{
  "0": [
    {"id":0,"is_plugin":false,"is_focused":true,"is_floating":false,
     "title":"claude [~/workspace/parley.nvim]",
     "terminal_command":"sh -c exec pair-wrap --scrollback-log /d/s.raw claude"},
    {"id":1,"is_plugin":false,"is_focused":false,"is_floating":false,
     "title":"draft",
     "terminal_command":"sh -c exec nvim -u /h/nvim/init.lua /d/draft-t.md"}
  ]
}`

func TestParseFlattensTabKeyedPanes(t *testing.T) {
	panes := Parse([]byte(realShape))
	if len(panes) != 2 {
		t.Fatalf("want 2 panes, got %d: %+v", len(panes), panes)
	}
	byID := map[string]Pane{}
	for _, p := range panes {
		byID[p.ID] = p
	}
	agent, ok := byID["0"]
	if !ok {
		t.Fatalf("missing agent pane id 0: %+v", panes)
	}
	if !agent.IsFocused || agent.IsPlugin || agent.IsFloating {
		t.Errorf("agent pane flags wrong: %+v", agent)
	}
	if agent.Title != "claude [~/workspace/parley.nvim]" {
		t.Errorf("agent title = %q", agent.Title)
	}
	draft, ok := byID["1"]
	if !ok {
		t.Fatalf("missing draft pane id 1")
	}
	if draft.IsFocused {
		t.Errorf("draft should not be focused: %+v", draft)
	}
	if draft.Title != "draft" {
		t.Errorf("draft title = %q", draft.Title)
	}
}

func TestParseSkipsNonPaneWrappers(t *testing.T) {
	// A layout wrapper object carries an id but none of the pane markers; it
	// must not be surfaced as a pane.
	js := `{"tab": {"id":9,"name":"main"}, "panes":[{"id":3,"is_focused":true}]}`
	panes := Parse([]byte(js))
	if len(panes) != 1 || panes[0].ID != "3" {
		t.Fatalf("want only pane id 3, got %+v", panes)
	}
}

func TestParseStringPaneID(t *testing.T) {
	js := `[{"pane_id":"terminal_7","is_focused":true,"terminal_command":"exec nvim x"}]`
	panes := Parse([]byte(js))
	if len(panes) != 1 || panes[0].ID != "terminal_7" {
		t.Fatalf("want pane id terminal_7, got %+v", panes)
	}
	if panes[0].TerminalCommand != "exec nvim x" {
		t.Errorf("terminal_command = %q", panes[0].TerminalCommand)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	if got := Parse([]byte("not json")); got != nil {
		t.Fatalf("invalid JSON → nil, got %+v", got)
	}
	if got := Parse(nil); got != nil {
		t.Fatalf("nil → nil, got %+v", got)
	}
}
