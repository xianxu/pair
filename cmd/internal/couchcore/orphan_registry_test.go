package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func leftRunningEntry(pid int, at time.Time) LeftRunningOrphan {
	return LeftRunningOrphan{Origin: "slot:/w/pair", Thread: ThreadAddress{RepoScope: "s", Tag: "t"}, Process: ProcessIdentity{PID: pid, Identity: fmt.Sprintf("id-%d", pid)}, Command: "sdlc", LeftAt: at}
}

func testRegistry(t *testing.T) *LeftRunningRegistry {
	t.Helper()
	ns, err := ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	return NewLeftRunningRegistry(ns)
}

// One entry per identity, the newest cap kept, oldest first.
func TestMergeLeftRunningDedupesAndEvictsOldest(t *testing.T) {
	t0 := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	old := []LeftRunningOrphan{leftRunningEntry(1, t0), leftRunningEntry(2, t0.Add(time.Minute))}
	again := leftRunningEntry(1, t0.Add(time.Hour))
	got := mergeLeftRunning(old, []LeftRunningOrphan{again, leftRunningEntry(3, t0.Add(2*time.Minute))}, 2)
	if len(got) != 2 || got[0].Process.PID != 2 || got[1].Process.PID != 3 {
		t.Fatalf("merge = %+v, want pids 2,3 (1 deduped then evicted as oldest)", got)
	}
}

// The registry is bounded on disk and drops what the table no longer holds.
func TestLeftRunningRegistryAddCapAndPrune(t *testing.T) {
	r := testRegistry(t)
	ctx := context.Background()
	t0 := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	var entries []LeftRunningOrphan
	for pid := 1; pid <= leftRunningCap+5; pid++ {
		entries = append(entries, leftRunningEntry(pid, t0.Add(time.Duration(pid)*time.Second)))
	}
	if err := r.Add(ctx, entries); err != nil {
		t.Fatal(err)
	}
	got, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != leftRunningCap || got[0].Process.PID != 6 {
		t.Fatalf("registry holds %d entries starting at pid %d", len(got), got[0].Process.PID)
	}
	if err := r.Prune(ctx, func(p ProcessIdentity) bool { return p.PID != 6 }); err != nil {
		t.Fatal(err)
	}
	if got, _ = r.Read(); len(got) != 1 || got[0].Process.PID != 6 {
		t.Fatalf("after prune: %+v", got)
	}
}

// Tolerant and preserving from birth: a newer build's members, at the file
// and the entry level, survive this build's rewrite.
func TestLeftRunningRegistryPreservesUnknownMembers(t *testing.T) {
	r := testRegistry(t)
	raw := []byte(`{"schema_version":1,"future":{"a":1},"orphans":[{"origin":"slot:/w/pair","thread":{"repo_scope":"s","tag":"t"},"process":{"pid":7,"identity":"id-7","gen":2},"command":"sdlc","left_at":"2026-10-19T00:00:00Z","later":"kept"}]}`)
	if err := os.MkdirAll(r.dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.Path(), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(context.Background(), []LeftRunningOrphan{leftRunningEntry(8, time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC))}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(r.Path())
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, after); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"future":{"a":1}`, `"later":"kept"`, `"gen":2`, `"pid":8`} {
		if !bytes.Contains(compact.Bytes(), []byte(want)) {
			t.Errorf("rewrite lost %s: %s", want, compact.String())
		}
	}
	if err := os.WriteFile(r.Path(), []byte(`{"schema_version":1,"orphans":[],"orphans":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(); err == nil {
		t.Fatal("a duplicate member was accepted")
	}
}

// Teardown registers what it left running, so it stays attributable.
func TestTeardownRegistersWhatItLeftRunning(t *testing.T) {
	f := newTeardownFixture(t)
	f.liveTree()
	if _, err := f.env.Couch.Teardown(context.Background(), f.thread, "op-1"); err != nil {
		t.Fatal(err)
	}
	got, err := NewLeftRunningRegistry(f.env.Couch.Namespace).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Process.PID != 502 || got[0].Operation != "op-1" || got[0].Thread != f.thread.Address {
		t.Fatalf("registry = %+v, want the setsid'd runner", got)
	}
}

// Decode checks values, not only types: a newer registry version is refused
// and left exactly as it was, and an entry without an exact identity refuses.
func TestLeftRunningRegistryRefusesUnsupportedValues(t *testing.T) {
	for name, raw := range map[string]string{
		"newer version":  `{"schema_version":2,"orphans":[]}`,
		"no version":     `{"orphans":[]}`,
		"pid zero":       `{"schema_version":1,"orphans":[{"origin":"o","thread":{"repo_scope":"s","tag":"t"},"process":{"pid":0,"identity":"x"},"command":"c","left_at":"2026-10-19T00:00:00Z"}]}`,
		"empty identity": `{"schema_version":1,"orphans":[{"origin":"o","thread":{"repo_scope":"s","tag":"t"},"process":{"pid":7,"identity":""},"command":"c","left_at":"2026-10-19T00:00:00Z"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := testRegistry(t)
			if err := os.MkdirAll(r.dir(), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(r.Path(), []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Read(); err == nil {
				t.Fatal("decoded")
			}
			if err := r.Add(context.Background(), []LeftRunningOrphan{leftRunningEntry(8, time.Now())}); err == nil {
				t.Fatal("rewrote a registry it refused to read")
			}
			if after, _ := os.ReadFile(r.Path()); string(after) != raw {
				t.Fatalf("refused registry changed: %s", after)
			}
		})
	}
}
