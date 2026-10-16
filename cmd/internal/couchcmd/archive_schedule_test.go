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
	if runs != 3 || len(notes) != 1 {
		t.Fatalf("stale day: runs=%d notes=%v, want two attempts and one notice", runs, notes)
	}
}
