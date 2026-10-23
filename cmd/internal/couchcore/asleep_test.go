package couchcore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestThreadAsleepIsTheMarkOfTheParkInForce(t *testing.T) {
	at := time.Date(2026, 10, 23, 1, 2, 3, 0, time.UTC)
	mark := &threadrecord.SleepMark{Nonce: "n1", At: at, Cause: threadrecord.SleepFullQuit}
	receipt := func(nonce string) *VerifiedPark { return &VerifiedPark{Identity: ParkIdentity{Nonce: nonce}} }
	for _, tc := range []struct {
		name   string
		record ThreadRecord
		want   bool
	}{
		{"no mark", ThreadRecord{VerifiedPark: receipt("n1")}, false},
		{"mark without receipt (rolled back, or resumed)", ThreadRecord{Asleep: mark}, false},
		{"mark of another park (re-parked individually, or kept by an older build)", ThreadRecord{Asleep: mark, VerifiedPark: receipt("n2")}, false},
		{"mark while a park is open", ThreadRecord{Asleep: mark, Park: &ParkTransaction{}}, false},
		{"mark of the receipt in force", ThreadRecord{Asleep: mark, VerifiedPark: receipt("n1")}, true},
	} {
		got, ok := ThreadAsleep(tc.record)
		if ok != tc.want || ok && !got.Equal(at) {
			t.Errorf("%s: asleep=%v at=%v, want %v", tc.name, ok, got, tc.want)
		}
	}
}

// The mark is a top-level member and born additive: a newer build's member
// inside it survives decode and rewrite, and a bad mark refuses the record.
func TestSleepMarkPersistsAdditively(t *testing.T) {
	record := validThreadRecord(t)
	record.Asleep = &threadrecord.SleepMark{Nonce: "n1", At: time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC), Cause: "drain"}
	raw, err := threadrecord.EncodePersisted(toPersistedThreadRecord(record))
	if err != nil {
		t.Fatal(err)
	}
	newer := bytes.Replace(raw, []byte(`"cause": "drain"`), []byte(`"cause": "drain", "future": {"x": 1}`), 1)
	if bytes.Equal(newer, raw) {
		t.Fatalf("fixture did not inject a member:\n%s", raw)
	}
	address := toPersistedThreadAddress(record.Address)
	decoded, err := threadrecord.DecodePersisted(newer, address, threadRecordValidators)
	if err != nil {
		t.Fatalf("newer mark refused: %v", err)
	}
	again, err := threadrecord.EncodePersisted(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(strings.Fields(string(again)), ""), `"future":{"x":1}`) {
		t.Fatalf("rewrite dropped the newer member:\n%s", again)
	}
	for _, bad := range []string{`"nonce": "n1"`, `"cause": "drain"`} {
		broken := bytes.Replace(raw, []byte(bad), []byte(strings.SplitN(bad, ":", 2)[0]+`: ""`), 1)
		if _, err := threadrecord.DecodePersisted(broken, address, threadRecordValidators); err == nil {
			t.Errorf("empty %s accepted", bad)
		}
	}
}

// A full quit marks exactly the parks it begins: a slot already parked stays
// plain, a rolled-back park's retry carries a fresh mark on its new nonce, and
// retiring the receipt clears the mark.
func TestLeaveParkMarksTheParksItBegins(t *testing.T) {
	withParallelism(t, 2)
	f := newLeaveParkFleet(t, 4)
	ctx := context.Background()
	individual := f.threads[0]
	if _, err := f.couch.PairLifecycle.Park(ctx, individual.Address); err != nil {
		t.Fatalf("individual park: %v", err)
	}
	f.lifecycle.failPublish[f.sessions[2]] = 1
	if _, err := f.couch.Leave(ctx, LeavePark); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	for i, thread := range f.threads {
		after, err := f.couch.Threads.GetThread(thread.Address)
		if err != nil {
			t.Fatal(err)
		}
		at, asleep := ThreadAsleep(after)
		if want := i != 0; asleep != want {
			t.Fatalf("%s asleep=%v want %v: %+v", thread.Address.Tag, asleep, want, after.Asleep)
		}
		if asleep && (at.IsZero() || after.Asleep.Cause != threadrecord.SleepFullQuit) {
			t.Fatalf("%s mark=%+v", thread.Address.Tag, after.Asleep)
		}
	}
	retried, _ := f.couch.Threads.GetThread(f.threads[2].Address)
	if len(retried.ParkHistory) != 2 || retried.Asleep.Nonce == retried.ParkHistory[0].Identity.Nonce {
		t.Fatalf("retry kept the rolled-back park's mark: %+v", retried.Asleep)
	}
	cleared, err := f.couch.Threads.ClearVerifiedPark(retried.Address, retried.Revision)
	if err != nil || cleared.Asleep != nil {
		t.Fatalf("reattach kept the mark: %+v %v", cleared.Asleep, err)
	}
}

