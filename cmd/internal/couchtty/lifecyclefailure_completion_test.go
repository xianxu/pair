package couchtty

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestLifecycleFailureMenuAdoptionUsesTerminalOwner(t *testing.T) {
	r := newRemoteRig(t)
	store := couchcore.NewFakeLaunchFailureStore()
	n := 0
	recorder := couchcore.NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) { n++; return fmt.Sprint(n), nil })
	attempt, err := recorder.Begin("resume", couchcore.LifecycleFailureTarget{StartingPath: "/project"})
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("adoption was not confirmed")
	r.attach = func(call couchcore.OperationCall) (any, error) {
		if couchcore.OperationAttemptFromContext(call.Context) != attempt {
			t.Fatal("attach did not borrow the original attempt")
		}
		return nil, cause
	}
	start, _ := attachStartResult(t, "started", menuAddress("target"))
	completed := operationRequest{attempt: attempt, origin: MenuOperationOrigin{Operation: "resume", Address: menuAddress("target")}}.complete(start, nil)
	completed = r.c.prepareOperationCompletion(completed)
	if len(store.Writes) != 0 {
		t.Fatal("adoption recorded before worker terminal publication")
	}
	completed = r.c.publishOperationCompletion(completed)
	r.c.finishOperation(completed)
	if len(store.Writes) != 1 || store.Writes[0].Nonce != attempt.Nonce() {
		t.Fatalf("lost adoption failure: %+v", store.Writes)
	}
}

func TestLifecycleFailureMenuEmptySlotUsesSelectedRowKey(t *testing.T) {
	r := newRemoteRig(t)
	store := couchcore.NewFakeLaunchFailureStore()
	n := 0
	recorder := couchcore.NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) { n++; return fmt.Sprint(n), nil })
	rows := []couchcore.ActionableThreadSummary{menuSlotRow(1, ""), menuSlotRow(2, "")}
	for i := range rows {
		rows[i].Address = couchcore.ThreadAddress{}
		rows[i].RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetSlot, SlotPath: rows[i].Target.Slot.WorktreeRoot}
	}
	r.c.menu = NewMenuState(rows, couchcore.ThreadAddress{})
	r.c.SetOperationAttemptFactory(func(call couchcore.OperationCall) (*couchcore.OperationAttempt, error) {
		return recorder.Begin(call.Name, *call.FailureTarget)
	})
	attempt, err := r.c.beginLifecycleAttempt("reboot", nil, couchcore.ThreadAddress{}, rows[1].RowKey)
	if err != nil {
		t.Fatal(err)
	}
	attempt.Finish(nil, errors.New("empty slot launch refused"))
	if len(store.Writes) != 1 || store.Writes[0].StartingPath != rows[1].Target.Slot.WorktreeRoot {
		t.Fatalf("empty slot failure bound to another row: %+v", store.Writes)
	}
}
