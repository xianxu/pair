package dispatcher

import (
	"os"
	"strings"
	"testing"
)

// Internal entrypoints are invoked by Pair's panes, agents or diagnostics. New
// operator-facing families must be documented instead of silently joining this list.
func TestOperatorFamiliesHaveREADMEUsage(t *testing.T) {
	internal := map[string]string{
		"launch":            "public launcher handoff, not a subcommand",
		"diagnostic append": "trace writer", "retention": "lease protocol",
		"context": "pane meter", "agent restart": "in-pane restart helper",
		"layout toggle-focused": "pane chord", "layout focus-terminal": "pane chord", "layout switch-terminal-tab": "pane chord",
		"slug": "background naming", "term": "workbench terminal", "hoprtt": "latency probe", "scribe": "PTY helper",
		"session-watch": "wrapper-owned observer", "session-log append": "submission protocol", "session-log commit": "submission protocol",
		"title": "background title poller", "continuation": "agent datatype writer",
		"review target": "review protocol", "review open": "pane helper", "review readiness": "review protocol",
		"scrollback render": "viewer renderer", "scrollback open": "viewer helper",
		"changelog render": "background distiller", "changelog open": "viewer helper",
		"clip copy-on-select": "clipboard chord", "clip clipboard-to-pane": "clipboard chord", "clip flash-pane": "pane feedback",
	}
	raw, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	families := map[string]bool{}
	for _, family := range Families() {
		families[family.Name] = true
		if internal[family.Name] != "" {
			continue
		}
		if !strings.Contains(string(raw), "pair "+family.Name) {
			t.Errorf("operator family %q needs README usage or an explicit internal classification", family.Name)
		}
	}
	for name := range internal {
		if !families[name] {
			t.Errorf("stale internal classification %q", name)
		}
	}
}
