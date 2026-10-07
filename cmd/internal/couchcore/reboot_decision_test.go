package couchcore

import "testing"

// expectedRebootPlan is the Spec's reboot table restated as an independent
// literal, so DecideReboot is checked against the decision the operator made
// and not against a copy of its own switch.
func expectedRebootPlan(slot bool, record RebootRecord, dir bool) RebootPlan {
	switch record {
	case RebootRecordReadable:
		if dir {
			return RebootArchiveAndStart
		}
		return RebootArchiveOnly
	case RebootRecordUnreadable:
		if !slot {
			return RebootArchiveOnly
		}
		if dir {
			return RebootArchiveAndStart
		}
		return RebootRefuse
	case RebootRecordNone:
		if slot && dir {
			return RebootStartOnly
		}
		return RebootRefuse
	case RebootRecordRolledBack:
		if dir {
			return RebootStartOnly
		}
		return RebootRefuse
	}
	return 0
}

func TestDecideRebootCoversEveryFactCombination(t *testing.T) {
	for _, slot := range []bool{false, true} {
		for _, record := range []RebootRecord{RebootRecordNone, RebootRecordReadable, RebootRecordUnreadable, RebootRecordRolledBack} {
			for _, dir := range []bool{false, true} {
				plan, reason := DecideReboot(RebootFacts{Slot: slot, Record: record, DirectoryPresent: dir})
				want := expectedRebootPlan(slot, record, dir)
				if plan != want {
					t.Errorf("slot=%v record=%v dir=%v: plan %v, want %v", slot, record, dir, plan, want)
				}
				if needsReason := plan == RebootArchiveOnly || plan == RebootRefuse; needsReason != (reason != "") {
					t.Errorf("slot=%v record=%v dir=%v: reason %q for plan %v", slot, record, dir, reason, plan)
				}
			}
		}
	}
}

// The reasons are what the operator reads on an archive-only or refused
// reboot, so the ones the Spec words are pinned here.
func TestDecideRebootReasons(t *testing.T) {
	cases := []struct {
		facts RebootFacts
		want  string
	}{
		// A :0's next step is its checkout, never add slot: add slot makes
		// :1+ slots and a :0 row never offers it there (pair#363 M2 review).
		// A :1+ slot reaches the missing reason only when reconcile could
		// not restore its directory (pair#387).
		{RebootFacts{Slot: false, Record: RebootRecordReadable}, RebootCheckoutMissing},
		{RebootFacts{Slot: false, Record: RebootRecordRolledBack}, RebootCheckoutMissing},
		{RebootFacts{Slot: true, Record: RebootRecordReadable}, RebootDirectoryMissing},
		{RebootFacts{Slot: false, Record: RebootRecordUnreadable, DirectoryPresent: true}, "record unreadable — start couch in its directory for a fresh agent"},
		{RebootFacts{Slot: true, Record: RebootRecordUnreadable}, RebootDirectoryMissing},
		{RebootFacts{Slot: false, Record: RebootRecordNone, DirectoryPresent: true}, "nothing to reboot"},
		{RebootFacts{Slot: true, Record: RebootRecordRolledBack}, RebootDirectoryMissing},
	}
	for _, tc := range cases {
		if _, got := DecideReboot(tc.facts); got != tc.want {
			t.Errorf("%+v: reason %q, want %q", tc.facts, got, tc.want)
		}
	}
}

// Reboot's admission is archive's: it runs the same retirement, so it admits
// exactly the rows nothing couch is acting on -- parked, detached and debris --
// and refuses `unusable/unknown`, which is ignorance and not a verdict, and
// `unusable/orphaned-server`, whose conversation is still running (#399).
func TestRebootableStateOverEveryClassification(t *testing.T) {
	admitted := 0
	for _, state := range AllThreadStates() {
		for _, reason := range append(AllThreadReasons(), "") {
			if (state == ThreadUnusable) != (reason != "") {
				continue
			}
			want := state == ThreadParked || state == ThreadDetached ||
				(state == ThreadUnusable && reason != ReasonUnknown && reason != ReasonOrphanedServer)
			if got := RebootableState(state, reason); got != want {
				t.Errorf("%s/%s: RebootableState=%v, want %v", state, reason, got, want)
			}
			if want {
				admitted++
			}
		}
	}
	if admitted == 0 {
		t.Fatal("nothing is admitted, so this table proves nothing")
	}
}
