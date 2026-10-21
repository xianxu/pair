package operationalrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sample(id string, at time.Time) Record {
	e := 0
	return Record{Version: 1, Key: RunKey{Hub: "hub", Task: "task", ID: id}, DeclarationRevision: "rev", Source: "direct", ScheduledAt: at, StartedAt: at, EndedAt: at, Exit: &e, Outcome: "completed"}
}
func TestPublishReadImmutableAndPreserving(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	ctx := context.Background()
	r := sample("one", time.Now())
	r.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "x": 2 }`)}
	r.Key.Unknown = map[string]json.RawMessage{"future_key": json.RawMessage(`true`)}
	if e := s.Publish(ctx, r.Key, r, []byte("output")); e != nil {
		t.Fatal(e)
	}
	got, out, e := s.Read(ctx, r.Key)
	if e != nil || string(out) != "output" || string(got.Unknown["future"]) != `{ "x": 2 }` || string(got.Key.Unknown["future_key"]) != "true" {
		t.Fatal(got, string(out), e)
	}
	if e = s.Publish(ctx, r.Key, r, []byte("output")); e != nil {
		t.Fatal(e)
	}
	if e = s.Publish(ctx, r.Key, r, []byte("changed")); e == nil {
		t.Fatal("overwrote immutable run")
	}
	rows, e := s.List(ctx, "hub", "task", 10)
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
	if e = s.Publish(ctx, RunKey{Hub: "../bad", Task: "task", ID: "x"}, r, nil); e == nil {
		t.Fatal("unsafe component accepted")
	}
}
func TestHeadTailBuffer(t *testing.T) {
	b := NewOutputBuffer()
	p := []byte("HEAD" + strings.Repeat("m", OutputLimit*2) + "TAIL")
	if n, e := b.Write(p); e != nil || n != len(p) {
		t.Fatal(n, e)
	}
	out := b.Bytes()
	if len(out) > OutputLimit || !strings.HasPrefix(string(out), "HEAD") || !strings.HasSuffix(string(out), "TAIL") {
		t.Fatal(len(out))
	}
}
func TestRetentionAgeCountAndProtected(t *testing.T) {
	ctx := context.Background()
	s := Store{Namespace: t.TempDir()}
	now := time.Now().UTC()
	old := sample("old", now.Add(-RetentionPeriod))
	if e := s.Publish(ctx, old.Key, old, nil); e != nil {
		t.Fatal(e)
	}
	active := sample("active", old.EndedAt)
	active.EndedAt = time.Time{}
	active.Outcome = "running"
	if e := s.Publish(ctx, active.Key, active, nil); e != nil {
		t.Fatal(e)
	}
	uncertain := sample("uncertain", old.EndedAt)
	uncertain.Outcome = "uncertain"
	if e := s.Publish(ctx, uncertain.Key, uncertain, nil); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < MaxRunsPerTask+1; i++ {
		r := sample(fmt.Sprintf("run-%04d", i), now.Add(time.Duration(i)*time.Millisecond))
		if e := s.Publish(ctx, r.Key, r, nil); e != nil {
			t.Fatal(e)
		}
	}
	var all []Row
	cur := Cursor{}
	for i := 0; i < 100; i++ {
		rows, next, done, e := s.SweepPage(ctx, now.Add(time.Second), cur, true, 23)
		if e != nil {
			t.Fatal(e)
		}
		all = append(all, rows...)
		if done {
			break
		}
		if next == cur {
			t.Fatal("stuck cursor")
		}
		cur = next
	}
	n := 0
	for _, r := range all {
		if r.Eligible {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("removed %d, want age plus count", n)
	}
	if _, _, e := s.Read(ctx, old.Key); !errors.Is(e, os.ErrNotExist) {
		t.Fatal(e)
	}
	if _, _, e := s.Read(ctx, active.Key); e != nil {
		t.Fatal(e)
	}
	if _, _, e := s.Read(ctx, uncertain.Key); e != nil {
		t.Fatal(e)
	}
}
func TestPublicationFaultAndUnsafeAncestry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s := Store{Namespace: root}
	r := sample("x", time.Now())
	s.fault = func(step string) error {
		if step == "publish-ready" {
			return errors.New("fault")
		}
		return nil
	}
	if e := s.Publish(ctx, r.Key, r, nil); e == nil {
		t.Fatal("fault ignored")
	}
	if _, e := os.Stat(filepath.Join(root, "schedule", "hub", "task", "runs", "x")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	s.fault = nil
	if e := s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	other := t.TempDir()
	bad := Store{Namespace: t.TempDir()}
	if e := os.Symlink(other, filepath.Join(bad.Namespace, "schedule")); e != nil {
		t.Fatal(e)
	}
	if e := bad.Publish(ctx, r.Key, r, nil); e == nil {
		t.Fatal("followed symlink")
	}
}
func TestCollectionRecoveryProtectsReplacement(t *testing.T) {
	for _, step := range []string{"journal", "renamed", "detached", "member-output", "member-record", "finalized"} {
		t.Run(step, func(t *testing.T) {
			ctx := context.Background()
			s := Store{Namespace: t.TempDir()}
			now := time.Now()
			r := sample("old", now.Add(-RetentionPeriod-time.Hour))
			if e := s.Publish(ctx, r.Key, r, []byte("old")); e != nil {
				t.Fatal(e)
			}
			s.fault = func(at string) error {
				if at == step {
					return errors.New("fault")
				}
				return nil
			}
			rows, _, _, e := s.SweepPage(ctx, now, Cursor{}, true, 100)
			if e != nil {
				t.Fatal(e)
			}
			if len(rows) == 0 {
				t.Fatal("no report")
			}
			s.fault = nil
			source := filepath.Join(s.Namespace, "schedule", "hub", "task", "runs", "old")
			replaced := false
			if _, e := os.Stat(source); os.IsNotExist(e) {
				replaced = true
				replacement := r
				replacement.EndedAt = now
				if e = s.Publish(ctx, r.Key, replacement, []byte("new")); e != nil {
					t.Fatal(e)
				}
			}
			if _, _, _, e = s.SweepPage(ctx, now, Cursor{}, true, 100); e != nil {
				t.Fatal(e)
			}
			_, out, e := s.Read(ctx, r.Key)
			if replaced {
				if e != nil || string(out) != "new" {
					t.Fatal("replacement lost", string(out), e)
				}
			} else if !errors.Is(e, os.ErrNotExist) {
				t.Fatal("original run not collected", e)
			}
		})
	}
}
func TestSweepLocalBlockerCursorAndTaskLock(t *testing.T) {
	ctx := context.Background()
	s := Store{Namespace: t.TempDir()}
	now := time.Now()
	for _, task := range []string{"a", "b"} {
		r := sample("old", now.Add(-31*24*time.Hour))
		r.Key.Task = task
		if e := s.Publish(ctx, r.Key, r, nil); e != nil {
			t.Fatal(e)
		}
	}
	bad := filepath.Join(s.Namespace, "schedule", "hub", "a", "runs", "old", "extra")
	if e := os.WriteFile(bad, []byte("unknown"), 0600); e != nil {
		t.Fatal(e)
	}
	rows, next, done, e := s.SweepPage(ctx, now, Cursor{}, true, 1)
	if e != nil || done || len(rows) != 1 || rows[0].Eligible {
		t.Fatal(rows, next, done, e)
	}
	for i := 0; i < 5 && !done; i++ {
		rows, next, done, e = s.SweepPage(ctx, now, next, true, 1)
		if e != nil {
			t.Fatal(e)
		}
	}
	if !done {
		t.Fatal("local blocker stranded peer")
	}
	if _, e = os.Stat(filepath.Join(s.Namespace, "schedule", "hub", "b", "runs", "old")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	task := filepath.Join(s.Namespace, "schedule", "hub", "a")
	unlock, e := taskLock(ctx, task, false)
	if e != nil {
		t.Fatal(e)
	}
	defer unlock()
	_, cur, _, e := s.SweepPage(ctx, now, Cursor{}, true, 100)
	if !errors.Is(e, ErrBusy) || cur != (Cursor{}) {
		t.Fatal(cur, e)
	}
}
func TestMalformedRunNamesAndEmptyTasksProgress(t *testing.T) {
	ctx := context.Background()
	s := Store{Namespace: t.TempDir()}
	r := sample("x", time.Now())
	if e := s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(s.Namespace, "schedule", "hub", "task", "runs", ".unknown"), 0700); e != nil {
		t.Fatal(e)
	}
	_, next, done, e := s.SweepPage(ctx, time.Now(), Cursor{}, false, 1)
	if e != nil || done {
		t.Fatal(next, done, e)
	}
	if _, _, _, e = s.SweepPage(ctx, time.Now(), next, false, 1); e != nil {
		t.Fatal("malformed cursor blocked next run", e)
	}
}
func TestPublicationCleansPrivateStage(t *testing.T) {
	ctx := context.Background()
	s := Store{Namespace: t.TempDir(), fault: func(string) error { return errors.New("fault") }}
	r := sample("x", time.Now())
	_ = s.Publish(ctx, r.Key, r, nil)
	ns, e := os.ReadDir(filepath.Join(s.Namespace, "schedule", "hub", "task"))
	if e != nil {
		t.Fatal(e)
	}
	for _, n := range ns {
		if strings.HasPrefix(n.Name(), ".run-stage-") {
			t.Fatal("private stage leaked")
		}
	}
}
func TestAbandonedPublicationStageRecovered(t *testing.T) {
	ctx := context.Background()
	s := Store{Namespace: t.TempDir()}
	r := sample("first", time.Now())
	if e := s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	task := filepath.Join(s.Namespace, "schedule", "hub", "task")
	stage, e := os.MkdirTemp(task, ".run-stage-")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(stage, "output"), []byte("unpublished"), 0600); e != nil {
		t.Fatal(e)
	}
	r.Key.ID = "second"
	if e = s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(stage); !os.IsNotExist(e) {
		t.Fatal("abandoned stage retained", e)
	}
}
func TestRecordJSONPreservesUnknown(t *testing.T) {
	r := sample("x", time.Now())
	r.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`true`)}
	r.Key.Unknown = map[string]json.RawMessage{"future_key": json.RawMessage(`2`)}
	raw, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	var back Record
	if e = json.Unmarshal(raw, &back); e != nil || len(back.Unknown) != 1 || len(back.Key.Unknown) != 1 {
		t.Fatal(string(raw), back, e)
	}
}
func TestPreparedCollectionRefusesSourceReplacement(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := Store{Namespace: t.TempDir()}
	r := sample("old", now.Add(-31*24*time.Hour))
	if e := s.Publish(ctx, r.Key, r, []byte("old")); e != nil {
		t.Fatal(e)
	}
	s.fault = func(step string) error {
		if step == "journal" {
			return errors.New("fault")
		}
		return nil
	}
	_, _, _, _ = s.SweepPage(ctx, now, Cursor{}, true, 100)
	s.fault = nil
	source := filepath.Join(s.Namespace, "schedule", "hub", "task", "runs", "old")
	if e := os.Rename(source, filepath.Join(s.Namespace, "original")); e != nil {
		t.Fatal(e)
	}
	r.EndedAt = now
	if e := s.Publish(ctx, r.Key, r, []byte("replacement")); e != nil {
		t.Fatal(e)
	}
	rows, _, _, e := s.SweepPage(ctx, now, Cursor{}, true, 100)
	if e != nil || len(rows) != 1 || rows[0].Eligible {
		t.Fatal(rows, e)
	}
	_, out, e := s.Read(ctx, r.Key)
	if e != nil || string(out) != "replacement" {
		t.Fatal(string(out), e)
	}
}
func TestJournalUnknownMembersSurviveRecovery(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := Store{Namespace: t.TempDir()}
	r := sample("old", now.Add(-31*24*time.Hour))
	if e := s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	s.fault = func(step string) error {
		if step == "journal" {
			return errors.New("fault")
		}
		return nil
	}
	_, _, _, _ = s.SweepPage(ctx, now, Cursor{}, true, 100)
	path := filepath.Join(s.Namespace, "schedule", "hub", "task", ".collection.json")
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	fields["future"] = json.RawMessage(`true`)
	raw, e = json.Marshal(fields)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	s.fault = func(step string) error {
		if step == "detached" {
			return errors.New("stop")
		}
		return nil
	}
	_, _, _, _ = s.SweepPage(ctx, now, Cursor{}, true, 100)
	raw, e = os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(raw), `"phase":"detached"`) || !strings.Contains(string(raw), `"future":true`) {
		t.Fatal(string(raw))
	}
}
func TestRecordRequiredAndNullFields(t *testing.T) {
	raw, e := encodeRecord(sample("x", time.Now()))
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"version", "key", "outcome", "started_at", "notice"} {
		t.Run(name, func(t *testing.T) {
			copy := map[string]json.RawMessage{}
			for k, v := range fields {
				copy[k] = v
			}
			copy[name] = json.RawMessage("null")
			bad, _ := json.Marshal(copy)
			var r Record
			if e := json.Unmarshal(bad, &r); e == nil {
				t.Fatal("null accepted", name)
			}
			delete(copy, name)
			bad, _ = json.Marshal(copy)
			if e := json.Unmarshal(bad, &r); e == nil {
				t.Fatal("missing accepted", name)
			}
		})
	}
}
func TestEmptyHubDiscoveryBounded(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	root := filepath.Join(s.Namespace, "schedule")
	for _, hub := range []string{"a", "b"} {
		if e := os.MkdirAll(filepath.Join(root, hub), 0700); e != nil {
			t.Fatal(e)
		}
	}
	_, next, done, e := s.SweepPage(context.Background(), time.Now(), Cursor{}, false, 1)
	if e != nil || done || next.Hub != "a" || !next.AfterHub {
		t.Fatal(next, done, e)
	}
}
func TestPublicationRefusesAtInventoryBound(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	ctx := context.Background()
	r := sample("first", time.Now())
	if e := s.Publish(ctx, r.Key, r, nil); e != nil {
		t.Fatal(e)
	}
	runs := filepath.Join(s.Namespace, "schedule", "hub", "task", "runs")
	for i := 1; i < 4096; i++ {
		if e := os.Mkdir(filepath.Join(runs, fmt.Sprintf("reserved-%04d", i)), 0700); e != nil {
			t.Fatal(e)
		}
	}
	r.Key.ID = "overflow"
	if e := s.Publish(ctx, r.Key, r, nil); e == nil {
		t.Fatal("published beyond readable inventory bound")
	}
	if _, e := os.Stat(filepath.Join(runs, "overflow")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
