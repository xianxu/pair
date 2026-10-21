package couchschedule

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStatusRefusedCatalogEntriesDoNotPoisonHealthyTasks(t *testing.T) {
	root := catalogRoot(t)
	catalogTask(t, root, "good", "")
	for _, name := range []string{"BadTask", "bad.name", "bad\nname", strings.Repeat("a", 129), "\xff", `:entry:ff`} {
		if err := os.Mkdir(filepath.Join(root, "scripts", "scheduled", name), 0700); err != nil {
			if errors.Is(err, syscall.EILSEQ) {
				continue
			}
			t.Fatal(err)
		}
	}
	catalogTask(t, root, "malformed", "")
	if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/malformed/schedule"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "scripts/scheduled/missing"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/regular"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "scripts/scheduled/linked")); err != nil {
		t.Fatal(err)
	}
	catalogTask(t, root, "foreign", `,"scope":"team","owner":"someone"`)
	if err := os.WriteFile(filepath.Join(root, "team.md"), []byte("## Operators\n- someone (Someone): owner\n"), 0600); err != nil {
		t.Fatal(err)
	}
	declarations, err := (Catalog{Git: &catalogGit{names: map[string]string{root: "Other Operator"}}}).Discover(t.Context(), []string{root})
	if err != nil {
		t.Fatal(err)
	}
	states := make([]TaskState, 0, len(declarations))
	for _, d := range declarations {
		state, _ := Advance(TaskState{}, Event{Kind: CatalogChanged, At: time.Now(), Declaration: &d})
		states = append(states, state)
	}
	snapshot := BuildStatus(states, "UTC")
	if err = snapshot.Validate(); err != nil {
		t.Fatalf("one refused entry poisoned healthy tasks: %v", err)
	}
	good := 0
	diagnostics := 0
	for _, row := range snapshot.Tasks {
		if row.Task == "good" {
			good++
			if row.State == "refused" {
				t.Fatal("valid task refused with bad siblings")
			}
		} else {
			diagnostics++
			if row.State != "refused" || row.Reason == "" {
				t.Fatalf("damaged sibling escaped refusal: %#v", row)
			}
			if row.Task == "" && (row.Entry == "" || row.EntryID == "") {
				t.Fatalf("invalid entry lacks diagnostic identity: %#v", row)
			}
			if row.Task != "" && !ValidTaskName(row.Task) {
				t.Fatalf("invalid entry gained routing: %#v", row)
			}
			if strings.IndexFunc(row.Entry, func(r rune) bool { return r < 0x20 || r > 0x7e }) >= 0 || len(row.Entry) > 1024 {
				t.Fatalf("unsafe entry display %q", row.Entry)
			}
		}
	}
	if good != 1 || diagnostics != len(declarations)-1 {
		t.Fatalf("lost rows: good=%d diagnostics=%d declarations=%d", good, diagnostics, len(declarations))
	}
	store := StateStore{Dir: filepath.Join(t.TempDir(), "schedule")}
	if err = store.Update(context.Background(), func(l *Ledger) error {
		for _, s := range states {
			l.Tasks[s.Key.ID()] = s
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.View(context.Background()); err != nil {
		t.Fatalf("refused names corrupted reopened authority: %v", err)
	}
}

func TestStatusRefusedIdentityPreservesRawBytesWithoutLiteralCollision(t *testing.T) {
	names := []string{"\xff", ":entry:ff", "BadTask", "bad\nname", strings.Repeat("x", 2048)}
	seen := map[string]bool{}
	var states []TaskState
	for _, name := range names {
		identity := refusedTaskIdentity(name)
		if seen[identity] || ValidTaskName(identity) {
			t.Fatalf("colliding/routable identity %q", identity)
		}
		seen[identity] = true
		states = append(states, TaskState{Key: TaskKey{Hub: "/hub", Task: identity}, Revision: "revision", Refusal: "invalid task name"})
		want := strconv.QuoteToASCII(name)
		if len(want) > 1024 {
			want = want[:1021] + "..."
		}
		if got := refusedTaskDisplay(identity); got != want {
			t.Fatalf("display=%q want=%q", got, want)
		}
	}
	store := StateStore{Dir: filepath.Join(t.TempDir(), "state")}
	if err := store.Update(t.Context(), func(l *Ledger) error {
		for _, s := range states {
			l.Tasks[s.Key.ID()] = s
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ledger, err := store.View(t.Context())
	if err != nil {
		t.Fatalf("raw-byte refusal did not reopen: %v", err)
	}
	if len(ledger.Tasks) != len(names) {
		t.Fatal("literal encoded name collided with raw name")
	}
	rows := make([]TaskState, 0, len(ledger.Tasks))
	for _, s := range ledger.Tasks {
		rows = append(rows, s)
	}
	if err := BuildStatus(rows, "UTC").Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStatusDiagnosticEntryCannotCarryRunnableAuthority(t *testing.T) {
	key := TaskKey{Hub: "/hub", Task: refusedTaskIdentity("BadTask")}
	base := BuildStatus([]TaskState{{Key: key, Revision: "revision", Refusal: "invalid task name"}}, "UTC")
	for _, tt := range []struct {
		name   string
		mutate func(*TaskStatus)
	}{
		{"runnable", func(r *TaskStatus) { r.State = "waiting" }},
		{"routing", func(r *TaskStatus) { r.Task = "good" }},
		{"raw task routing", func(r *TaskStatus) { r.Task = "BadTask"; r.Entry = ""; r.EntryID = "" }},
		{"no reason", func(r *TaskStatus) { r.Reason = "" }},
		{"active", func(r *TaskStatus) { r.Active = &Attempt{ID: "run"} }},
		{"history", func(r *TaskStatus) { r.Last = &RunSummary{ID: "run"} }},
		{"due", func(r *TaskStatus) { r.NextDue = time.Now() }},
		{"control display", func(r *TaskStatus) { r.Entry = "bad\nname" }},
		{"unbounded display", func(r *TaskStatus) { r.Entry = strings.Repeat("a", 1025) }},
		{"invalid ID", func(r *TaskStatus) { r.EntryID = "not-a-hash" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := base
			s.Tasks = append([]TaskStatus(nil), base.Tasks...)
			tt.mutate(&s.Tasks[0])
			if s.Validate() == nil {
				t.Fatal("diagnostic entry acquired authority or unsafe display")
			}
		})
	}
	base.Tasks = append(base.Tasks, base.Tasks[0])
	if base.Validate() == nil {
		t.Fatal("accepted duplicate refused identities")
	}
}

func TestStatusSnapshotIsSortedDetachedAndBounded(t *testing.T) {
	s, due := taskFixture(t, true)
	s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "first"})
	other := s
	other.Key.Hub = "/another"
	other.Active = nil
	other.Allowed = false
	other.Refusal = "owner missing"
	snapshot := BuildStatus([]TaskState{s, other}, "UTC")
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tasks) != 2 || snapshot.Tasks[0].Hub != "/another" || snapshot.Tasks[0].State != "refused" || snapshot.Tasks[1].State != "reserved" {
		t.Fatalf("unexpected projection: %+v", snapshot)
	}
	snapshot.Tasks[1].Active.Stage = "changed"
	if s.Active.Stage != "reserved" {
		t.Fatal("snapshot aliases authoritative attempt")
	}
	code := 10
	s.Last = &RunSummary{Exit: &code}
	snapshot = BuildStatus([]TaskState{s}, "UTC")
	*snapshot.Tasks[0].Last.Exit = 7
	if *s.Last.Exit != 10 {
		t.Fatal("snapshot aliases the last run exit")
	}
	snapshot.Version = 2
	if snapshot.Validate() == nil {
		t.Fatal("accepted unknown snapshot version")
	}
}
