package threadactivity

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchperftest"
)

type idleInvariantFixture struct {
	runtime              OSRuntime
	rows                 []Thread
	ledgers, transcripts []string
	counts               *workbenchperftest.Counter
}

func newIdleInvariantFixture(t testing.TB, slots, ledgerBytes int) idleInvariantFixture {
	t.Helper()
	fixture := workbenchperftest.NewFixture(t, slots, ledgerBytes)
	f := idleInvariantFixture{runtime: NewOSRuntime(fixture.Home).WithWorkObserver(fixture.Counts.Observer("")), counts: fixture.Counts}
	for _, owner := range fixture.Owners {
		f.rows = append(f.rows, Thread{ScopeDir: owner.ScopeDir, Scope: owner.Scope, Tag: owner.Tag, Agent: owner.Agent})
		f.ledgers = append(f.ledgers, owner.Ledger)
		f.transcripts = append(f.transcripts, owner.Transcript)
	}
	return f
}
func (f idleInvariantFixture) tick(t testing.TB) {
	t.Helper()
	for _, row := range f.rows {
		at, ok := f.runtime.SessionActivity(context.Background(), row)
		if !ok || at.IsZero() {
			t.Fatalf("activity unavailable scope=%s at=%v", row.Scope, at)
		}
	}
}
func TestIdleActivityTickUnchangedFilesDecodeZeroBytes(t *testing.T) {
	// The Console consumer owns the 20 x 30MB scale; retain invalidation here.
	const slots, ledgerBytes = 3, 4096
	f := newIdleInvariantFixture(t, slots, ledgerBytes)
	f.tick(t)
	cold := f.counts.Snapshot()
	if cold.ReadFamily("ledger") < slots*ledgerBytes || cold.DecodeFamily("ledger") < slots*ledgerBytes || cold.ReadFamily("native") == 0 || cold.DecodeFamily("native") == 0 || cold.DecodeFamily("catalog") == 0 {
		t.Fatalf("cold controls disconnected: %+v", cold)
	}
	for i := 0; i < 3; i++ {
		before := f.counts.Snapshot()
		f.tick(t)
		f.counts.Snapshot().Delta(before).RequireZero(t)
	}
	before := f.counts.Snapshot()
	file, e := os.OpenFile(f.ledgers[0], os.O_APPEND|os.O_WRONLY, 0)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = file.WriteString(" \n"); e != nil {
		t.Fatal(e)
	}
	if e = file.Close(); e != nil {
		t.Fatal(e)
	}
	f.tick(t)
	delta := f.counts.Snapshot().Delta(before)
	if delta.ReadFamily("ledger") < ledgerBytes || delta.ReadFamily("ledger") >= 2*ledgerBytes || delta.DecodeFamily("ledger") == 0 {
		t.Fatalf("mutation must parse just changed owner: %+v", delta)
	}
	before = f.counts.Snapshot()
	f.tick(t)
	f.counts.Snapshot().Delta(before).RequireZero(t)
	// Same-size replacement with restored mtime must not preserve success.
	info, e := os.Stat(f.transcripts[0])
	if e != nil {
		t.Fatal(e)
	}
	bad := strings.Repeat("x", int(info.Size()))
	replacement := f.transcripts[0] + ".new"
	if e = os.WriteFile(replacement, []byte(bad), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Chtimes(replacement, info.ModTime(), info.ModTime()); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(replacement, f.transcripts[0]); e != nil {
		t.Fatal(e)
	}
	if at, ok := f.runtime.SessionActivity(context.Background(), f.rows[0]); ok {
		t.Fatalf("replaced transcript retained cached activity %v", at)
	}
	if !t.Failed() {
		t.Logf("cold work=%+v; unchanged %d-slot ticks with %d-byte ledgers decode/read zero", cold, slots, ledgerBytes)
	}
}

func TestIdleActivityTickWarmAllocationIndependentOfLedgerHistory(t *testing.T) {
	var counts [2]float64
	for i, size := range []int{1024, 30_000_000} {
		f := newIdleInvariantFixture(t, 1, size)
		f.tick(t)
		counts[i] = testing.AllocsPerRun(5, func() {
			at, ok := f.runtime.SessionActivity(context.Background(), f.rows[0])
			if !ok || at.IsZero() {
				panic("lost activity")
			}
		})
	}
	if counts[1] > counts[0]+8 {
		t.Fatalf("warm allocations scale with history small=%v large=%v", counts[0], counts[1])
	}
	t.Logf("warm allocations small=%v large=%v", counts[0], counts[1])
}

func TestIdleActivityUsesCurrentLaunchNotResumeFallback(t *testing.T) {
	f := newIdleInvariantFixture(t, 1, 1024)
	f.tick(t)
	row := f.rows[0]
	fresh := fmt.Sprintf("{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":%q,\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":\"019d2222-2222-7222-8222-222222222222\",\"request_origin\":\"chosen-id\",\"baseline_complete\":true}\n", row.Scope, row.Tag)
	file, e := os.OpenFile(f.ledgers[0], os.O_APPEND|os.O_WRONLY, 0)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = file.WriteString(fresh); e != nil {
		t.Fatal(e)
	}
	if e = file.Close(); e != nil {
		t.Fatal(e)
	}
	if at, ok := f.runtime.SessionActivity(context.Background(), row); ok {
		t.Fatalf("activity inherited older launch %v", at)
	}
}

func BenchmarkIdleActivityTickWarm(b *testing.B) {
	f := newIdleInvariantFixture(b, 20, 30_000_000)
	f.tick(b)
	before := f.counts.Snapshot()
	b.ReportAllocs()
	b.ResetTimer()
	pprof.Do(context.Background(), pprof.Labels("phase", "warm"), func(context.Context) {
		for i := 0; i < b.N; i++ {
			f.tick(b)
		}
	})
	b.StopTimer()
	delta := f.counts.Snapshot().Delta(before)
	var read, decoded int64
	for _, n := range delta.Read {
		read += n
	}
	for _, n := range delta.Decoded {
		decoded += n
	}
	b.ReportMetric(float64(read)/float64(b.N), "body-B/op")
	b.ReportMetric(float64(decoded)/float64(b.N), "decoded-B/op")
}
