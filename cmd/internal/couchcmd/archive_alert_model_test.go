package couchcmd

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

func TestArchiveWarningGateExactKey(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	base := archiveWarningKey{Source: historyarchive.SourceIdentity{Host: "host", Root: "root", Repository: "repo", Worktree: "tree"}, Kind: "codex", Path: "session"}
	keys := []archiveWarningKey{base}
	for _, change := range []func(*archiveWarningKey){
		func(k *archiveWarningKey) { k.Source.Host += "2" },
		func(k *archiveWarningKey) { k.Source.Root += "2" },
		func(k *archiveWarningKey) { k.Source.Repository += "2" },
		func(k *archiveWarningKey) { k.Source.Worktree += "2" },
		func(k *archiveWarningKey) { k.Kind += "2" },
		func(k *archiveWarningKey) { k.Path += "2" },
	} {
		k := base
		change(&k)
		keys = append(keys, k)
	}
	// Delimiter-like strings must retain their field boundaries.
	keys = append(keys, archiveWarningKey{Kind: "a:b", Path: "c"}, archiveWarningKey{Kind: "a", Path: "b:c"}, archiveWarningKey{})
	var gate archiveWarningGate
	for i, k := range keys {
		if admit, overflow := gate.Reserve(now, k); !admit || overflow {
			t.Fatalf("key %d: got (%v,%v), want (true,false)", i, admit, overflow)
		}
		if admit, overflow := gate.Reserve(now, k); admit || overflow {
			t.Fatalf("repeat %d: got (%v,%v)", i, admit, overflow)
		}
	}
}

func TestArchiveWarningGateUTCDayAndRollback(t *testing.T) {
	var gate archiveWarningGate
	key := archiveWarningKey{Path: "session"}
	checks := []struct {
		at    string
		key   archiveWarningKey
		admit bool
	}{
		{"2026-10-19T23:59:59Z", key, true},
		{"2026-10-20T08:59:59+09:00", key, false}, // Same instant and UTC day.
		{"2026-10-19T17:00:00-07:00", key, true},  // UTC midnight, local date unchanged.
		{"2026-10-19T23:59:59Z", archiveWarningKey{Path: "new"}, false},
		{"2026-10-20T01:00:00Z", key, false},
		{"2026-10-20T01:00:00Z", archiveWarningKey{Path: "new"}, true},
		{"2026-11-01T00:00:00Z", key, true},
		{"2026-10-21T00:00:00Z", archiveWarningKey{Path: "other"}, false},
		{"2026-11-01T01:00:00Z", archiveWarningKey{Path: "other"}, true},
		{"2027-01-01T00:00:00Z", key, true},
	}
	for i, c := range checks {
		now, err := time.Parse(time.RFC3339, c.at)
		if err != nil {
			t.Fatal(err)
		}
		if admit, overflow := gate.Reserve(now, c.key); admit != c.admit || overflow {
			t.Fatalf("step %d (%s): got (%v,%v), want (%v,false)", i, c.at, admit, overflow, c.admit)
		}
	}
}

func TestArchiveWarningGateRestart(t *testing.T) {
	now := time.Time{}
	key := archiveWarningKey{}
	var old, restarted archiveWarningGate
	for i, g := range []*archiveWarningGate{&old, &restarted} {
		if admit, overflow := g.Reserve(now, key); !admit || overflow {
			t.Fatalf("instance %d initial reservation: (%v,%v)", i, admit, overflow)
		}
		// The reservation itself consumes admission; no output acknowledgment exists.
		if admit, overflow := g.Reserve(now, key); admit || overflow {
			t.Fatalf("instance %d repeated reservation: (%v,%v)", i, admit, overflow)
		}
	}
}

