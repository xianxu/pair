package couchcore

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestResumeCandidatesPathNeverProvisionsOrEnrolls(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	slot := *f.Couch.Threads.slot
	f.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{slot.PrimaryRoot: {Identity: WorkspaceIdentity{RepoIdentity: slot.RepoIdentity, PrimaryRoot: slot.PrimaryRoot}, Slots: []SlotCandidate{{Identity: slot, Verified: true}}}}}
	ensures := 0
	f.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		ensures++
		return ProvisionResult{}, errors.New("preview must not ensure")
	})
	before, err := snapshotDirectoryForTest(slot.EnvironmentRoot)
	if err != nil {
		t.Fatal(err)
	}
	beforeNamespace, err := snapshotDirectoryForTest(f.Couch.Namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.Couch.ResumeCandidates(context.Background(), ResumeTarget{Path: slot.WorktreeRoot})
	if err != nil {
		t.Fatal(err)
	}
	if ensures != 0 {
		t.Fatalf("read-only preview called workspace Ensure %d times", ensures)
	}
	if preview.Address != f.record.Address || len(preview.Candidates) != 1 {
		t.Fatalf("read-only path did not resolve retained record: %+v", preview)
	}
	after, err := snapshotDirectoryForTest(slot.EnvironmentRoot)
	if err != nil {
		t.Fatal(err)
	}
	afterNamespace, err := snapshotDirectoryForTest(f.Couch.Namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeNamespace, afterNamespace) {
		t.Fatal("preview changed slot or global store")
	}
}

func TestResumeCandidatesMissingSlotStoreStaysAbsent(t *testing.T) {
	local := testLocalThreadStore(t)
	env := newTestEnv(t, local.slot.WorktreeRoot)
	env.Couch.Threads = local
	slot := *local.slot
	env.Couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{slot.PrimaryRoot: {Identity: WorkspaceIdentity{RepoIdentity: slot.RepoIdentity, PrimaryRoot: slot.PrimaryRoot}, Slots: []SlotCandidate{{Identity: slot, Verified: true}}}}}
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		t.Error("preview provisioned absent store")
		return ProvisionResult{}, errors.New("not authorized")
	})
	if _, err := os.Stat(local.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture store already exists: %v", err)
	}
	preview, err := env.Couch.ResumeCandidates(context.Background(), ResumeTarget{Path: slot.WorktreeRoot})
	if err != nil || !preview.Incomplete {
		t.Fatalf("missing store preview=%+v err=%v", preview, err)
	}
	if _, err := os.Stat(local.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created absent store: %v", err)
	}
}
