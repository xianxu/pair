package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The switcher's rows while the pass owns them (pair#206 plan, Task 9): the
// state column comes from the pass, and a pending row is greyed.
func TestTheSwitcherShowsEachPassStateOnItsRow(t *testing.T) {
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("couch-a", couchcore.ThreadDetached, 1), // will attach
		reattachRow("couch-b", couchcore.ThreadDetached, 2), // will fail
		reattachRow("couch-c", couchcore.ThreadDetached, 3), // will be loading
		reattachRow("couch-d", couchcore.ThreadDetached, 4), // will be queued
		reattachRow("couch-parked", couchcore.ThreadParked, 5),
	}
	state, effects := passMenu(t, inventory)
	resolve := func(event MenuEvent) {
		t.Helper()
		var next []MenuEffect
		state, next = ReduceMenu(state, event)
		effects = next
	}
	resolve(MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true, ProjectionAfterGeneration: 1})
	resolve(MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-b"), Diagnostic: couchcore.ResumePathMissing})
	if state.Reattach.Loading != menuAddress("couch-c") {
		t.Fatalf("setup: loading %v, want couch-c", state.Reattach.Loading)
	}

	lines, extents := renderRootMenuFrame(state, state.Frames[0], 120, 40, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC), true)
	rowOf := map[couchcore.ThreadAddress]string{}
	for _, extent := range extents {
		rowOf[extent.Thread] = lines[extent.Start]
	}
	for tag, want := range map[string]string{
		"couch-a": "live",
		"couch-b": "reattach failed: resume-path-missing",
		"couch-c": "reattaching…",
		"couch-d": "queued",
	} {
		if row := rowOf[menuAddress(tag)]; !strings.Contains(row, want) {
			t.Fatalf("%s row = %q, want it to read %q", tag, row, want)
		}
	}
	for _, tag := range []string{"couch-c", "couch-d"} {
		if row := rowOf[menuAddress(tag)]; !strings.Contains(row, placeholderSGR) {
			t.Fatalf("pending %s row = %q is not greyed", tag, row)
		}
	}
	// A row outside the pass is untouched.
	if row := rowOf[menuAddress("couch-parked")]; !strings.Contains(row, "parked") {
		t.Fatalf("parked row = %q, want today's text", row)
	}
}
