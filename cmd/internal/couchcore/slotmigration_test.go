package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func migrationFixture(t *testing.T) (*ThreadStore, SlotRepository, ThreadRecord) {
	t.Helper()
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newTestThreadStore(t)
	r := validThreadRecord(t)
	r.StartingPath, r.WorkingPath = f.host(1), f.host(1)
	r, err = s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	return s, repository, r
}

func TestSlotMigrationPreservesAddressAndCutsOver(t *testing.T) {
	s, repository, r := migrationFixture(t)
	preference := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: repository.Identity.RepoIdentity, PhysicalPath: r.StartingPath, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"--sandbox", "workspace-write"}}, Revision: 1}
	if err := writePathLaunchPreferenceForTest(s, preference); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	backend, err := s.storeForAddress(r.Address)
	if err != nil || backend == s {
		t.Fatalf("route = %v, %v", backend, err)
	}
	got, err := backend.GetThread(r.Address)
	if err != nil || got.Address != r.Address || got.Revision != r.Revision {
		t.Fatalf("record = %+v, %v", got, err)
	}
	pref, found, err := backend.GetPathLaunchPreference(preference.RepoIdentity, r.StartingPath)
	if err != nil || !found || pref.Revision != preference.Revision {
		t.Fatalf("preference %+v %v %v", pref, found, err)
	}
	if _, err := os.Stat(s.recordPath(r.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy payload retained: %v", err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backend.recordPath(r.Address), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.storeForAddress(r.Address); err == nil {
		t.Fatal("corrupt local record fell back to global")
	}
}

func TestSlotMigrationInterruptedCutoverReplays(t *testing.T) {
	for _, after := range []int{-1, 0, 1} {
		t.Run(string(rune('a'+after+1)), func(t *testing.T) {
			s, repository, r := migrationFixture(t)
			boom := errors.New("interrupted migration")
			if after == -1 {
				s.hooks.AfterJournal = func() error { return boom }
			} else {
				s.hooks.AfterTarget = func(n int) error {
					if n == after {
						return boom
					}
					return nil
				}
			}
			if err := s.EnrollSlotRepository(context.Background(), repository); !errors.Is(err, boom) {
				t.Fatalf("failure: %v", err)
			}
			s.hooks = threadStoreHooks{}
			if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
				t.Fatal(err)
			}
			local, err := s.storeForAddress(r.Address)
			if err != nil || local == s {
				t.Fatalf("missing local authority: %v", err)
			}
			if _, err := local.GetThread(r.Address); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSlotMigrationRejectsAmbiguousAndConflictingRecords(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "multiple-global", true: "conflicting-local"}[conflict], func(t *testing.T) {
			s, repository, r := migrationFixture(t)
			other := r
			other.Address.Tag = "couch-1111111111111111"
			if conflict {
				local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
				if _, err := local.CreateThread(other); err != nil {
					t.Fatal(err)
				}
			} else if _, err := s.CreateThread(other); err != nil {
				t.Fatal(err)
			}
			if err := s.EnrollSlotRepository(context.Background(), repository); err == nil {
				t.Fatal("ambiguous enrollment succeeded")
			}
			if _, err := os.Stat(s.recordPath(r.Address)); err != nil {
				t.Fatal("lost global source", err)
			}
			if local, err := s.storeForPath(r.StartingPath); err != nil || local != s {
				t.Fatalf("failed enrollment cut over: %v", err)
			}
		})
	}
}

func TestSlotRoutingRebuildsAndMissingCurrentDoesNotInventAddress(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
	if err := os.Remove(local.recordPath(r.Address)); err != nil {
		t.Fatal(err)
	}
	reopened := NewThreadStore(s.namespace)
	stores, err := reopened.discoveredBackends()
	if err != nil || len(stores) != 1 {
		t.Fatalf("discovery %d %v", len(stores), err)
	}
	got, err := reopened.storeForPath(r.StartingPath)
	if err != nil || got.root != local.root {
		t.Fatalf("missing-current slot lost: %v", err)
	}
	if _, err := reopened.storeForAddress(r.Address); err == nil {
		t.Fatal("missing current granted an address")
	}
}

