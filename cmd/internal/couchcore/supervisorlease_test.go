package couchcore

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func testCouchNamespace(t *testing.T) CouchNamespace {
	t.Helper()
	ns, err := ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace: %v", err)
	}
	return ns
}

func testOwnerProc(t *testing.T, identity string) *FakeProcOps {
	t.Helper()
	proc := NewFakeProcOps()
	proc.Set(os.Getpid(), identity)
	return proc
}

func TestSupervisorLeaseRefusesASecondOwnerWithVerifiedIdentity(t *testing.T) {
	ns := testCouchNamespace(t)
	proc := testOwnerProc(t, "this-process")
	lease, err := AcquireSupervisorLease(ns, proc)
	if err != nil {
		t.Fatalf("AcquireSupervisorLease(first): %v", err)
	}
	defer lease.Close()

	_, err = AcquireSupervisorLease(ns, proc)
	var held *SupervisorLeaseHeldError
	if !errors.As(err, &held) {
		t.Fatalf("second acquire err = %v, want *SupervisorLeaseHeldError", err)
	}
	if held.Owner == nil || held.Owner.PID != os.Getpid() || held.Owner.Identity != "this-process" {
		t.Fatalf("verified owner = %+v", held.Owner)
	}
	if !strings.Contains(err.Error(), "this-process") {
		t.Fatalf("refusal omits verified process-start identity: %v", err)
	}
}

func TestVerifiedOwnerWithholdsStaleOrRecycledProcessMetadata(t *testing.T) {
	ns := testCouchNamespace(t)
	proc := testOwnerProc(t, "first-process")
	lease, err := AcquireSupervisorLease(ns, proc)
	if err != nil {
		t.Fatalf("AcquireSupervisorLease: %v", err)
	}
	defer lease.Close()

	proc.Set(os.Getpid(), "recycled-process")
	owner, held, err := VerifiedOwner(ns, proc)
	if err == nil {
		t.Fatal("identity mismatch must be observable")
	}
	if held {
		t.Fatalf("held = true with unverified owner %+v", owner)
	}
	if owner != (SupervisorOwner{}) {
		t.Fatalf("unverified metadata leaked as owner: %+v", owner)
	}
}

func TestSupervisorLeaseCloseReleasesNamespace(t *testing.T) {
	ns := testCouchNamespace(t)
	proc := testOwnerProc(t, "this-process")
	first, err := AcquireSupervisorLease(ns, proc)
	if err != nil {
		t.Fatalf("AcquireSupervisorLease(first): %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	second, err := AcquireSupervisorLease(ns, proc)
	if err != nil {
		t.Fatalf("AcquireSupervisorLease(after close): %v", err)
	}
	defer second.Close()
}
