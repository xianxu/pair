package couchschedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func storedTask(t *testing.T) TaskState {
	t.Helper()
	spec, err := ParseTaskSpec([]byte(`{"version":1,"cron":"0 7 * * *","timeout":"1m"}`))
	if err != nil {
		t.Fatal(err)
	}
	return TaskState{Key: TaskKey{Hub: "/hub", Task: "task"}, Revision: "rev", Spec: spec, Allowed: true, NextDue: time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)}
}
func TestStoreMissingViewAndConcurrentWriters(t *testing.T) {
	dir := filepath.Join(catalogRoot(t), "new")
	s := StateStore{Dir: dir}
	v, err := s.View(t.Context())
	if err != nil || v.Version != 1 || len(v.Tasks) != 0 {
		t.Fatalf("missing: %+v %v", v, err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("View created store: %v", err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- s.Update(t.Context(), func(l *Ledger) error {
				task := storedTask(t)
				task.Key.Task = string(rune('a' + i))
				l.Tasks[task.Key.ID()] = task
				return nil
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	v, err = s.View(t.Context())
	if err != nil || len(v.Tasks) != 12 {
		t.Fatalf("lost updates: %d %v", len(v.Tasks), err)
	}
}

func TestStateJSONPreservesEveryNestedUnknown(t *testing.T) {
	task := storedTask(t)
	task.Active = &Attempt{ID: "run", Revision: "rev", Source: "manual", Stage: "running", StartedAt: task.NextDue, Deadline: task.NextDue.Add(time.Minute)}
	task.Last = &RunSummary{ID: "past", Source: "manual", Outcome: "quiet", StartedAt: task.NextDue, EndedAt: task.NextDue}
	l := Ledger{Version: 1, Tasks: map[string]TaskState{task.Key.ID(): task}}
	raw, err := EncodeLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	// Inject distinct raw formatting at all schema nesting levels.
	var top map[string]json.RawMessage
	json.Unmarshal(raw, &top)
	var tasks map[string]json.RawMessage
	json.Unmarshal(top["tasks"], &tasks)
	var row map[string]json.RawMessage
	json.Unmarshal(tasks[task.Key.ID()], &row)
	marker := json.RawMessage(`{ "future" : [1, 2] }`)
	for _, name := range []string{"key", "spec", "active", "last"} {
		var obj map[string]json.RawMessage
		json.Unmarshal(row[name], &obj)
		obj["future_"+name] = marker
		row[name], _ = json.Marshal(obj)
	}
	row["future_task"] = marker
	tasks[task.Key.ID()], _ = json.Marshal(row)
	top["tasks"], _ = json.Marshal(tasks)
	top["future_ledger"] = marker
	raw, _ = json.Marshal(top)
	raw = bytes.ReplaceAll(raw, []byte(`{"future":[1,2]}`), marker)
	if count := bytes.Count(raw, marker); count != 6 {
		t.Fatalf("fixture markers: %d", count)
	}
	// Preserve the exact raw member bytes that actually reach the decoder.
	decoded, err := DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	changed := decoded.Tasks[task.Key.ID()]
	changed.Last.Outcome = "failed"
	decoded.Tasks[task.Key.ID()] = changed
	next, err := EncodeLedger(decoded)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"key", "spec", "active", "last", "task", "ledger"} {
		if !bytes.Contains(next, []byte(`"future_`+name+`"`)) {
			t.Errorf("lost unknown at %s: %s", name, next)
		}
	}
	if count := bytes.Count(next, marker); count != 6 {
		t.Fatalf("unknown values lost formatting: count %d, %s", count, next)
	}
	again, err := DecodeLedger(next)
	if err != nil || again.Tasks[task.Key.ID()].Last.Outcome != "failed" {
		t.Fatalf("rewrite: %+v %v", again, err)
	}
}

func TestStoreLockCancellationAndDirectoryDurabilityFailure(t *testing.T) {
	root := catalogRoot(t)
	s := StateStore{Dir: root}
	task := storedTask(t)
	dir, err := openScheduleDir(t.Context(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	release, err := lockSchedule(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	called := false
	err = s.Update(ctx, func(*Ledger) error { called = true; return nil })
	release()
	if !errors.Is(err, context.DeadlineExceeded) || called {
		t.Fatalf("lock wait: called=%v err=%v", called, err)
	}
	s.BeforeDirectorySync = func() error { return errors.New("directory sync failed") }
	err = s.Update(t.Context(), func(l *Ledger) error { l.Tasks[task.Key.ID()] = task; return nil })
	if err == nil {
		t.Fatal("directory sync error hidden")
	}
	// Rename may have committed: failure cannot be reported as definitely unrun.
	current, err := s.View(t.Context())
	if err != nil || len(current.Tasks) != 1 {
		t.Fatalf("post-rename evidence missing: %+v %v", current, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("leaked stage %s", entry.Name())
		}
	}
}

func TestStoreNoFollowAndAnchoredPublication(t *testing.T) {
	root, outside := catalogRoot(t), catalogRoot(t)
	live := filepath.Join(root, "live")
	s := StateStore{Dir: live}
	task := storedTask(t)
	if err := s.Update(t.Context(), func(l *Ledger) error { l.Tasks[task.Key.ID()] = task; return nil }); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved")
	err := s.Update(t.Context(), func(l *Ledger) error {
		if err := os.Rename(live, moved); err != nil {
			return err
		}
		if err := os.Symlink(outside, live); err != nil {
			return err
		}
		task.Revision = "changed"
		l.Tasks[task.Key.ID()] = task
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "state.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("redirected write: %v", err)
	}
	if _, err := s.View(t.Context()); err == nil {
		t.Fatal("followed replaced ancestor")
	}
	state, err := (StateStore{Dir: moved}).View(t.Context())
	if err != nil || state.Tasks[task.Key.ID()].Revision != "changed" {
		t.Fatalf("anchored commit: %+v %v", state, err)
	}
	lock := filepath.Join(moved, ".state.lock")
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(moved, "state.json"), lock); err != nil {
		t.Fatal(err)
	}
	if err := (StateStore{Dir: moved}).Update(t.Context(), func(*Ledger) error { return nil }); err == nil {
		t.Fatal("followed lock symlink")
	}
}

func TestStateJSONRefusesNestedCorruption(t *testing.T) {
	task := storedTask(t)
	task.Active = &Attempt{ID: "run", Revision: "rev", Source: "manual", Stage: "running", StartedAt: task.NextDue, Deadline: task.NextDue.Add(time.Minute)}
	task.Last = &RunSummary{ID: "past", Source: "manual", Outcome: "quiet", StartedAt: task.NextDue, EndedAt: task.NextDue}
	raw, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{task.Key.ID(): task}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ old, new string }{
		{`"hub":"/hub"`, `"hub":null`},
		{`"Timeout":60000000000`, `"Timeout":null`},
		{`"controller_pid":0`, `"controller_pid":null`},
		{`"stage":"running"`, `"stage":"invented"`},
		{`"next_due":"2026-01-01T07:00:00Z"`, `"next_due":"yesterday"`},
		{`"outcome":"quiet"`, `"outcome":null`},
		{`"hub":"/hub"`, `"hub":"/hub","hub":"/other"`},
	} {
		if !bytes.Contains(raw, []byte(tc.old)) {
			t.Fatalf("fixture lacks %s", tc.old)
		}
		bad := bytes.Replace(raw, []byte(tc.old), []byte(tc.new), 1)
		if _, err := DecodeLedger(bad); err == nil {
			t.Errorf("accepted %s", tc.new)
		}
	}
}

func TestStoreRefusesCorruptionFaultsAndSymlinks(t *testing.T) {
	root := catalogRoot(t)
	s := StateStore{Dir: root}
	task := storedTask(t)
	if err := s.Update(t.Context(), func(l *Ledger) error { l.Tasks[task.Key.ID()] = task; return nil }); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.BeforePersist = func() error { return errors.New("publication fault") }
	if err := s.Update(t.Context(), func(l *Ledger) error { delete(l.Tasks, task.Key.ID()); return nil }); err == nil {
		t.Fatal("ignored publication fault")
	}
	after, _ := os.ReadFile(filepath.Join(root, "state.json"))
	if !bytes.Equal(original, after) {
		t.Fatal("publication fault changed state")
	}
	for _, raw := range []string{`{}`, `{"version":2,"tasks":{}}`, `{"version":1,"tasks":null}`, `{"version":1,"version":1,"tasks":{}}`, "{", strings.Repeat(" ", 2*1024*1024+1)} {
		if err := os.WriteFile(filepath.Join(root, "state.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.View(t.Context()); err == nil {
			t.Errorf("accepted corrupt state %q", raw[:min(len(raw), 100)])
		}
		called := false
		if err := s.Update(t.Context(), func(*Ledger) error { called = true; return nil }); err == nil || called {
			t.Errorf("corruption reset authority: called=%v err=%v", called, err)
		}
	}
	if err := os.Remove(filepath.Join(root, "state.json")); err != nil {
		t.Fatal(err)
	}
	outside := catalogRoot(t)
	if err := os.WriteFile(filepath.Join(outside, "state"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "state"), filepath.Join(root, "state.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.View(t.Context()); err == nil {
		t.Fatal("followed state symlink")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.Update(ctx, func(*Ledger) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("ignored cancellation: %v", err)
	}
}

func FuzzStateJSON(f *testing.F) {
	f.Add([]byte(`{"version":1,"tasks":{},"future":{ "v" : [1, 2] }}`))
	f.Add([]byte(`{"version":2,"tasks":{}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		l, err := DecodeLedger(raw)
		if err != nil {
			return
		}
		next, err := EncodeLedger(l)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeLedger(next)
		if err != nil || len(again.Tasks) != len(l.Tasks) {
			t.Fatalf("roundtrip: %v", err)
		}
	})
}
