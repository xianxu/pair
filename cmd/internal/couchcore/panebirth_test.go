package couchcore

import (
	"testing"
	"time"
)

func TestPaneMarksBornIn(t *testing.T) {
	old := time.Unix(100, 0)
	fresh := time.Unix(200, 0)
	const ours, twin = "/s/pane-T-claude.json", "/s/pane-T-codex.json"
	for _, c := range []struct {
		name        string
		before, now PaneMarks
		born        bool
	}{
		{"nothing before, nothing now", PaneMarks{}, PaneMarks{}, false},
		{"a sidecar appears", PaneMarks{}, PaneMarks{ours: fresh}, true},
		{"unchanged is not a birth", PaneMarks{ours: old}, PaneMarks{ours: old}, false},
		{"a rewrite moves the mtime", PaneMarks{ours: old}, PaneMarks{ours: fresh}, true},
		{"the launcher's clear is not a birth", PaneMarks{ours: old}, PaneMarks{}, false},
		{"a stale twin sitting still is not a birth", PaneMarks{twin: old}, PaneMarks{twin: old}, false},
		{"cleared ours beside a still twin", PaneMarks{ours: old, twin: old}, PaneMarks{twin: old}, false},
		{"reborn ours beside a still twin", PaneMarks{ours: old, twin: old}, PaneMarks{ours: fresh, twin: old}, true},
		{"a nil baseline treats any sidecar as born", nil, PaneMarks{ours: old}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.before.BornIn(c.now); got != c.born {
				t.Fatalf("BornIn = %v, want %v", got, c.born)
			}
		})
	}
}
