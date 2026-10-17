package couchtty

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/threadactivity"
	"github.com/xianxu/pair/cmd/internal/workbenchperftest"
)

// Issue #628: the historical partial-results failure was four probes exceeding
// their per-pass deadline, reported as failed rows while the rest observed.
// The production deadline stays untouched; this test shrinks it through the
// Console seam and pins what a deadline overrun owes the operator: the row
// surfaces as failed (never silently dropped), a cold row stays without
// activity, an established row keeps its previous evidence, and the next
// healthy pass recovers the row.
type deadlineProbeRuntime struct {
	base  threadactivity.OSRuntime
	slow  string
	delay time.Duration
}

func (r *deadlineProbeRuntime) ModTime(path string) (time.Time, bool) { return r.base.ModTime(path) }

func (r *deadlineProbeRuntime) SessionActivity(ctx context.Context, thread threadactivity.Thread) (time.Time, bool) {
	if thread.Scope == r.slow {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return time.Time{}, false
		}
	}
	return r.base.SessionActivity(ctx, thread)
}

func TestActivityProbeDeadlineSurfacesFailureAndRecovers(t *testing.T) {
	const slots = 2
	evidence := workbenchperftest.NewFixture(t, slots, 4096)
	rt := &deadlineProbeRuntime{base: threadactivity.NewOSRuntime(evidence.Home), slow: evidence.Owners[0].Scope, delay: 400 * time.Millisecond}
	console := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(func() { console.Stop(); console.workers.Wait() })
	// A first query costs ~40ms (native-root bootstrap) before any ledger
	// bytes, so the injected bound must sit above that floor; the stall is 4x.
	console.probeTimeout = 100 * time.Millisecond
	rows := make([]couchcore.ActionableThreadSummary, slots)
	for i, owner := range evidence.Owners {
		address := couchcore.ThreadAddress{RepoScope: owner.Scope, Tag: couchcore.ThreadTag(owner.Tag)}
		target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: address}
		key, err := target.RowKey()
		if err != nil {
			t.Fatal(err)
		}
		rows[i] = couchcore.ActionableThreadSummary{Address: address, Target: target, RowKey: key, Agent: owner.Agent, State: couchcore.ThreadLive}
	}
	console.menu = NewMenuState(rows, rows[0].Address)
	console.SetActivityProbe(ThreadActivityProbe(evidence.DataDir, rt))
	tick := func(failedWant map[couchcore.ThreadAddress]bool) map[couchcore.ThreadAddress]time.Time {
		t.Helper()
		select {
		case <-console.activityRequests:
		default:
			t.Fatal("no activity request")
		}
		console.advanceActivity(RefreshScheduleEvent{Kind: RefreshRequested})
		var result activityResult
		select {
		case result = <-console.activityResults:
		case <-time.After(5 * time.Second):
			t.Fatal("activity worker did not finish")
		}
		if len(result.failed) != len(failedWant) {
			t.Fatalf("failed rows %v, want %v", result.failed, failedWant)
		}
		for address := range failedWant {
			if !result.failed[address] {
				t.Fatalf("failed rows %v, want %v", result.failed, failedWant)
			}
		}
		console.finishActivity(result)
		if console.activitySchedule.Running != 0 {
			t.Fatal("activity generation did not complete")
		}
		return result.observed
	}
	deadlined := rows[0].Address
	healthy := rows[1].Address

	// A cold deadline overrun is visible: the row reports failed and the UI
	// carries no activity for it — the retained #628 assertion's own shape.
	observed := tick(map[couchcore.ThreadAddress]bool{deadlined: true})
	if len(observed) != 1 || observed[healthy].IsZero() {
		t.Fatalf("observed %v, want only the healthy row", observed)
	}
	cold := console.menuSnapshot().Activity
	if !cold[deadlined].IsZero() || cold[healthy].IsZero() {
		t.Fatalf("cold UI activity %v", cold)
	}

	// The next healthy pass recovers the row.
	rt.slow = ""
	console.requestActivity()
	observed = tick(nil)
	if len(observed) != slots {
		t.Fatalf("observed %v, want every row", observed)
	}
	recovered := console.menuSnapshot().Activity
	if recovered[deadlined].IsZero() || recovered[healthy].IsZero() {
		t.Fatalf("recovered UI activity %v", recovered)
	}

	// Once established, a deadline overrun keeps the previous evidence instead
	// of dropping the row (pair#247's fade must not flicker to unknown).
	rt.slow = evidence.Owners[0].Scope
	kept := recovered[deadlined]
	console.requestActivity()
	observed = tick(map[couchcore.ThreadAddress]bool{deadlined: true})
	if len(observed) != 1 || observed[healthy].IsZero() {
		t.Fatalf("observed %v, want only the healthy row", observed)
	}
	if after := console.menuSnapshot().Activity[deadlined]; !after.Equal(kept) {
		t.Fatalf("established row lost its previous evidence: %v then %v", kept, after)
	}
}
