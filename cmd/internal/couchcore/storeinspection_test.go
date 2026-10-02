package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreInspectionAuthorityExpiresAndRejectsReplacedSlotLock(t *testing.T) {
	s, repository, _ := migrationFixture(t)
	if e := s.EnrollSlotRepository(context.Background(), repository); e != nil {
		t.Fatal(e)
	}
	var saved *StoreInspection
	e := WithStoreInspectionLocks(context.Background(), []CouchNamespace{s.namespace}, nil, func(i *StoreInspection) error {
		saved = i
		if _, e := i.Snapshot(s.namespace); e != nil {
			return e
		}
		root := filepath.Join(repository.Slots[0].Identity.EnvironmentRoot, ".couch")
		if e := os.Rename(filepath.Join(root, "store.lock"), filepath.Join(root, "store.old")); e != nil {
			return e
		}
		if e := os.WriteFile(filepath.Join(root, "store.lock"), nil, 0600); e != nil {
			return e
		}
		if _, e := i.Snapshot(s.namespace); e == nil {
			t.Fatal("replacement lock accepted")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = saved.Snapshot(s.namespace); e == nil {
		t.Fatal("expired authority accepted")
	}
}
func TestMigrationProcessesIncludesCreatingOwnersAndUnknownIdentity(t *testing.T) {
	for _, state := range []IncarnationState{IncarnationLive, IncarnationUnknown, IncarnationCreating} {
		proc := NewFakeProcOps()
		proc.Set(123, "wrapper")
		record := ThreadRecord{Incarnations: []ThreadIncarnation{{State: state, PID: 123, Identity: "wrapper"}}}
		got, e := ObserveMigrationProcesses(context.Background(), proc, []ThreadRecord{record})
		if e != nil || len(got) != 1 || got[0].Liveness != Live {
			t.Fatal(state, got, e)
		}
	}
	for _, live := range []Liveness{Live, Dead, Unknown} {
		proc := NewFakeProcOps()
		if live == Live {
			proc.Set(456, "owner")
		} else if live == Unknown {
			proc.SetUnknown(456)
		}
		record := ThreadRecord{Incarnations: []ThreadIncarnation{{State: IncarnationCreating, Start: &ThreadStartClaim{OwnerPID: 456, OwnerIdentity: "owner"}}}}
		got, e := ObserveMigrationProcesses(context.Background(), proc, []ThreadRecord{record})
		if e != nil || len(got) != 1 || got[0].Liveness != live {
			t.Fatal(got, e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := ObserveMigrationProcesses(ctx, NewFakeProcOps(), []ThreadRecord{{Incarnations: []ThreadIncarnation{{}}}}); e == nil {
		t.Fatal("ignored cancelled scan")
	}
}
