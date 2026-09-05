package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStartTransactionIntegratesWithFakeRunnerLifecycle(t *testing.T) {
	runner := NewFakeRunner()
	record := admittedStartRecord(t)
	record, err := AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := runner.StartBlocked(context.Background(), "/repo", []string{"pair"}, nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	record, err = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: h.PID(), Identity: h.Identity()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if runner.Child(h.ID()).ExecCount != 0 {
		t.Fatal("fake target execed before durable helper transition")
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	record, err = AdvanceStartTransaction(record, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if err != nil || record.Incarnations[0].State != IncarnationLive || runner.Child(h.ID()).ExecCount != 1 {
		t.Fatalf("final record/child = %+v / %+v, %v", record.Incarnations[0], runner.Child(h.ID()), err)
	}
}

func TestStartTransactionIntegratesWithThreadStoreCAS(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := admittedStartRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := store.AdvanceStart(created.Address, created.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	if err != nil || claimed.Revision != created.Revision+1 {
		t.Fatalf("claim = %+v, %v", claimed, err)
	}
	if err := store.DeletePristineThread(created.Address); err == nil {
		t.Fatal("pristine rollback removed a nonce-tracked start")
	}
	helper, err := store.AdvanceStart(created.Address, claimed.Revision, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	if err != nil || helper.Revision != claimed.Revision+1 {
		t.Fatalf("helper = %+v, %v", helper, err)
	}
	if err := store.DeleteStart(created.Address, claimed.Revision, "start-0123456789abcdef"); err == nil {
		t.Fatal("stale delete removed a newer helper record")
	}
	if err := store.DeleteStart(created.Address, helper.Revision, "wrong"); err == nil {
		t.Fatal("wrong nonce removed start")
	}
	if err := store.DeleteStart(created.Address, helper.Revision, "start-0123456789abcdef"); err != nil {
		t.Fatalf("DeleteStart: %v", err)
	}
	if _, err := store.GetThread(created.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("deleted start lookup = %v", err)
	}
}
