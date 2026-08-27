package couchcore

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestThreadStoreUpsertsStandalonePairWithoutLosingMetadata(t *testing.T) {
	store, ns := newTestThreadStore(t)
	now := time.Unix(1_700_000_000, 0).UTC()
	registration := launcher.StandaloneThreadRegistration{
		RepoScope: "0123456789abcdef", Tag: "work", WorkingPath: ns.Dir(), CreatedAt: now,
		Agent: "codex", Argv: []string{"--sandbox", "workspace-write"},
	}
	firstProcess := ProcessIdentity{PID: 41, Identity: "first-process"}
	created, err := store.UpsertStandalonePair(registration, firstProcess)
	if err != nil {
		t.Fatal(err)
	}
	if created.Reservation || len(created.Incarnations) != 1 {
		t.Fatalf("created record = %+v", created)
	}
	wantProfile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	if got := created.Incarnations[0]; got.State != IncarnationLive || got.PID != 41 || got.Identity != "first-process" || !reflect.DeepEqual(*got.LaunchProfile, wantProfile) {
		t.Fatalf("created incarnation = %+v", got)
	}

	named, err := store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: stringPointer("compiler")})
	if err != nil {
		t.Fatal(err)
	}
	registration.CreatedAt = now.Add(time.Hour)
	registration.Agent = "claude"
	registration.Argv = []string{"--model", "opus"}
	updated, err := store.UpsertStandalonePair(registration, ProcessIdentity{PID: 42, Identity: "second-process"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "compiler" || updated.CreatedAt != created.CreatedAt || updated.StartingPath != created.StartingPath {
		t.Fatalf("upsert lost immutable/metadata fields: %+v", updated)
	}
	if updated.Revision != named.Revision+1 || len(updated.Incarnations) != 2 {
		t.Fatalf("updated record = %+v", updated)
	}
	wantSecond := LaunchProfile{Agent: "claude", Argv: []string{"--model", "opus"}}
	if got := updated.Incarnations[1]; got.PID != 42 || got.Identity != "second-process" || !reflect.DeepEqual(*got.LaunchProfile, wantSecond) {
		t.Fatalf("second incarnation = %+v", got)
	}
}

func TestThreadStoreStandalonePairUpsertIsIdempotentForOneProcess(t *testing.T) {
	store, ns := newTestThreadStore(t)
	registration := launcher.StandaloneThreadRegistration{
		RepoScope: "0123456789abcdef", Tag: "work", WorkingPath: ns.Dir(), CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
		Agent: "codex", Argv: []string{},
	}
	process := ProcessIdentity{PID: 41, Identity: "same-process"}
	first, err := store.UpsertStandalonePair(registration, process)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.UpsertStandalonePair(registration, process)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Incarnations) != 1 || second.Revision != first.Revision {
		t.Fatalf("idempotent upsert = %+v after %+v", second, first)
	}
	registration.Agent = "claude"
	registration.Argv = []string{"--model", "opus"}
	third, err := store.UpsertStandalonePair(registration, process)
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Incarnations) != 1 || third.Revision != second.Revision+1 || third.Incarnations[0].LaunchProfile.Agent != "claude" {
		t.Fatalf("same-process profile refresh = %+v", third)
	}
}

func stringPointer(value string) *string { return &value }
