package threadactivity

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
)

type activityWorkSnapshot struct{ Read, Decoded map[string]int64 }
type activityWorkCounter struct {
	mu            sync.Mutex
	read, decoded map[string]int64
}

func newActivityWorkCounter() *activityWorkCounter {
	return &activityWorkCounter{read: map[string]int64{}, decoded: map[string]int64{}}
}
func activityWorkFamily(a sessioninventory.Artifact) string {
	if a.StorageRoot != "pair-data" {
		return "native"
	}
	if strings.HasPrefix(a.RelativePath, "ledger-") {
		return "ledger"
	}
	return "catalog"
}
func (c *activityWorkCounter) ReadWork(a sessioninventory.Artifact, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.read[activityWorkFamily(a)] += n
}
func (c *activityWorkCounter) DecodeWork(a sessioninventory.Artifact, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.decoded[activityWorkFamily(a)] += n
}
func (c *activityWorkCounter) snapshot() activityWorkSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := activityWorkSnapshot{Read: map[string]int64{}, Decoded: map[string]int64{}}
	for k, v := range c.read {
		s.Read[k] = v
	}
	for k, v := range c.decoded {
		s.Decoded[k] = v
	}
	return s
}
func (s activityWorkSnapshot) delta(prior activityWorkSnapshot) activityWorkSnapshot {
	d := activityWorkSnapshot{Read: map[string]int64{}, Decoded: map[string]int64{}}
	for k, v := range s.Read {
		d.Read[k] = v - prior.Read[k]
	}
	for k, v := range s.Decoded {
		d.Decoded[k] = v - prior.Decoded[k]
	}
	return d
}
func (s activityWorkSnapshot) requireZero(t *testing.T) {
	t.Helper()
	for family, n := range s.Read {
		if n != 0 {
			t.Errorf("unchanged idle tick read %s bytes=%d", family, n)
		}
	}
	for family, n := range s.Decoded {
		if n != 0 {
			t.Errorf("unchanged idle tick decoded %s bytes=%d", family, n)
		}
	}
}

type idleInvariantFixture struct {
	runtime              OSRuntime
	rows                 []Thread
	ledgers, transcripts []string
	counts               *activityWorkCounter
}

func newIdleInvariantFixture(t testing.TB, slots, ledgerBytes int) idleInvariantFixture {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	data := filepath.Join(root, "pair")
	counts := newActivityWorkCounter()
	f := idleInvariantFixture{runtime: NewOSRuntime(home).WithWorkObserver(counts), counts: counts}
	native := filepath.Join(home, ".codex", "sessions", "2026", "10", "14")
	if e := os.MkdirAll(native, 0700); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < slots; i++ {
		scope := fmt.Sprintf("%016x", i+1)
		dir := filepath.Join(data, "repos", scope)
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		id := fmt.Sprintf("019d1111-1111-7111-8111-%012d", i+1)
		ledger := filepath.Join(dir, "ledger-work.jsonl")
		rows := fmt.Sprintf("{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":\"work\",\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":%q,\"request_origin\":\"chosen-id\",\"baseline_complete\":true}\n{\"v\":3,\"kind\":\"binding\",\"scope_key\":%q,\"tag\":\"work\",\"agent\":\"codex\",\"launch_ordinal\":1,\"root_native_id\":%q,\"confirmation_reason\":\"chosen-id\"}\n", scope, id, scope, id)
		file, e := os.OpenFile(ledger, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if e != nil {
			t.Fatal(e)
		}
		// Stream valid JSON whitespace inside the first record. Logical30 MB files
		// exercise the real parser without retaining20 source buffers in memory.
		padding := ledgerBytes - len(rows)
		chunk := strings.Repeat(" ", 64<<10)
		for padding > 0 {
			n := len(chunk)
			if padding < n {
				n = padding
			}
			if _, e = file.WriteString(chunk[:n]); e != nil {
				t.Fatal(e)
			}
			padding -= n
		}
		if _, e = file.WriteString(rows); e != nil {
			t.Fatal(e)
		}
		if e = file.Close(); e != nil {
			t.Fatal(e)
		}
		transcript := filepath.Join(native, "rollout-test-"+id+".jsonl")
		body := fmt.Sprintf("{\"timestamp\":\"2026-10-14T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"parent_thread_id\":null,\"source\":\"cli\"}}\n", id)
		if e = os.WriteFile(transcript, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		f.rows = append(f.rows, Thread{ScopeDir: dir, Scope: scope, Tag: "work", Agent: "codex"})
		f.ledgers = append(f.ledgers, ledger)
		f.transcripts = append(f.transcripts, transcript)
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
	f := newIdleInvariantFixture(t, 20, 30_000_000)
	f.tick(t)
	cold := f.counts.snapshot()
	if cold.Read["ledger"] < 20*30_000_000 || cold.Decoded["ledger"] < 20*30_000_000 || cold.Read["native"] == 0 || cold.Decoded["native"] == 0 || cold.Decoded["catalog"] == 0 {
		t.Fatalf("cold controls disconnected: %+v", cold)
	}
	for i := 0; i < 3; i++ {
		before := f.counts.snapshot()
		f.tick(t)
		f.counts.snapshot().delta(before).requireZero(t)
	}
	before := f.counts.snapshot()
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
	delta := f.counts.snapshot().delta(before)
	if delta.Read["ledger"] < 30_000_000 || delta.Read["ledger"] >= 60_000_000 || delta.Decoded["ledger"] == 0 {
		t.Fatalf("mutation must parse just changed owner: %+v", delta)
	}
	before = f.counts.snapshot()
	f.tick(t)
	f.counts.snapshot().delta(before).requireZero(t)
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
		t.Logf("cold work=%+v; unchanged20-slot ticks decode/read zero", cold)
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
	fresh := fmt.Sprintf("{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":\"work\",\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":\"019d2222-2222-7222-8222-222222222222\",\"request_origin\":\"chosen-id\",\"baseline_complete\":true}\n", row.Scope)
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
	before := f.counts.snapshot()
	b.ReportAllocs()
	b.ResetTimer()
	pprof.Do(context.Background(), pprof.Labels("phase", "warm"), func(context.Context) {
		for i := 0; i < b.N; i++ {
			f.tick(b)
		}
	})
	b.StopTimer()
	delta := f.counts.snapshot().delta(before)
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
