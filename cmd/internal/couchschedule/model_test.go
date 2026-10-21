package couchschedule

import (
	"reflect"
	"testing"
	"time"
)

func taskFixture(t *testing.T, direct bool) (TaskState, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	spec := TaskSpec{Version: 1, Cron: "* * * * *", Timeout: time.Minute, Scope: "machine", Missed: "skip", ScriptRun: direct}
	d := Declaration{Key: TaskKey{Hub: "/ops", Task: "report"}, Revision: "r1", Spec: spec, Allowed: true}
	s, effects := Advance(TaskState{}, Event{Kind: CatalogChanged, At: now, Declaration: &d})
	if len(effects) != 0 || !s.NextDue.Equal(now.Add(time.Minute)) {
		t.Fatalf("first discovery must only arm future due: %+v %v", s, effects)
	}
	return s, now.Add(time.Minute)
}

func TestAdvanceOfflineNotifySkipsButDirectReserves(t *testing.T) {
	for _, direct := range []bool{false, true} {
		s, due := taskFixture(t, direct)
		next, effects := Advance(s, Event{Kind: ClockObserved, At: due, RunID: "run-1"})
		if !next.NextDue.After(due) || len(effects) != 0 {
			t.Fatalf("must durably consume due before any effect: %+v %v", next, effects)
		}
		if direct {
			if next.Active == nil || next.Active.Stage != "reserved" {
				t.Fatalf("offline script must reserve: %+v", next)
			}
			running, starts := Advance(next, Event{Kind: ReservationDurable, At: due, Attempt: "run-1", Revision: "r1"})
			if len(starts) != 1 || starts[0].Kind != StartTask || running.Active.Stage != "running" {
				t.Fatalf("durable direct reservation starts despite offline hub: %+v %v", running, starts)
			}
		} else if next.Active != nil || next.Last == nil || next.Last.Outcome != "skipped" || next.Last.Notice != "hub offline, skipped" {
			t.Fatalf("notify only must skip offline: %+v", next)
		}
	}
}

func TestAdvanceDirectExitSemanticsAndNoReplay(t *testing.T) {
	for _, code := range []int{0, 10, 2} {
		for _, online := range []bool{false, true} {
			s, due := taskFixture(t, true)
			s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "run-1", HubOnline: online})
			s, _ = Advance(s, Event{Kind: ReservationDurable, At: due, Attempt: "run-1", Revision: "r1", HubOnline: online})
			before := s
			s, effects := Advance(s, Event{Kind: TaskFinished, At: due.Add(time.Second), Attempt: "run-1", Revision: "r1", Exit: code, HubOnline: online})
			want := "failed"
			if code == 0 {
				want = "quiet"
			}
			if code == 10 {
				want = "wake"
			}
			if s.Last == nil || s.Last.Outcome != want {
				t.Fatalf("exit %d: %+v", code, s)
			}
			if !reflect.DeepEqual(before.Active.Stage, "running") {
				t.Fatal("pure reducer mutated its input")
			}
			if code != 0 && online {
				if len(effects) != 0 || s.Active == nil || s.Active.Stage != "notifying" {
					t.Fatalf("notice must first persist intent: %v %+v", effects, s)
				}
				s, effects = Advance(s, Event{Kind: NoticeIntentDurable, At: due.Add(time.Second), Attempt: "run-1", Revision: "r1", HubOnline: true})
				if len(effects) != 1 || effects[0].Kind != SendNotice {
					t.Fatalf("want one notice: %v", effects)
				}
			} else if len(effects) != 0 || s.Active != nil {
				t.Fatalf("quiet/offline completion must finish: %+v %v", s, effects)
			}
			if code != 0 && !online && s.Last.Notice != "dropped: hub offline" {
				t.Fatalf("missing dropped status: %+v", s.Last)
			}
			duplicate, replay := Advance(s, Event{Kind: TaskFinished, At: due.Add(2 * time.Second), Attempt: "run-1", Revision: "r1", Exit: code, HubOnline: true})
			if !reflect.DeepEqual(duplicate, s) || len(replay) != 0 {
				t.Fatalf("duplicate completion replayed effects: %v", replay)
			}
		}
	}
}

