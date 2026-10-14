package couchtty

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// A retained continuation is information about its thread, never a
// replacement for it, in EVERY phase (#280): the state always shows, and an
// in-flight request is appended. The phases come from the vocabulary, so a
// new one cannot slip past the table. A failed request never reaches a row
// (#522: the record reads it as its line), so it adds nothing here.
func TestContinuationPhaseComposesWithTheStateText(t *testing.T) {
	now := time.Date(2026, 9, 17, 22, 0, 0, 0, time.UTC)
	active := now.Add(-3 * time.Hour)
	labels := map[checkpoint.Phase]string{checkpoint.Pending: "continuation queued", checkpoint.Running: "continuing…", checkpoint.Failed: "", checkpoint.Complete: ""}
	for _, state := range couchcore.AllThreadStates() {
		reason := couchcore.ThreadReason("")
		if state == couchcore.ThreadUnusable {
			reason = couchcore.ReasonBindingLost
		}
		plain := rootStateText(stateTextRow(state, reason, active), now)
		for _, phase := range checkpoint.AllPhases() {
			label, known := labels[phase]
			if !known {
				t.Fatalf("phase %q has no expected label; decide how it renders", phase)
			}
			want := plain
			if label != "" {
				want = plain + " · " + label
			}
			row := stateTextRow(state, reason, active)
			row.Continuation = &couchcore.ContinuationStatus{Address: row.Address, RequestID: "request", Phase: phase}
			if got := rootStateText(row, now); got != want {
				t.Errorf("%s + %s = %q, want %q", state, phase, got, want)
			}
		}
		row := stateTextRow(state, reason, active)
		row.ContinuationMissed = &checkpoint.Miss{RequestID: "request", Path: "/repo/c.md", Reason: "fork failed", At: active}
		if got := rootStateText(row, now); got != plain+" · compaction didn't finish" {
			t.Errorf("%s + a miss = %q", state, got)
		}
	}
}

// A compaction that didn't finish blocks nothing (#522): its row keeps exactly
// the actions it would have without it, and the focused row says the one line
// -- what happened, where the continuation is, and the action for its state.
// An in-flight request still owns a live row until Couch finishes it.
func TestAMissedRowKeepsItsActionsAndShowsItsLine(t *testing.T) {
	address := menuAddress("pair")
	for _, state := range []couchcore.ActionableThreadState{couchcore.ThreadLive, couchcore.ThreadDetached} {
		plain := couchcore.ActionableThreadSummary{Address: address, State: state}
		missed := plain
		missed.ContinuationMissed = &checkpoint.Miss{RequestID: "request", Path: "/repo/workshop/continuation/c.md", Reason: "fork unavailable", At: time.Unix(1, 0)}
		if got, want := menuActionItems(missed), menuActionItems(plain); !slices.Equal(got, want) {
			t.Errorf("%s: a miss changed the actions %v -> %v", state, want, got)
		}
		line := menuFocusSummary(missed)
		want := "Resume the slot"
		if state == couchcore.ThreadLive {
			want = "Compact again"
		}
		if !strings.Contains(line, "compaction didn't finish: fork unavailable; continuation at /repo/workshop/continuation/c.md") || !strings.Contains(line, want) {
			t.Errorf("%s: focus line %q, want the line with %q", state, line, want)
		}
	}
	for _, tc := range []struct {
		state couchcore.ActionableThreadState
		phase checkpoint.Phase
		want  []string
	}{
		{couchcore.ThreadLive, checkpoint.Running, nil},
		{couchcore.ThreadLive, checkpoint.Pending, nil},
		// Not live: resume routes the request to its own executor, reboot
		// retires it with the conversation.
		{couchcore.ThreadUnusable, checkpoint.Running, []string{"recover", "resume", "reboot"}},
	} {
		r := couchcore.ActionableThreadSummary{Address: address, State: tc.state,
			Continuation: &couchcore.ContinuationStatus{Address: address, RequestID: "request", Phase: tc.phase}}
		if tc.state == couchcore.ThreadUnusable {
			r.Reason = couchcore.ReasonSessionGone
		}
		if got := menuActionItems(r); !slices.Equal(got, tc.want) {
			t.Errorf("%s + %s = %v, want %v", tc.state, tc.phase, got, tc.want)
		}
	}
}
