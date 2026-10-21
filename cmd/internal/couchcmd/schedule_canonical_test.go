package couchcmd

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleCachedCanonicalRecoveryAfterFailedCommit(t *testing.T) {
	for _, outcome := range []string{"skipped", "refused", "completed"} {
		t.Run(outcome, func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			c.roots = roots[:1]
			a := newAutomaticScheduler(c, nil, offlineScheduleNotices{})
			*now = now.Add(time.Minute)
			key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
			var state couchschedule.TaskState
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				state, _ = couchschedule.Advance(l.Tasks[key.ID()], couchschedule.Event{Kind: couchschedule.ClockObserved, At: *now, RunID: "canonical-passive", Location: time.UTC})
				state.Last.Outcome = outcome
				state.Last.Notice = "none"
				l.Tasks[key.ID()] = state
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			record := runRecordForSummary(state)
			record.StartedAt = record.StartedAt.Add(-20 * time.Second)
			record.EndedAt = record.EndedAt.Add(-10 * time.Second)
			if err := a.history().Publish(t.Context(), record.Key, record, nil); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("authority commit failed after canonical read")
			c.store.BeforePersist = func() error { return failure }
			if err := a.step(t.Context()); !errors.Is(err, failure) {
				t.Fatalf("missing commit failure: %v", err)
			}
			if proof, ok := a.passiveDurable[key.ID()]; !ok || !proof.EndedAt.Equal(record.EndedAt) {
				t.Fatal("canonical proof not retained")
			}
			c.store.BeforePersist = nil
			*now = now.Add(time.Second)
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			ledger, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := ledger.Tasks[key.ID()]
			canonical := runSummaryForRecord(record)
			known := *got.Last
			known.Unknown = nil
			if !reflect.DeepEqual(&known, &canonical) {
				t.Fatalf("cached recovery differs from history: ledger=%+v history=%+v", got.Last, canonical)
			}
			a.passiveDurable = map[string]couchschedule.RunSummary{}
			direct, err := a.ensurePassive(t.Context(), state)
			if err != nil {
				t.Fatal(err)
			}
			directKnown := *direct.Last
			directKnown.Unknown = nil
			if !reflect.DeepEqual(&directKnown, &known) {
				t.Fatalf("direct/cache mismatch: direct=%+v cached=%+v err=%v", direct.Last, got.Last, err)
			}
			stored, output, err := a.history().Read(t.Context(), record.Key)
			storedJSON, _ := json.Marshal(stored)
			recordJSON, _ := json.Marshal(record)
			if err != nil || string(storedJSON) != string(recordJSON) || len(output) != 0 {
				t.Fatalf("recovery rewrote history: %+v %q %v", stored, output, err)
			}
			if len(a.jobs) != 0 || got.Active != nil || !got.NextDue.Equal(state.NextDue) {
				t.Fatal("recovery re-admitted work or moved due")
			}
		})
	}
}
