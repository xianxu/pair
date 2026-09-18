package couchtty

import (
	"slices"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// A retained continuation is information about its thread, never a
// replacement for it, in EVERY phase (#280): the state always shows, and a
// non-complete request is appended. The phases come from the vocabulary, so a
// new one cannot slip past the table.
func TestContinuationPhaseComposesWithTheStateText(t *testing.T) {
	now := time.Date(2026, 9, 17, 22, 0, 0, 0, time.UTC)
	active := now.Add(-3 * time.Hour)
	labels := map[checkpoint.Phase]string{checkpoint.Pending: "continuation queued", checkpoint.Running: "continuing…", checkpoint.Failed: "continuation failed", checkpoint.Complete: ""}
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
	}
}

// A failed request composed with a live row keeps the row's own actions minus
// exactly what the guard refuses -- couchcore's list, not a restatement -- and
// adds both exits. Other phases and non-live rows keep their shapes.
func TestFailedContinuationComposesWithALiveRowsActions(t *testing.T) {
	address := menuAddress("pair")
	row := func(state couchcore.ActionableThreadState, phase checkpoint.Phase) couchcore.ActionableThreadSummary {
		return couchcore.ActionableThreadSummary{Address: address, State: state,
			Continuation: &couchcore.ContinuationStatus{Address: address, RequestID: "request", Phase: phase}}
	}
	live := menuActionItems(row(couchcore.ThreadLive, checkpoint.Failed))
	if want := []string{"detach", "retry-continuation", "dismiss-continuation", "park", "name", "describe"}; !slices.Equal(live, want) {
		t.Fatalf("live + failed = %v, want %v", live, want)
	}
	for _, op := range live {
		if couchcore.ContinuationRefuses(op) {
			t.Fatalf("offered %q, which the continuation guard refuses", op)
		}
	}
	for _, tc := range []struct {
		state couchcore.ActionableThreadState
		phase checkpoint.Phase
		want  []string
	}{
		{couchcore.ThreadBusy, checkpoint.Failed, []string{"retry-continuation", "dismiss-continuation", "name", "describe"}},
		{couchcore.ThreadLive, checkpoint.Running, []string{"retry-continuation", "name", "describe"}},
		{couchcore.ThreadLive, checkpoint.Pending, []string{"name", "describe"}},
	} {
		if got := menuActionItems(row(tc.state, tc.phase)); !slices.Equal(got, tc.want) {
			t.Errorf("%s + %s = %v, want %v", tc.state, tc.phase, got, tc.want)
		}
	}
	recovery := row(couchcore.ThreadDetached, checkpoint.Failed)
	recovery.Recovery = &couchcore.RecoveryDecision{Recover: true}
	got := menuActionItems(recovery)
	if i := slices.Index(got, "retry-continuation"); i < 0 || i+1 >= len(got) || got[i+1] != "dismiss-continuation" {
		t.Fatalf("recovery row with a failed request must offer dismiss after retry: %v", got)
	}
}

// Dismiss from the switcher crosses the PRODUCTION dispatcher and deletes the
// request from the real store -- addressed by the exact tag alone, the dialect
// retry got wrong (#280).
func TestDismissFromTheSwitcherDeletesTheRequestThroughTheRealDispatcher(t *testing.T) {
	c, address, requestID := failedContinuationCouch(t)
	row := couchcore.ActionableThreadSummary{Address: address, State: couchcore.ThreadLive,
		Continuation: &couchcore.ContinuationStatus{Address: address, RequestID: requestID, Phase: checkpoint.Failed}}
	if !slices.Contains(menuActionItems(row), "dismiss-continuation") {
		t.Fatalf("a failed live row must offer dismiss: %v", menuActionItems(row))
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, address)
	_, effects := dispatchThreadOperation(state, "dismiss-continuation", address)
	if len(effects) != 1 || effects[0].Args["request-id"] != requestID || effects[0].Args["ref"] != "" {
		t.Fatalf("dismiss effect must carry the exact tag and request, never a ref: %v", effects)
	}
	result, err := couchcore.DispatchOperation(couchcore.OperationExecutors{DirectStore: couchcore.DirectStoreExecutor(c)},
		couchcore.OperationCall{Name: effects[0].Operation, Args: effects[0].Args, Implicit: true})
	if err != nil {
		t.Fatal(err)
	}
	if record, ok := result.(couchcore.ThreadRecord); !ok || record.Continuation != nil {
		t.Fatalf("dismiss result = %#v", result)
	}
	stored, err := c.Threads.GetThread(address)
	if err != nil || stored.Continuation != nil {
		t.Fatalf("stored continuation after dismiss = %+v, %v", stored.Continuation, err)
	}
}
