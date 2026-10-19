package diagnosticlog

import (
	"context"
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

func TestDiagnosticRecentGenerationsHaveNoCountCliff(t *testing.T) {
	path, clock, opts := fixture(t)
	opts.MaxBytes = 1
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for n := range 80 {
		*clock = clock.Add(time.Second)
		if count, err := w.Write([]byte{byte(n)}); err != nil || count != 1 {
			t.Fatalf("record %d rejected by stale count authority: %d %v", n, count, err)
		}
	}
	rows, err := Preview(path, opts, 100)
	if err != nil || len(rows) != 80 {
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
			for n := range count {
				*clock = clock.Add(time.Second)
				if written, err := w.Write([]byte{byte(n)}); err != nil || written != 1 {
					t.Fatalf("seed %d: %d %v", n, written, err)
				}
			}
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
