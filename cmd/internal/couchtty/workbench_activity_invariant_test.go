package couchtty

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/threadactivity"
	"github.com/xianxu/pair/cmd/internal/workbenchperftest"
)

// Each scoped observation decorates the same runtime/cache owner. Attribution
// changes, but neither activity nor launch cache is rebuilt between rows/ticks.
type countedActivityRuntime struct {
	base   threadactivity.OSRuntime
	counts *workbenchperftest.Counter
	logf   func(string, ...any)
}

func (r countedActivityRuntime) ModTime(path string) (time.Time, bool) { return r.base.ModTime(path) }
func (r countedActivityRuntime) SessionActivity(ctx context.Context, thread threadactivity.Thread) (time.Time, bool) {
	started := time.Now()
	at, ok := r.base.WithWorkObserver(r.counts.Observer(thread.Scope)).SessionActivity(ctx, thread)
	if r.logf != nil {
		r.logf("activity probe scope=%s elapsed=%s available=%t context=%v", thread.Scope, time.Since(started), ok, ctx.Err())
	}
	return at, ok
}

func TestWorkbenchPerformanceIdleUITick(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("host state: %s", testwait.HostState())
		}
	})
	const slots, ledgerBytes = 20, 30_000_000
	evidence := workbenchperftest.NewFixture(t, slots, ledgerBytes)
	runtime := countedActivityRuntime{base: threadactivity.NewOSRuntime(evidence.Home), counts: evidence.Counts, logf: t.Logf}
	console := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	// This test owns byte/decode counts, not the product latency ceiling.
	// Dedicated deadline tests retain the unchanged production bound.
	console.probeTimeout = testwait.Budget(t)
	t.Cleanup(func() { console.Stop(); console.workers.Wait() })
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
	console.SetActivityProbe(ThreadActivityProbe(evidence.DataDir, runtime))
	tick := func() {
		t.Helper()
		select {
		case <-console.activityRequests:
		default:
			t.Fatal("no activity request")
		}
		// Sole scheduler owner, exactly the same event used by Console.Run's tick.
		started := time.Now()
		console.advanceActivity(RefreshScheduleEvent{Kind: RefreshRequested})
		select {
		case result := <-console.activityResults:
			if len(result.observed) != slots || len(result.failed) != 0 {
				t.Fatalf("pass results observed=%d failed=%v", len(result.observed), result.failed)
			}
			t.Logf("activity generation=%d completed=%d elapsed=%s", result.generation, len(result.observed), time.Since(started))
			console.finishActivity(result)
		case <-testwait.After(t):
			t.Fatal("activity worker did not finish")
		}
		if console.activitySchedule.Running != 0 {
			t.Fatal("activity generation did not complete")
		}
		activity := console.menuSnapshot().Activity
		if len(activity) != slots {
			t.Fatalf("UI reducer has %d activity rows, want %d", len(activity), slots)
		}
		for _, row := range rows {
			if activity[row.Address].IsZero() {
				t.Fatalf("missing UI activity for %+v", row.Address)
			}
		}
	}
	tick()
	cold := evidence.Counts.Snapshot()
	for _, family := range []string{"ledger", "native", "catalog"} {
		if cold.ReadFamily(family) == 0 || cold.DecodeFamily(family) == 0 {
			t.Fatalf("disconnected cold %s accounting: %+v", family, cold)
		}
	}
	if cold.ReadFamily("ledger") < slots*ledgerBytes || cold.DecodeFamily("ledger") < slots*ledgerBytes {
		t.Fatalf("cold ledger scale missing: %+v", cold)
	}
	for range 3 {
		before := evidence.Counts.Snapshot()
		console.requestActivity()
		tick()
		evidence.Counts.Snapshot().Delta(before).RequireZero(t)
	}
	changed := evidence.Owners[7]
	file, err := os.OpenFile(changed.Ledger, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString(" \n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	before := evidence.Counts.Snapshot()
	console.requestActivity()
	tick()
	mutation := evidence.Counts.Snapshot().Delta(before)
	if mutation.ReadFamily("ledger") < ledgerBytes || mutation.DecodeFamily("ledger") == 0 {
		t.Fatalf("mutation accounting disconnected: %+v", mutation)
	}
	for key, n := range mutation.Read {
		if n != 0 && key.Owner != changed.Scope {
			t.Fatalf("mutation read unchanged owner %s: %d bytes", key.Owner, n)
		}
	}
	for key, n := range mutation.Decoded {
		if n != 0 && key.Owner != changed.Scope {
			t.Fatalf("mutation decoded unchanged owner %s: %d bytes", key.Owner, n)
		}
	}
	before = evidence.Counts.Snapshot()
	console.requestActivity()
	tick()
	evidence.Counts.Snapshot().Delta(before).RequireZero(t)
	t.Logf("20 x30MB; cold read=%d decoded=%d; unchanged completed UI ticks read=0 decoded=0", cold.ReadBytes(), cold.DecodeBytes())
}
