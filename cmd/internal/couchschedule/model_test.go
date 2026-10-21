package couchschedule

import (
	"bytes"
	"encoding/json"
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

func TestAdvanceInvalidOutputRollsBackWithoutMutatingInput(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "running")
	s.Last = &RunSummary{ID: "previous", Source: "scheduled", Outcome: "quiet"}
	before, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	next, effects := Advance(s, Event{Kind: TaskFinished, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, Exit: 10, Outcome: "invalid-outcome", HubOnline: true})
	after, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !reflect.DeepEqual(next, s) || len(effects) != 0 {
		t.Fatalf("invalid transition mutated or advanced input: before=%s after=%s effects=%v", before, after, effects)
	}
}

func TestAdvanceChangedDeclarationKeepsOldExecutionPhaseLegal(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "running")
	replacement := Declaration{Key: s.Key, Revision: "new-revision", Spec: s.Spec, Allowed: true}
	replacement.Spec.ScriptRun = false
	next, effects := Advance(s, Event{Kind: CatalogChanged, At: s.Active.StartedAt.Add(time.Second), Declaration: &replacement})
	if len(effects) != 1 || effects[0].Kind != StopTask || next.Active == nil || next.Active.Stage != "stopping" || next.Active.Revision == next.Revision {
		t.Fatalf("old execution not retained for cleanup: %+v %v", next, effects)
	}
	if _, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{next.Key.ID(): next}}); err != nil {
		t.Fatalf("valid cross-revision cleanup refused: %v", err)
	}
}

func TestAdvanceNoticeCancellationTerminatesPendingCompanion(t *testing.T) {
	for _, kind := range []EventKind{CancelRequested, CatalogChanged} {
		s := phaseStateFixture(t, "scheduled", "notifying")
		next, effects := Advance(s, Event{Kind: kind, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision})
		if next.Active != nil || next.Last == nil || next.Last.Notice != "dropped: task cancelled" || len(effects) != 0 {
			t.Fatalf("cancellation left dangling notice: %+v %v", next, effects)
		}
	}
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
				s = freezeForTest(s, due.Add(time.Second))
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

func TestAdvanceRemovedTaskDisarmsAndSameRevisionRestorationRearms(t *testing.T) {
	s, due := taskFixture(t, true)
	declaration := Declaration{Key: s.Key, Revision: s.Revision, Spec: s.Spec, Allowed: true}
	s, _ = Advance(s, Event{Kind: CatalogChanged, At: due})
	if !s.NextDue.IsZero() || !s.RemovedAt.Equal(due) || s.Allowed {
		t.Fatalf("removed task remains armed: %+v", s)
	}
	s, _ = Advance(s, Event{Kind: CatalogChanged, At: due.Add(time.Minute)})
	if !s.RemovedAt.Equal(due) {
		t.Fatal("catalog scans extended removal retention")
	}
	s, _ = Advance(s, Event{Kind: CatalogChanged, At: due.Add(2 * time.Minute), Declaration: &declaration})
	if !s.NextDue.After(due.Add(2*time.Minute)) || !s.RemovedAt.IsZero() || !s.Allowed {
		t.Fatalf("restored task did not rearm: %+v", s)
	}
}

func TestAdvanceMissingNativeExitIsNullable(t *testing.T) {
	s, due := taskFixture(t, true)
	s, _ = Advance(s, Event{Kind: ManualRequested, At: due, RunID: "manual"})
	s, _ = Advance(s, Event{Kind: ReservationDurable, At: due, Attempt: "manual", Revision: "r1"})
	s, _ = Advance(s, Event{Kind: TaskFinished, At: due.Add(time.Second), Attempt: "manual", Revision: "r1", Exit: -1, Outcome: "cancelled"})
	if s.Last == nil || s.Last.Exit != nil {
		t.Fatalf("no native exit was invented as a number: %+v", s.Last)
	}
}

func TestAdvanceMalformedNoticeStateCannotPanicOrEmit(t *testing.T) {
	for _, stage := range []string{"notifying", "delivery"} {
		for _, kind := range []EventKind{OwnerUnknown, NoticeIntentDurable, NoticeObserved, CancelRequested, CatalogChanged} {
			t.Run(stage+"/"+string(kind), func(t *testing.T) {
				s := phaseStateFixture(t, "manual", stage)
				s.Last = nil
				next, effects := Advance(s, Event{Kind: kind, At: s.Active.StartedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, HubOnline: true, Notice: "submitted"})
				if !reflect.DeepEqual(s, next) || len(effects) != 0 {
					t.Fatalf("malformed authority advanced: %+v %v", next, effects)
				}
			})
		}
	}
}

func TestAdvanceOwnsExecutionIdentityBeforeAuthorization(t *testing.T) {
	s, due := taskFixture(t, true)
	s, _ = Advance(s, Event{Kind: ManualRequested, At: due, RunID: "manual-owner"})
	before := s
	owner := OwnerIdentity{PID: 42, Start: "exact-start"}
	recorded, effects := Advance(s, Event{Kind: OwnerRecorded, At: due, Attempt: s.Active.ID, Revision: s.Active.Revision, Owner: &owner})
	if len(effects) != 0 || recorded.Active.ControllerPID != 42 || recorded.Active.ControllerStart != "exact-start" || before.Active.ControllerPID != 0 {
		t.Fatalf("owner not transition-owned/pure: %+v %v", recorded.Active, effects)
	}
	running, effects := Advance(recorded, Event{Kind: ReservationDurable, At: due, Attempt: s.Active.ID, Revision: s.Active.Revision})
	if len(effects) != 1 || effects[0].Kind != StartTask || running.Active.ControllerPID != 42 {
		t.Fatalf("authorization lost owner: %+v %v", running.Active, effects)
	}
	for _, tc := range []struct {
		name              string
		state             TaskState
		attempt, revision string
		owner             *OwnerIdentity
	}{
		{"missing owner", s, s.Active.ID, s.Active.Revision, nil},
		{"bad PID", s, s.Active.ID, s.Active.Revision, &OwnerIdentity{PID: -1, Start: "token"}},
		{"missing token", s, s.Active.ID, s.Active.Revision, &OwnerIdentity{PID: 42}},
		{"stale attempt", s, "foreign", s.Active.Revision, &owner},
		{"stale revision", s, s.Active.ID, "foreign", &owner},
		{"replaced identity", recorded, s.Active.ID, s.Active.Revision, &OwnerIdentity{PID: 43, Start: "other"}},
		{"already running", running, s.Active.ID, s.Active.Revision, &OwnerIdentity{PID: 43, Start: "other"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, effects := Advance(tc.state, Event{Kind: OwnerRecorded, At: due, Attempt: tc.attempt, Revision: tc.revision, Owner: tc.owner})
			if !reflect.DeepEqual(next, tc.state) || len(effects) != 0 {
				t.Fatalf("invalid identity observation advanced: %+v %v", next, effects)
			}
		})
	}
}
