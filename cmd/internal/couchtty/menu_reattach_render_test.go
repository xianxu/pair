package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
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
	if state.Reattach.soleLoading() != menuAddress("couch-c") {
		t.Fatalf("setup: loading %v, want couch-c", state.Reattach.soleLoading())
	}

	lines, extents := renderRootMenuFrame(state, state.Frames[0], 120, 40, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC), true)
	rowOf := map[couchcore.ThreadAddress]string{}
	for _, extent := range extents {
		rowOf[extent.Thread] = lines[extent.Start]
	}
	for tag, want := range map[string]string{
		"couch-a": "Continue",
		"couch-b": "Continue · pane failed: resume-path-missing (retry)",
		"couch-c": "Continue · attaching…",
		"couch-d": "Continue · queued",
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
	if row := rowOf[menuAddress("couch-parked")]; !strings.Contains(row, "Resume from disk") {
		t.Fatalf("parked row = %q, want today's text", row)
	}
}

// A failed row can carry an error's own text (decision 11), which is neither
// short nor trustworthy. Its state column is sanitized, and it never takes the
// label's columns, at any width the switcher supports.
func TestAPassSuffixIsSanitizedAndLeavesTheLabelItsColumns(t *testing.T) {
	failed := PassView{State: PassFailed, Diagnostic: "\x1b]0;hijacked\x07spawn failed: " + strings.Repeat("a very long reason ", 20)}
	for _, width := range []int{120, 60, 40} {
		suffix := passSuffix(failed, true, width, "Continue")
		if strings.ContainsAny(suffix, "\x1b\x07") {
			t.Fatalf("width %d: suffix %q carries the error's control bytes", width, suffix)
		}
		if got, limit := textwidth.Width(suffix), width-menuLabelFloor; got > limit {
			t.Fatalf("width %d: suffix is %d columns, leaving the label fewer than %d: %q", width, got, menuLabelFloor, suffix)
		}
		if !strings.HasPrefix(suffix, "  Continue · pane ") {
			t.Fatalf("width %d: suffix %q no longer says the reattach failed", width, suffix)
		}
	}
	if got := passSuffix(PassView{State: PassQueued}, true, 120, "Continue"); got != "  Continue · queued" {
		t.Fatalf("queued suffix = %q", got)
	}
	if got := passSuffix(PassView{}, false, 120, "Continue"); got != "" {
		t.Fatalf("a row the pass does not own got suffix %q", got)
	}
}

func TestContinuePaneAttachmentRenderedSnapshots(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	row := reattachRow("worker", couchcore.ThreadDetached, 5)
	for _, test := range []struct {
		name string
		pass ReattachPass
		want string
	}{
		{name: "loading", pass: ReattachPass{InFlight: map[uint64]couchcore.ThreadAddress{1: row.Address}}, want: "  worker  (path unavailable)  Continue · attaching…"},
		{name: "attached", pass: ReattachPass{Attached: map[couchcore.ThreadAddress]uint64{row.Address: 1}}, want: "  worker  (path unavailable)  Continue"},
		{name: "failed", pass: ReattachPass{Failed: map[couchcore.ThreadAddress]string{row.Address: "temporary helper failure"}}, want: "  worker  (path unavailable)  Continue · pane failed: temporary helper failure (retry)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
			state.Reattach = test.pass
			frame := state.Frames[0]
			frame.SelectedAddress = couchcore.ThreadAddress{}
			frame.SelectedKey = couchcore.ThreadRowKey{}
			lines, extents := renderRootMenuFrame(state, frame, 180, 20, now, false)
			if len(extents) != 1 {
				t.Fatalf("row extents=%+v lines=%q", extents, lines)
			}
			if got := lines[extents[0].Start]; got != test.want {
				t.Fatalf("rendered row snapshot:\n got: %q\nwant: %q", got, test.want)
			}
		})
	}
}
