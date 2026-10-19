package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	s.launch = func(context.Context) (historyarchive.ContinuousCommandReport, error) {
		runs++
		return historyarchive.ContinuousCommandReport{Outcome: historyarchive.ContinuousComplete}, fail
	}
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

func TestArchiveSchedulerTypedStalePartialOncePerDay(t *testing.T) {
	data, fixture := t.TempDir(), t.TempDir()
	t.Setenv("PAIR_ARCHIVE_FIXTURE", fixture)
	t.Setenv("PATH", fixture+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\nprintf 'start\\n' >> \"$PAIR_ARCHIVE_FIXTURE/starts\"\nprintf '%s\\n' \"$*\" > \"$PAIR_ARCHIVE_FIXTURE/args\"\ncat \"$PAIR_ARCHIVE_FIXTURE/report\"\nprintf 'fixture partial failure\\n' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(fixture, "pair"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	row := historyarchive.DatabaseBackupSummary{Source: historyarchive.SourceIdentity{Host: "h", Root: "/fixture"}, Kind: "codex-database", Path: "stale.sqlite", Status: historyarchive.DatabaseDeferred, Detail: "deadline"}
	raw, err := historyarchive.EncodeContinuousReport(historyarchive.ContinuousCommandReport{Version: historyarchive.ContinuousReportVersion, Outcome: historyarchive.ContinuousFailed, MaxDatabaseAge: 48 * time.Hour, Report: historyarchive.ContinuousReport{Databases: []historyarchive.DatabaseBackupSummary{row}}})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(fixture, "report"), raw, 0600)
	var notes []string
	s := newArchiveScheduler(data, func(n string) { notes = append(notes, n) })
	now := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	os.MkdirAll(s.store, 0700)
	os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0600)
	count := func() int {
		n := 0
		for _, note := range notes {
			if strings.Contains(note, "stale.sqlite") && strings.Contains(note, "Ask the hub") {
				n++
			}
		}
		return n
	}
	s.tick(context.Background())
	if count() != 1 {
		t.Fatalf("stale partial pass omitted its database warning: %v", notes)
	}
	s.tick(context.Background())
	now = now.Add(6 * time.Hour)
	s.tick(context.Background())
	if count() != 1 {
		t.Fatalf("repeated daily warning: %v", notes)
	}
	now = now.Add(24 * time.Hour)
	s.tick(context.Background())
	if count() != 2 {
		t.Fatalf("next day did not warn: %v", notes)
	}
	args, _ := os.ReadFile(filepath.Join(fixture, "args"))
	if !strings.Contains(string(args), "archive continuous --json") {
		t.Fatalf("argv=%s", args)
	}
	log, _ := os.ReadFile(filepath.Join(s.store, "continuous.log"))
	if !strings.Contains(string(log), "stale.sqlite") {
		t.Fatalf("report missing: %s", log)
	}
}

func TestArchiveSchedulerCountedWarningRows(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s := newArchiveScheduler(t.TempDir(), func(string) {})
			os.MkdirAll(s.store, 0700)
			os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0600)
			now := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)
			clocks := 0
			s.now = func() time.Time { clocks++; return now }
			rows := make([]historyarchive.DatabaseBackupSummary, n)
			for i := range rows {
				rows[i] = historyarchive.DatabaseBackupSummary{Source: historyarchive.SourceIdentity{Host: "fixture", Root: "/nonexistent-must-not-open"}, Kind: "codex-database", Path: fmt.Sprintf("db_%d.sqlite", i), Status: historyarchive.DatabaseDeferred}
			}
			starts, outputs := 0, 0
			s.launch = func(context.Context) (historyarchive.ContinuousCommandReport, error) {
				starts++
				return historyarchive.ContinuousCommandReport{Outcome: historyarchive.ContinuousComplete, MaxDatabaseAge: 48 * time.Hour, Report: historyarchive.ContinuousReport{Databases: rows}}, nil
			}
			s.warn = func(_ context.Context, body string) (string, error) {
				outputs++
				return body, errors.New("uncertain fixture output")
			}
			s.tick(context.Background())
			if starts != 1 || outputs != n || clocks != n || len(s.warnings.keys) != n {
				t.Fatalf("N=%d starts=%d outputs=%d clocks=%d keys=%d", n, starts, outputs, clocks, len(s.warnings.keys))
			}
			s.tick(context.Background())
			if starts != 2 || outputs != n || clocks != 2*n {
				t.Fatalf("uncertain output retried: starts=%d outputs=%d clocks=%d", starts, outputs, clocks)
			}
			restarted := newArchiveScheduler(filepath.Dir(s.store), func(string) {})
			restarted.now = s.now
			restarted.launch = s.launch
			restarted.warn = s.warn
			restarted.tick(context.Background())
			if outputs != 2*n {
				t.Fatalf("restart did not allow fresh warnings: %d", outputs)
			}
		})
	}
}