func TestArchiveWarningGateCapAndDayReset(t *testing.T) {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	var gate archiveWarningGate
	for i := 0; i < historyarchive.MaxFiles; i++ {
		if admit, overflow := gate.Reserve(now, archiveWarningKey{Path: fmt.Sprint(i)}); !admit || overflow {
			t.Fatalf("key %d: (%v,%v)", i, admit, overflow)
		}
	}
	for i, c := range []struct {
		at              time.Time
		path            string
		admit, overflow bool
	}{
		{now, "0", false, false},
		{now, "overflow", false, true},
		{now, "overflow2", false, false},
		{now.Add(-24 * time.Hour), "old", false, false},
		{now, "overflow", false, false},
		{now.Add(24 * time.Hour), "overflow", true, false},
		{now.Add(24 * time.Hour), "0", true, false},
	} {
		if admit, overflow := gate.Reserve(c.at, archiveWarningKey{Path: c.path}); admit != c.admit || overflow != c.overflow {
			t.Fatalf("step %d: (%v,%v), want (%v,%v)", i, admit, overflow, c.admit, c.overflow)
		}
	}
	if len(gate.keys) != 2 {
		t.Fatalf("new day retains %d keys, want only its 2 reservations", len(gate.keys))
	}
	nextDay := now.Add(24 * time.Hour)
	for i := 1; i < historyarchive.MaxFiles-1; i++ {
		if admit, overflow := gate.Reserve(nextDay, archiveWarningKey{Path: fmt.Sprint(i)}); !admit || overflow {
			t.Fatalf("new day key %d: (%v,%v)", i, admit, overflow)
		}
	}
	for i := 0; i < 100; i++ {
		if admit, overflow := gate.Reserve(nextDay, archiveWarningKey{Path: fmt.Sprintf("excess-%d", i)}); admit || overflow != (i == 0) {
			t.Fatalf("new day overflow %d: (%v,%v)", i, admit, overflow)
		}
	}
	if len(gate.keys) != historyarchive.MaxFiles {
		t.Fatalf("overflow retained %d keys, want cap %d", len(gate.keys), historyarchive.MaxFiles)
	}
}

func TestArchiveWarningGateAdmissionCounts(t *testing.T) {
	for _, n := range []int{historyarchive.MaxFiles / 2, historyarchive.MaxFiles} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var gate archiveWarningGate
			now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
			count := 0
			for pass := 0; pass < 2; pass++ {
				for i := 0; i < n; i++ {
					admit, overflow := gate.Reserve(now, archiveWarningKey{Path: fmt.Sprint(i)})
					if admit {
						count++
					}
					if overflow {
						t.Fatal("unexpected overflow")
					}
				}
			}
			if count != n {
				t.Fatalf("2N calls admitted %d, want N=%d", count, n)
			}
		})
	}
}

// The reference keeps admitted events rather than the production gate's state.
func TestArchiveWarningGateGeneratedSequence(t *testing.T) {
	rng := rand.New(rand.NewSource(700))
	var gate archiveWarningGate
	highDay := ""
	admitted := map[archiveWarningKey]string{}
	overflowDay := ""
	dailyCount := 0
	start := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	for step := 0; step < 50000; step++ {
		day := step / 15000
		offset := day
		if step%7 == 0 {
			offset--
		} // Rollbacks interspersed with duplicates and new keys.
		now := start.AddDate(0, 0, offset).Add(time.Duration(rng.Intn(86400)) * time.Second).In(time.FixedZone("generated", (rng.Intn(25)-12)*3600))
		key := archiveWarningKey{Source: historyarchive.SourceIdentity{Host: fmt.Sprint(rng.Intn(3)), Root: "root", Repository: "repo", Worktree: fmt.Sprint(rng.Intn(2))}, Kind: fmt.Sprint(rng.Intn(2)), Path: fmt.Sprint(rng.Intn(2500))}
		date := now.UTC().Format("2006-01-02")
		wantAdmit, wantOverflow := false, false
		if date >= highDay {
			if date > highDay {
				highDay = date
				dailyCount = 0
			}
			if admitted[key] != date {
				if dailyCount < historyarchive.MaxFiles {
					wantAdmit = true
					admitted[key] = date
					dailyCount++
				} else if overflowDay != date {
					wantOverflow = true
					overflowDay = date
				}
			}
		}
		if admit, overflow := gate.Reserve(now, key); admit != wantAdmit || overflow != wantOverflow {
			t.Fatalf("seed 700 step %d (%s): (%v,%v), want (%v,%v)", step, now, admit, overflow, wantAdmit, wantOverflow)
		}
	}
}
