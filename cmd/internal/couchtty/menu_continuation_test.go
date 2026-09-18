package couchtty

import (
	"slices"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// A retained continuation is information about its thread, not a replacement
// for it (#280). State × phase, exhaustively: a FAILED request composes with
// whatever the state says; pending and running displace it on purpose (in
// flight, bounded); none and complete leave the plain state.
func TestContinuationPhaseComposesOrDisplacesTheStateText(t *testing.T) {
	now := time.Date(2026, 9, 17, 22, 0, 0, 0, time.UTC)
	active := now.Add(-3 * time.Hour)
	for _, state := range couchcore.AllThreadStates() {
		reason := couchcore.ThreadReason("")
		if state == couchcore.ThreadUnusable {
			reason = couchcore.ReasonBindingLost
		}
		plain := rootStateText(stateTextRow(state, reason, active), now)
		for _, tc := range []struct {
			phase checkpoint.Phase
			want  string
		}{
			{"", plain},
			{checkpoint.Complete, plain},
			{checkpoint.Pending, "continuation queued"},
			{checkpoint.Running, "continuing…"},
			{checkpoint.Failed, plain + " · continuation failed"},
		} {
			row := stateTextRow(state, reason, active)
			if tc.phase != "" {
				row.Continuation = &couchcore.ContinuationStatus{Address: row.Address, RequestID: "request", Phase: tc.phase}
			}
			if got := rootStateText(row, now); got != tc.want {
				t.Errorf("%s + %q = %q, want %q", state, tc.phase, got, tc.want)
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
