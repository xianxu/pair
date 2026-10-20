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

func TestArchiveDriveCountRootDailyAdmission(t *testing.T) {
	s := newArchiveScheduler(t.TempDir(), func(string) {})
	if err := os.MkdirAll(s.store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.store, "continuous.log"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	observation := &historyarchive.DriveInventoryObservation{Destination: "remote:root", FolderID: "root-1", Phase: "after", Files: 20001}
	report := historyarchive.ContinuousCommandReport{Outcome: historyarchive.ContinuousComplete, MaxDriveFiles: 20000, Report: historyarchive.ContinuousReport{Drive: &historyarchive.DriveReport{Inventory: observation}}}
	s.launch = func(context.Context) (historyarchive.ContinuousCommandReport, error) { return report, nil }
	var outputs []string
	s.warn = func(_ context.Context, body string) (string, error) {
		outputs = append(outputs, body)
		return body, errors.New("uncertain fixture")
	}
	check := func(want int) {
		t.Helper()
		s.tick(context.Background())
		if len(outputs) != want {
			t.Fatalf("outputs=%d want %d: %v", len(outputs), want, outputs)
		}
	}
	check(1)
	observation.FolderID = "replacement-root" // provenance cannot rearm configured root
	observation.Phase = "before"
	check(1)
	observation.Files = 0
	check(1)
	observation.Files = 30000
	report.MaxDriveFiles = 25000
	check(1)
	observation.Destination = "remote:other"
	check(2)
	now = now.Add(24 * time.Hour)
	check(3)
	report.Report.Drive.Inventory = nil
	check(3)
	report.Report.Drive.Inventory = observation
	observation.Files = report.MaxDriveFiles // strict greater-than
	now = now.Add(24 * time.Hour)
	check(3)
	observation.Files++
	check(4)
	if !strings.Contains(outputs[0], "Ask the hub to arrange year subfolders on Drive.") || !strings.Contains(outputs[0], "20001") {
		t.Fatal(outputs[0])
	}
	log, err := os.ReadFile(filepath.Join(s.store, "continuous.log"))
	if err != nil || !strings.Contains(string(log), "notification outcome uncertain") {
		t.Fatalf("log=%s err=%v", log, err)
	}
	restarted := newArchiveScheduler(filepath.Dir(s.store), func(string) {})
	restarted.now = s.now
	restarted.launch = s.launch
	restarted.warn = s.warn
	restarted.tick(context.Background())
	if len(outputs) != 5 {
		t.Fatalf("restart should allow one fresh warning: %v", outputs)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	now = now.Add(24 * time.Hour)
	s.tick(cancelled)
	if len(outputs) != 5 {
		t.Fatal("cancelled pass emitted warning")
	}
}

// A reference model ignores physical folder IDs and tracks only configured
// roots and the latest UTC day, including clock rollback and uncertain output.
func TestArchiveDriveCountAdmissionMatchesSequenceModel(t *testing.T) {
	s := newArchiveScheduler(t.TempDir(), func(string) {})
	if err := os.MkdirAll(s.store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	now := base
	s.now = func() time.Time { return now }
	var report historyarchive.ContinuousCommandReport
	s.launch = func(context.Context) (historyarchive.ContinuousCommandReport, error) { return report, nil }
	attempts := 0
	s.warn = func(_ context.Context, text string) (string, error) { attempts++; return text, errors.New("uncertain") }
	day := -1
	reserved := map[string]bool{}
	want := 0
	for i := 0; i < 240; i++ {
		currentDay := i / 40
		if i%17 == 0 && currentDay > 0 {
			currentDay--
		}
		now = base.Add(time.Duration(currentDay) * 24 * time.Hour)
		root := fmt.Sprintf("remote:root-%d", (i*7)%5)
		limit := 1 + i%13
		files := (i * 11) % 29
		observation := &historyarchive.DriveInventoryObservation{Destination: root, FolderID: fmt.Sprintf("physical-%d", i), Phase: []string{"before", "after"}[i%2], Files: files}
		if i%9 == 0 {
			observation = nil
		}
		report = historyarchive.ContinuousCommandReport{Outcome: historyarchive.ContinuousComplete, MaxDriveFiles: limit, Report: historyarchive.ContinuousReport{Drive: &historyarchive.DriveReport{Inventory: observation}}}
		if observation != nil && files > limit {
			if currentDay > day {
				day = currentDay
				reserved = map[string]bool{}
			}
			if currentDay == day && !reserved[root] {
				want++
				reserved[root] = true
			}
		}
		s.tick(context.Background())
		if attempts != want {
			t.Fatalf("event %d day=%d root=%s files=%d limit=%d attempts=%d want=%d", i, currentDay, root, files, limit, attempts, want)
		}
	}
}