func TestAdvanceMissedSlotsCoalesceAndOwnerRefuses(t *testing.T) {
	for _, once := range []bool{false, true} {
		s, due := taskFixture(t, true)
		if once {
			s.Spec.Missed = "once"
		}
		now := due.Add(1000 * time.Minute)
		next, effects := Advance(s, Event{Kind: ClockObserved, At: now, RunID: "catchup"})
		if len(effects) != 0 || !next.NextDue.After(now) {
			t.Fatalf("catchup must reserve once/advance directly: %+v %v", next, effects)
		}
		if (next.Active != nil) != once {
			t.Fatalf("missed=%v active=%+v", once, next.Active)
		}
	}
	s, due := taskFixture(t, true)
	s.Allowed = false
	s.Refusal = "owner absent from Operators"
	s, effects := Advance(s, Event{Kind: ClockObserved, At: due, RunID: "refused", HubOnline: true})
	if s.Active != nil || len(effects) != 0 || s.Last.Outcome != "refused" || !s.NextDue.After(due) {
		t.Fatalf("owner refusal: %+v %v", s, effects)
	}
}

func TestAdvanceManualAndStaleEventsCannotConsumeSchedule(t *testing.T) {
	s, due := taskFixture(t, true)
	s, effects := Advance(s, Event{Kind: ManualRequested, At: due.Add(-time.Second), RunID: "manual"})
	if len(effects) != 0 || s.Active == nil || s.Active.Source != "manual" || !s.NextDue.Equal(due) {
		t.Fatalf("manual reservation: %+v %v", s, effects)
	}
	old := s
	s, effects = Advance(s, Event{Kind: ReservationDurable, At: due, Attempt: "foreign", Revision: "r1"})
	if !reflect.DeepEqual(old, s) || len(effects) != 0 {
		t.Fatal("foreign attempt mutated state")
	}
	s, effects = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "overlap", HubOnline: true})
	if s.Active.ID != "manual" || !s.NextDue.After(due) || len(effects) != 0 {
		t.Fatalf("overlap: %+v %v", s, effects)
	}
	other := TaskKey{Hub: "/other-ops", Task: s.Key.Task}
	if other.ID() == s.Key.ID() || other.HubID() == s.Key.HubID() {
		t.Fatal("two hubs share task identity")
	}
}

func TestAdvanceRecoveryUnknownNeverRetries(t *testing.T) {
	s, due := taskFixture(t, true)
	s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "run-1"})
	s, _ = Advance(s, Event{Kind: ReservationDurable, At: due, Attempt: "run-1", Revision: "r1"})
	s, effects := Advance(s, Event{Kind: OwnerUnknown, At: due.Add(time.Second), Attempt: "run-1", Revision: "r1"})
	if len(effects) != 0 || s.Active == nil || s.Active.Stage != "uncertain" || s.Last.Outcome != "unknown" {
		t.Fatalf("uncertain must retain blocking authority: %+v %v", s, effects)
	}
	s, effects = Advance(s, Event{Kind: ClockObserved, At: due.Add(time.Minute), RunID: "never-retry"})
	if s.Active.ID != "run-1" || len(effects) != 0 {
		t.Fatalf("unknown replayed: %+v %v", s, effects)
	}
}

func TestAdvanceCalendarWorkIsLinearAndIndependentOfMissedSlots(t *testing.T) {
	for _, n := range []int{8, 16} {
		calls := 0
		for i := 0; i < n; i++ {
			s, due := taskFixture(t, true)
			now := due.Add(1000 * time.Minute)
			_, effects := Advance(s, Event{Kind: ClockObserved, At: now, RunID: "missed", Next: func(at time.Time) time.Time { calls++; return at.Add(time.Minute) }})
			if len(effects) != 0 {
				t.Fatal("default missed slot emitted work")
			}
		}
		if calls != n {
			t.Fatalf("%d tasks caused %d recurrence evaluations", n, calls)
		}
	}
}