func TestSlotMigrationZeroCurrentPreservesPreferencesAndHistory(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	preference := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: repository.Identity.RepoIdentity, PhysicalPath: r.StartingPath, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {}}, Revision: 2}
	if err := writePathLaunchPreferenceForTest(s, preference); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local, err := s.storeForAddress(r.Address)
	if err != nil || local == s {
		t.Fatalf("archive not routed locally: %v", err)
	}
	records, err := local.ArchivedThreads()
	if err != nil || len(records) != 1 || records[0].Address != r.Address {
		t.Fatalf("history %+v %v", records, err)
	}
	if _, err := local.readArchiveGraceLocked(r.Address); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(local.recordPath(r.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invented current: %v", err)
	}
	pref, found, err := local.GetPathLaunchPreference(preference.RepoIdentity, r.StartingPath)
	if err != nil || !found || pref.Revision != 2 {
		t.Fatalf("preference %+v %v %v", pref, found, err)
	}
}

func TestSlotMigrationLeavesPrimaryArchiveRoutable(t *testing.T) {
	s, repository, r := migrationFixture(t)
	primary := r
	primary.Address.Tag = "couch-2222222222222222"
	primary.StartingPath, primary.WorkingPath = repository.Identity.PrimaryRoot, repository.Identity.PrimaryRoot
	if _, err := s.CreateThread(primary); err != nil {
		t.Fatal(err)
	}
	if err := s.ArchiveThread(primary.Address); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	backend, err := s.storeForAddress(primary.Address)
	if err != nil || backend != s {
		t.Fatalf("primary archive route %v %v", backend, err)
	}
}

func TestSlotMigrationStagingInterruptionKeepsGlobalAuthority(t *testing.T) {
	s, repository, r := migrationFixture(t)
	interrupted := errors.New("local target interruption")
	s.hooks.AfterPublicationWrite = func(path string) error {
		if path == filepath.Join(repository.Slots[0].Identity.EnvironmentRoot, ".couch", "thread.json") {
			return interrupted
		}
		return nil
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); !errors.Is(err, interrupted) {
		t.Fatalf("did not interrupt local stage: %v", err)
	}
	backend, err := s.storeForAddress(r.Address)
	if err != nil || backend != s {
		t.Fatalf("staging became authoritative: %v", err)
	}
	s.hooks = threadStoreHooks{}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	backend, err = s.storeForAddress(r.Address)
	if err != nil || backend == s {
		t.Fatalf("retry did not cut over: %v", err)
	}
}

func TestSlotMigrationEmptyRepositoryCanEnrollBeforeCreation(t *testing.T) {
	f := newProvisionFixture(t)
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newTestThreadStore(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local, err := s.storeForPath(f.host(1))
	if err != nil || local == s || local.slot.Number != 1 {
		t.Fatalf("uncreated slot routed global: %v", err)
	}
	primary, err := s.storeForPath(f.Primary)
	if err != nil || primary != s {
		t.Fatalf("primary moved: %v", err)
	}
}

func TestSlotRoutingCorruptSlotLeavesPrimaryAvailable(t *testing.T) {
	s, repository, r := migrationFixture(t)
	primary := r
	primary.Address.Tag = "couch-3333333333333333"
	primary.StartingPath, primary.WorkingPath = repository.Identity.PrimaryRoot, repository.Identity.PrimaryRoot
	if _, err := s.CreateThread(primary); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
	if err := os.WriteFile(local.recordPath(r.Address), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	backend, err := s.storeForAddress(primary.Address)
	if err != nil || backend != s {
		t.Fatalf("unrelated corruption hid primary: %v", err)
	}
}

func TestSlotMigrationRefusesUnmatchedLocalCurrent(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
	other := r
	other.Address.Tag = "couch-4444444444444444"
	if _, err := local.CreateThread(other); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err == nil {
		t.Fatal("unmatched local current silently adopted")
	}
}

func TestSlotEnrollmentRebuildsLostRootDiscoveryFromLocalAuthority(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
	before, err := os.ReadFile(local.recordPath(r.Address))
	if err != nil {
		t.Fatal(err)
	}
	// Root locations must be re-supplied explicitly. Existing local state remains
	// authoritative when there is no legacy source to migrate or conflict with.
	if err := os.Remove(s.manifestPath()); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetThread(r.Address)
	if err != nil || got.Address != r.Address {
		t.Fatalf("reopened=%+v %v", got, err)
	}
	after, err := os.ReadFile(local.recordPath(r.Address))
	if err != nil || string(after) != string(before) {
		t.Fatalf("local authority changed: %v", err)
	}
}
