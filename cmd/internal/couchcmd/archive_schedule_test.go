package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

func TestArchiveSchedulerRunsDailyOnlyWhenConfigured(t *testing.T) {
	data := t.TempDir()
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	var notes []string
	s := newArchiveScheduler(data, func(n string) { notes = append(notes, n) })
	s.now = func() time.Time { return now }
	runs := 0
	var fail error
	s.launch = func(context.Context) error { runs++; return fail }
	s.tick(context.Background())
	if runs != 0 {
		t.Fatal("ran without a config")
	}
	os.MkdirAll(s.store, 0o700)
	os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0o600)
	s.tick(context.Background())
	if runs != 1 {
		t.Fatalf("configured, never run: runs=%d", runs)
	}
	os.WriteFile(filepath.Join(s.store, historyarchive.LastRunName), []byte(now.Add(-time.Hour).Format(time.RFC3339)+"\n"), 0o600)
	s.tick(context.Background())
	if runs != 1 {
		t.Fatal("ran again within a day")
	}
	os.WriteFile(filepath.Join(s.store, historyarchive.LastRunName), []byte(now.Add(-25*time.Hour).Format(time.RFC3339)+"\n"), 0o600)
	fail = errors.New("boom")
	s.tick(context.Background())
	s.tick(context.Background())
	if runs != 2 || len(notes) != 1 {
		t.Fatalf("stale day: runs=%d notes=%v, want one failed attempt, one notice, and no hourly retry", runs, notes)
	}
	now = now.Add(archiveRetryAfter)
	fail = nil
	s.tick(context.Background())
	if runs != 3 {
		t.Fatalf("no retry after the backoff: runs=%d", runs)
	}
}

func TestArchiveLogHoldsTheLatestPassOnly(t *testing.T) {
	store := t.TempDir()
	for _, line := range []string{"first pass, a long line\n", "second\n"} {
		f, err := openArchiveLog(store)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(line)
		f.Close()
	}
	if raw, _ := os.ReadFile(filepath.Join(store, "continuous.log")); string(raw) != "second\n" {
		t.Fatalf("log %q, want only the latest pass", raw)
	}
}
