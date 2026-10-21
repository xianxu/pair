package couchschedule

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestSummaryCapturesDeclarationRevision(t *testing.T) {
	s, due := taskFixture(t, false)
	skipped, _ := Advance(s, Event{Kind: ClockObserved, At: due, RunID: "skipped"})
	if skipped.Last == nil || skipped.Last.Revision != s.Revision {
		t.Fatal("passive result lost declaration revision")
	}
	running := phaseStateFixture(t, "scheduled", "running")
	done, _ := Advance(running, Event{Kind: TaskFinished, At: running.Active.StartedAt.Add(time.Second), Attempt: running.Active.ID, Revision: running.Active.Revision, Exit: 0})
	if done.Last == nil || done.Last.Revision != running.Active.Revision {
		t.Fatal("execution result lost attempt revision")
	}
}
func TestRecoveredSummaryCanonicalFactsWinWithoutReplay(t *testing.T) {
	for _, notice := range []string{"pending", "submitted", "dropped: offline", "uncertain", ""} {
		s := phaseStateFixture(t, "scheduled", "reserved")
		code := 10
		canonical := &RunSummary{ID: s.Active.ID, Revision: "canonical-revision", Source: "scheduled", DueAt: s.Active.DueAt.Add(-time.Hour), StartedAt: s.Active.StartedAt.Add(-time.Minute), EndedAt: s.Active.StartedAt.Add(-time.Second), Exit: &code, Outcome: "wake", Notice: notice, Unknown: map[string]json.RawMessage{"future": json.RawMessage(`{ "x" : 1 }`)}}
		next, effects := Advance(s, Event{Kind: ResultRecovered, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: canonical})
		if len(effects) != 0 || !reflect.DeepEqual(next.Last, canonical) {
			t.Fatalf("canonical %q facts not adopted: %+v", notice, next)
		}
		if notice == "pending" {
			if next.Active == nil || next.Active.Stage != "notifying" || next.Active.Revision != canonical.Revision || !next.Active.StartedAt.Equal(canonical.StartedAt) || !next.Active.DueAt.Equal(canonical.DueAt) || !next.Active.NoticeDeadline.Equal(canonical.EndedAt.Add(time.Minute)) {
				t.Fatal("pending authority not based on canonical facts")
			}
		} else if next.Active != nil {
			t.Fatal("terminal canonical result retained active authority")
		}
		*next.Last.Exit = 3
		next.Last.Unknown["future"][0] = '['
		next.Last.Unknown["added"] = json.RawMessage(`1`)
		if *canonical.Exit != 10 || string(canonical.Unknown["future"]) != `{ "x" : 1 }` || canonical.Unknown["added"] != nil {
			t.Fatal("canonical payload aliased adopted state")
		}
	}
}
func TestRecoveredUnknownRetainsConservativeAuthority(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "running")
	canonical := &RunSummary{ID: s.Active.ID, Revision: "old-revision", Source: "manual", StartedAt: s.Active.StartedAt.Add(-time.Second), EndedAt: s.Active.StartedAt, Outcome: "unknown", Notice: "owner unavailable"}
	next, effects := Advance(s, Event{Kind: ResultRecovered, At: s.Active.StartedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: canonical})
	if len(effects) != 0 || !reflect.DeepEqual(next.Last, canonical) || next.Active == nil || next.Active.Stage != "uncertain" || next.Active.Source != "manual" || next.Active.Revision != canonical.Revision {
		t.Fatal("unknown canonical result lost authority")
	}
}
func TestRecoveredSummaryRefusesMalformedDomainFacts(t *testing.T) {
	for _, mutate := range []func(*RunSummary){
		func(r *RunSummary) { r.ID = "other" }, func(r *RunSummary) { r.Revision = "" }, func(r *RunSummary) { r.Source = "alien" }, func(r *RunSummary) { r.Outcome = "alien" }, func(r *RunSummary) { r.Notice = "alien" }, func(r *RunSummary) { r.Source = "manual"; r.Notice = "pending" }, func(r *RunSummary) { r.Exit = new(int); *r.Exit = -1 },
	} {
		s := phaseStateFixture(t, "scheduled", "reserved")
		canonical := &RunSummary{ID: s.Active.ID, Revision: "canonical", Source: "scheduled", StartedAt: s.Active.StartedAt, EndedAt: s.Active.StartedAt.Add(time.Second), Outcome: "wake", Notice: "submitted"}
		mutate(canonical)
		next, effects := Advance(s, Event{Kind: ResultRecovered, At: s.Active.StartedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: canonical})
		if len(effects) != 0 || !reflect.DeepEqual(next, s) {
			t.Fatal("malformed canonical result advanced authority")
		}
	}
}
func TestSummaryRevisionCodecLegacyAndAdditive(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "uncertain")
	s.Last.Revision = "canonical-revision"
	raw := phaseWire(t, s)
	var top map[string]json.RawMessage
	json.Unmarshal(raw, &top)
	var tasks map[string]json.RawMessage
	json.Unmarshal(top["tasks"], &tasks)
	var task map[string]json.RawMessage
	json.Unmarshal(tasks[s.Key.ID()], &task)
	var last map[string]json.RawMessage
	json.Unmarshal(task["last"], &last)
	delete(last, "revision")
	last["future"] = json.RawMessage(`{"v":1}`)
	task["last"], _ = json.Marshal(last)
	tasks[s.Key.ID()], _ = json.Marshal(task)
	top["tasks"], _ = json.Marshal(tasks)
	raw, _ = json.Marshal(top)
	ledger, err := DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[s.Key.ID()]
	if got.Last.Revision != "" || string(got.Last.Unknown["future"]) != `{"v":1}` {
		t.Fatal("legacy or unknown summary facts lost")
	}
	got.Last.Revision = "restored"
	ledger.Tasks[s.Key.ID()] = got
	raw, err = EncodeLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err = DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Tasks[s.Key.ID()].Last.Revision != "restored" {
		t.Fatal("revision did not round trip")
	}
}