// An interrupted full quit keeps its mark through Recover (same nonce); an
// individual park left open and finished by the quit's Recover stays plain.
func TestLeaveParkRecoverKeepsWhatTheBeginWrote(t *testing.T) {
	for _, individual := range []bool{false, true} {
		t.Run(map[bool]string{false: "full-quit", true: "individual"}[individual], func(t *testing.T) {
			withParallelism(t, 1)
			f := newLeaveParkFleet(t, 1)
			thread := f.threads[0]
			f.lifecycle.failPublish[f.sessions[0]] = 1
			refuse := errors.New("rollback write refused")
			refusals := 0
			f.couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
				if old.Park != nil && next.Park == nil && next.VerifiedPark == nil && refusals == 0 {
					refusals++
					return refuse
				}
				return nil
			}
			if individual {
				if _, err := f.couch.PairLifecycle.Park(context.Background(), thread.Address); err == nil {
					t.Fatal("individual park succeeded; want it left open")
				}
				open, _ := f.couch.Threads.GetThread(thread.Address)
				if open.Park == nil || open.Asleep != nil {
					t.Fatalf("individual park not left open and plain: %+v", open)
				}
			}
			if _, err := f.couch.Leave(context.Background(), LeavePark); err != nil {
				t.Fatalf("Leave: %v", err)
			}
			after, _ := f.couch.Threads.GetThread(thread.Address)
			if _, asleep := ThreadAsleep(after); asleep == individual {
				t.Fatalf("asleep=%v after recover, individual=%v: %+v", asleep, individual, after.Asleep)
			}
		})
	}
}

// Wake's targets are the slot rows a full quit put to sleep: a plain parked
// slot and an asleep thread outside any slot are not among them.
func TestAsleepSlotRowsAreTheSleepingSlots(t *testing.T) {
	at := time.Date(2026, 10, 23, 1, 2, 3, 0, time.UTC)
	record := func(tag ThreadTag, mark bool) ThreadRecord {
		r := validThreadRecord(t)
		r.Address.Tag = tag
		r.VerifiedPark = &VerifiedPark{Identity: ParkIdentity{Nonce: string(tag)}}
		if mark {
			r.Asleep = &threadrecord.SleepMark{Nonce: string(tag), At: at, Cause: threadrecord.SleepFullQuit}
		}
		return r
	}
	asleep, plain, loose := record("asleep", true), record("plain", false), record("loose", true)
	input := ThreadProjectionInput{
		Records: []ThreadRecord{asleep, plain, loose},
		Slots: []SlotInventoryObservation{
			{Identity: allocationCandidate(1).Identity, Address: asleep.Address},
			{Identity: allocationCandidate(2).Identity, Address: plain.Address},
		},
	}
	rows := asleepSlotRows(input)
	if len(rows) != 1 || rows[0].Address != asleep.Address || !rows[0].AsleepAt.Equal(at) || rows[0].Target.Slot.Number != 1 {
		t.Fatalf("asleep slots = %+v", rows)
	}
}

// Only the main hub wakes, and only another hub's own slot.
func TestOnlyTheMainHubWakesTheOtherHubs(t *testing.T) {
	state, err := NewHubOwnership(hubFixture(t, "", "pair\n", ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		caller, target, refusal string
	}{
		{"/work/ops", "/work/bench", ""},
		{"/work/ops", "/work/other", ""},
		{"/work/bench", "/work/other", "only the main hub"},
		{"", "/work/bench", "only the main hub"},
		{"/work/ops", "/work/ops", "already awake"},
		{"/work/ops", "/work/pair-slot1", "not a hub"},
	} {
		err := state.Wakes(tc.caller, tc.target)
		if tc.refusal == "" && err != nil || tc.refusal != "" && (err == nil || !strings.Contains(err.Error(), tc.refusal)) {
			t.Errorf("Wakes(%q, %q) = %v, want %q", tc.caller, tc.target, err, tc.refusal)
		}
	}
}
