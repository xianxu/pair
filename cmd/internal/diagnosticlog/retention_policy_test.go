package diagnosticlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiagnosticDefaultSevenDayBoundary(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	for _, age := range []time.Duration{-time.Hour, 0, 7*24*time.Hour - 1, 7 * 24 * time.Hour, 8 * 24 * time.Hour} {
		want := age >= 7*24*time.Hour
		if got := DecideSegment(now, now.Add(-age)); got != want {
			t.Errorf("age=%s eligible=%t want=%t", age, got, want)
		}
	}
	if DecideSegment(now, time.Time{}) {
		t.Fatal("unknown clock granted deletion authority")
	}
}

func TestDiagnosticExpiredGenerationNeedsNoBackup(t *testing.T) {
	path, clock, opts := fixture(t)
	started := *clock
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	if _, err := w.Write([]byte("expired")); err != nil {
		t.Fatal(err)
	}
	*clock = started.Add(25 * time.Hour)
	if _, err := w.Write([]byte("recent")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	*clock = started.Add(7*24*time.Hour + time.Hour)
	removed, err := Collect(path, opts, 100)
	if err != nil || len(removed) != 1 || removed[0].Bytes != int64(len("expired")) {
		t.Fatalf("expired generation not collected: %+v err=%v", removed, err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "recent" {
		t.Fatalf("recent current changed: %q %v", body, err)
	}
}

func TestDiagnosticYoungOversizedLegacyCurrentEvicted(t *testing.T) {
	path, clock, opts := fixture(t)
	opts.MaxBytes = 4
	if err := os.WriteFile(path, []byte("oversized legacy"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, *clock, *clock); err != nil {
		t.Fatal(err)
	}
	neighbor := filepath.Join(filepath.Dir(path), "authored-history.md")
	if err := os.WriteFile(neighbor, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	removed, err := Collect(path, opts, 100)
	if err != nil || len(removed) != 1 || removed[0].Bytes != int64(len("oversized legacy")) {
		t.Fatalf("young legacy excess not evicted: %+v %v", removed, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("oversized payload remains: %v", err)
	}
	if body, err := os.ReadFile(neighbor); err != nil || string(body) != "keep" {
		t.Fatalf("unrelated history changed: %q %v", body, err)
	}
}

// Seed certified real payload/metadata without exercising hundreds of unrelated
// durable append/rotation transactions. The collector still performs every real
// unlink and journal write. Rotation itself retains its separate 70-record test.
func seedSmallGenerations(t *testing.T, path string, opts Options, count int) {
	t.Helper()
	for n := range count {
		name := fmt.Sprintf("segment-%04x%028x.log", n, n)
		p := segmentPath(path, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte{byte(n)}, 0600); err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		g := generation{Name: name, Identity: fileIdentity(st), Start: opts.Now(), LastWrite: opts.Now(), Size: 1, ModTime: st.ModTime()}
		b, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p+".json", b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestDiagnosticRecentGenerationsHaveNoCountCliff(t *testing.T) {
	path, clock, opts := fixture(t)
	opts.MaxBytes = 1
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.Write([]byte("a")); err != nil {
		t.Fatal(err)
	}
	seedSmallGenerations(t, w.path, opts, 80)
	*clock = clock.Add(time.Second)
	if n, err := w.Write([]byte("b")); err != nil || n != 1 {
		t.Fatalf("rotation blocked by dormant count authority: %d %v", n, err)
	}
	rows, err := Preview(path, opts, 100)
	if err != nil || len(rows) != 82 {
		t.Fatalf("recent generations lost: %d %v", len(rows), err)
	}
	for _, row := range rows {
		if row.Eligible || row.Bytes != 1 {
			t.Fatalf("recent bounded generation eligible: %+v", row)
		}
	}
}

func TestDiagnosticAgeSweepCountedIO(t *testing.T) {
	for _, count := range []int{80, 160} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			path, clock, opts := fixture(t)
			opts.MaxBytes = 1
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = w.Close() })
			if _, err := w.Write([]byte("a")); err != nil {
				t.Fatal(err)
			}
			seedSmallGenerations(t, w.path, opts, count-1)
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			*clock = clock.Add(8 * 24 * time.Hour)
			probe := &fakeInspection{}
			proofs, writes, metadataBytes, payloadCopies := 0, 0, 0, 0
			opts.Proof = func(ctx context.Context, path string, registrations []Registration) error {
				proofs++
				return VerifyWriters(ctx, path, registrations, probe)
			}
			opts.ObserveIO = func(effect IOEffect) {
				if effect.Operation != "write" {
					return
				}
				writes++
				metadataBytes += effect.Bytes
				if effect.Path == path || validSegment(filepath.Base(effect.Path)) {
					payloadCopies += effect.Bytes
				}
			}
			cursor, pages, removed, removedBytes := "", 0, 0, int64(0)
			for pages < 32 {
				rows, next, complete, err := collectManagedPage(path, opts, cursor, 17)
				pages++
				if err != nil {
					t.Fatal(err)
				}
				removed += len(rows)
				for _, row := range rows {
					removedBytes += row.Bytes
				}
				if complete {
					break
				}
				if next == cursor {
					t.Fatal("page cursor did not advance")
				}
				cursor = next
			}
			if removed != count || removedBytes != int64(count) {
				t.Fatalf("removed=%d bytes=%d want=%d", removed, removedBytes, count)
			}
			if pages >= 32 || proofs > 2*pages+2 || writes > 4*count+8 || metadataBytes > 8192*(count+1) || payloadCopies != 0 {
				t.Fatalf("sweep exceeded counted budget: pages=%d proofs=%d writes=%d metadata=%d payload=%d", pages, proofs, writes, metadataBytes, payloadCopies)
			}
			t.Logf("generations=%d removed=%d bytes=%d pages=%d proofs=%d writes=%d metadata=%d payload-copies=%d", count, removed, removedBytes, pages, proofs, writes, metadataBytes, payloadCopies)
		})
	}
}

func TestDiagnosticRegisteredOpenHolderBlocksCollection(t *testing.T) {
	path, clock, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(8 * 24 * time.Hour)
	probe := &fakeInspection{holders: []Registration{w.registration}}
	opts.Proof = func(ctx context.Context, path string, registrations []Registration) error {
		return VerifyWriters(ctx, path, registrations, probe)
	}
	if rows, err := Collect(path, opts, 100); err == nil || len(rows) != 0 {
		t.Fatalf("live registered open holder was not blocked: %+v %v", rows, err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "old" {
		t.Fatalf("held payload changed: %q %v", body, err)
	}
	probe.holders = nil
	if rows, err := Collect(path, opts, 100); err != nil || len(rows) != 1 {
		t.Fatalf("released holder did not permit collection: %+v %v", rows, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("released expired payload remains: %v", err)
	}
}

func FuzzDiagnosticRetentionPolicy(f *testing.F) {
	for _, value := range []string{"", "1", "7", "365", "366", "-1", "0", "bad", "99999999999999999999999"} {
		f.Add(value, int64(0), int64(64))
	}
	f.Fuzz(func(t *testing.T, value string, age, size int64) {
		period, err := ParseRetentionDays(value)
		if err != nil {
			return
		}
		if period < 24*time.Hour || period > 365*24*time.Hour || period%(24*time.Hour) != 0 {
			t.Fatal("parser escaped allowed day domain", period)
		}
		now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
		// Keep arithmetic within independently representable clocks; fuzz both sides
		// of the age boundary and both sides of the size cap without duration wrap.
		age = age % (400 * 24 * int64(time.Hour))
		stamp := now.Add(-time.Duration(age))
		want := age >= 0 && size >= 0 && (time.Duration(age) >= period || size > 64)
		if got := DecideGeneration(now, stamp, size, 64, period); got != want {
			t.Fatalf("age=%d size=%d period=%s got=%t want=%t", age, size, period, got, want)
		}
	})
}