func TestRecoveredPassiveLastCanonicalAdoption(t *testing.T) {
	for _, notice := range []string{"none", "manual", "pending", "uncertain"} {
		s, due := taskFixture(t, false)
		s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "passive"})
		canonical := &RunSummary{ID: s.Last.ID, Revision: "canonical", Source: "scheduled", DueAt: due.Add(-time.Minute), StartedAt: due.Add(-time.Second), EndedAt: due, Outcome: "skipped", Notice: notice}
		next, effects := Advance(s, Event{Kind: ResultRecovered, At: due, Recovered: canonical})
		if notice == "pending" {
			if !reflect.DeepEqual(next, s) || len(effects) != 0 {
				t.Fatal("passive pending invented authority")
			}
			continue
		}
		if next.Active != nil || len(effects) != 0 || !reflect.DeepEqual(next.Last, canonical) {
			t.Fatalf("passive canonical %s not adopted", notice)
		}
	}
	s, due := taskFixture(t, false)
	s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "passive"})
	for _, canonical := range []*RunSummary{
		{ID: s.Last.ID, Revision: "canonical", Source: "scheduled", StartedAt: due, EndedAt: due, Outcome: "unknown", Notice: "none"},
		{ID: "different", Revision: "canonical", Source: "scheduled", StartedAt: due, EndedAt: due, Outcome: "quiet", Notice: "none"},
	} {
		next, effects := Advance(s, Event{Kind: ResultRecovered, At: due, Recovered: canonical})
		if !reflect.DeepEqual(next, s) || len(effects) != 0 {
			t.Fatal("unowned or unknown passive record adopted")
		}
	}
}
func TestRecoveredCanonicalOperationalNoticeValues(t *testing.T) {
	for _, source := range []string{"manual", "scheduled"} {
		for _, notice := range []string{"manual", "none"} {
			s := phaseStateFixture(t, source, "running")
			canonical := &RunSummary{ID: s.Active.ID, Revision: s.Active.Revision, Source: source, DueAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: s.Active.StartedAt.Add(time.Second), Outcome: "quiet", Notice: notice}
			next, effects := Advance(s, Event{Kind: ResultRecovered, At: canonical.EndedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: canonical})
			if next.Active != nil || len(effects) != 0 || !reflect.DeepEqual(next.Last, canonical) {
				t.Fatalf("real operational notice value %s/%s refused", source, notice)
			}
		}
	}
}
func TestRecoveredPendingNoticeDeadlinePreservesAdmissionBounds(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, existing := range []time.Duration{0, 10 * time.Second} {
			s := phaseStateFixture(t, "scheduled", "reserved")
			s.Spec.ScriptRun = direct
			start := s.Active.StartedAt
			s.Active.Deadline = start.Add(20 * time.Second)
			if existing != 0 {
				s.Active.Stage = "notifying"
				s.Active.NoticeDeadline = start.Add(existing)
				s.Active.NoticeMayHaveHappened = true
				s.Last = summary(s.Active, start, nil, "completed", "pending")
			}
			canonical := &RunSummary{ID: s.Active.ID, Revision: s.Active.Revision, Source: "scheduled", DueAt: s.Active.DueAt, StartedAt: start, EndedAt: start.Add(5 * time.Second), Outcome: "completed", Notice: "pending"}
			next, effects := Advance(s, Event{Kind: ResultRecovered, At: start.Add(6 * time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, Recovered: canonical})
			want := canonical.EndedAt.Add(time.Minute)
			if !direct {
				want = s.Active.Deadline
				if existing != 0 {
					want = start.Add(existing)
				}
			}
			if len(effects) != 0 || next.Active == nil || !next.Active.NoticeDeadline.Equal(want) {
				t.Fatalf("direct=%v existing=%s wanted %s, got %+v", direct, existing, want, next.Active)
			}
		}
	}
}
