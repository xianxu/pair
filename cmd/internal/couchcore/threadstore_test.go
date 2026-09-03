package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestThreadStoreReadsStrictDefensivePathLaunchPreference(t *testing.T) {
	store, ns := newTestThreadStore(t)
	physicalPath := ns.Dir()
	preference := PathLaunchPreference{
		SchemaVersion: PathLaunchPreferenceSchemaVersion,
		RepoIdentity:  "repo-identity",
		PhysicalPath:  physicalPath,
		LastAgent:     "codex",
		ArgvByAgent: map[string][]string{
			"claude": {"--model", "opus"},
			"codex":  {"--sandbox", "workspace-write"},
		},
		Revision: 3,
	}
	if err := writePathLaunchPreferenceForTest(store, preference); err != nil {
		t.Fatal(err)
	}

	got, found, err := store.GetPathLaunchPreference(preference.RepoIdentity, physicalPath)
	if err != nil || !found {
		t.Fatalf("GetPathLaunchPreference = %+v, %v, %v", got, found, err)
	}
	got.ArgvByAgent["codex"][0] = "mutated"
	again, found, err := store.GetPathLaunchPreference(preference.RepoIdentity, physicalPath)
	if err != nil || !found || !reflect.DeepEqual(again.ArgvByAgent["codex"], []string{"--sandbox", "workspace-write"}) {
		t.Fatalf("stored preference aliased caller: %+v, %v, %v", again, found, err)
	}

	missing, found, err := store.GetPathLaunchPreference("other-repo", physicalPath)
	if err != nil || found || missing.Revision != 0 {
		t.Fatalf("missing preference = %+v, %v, %v", missing, found, err)
	}
}

func TestThreadStoreRejectsPathLaunchPreferenceAtWrongAddress(t *testing.T) {
	store, ns := newTestThreadStore(t)
	preference := PathLaunchPreference{
		SchemaVersion: PathLaunchPreferenceSchemaVersion,
		RepoIdentity:  "repo-identity",
		PhysicalPath:  ns.Dir(),
		LastAgent:     "codex",
		ArgvByAgent:   map[string][]string{"codex": {}},
		Revision:      1,
	}
	if err := writePathLaunchPreferenceForTest(store, preference); err != nil {
		t.Fatal(err)
	}
	path := store.pathLaunchPreferencePath(preference.RepoIdentity, preference.PhysicalPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := strings.Replace(string(raw), `"physical_path": "`+preference.PhysicalPath+`"`, `"physical_path": "/other"`, 1)
	if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetPathLaunchPreference(preference.RepoIdentity, preference.PhysicalPath); err == nil {
		t.Fatal("path preference stored at the wrong address was accepted")
	}
}

func TestThreadStoreRejectsMalformedPathLaunchPreference(t *testing.T) {
	tests := map[string]func(string) string{
		"unknown field": func(raw string) string {
			return strings.Replace(raw, `"revision": 1`, `"revision": 1, "unknown": true`, 1)
		},
		"missing last agent": func(raw string) string {
			return strings.Replace(raw, `"last_agent": "codex",`, `"last_agent": "",`, 1)
		},
		"null argv map": func(raw string) string {
			start := strings.Index(raw, `"argv_by_agent":`)
			end := strings.Index(raw[start:], `},`)
			return raw[:start] + `"argv_by_agent": null` + raw[start+end+1:]
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			preference := PathLaunchPreference{
				SchemaVersion: PathLaunchPreferenceSchemaVersion,
				RepoIdentity:  "repo-identity",
				PhysicalPath:  ns.Dir(),
				LastAgent:     "codex",
				ArgvByAgent:   map[string][]string{"codex": {"--sandbox", "workspace-write"}},
				Revision:      1,
			}
			if err := writePathLaunchPreferenceForTest(store, preference); err != nil {
				t.Fatal(err)
			}
			path := store.pathLaunchPreferencePath(preference.RepoIdentity, preference.PhysicalPath)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(mutate(string(raw))), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := store.GetPathLaunchPreference(preference.RepoIdentity, preference.PhysicalPath); err == nil {
				t.Fatal("malformed path preference accepted")
			}
		})
	}
}

func writePathLaunchPreferenceForTest(store *ThreadStore, preference PathLaunchPreference) error {
	raw, err := json.MarshalIndent(preference, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomicBytes(store.pathLaunchPreferencePath(preference.RepoIdentity, preference.PhysicalPath), append(raw, '\n'))
}

func newTestThreadStore(t *testing.T) (*ThreadStore, CouchNamespace) {
	t.Helper()
	ns := testCouchNamespace(t)
	return NewThreadStore(ns), ns
}

func TestThreadStoreRejectsDuplicatePersistedFields(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	path := store.recordPath(created.Address)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := strings.Replace(string(raw), `"revision": 1`, `"revision": 1, "revision": 2`, 1)
	if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetThread(created.Address); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("GetThread duplicate-field err = %v", err)
	}
}

func TestThreadStoreRejectsCompositeTraversalBeforeAnyPathLookup(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.Address.RepoScope = "../outside"
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	_, err := store.CreateThread(record)
	if err == nil || !strings.Contains(err.Error(), "invalid thread repo scope") {
		t.Fatalf("CreateThread traversal err = %v", err)
	}
	var exists *ThreadExistsError
	if errors.As(err, &exists) {
		t.Fatalf("invalid address reached filesystem collision lookup: %v", err)
	}
}

func TestThreadStoreUpdateExistingThreadUsesRevisionWithoutChangingManifest(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	before, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	other := NewThreadStore(ns)
	updated, err := other.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Description = "first description"
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateExistingThread: %v", err)
	}
	if updated.Revision != created.Revision+1 || updated.Description != "first description" {
		t.Fatalf("updated = %+v", updated)
	}
	after, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot(after): %v", err)
	}
	if after.Generation != before.Generation {
		t.Fatalf("single-record update changed manifest generation: %d -> %d", before.Generation, after.Generation)
	}

	_, err = store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Description = "stale overwrite"
		return nil
	})
	var conflict *ThreadRevisionError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale update err = %v, want *ThreadRevisionError", err)
	}
	got, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.Description != "first description" {
		t.Fatalf("stale writer changed record: %+v", got)
	}
}

func TestThreadStoreUpdateMissingPreservesNotFoundContract(t *testing.T) {
	store, _ := newTestThreadStore(t)
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	_, err := store.UpdateExistingThread(address, 1, func(*ThreadRecord) error { return nil })
	if !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("missing update err = %T %v, want ErrThreadNotFound", err, err)
	}
}

func TestThreadStoreReturnsDefensiveIncarnationCopies(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "original", State: IncarnationUnknown}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	created.Incarnations[0].Identity = "caller mutation"
	got, err := store.GetThread(record.Address)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.Incarnations[0].Identity != "original" {
		t.Fatalf("stored incarnation aliased caller: %+v", got.Incarnations)
	}
	got.Incarnations[0].Identity = "read mutation"
	again, _ := store.GetThread(record.Address)
	if again.Incarnations[0].Identity != "original" {
		t.Fatalf("read result aliased store: %+v", again.Incarnations)
	}
}

func TestThreadStorePersistsAndDefensivelyCopiesRecoverableStartClaim(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce:         "start-0123456789abcdef",
			OwnerPID:      41,
			OwnerIdentity: "owner-start-token",
		},
	}}

	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	created.Incarnations[0].Start.OwnerIdentity = "caller mutation"

	got, err := store.GetThread(record.Address)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.Incarnations[0].Start == nil || got.Incarnations[0].Start.OwnerIdentity != "owner-start-token" {
		t.Fatalf("stored start claim aliased caller: %+v", got.Incarnations[0].Start)
	}
	got.Incarnations[0].Start.Nonce = "read mutation"
	again, err := store.GetThread(record.Address)
	if err != nil {
		t.Fatalf("GetThread again: %v", err)
	}
	if again.Incarnations[0].Start.Nonce != "start-0123456789abcdef" {
		t.Fatalf("read start claim aliased store: %+v", again.Incarnations[0].Start)
	}
}

func TestThreadStorePersistsAndDefensivelyCopiesIncarnationLaunchProfile(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "helper", State: IncarnationLive,
		LaunchProfile: &LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}},
	}}

	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	created.Incarnations[0].LaunchProfile.Argv[0] = "mutated"
	got, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	want := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	if got.Incarnations[0].LaunchProfile == nil || !reflect.DeepEqual(*got.Incarnations[0].LaunchProfile, want) {
		t.Fatalf("stored launch profile = %+v, want %+v", got.Incarnations[0].LaunchProfile, want)
	}
	got.Incarnations[0].LaunchProfile.Argv[1] = "mutated"
	again, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*again.Incarnations[0].LaunchProfile, want) {
		t.Fatalf("read launch profile aliases caller: %+v", again.Incarnations[0].LaunchProfile)
	}
}

func TestThreadStoreRegistrationAtomicallyCommitsThreadAndPathLaunchPreference(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating, RepoIdentity: "repo-identity",
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	claimed, err := store.AdvanceStart(created.Address, created.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner"}, Profile: &profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	helper, err := store.AdvanceStart(claimed.Address, claimed.Revision, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.GetPathLaunchPreference("repo-identity", ns.Dir()); err != nil || found {
		t.Fatalf("unsuccessful start wrote path preference: found=%v err=%v", found, err)
	}
	registered, err := store.AdvanceStart(helper.Address, helper.Revision, StartEvent{
		Kind: StartRegistered, Nonce: "start-0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if registered.Incarnations[0].LaunchProfile == nil || !reflect.DeepEqual(*registered.Incarnations[0].LaunchProfile, profile) {
		t.Fatalf("registered profile = %+v", registered.Incarnations[0].LaunchProfile)
	}
	preference, found, err := store.GetPathLaunchPreference("repo-identity", ns.Dir())
	if err != nil || !found || preference.LastAgent != "codex" || !reflect.DeepEqual(preference.ArgvByAgent["codex"], profile.Argv) {
		t.Fatalf("path preference = %+v found=%v err=%v", preference, found, err)
	}
}

func TestThreadStoreRecoversInterruptedThreadAndPathPreferenceCommit(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating, RepoIdentity: "repo-identity",
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	profile := LaunchProfile{Agent: "muse", Argv: []string{"--model", "spark"}}
	claimed, err := store.AdvanceStart(created.Address, created.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner"}, Profile: &profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	helper, err := store.AdvanceStart(claimed.Address, claimed.Revision, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{AfterTarget: func(index int) error {
		if index == 0 {
			return errors.New("injected crash after thread image")
		}
		return nil
	}})
	if _, err := crashing.AdvanceStart(helper.Address, helper.Revision, StartEvent{
		Kind: StartRegistered, Nonce: "start-0123456789abcdef",
	}); err == nil {
		t.Fatal("interrupted commit returned success")
	}

	recovered := NewThreadStore(ns)
	if err := recovered.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	thread, err := recovered.GetThread(helper.Address)
	if err != nil || thread.Incarnations[0].LaunchProfile == nil || !reflect.DeepEqual(*thread.Incarnations[0].LaunchProfile, profile) {
		t.Fatalf("recovered thread = %+v, %v", thread, err)
	}
	preference, found, err := recovered.GetPathLaunchPreference("repo-identity", ns.Dir())
	if err != nil || !found || preference.LastAgent != "muse" || !reflect.DeepEqual(preference.ArgvByAgent["muse"], profile.Argv) {
		t.Fatalf("recovered preference = %+v, %v, %v", preference, found, err)
	}
}

func TestThreadStoreIndependentInstancesSerializeRevisionUpdates(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}

	stores := []*ThreadStore{NewThreadStore(ns), NewThreadStore(ns)}
	var wg sync.WaitGroup
	errs := make([]error, len(stores))
	for i := range stores {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = stores[i].UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
				next.Description = string(rune('a' + i))
				return nil
			})
		}(i)
	}
	wg.Wait()
	var successes, conflicts int
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			conflicts++
		}
	}
	if !reflect.DeepEqual([]int{successes, conflicts}, []int{1, 1}) {
		t.Fatalf("successes/conflicts = %d/%d, errors=%v", successes, conflicts, errs)
	}
}

func createParkableThread(t *testing.T, store *ThreadStore, ns CouchNamespace, nonce string) (ThreadRecord, ParkIdentity, LaunchProfile) {
	t.Helper()
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record.Incarnations = []ThreadIncarnation{
		{PID: 42, Identity: "park-helper", State: IncarnationLive, LaunchProfile: &profile},
	}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{
		Nonce: nonce, Address: created.Address, PID: 42, ProcessIdentity: "park-helper",
	}
	return created, identity, profile
}

func TestThreadStoreParkLifecycleUsesRevisionCASAndFinalizesExactIncarnation(t *testing.T) {
	store, ns := newTestThreadStore(t)
	created, identity, profile := createParkableThread(t, store, ns, "park-0123456789abcdef")
	previousActive := time.Unix(50, 0).UTC()
	created, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.LastActiveAt = previousActive
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	begun, err := store.BeginPark(created.Address, created.Revision, identity)
	if err != nil {
		t.Fatalf("BeginPark: %v", err)
	}
	if begun.Park == nil || begun.Park.BaseRevision != created.Revision || begun.Park.RecordRevision != begun.Revision || begun.Park.Phase != ParkRequested {
		t.Fatalf("begun = %+v", begun)
	}
	if begun.LatestLaunchProfile == nil || !reflect.DeepEqual(*begun.LatestLaunchProfile, profile) {
		t.Fatalf("BeginPark did not capture legacy live profile: %+v", begun.LatestLaunchProfile)
	}
	profile.Argv[0] = "caller-mutated"
	if begun.LatestLaunchProfile.Argv[0] != "--sandbox" {
		t.Fatal("captured profile aliases incarnation profile")
	}

	awaiting, err := store.AdvancePark(begun.Address, begun.Revision, ParkEvent{
		Kind: ParkRequestCommitted, Identity: identity, Attempt: 1,
	})
	if err != nil || awaiting.Park.Phase != ParkAwaitingCompletion || awaiting.Park.RecordRevision != awaiting.Revision {
		t.Fatalf("AdvancePark(request): %+v, %v", awaiting, err)
	}
	timedOut, err := store.AdvancePark(awaiting.Address, awaiting.Revision, ParkEvent{
		Kind: ParkFailureObserved, Identity: identity, Attempt: 1,
		Failure: &ParkFailure{Code: "timeout", Diagnostic: "cleanup deadline"},
	})
	if err != nil || !timedOut.Park.Attempts[0].TimedOut {
		t.Fatalf("AdvancePark(timeout): %+v, %v", timedOut, err)
	}
	second, err := store.AppendParkAttempt(timedOut.Address, timedOut.Revision, identity)
	if err != nil || len(second.Park.Attempts) != 2 || second.Park.Attempts[1].Number != 2 {
		t.Fatalf("AppendParkAttempt: %+v, %v", second, err)
	}
	second, err = store.AdvancePark(second.Address, second.Revision, ParkEvent{
		Kind: ParkRequestCommitted, Identity: identity, Attempt: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	parkedAt := time.Unix(40, 0).UTC()
	finalized, err := store.FinalizePark(second.Address, second.Revision, identity, 1, parkedAt)
	if err != nil {
		t.Fatalf("FinalizePark: %v", err)
	}
	if finalized.Park != nil || len(finalized.Incarnations) != 0 {
		t.Fatalf("finalization removed wrong state: %+v", finalized)
	}
	if finalized.VerifiedPark == nil || finalized.VerifiedPark.Identity != identity || finalized.VerifiedPark.Attempt != 1 || finalized.VerifiedPark.ParkedAt != parkedAt {
		t.Fatalf("verified park = %+v", finalized.VerifiedPark)
	}
	if finalized.LastActiveAt != previousActive {
		t.Fatalf("last_active_at moved backward: %v", finalized.LastActiveAt)
	}
	if len(finalized.ParkHistory) != 1 || !finalized.ParkHistory[0].Closed || finalized.ParkHistory[0].Tombstoned || finalized.ParkHistory[0].SuccessfulAttempt != 1 {
		t.Fatalf("park history = %+v", finalized.ParkHistory)
	}
	if finalized.LatestLaunchProfile == nil || finalized.LatestLaunchProfile.Argv[0] != "--sandbox" {
		t.Fatalf("latest profile lost: %+v", finalized.LatestLaunchProfile)
	}
}

func TestThreadStoreParkConflictsAndAbandonNeverReleaseAdmission(t *testing.T) {
	store, ns := newTestThreadStore(t)
	created, identity, _ := createParkableThread(t, store, ns, "park-1111111111111111")
	concurrent, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Description = "competing writer"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BeginPark(created.Address, created.Revision, identity); err == nil {
		t.Fatal("stale BeginPark succeeded")
	}
	kept, _ := store.GetThread(created.Address)
	if len(kept.Incarnations) != 1 || kept.Park != nil {
		t.Fatalf("stale begin released admission: %+v", kept)
	}

	begun, err := store.BeginPark(concurrent.Address, concurrent.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	competing, err := store.UpdateExistingThread(begun.Address, begun.Revision, func(next *ThreadRecord) error {
		next.Name = "concurrent name"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalizePark(begun.Address, begun.Revision, identity, 1, time.Unix(60, 0).UTC()); err == nil {
		t.Fatal("stale FinalizePark succeeded")
	}
	kept, _ = store.GetThread(begun.Address)
	if kept.Park == nil || len(kept.Incarnations) != 1 || kept.VerifiedPark != nil {
		t.Fatalf("stale finalize released admission: %+v", kept)
	}

	abandoned, err := store.AbandonPark(competing.Address, competing.Revision, identity)
	if err != nil {
		t.Fatalf("AbandonPark: %v", err)
	}
	if abandoned.Park != nil || len(abandoned.Incarnations) != 1 || abandoned.VerifiedPark != nil ||
		len(abandoned.ParkHistory) != 1 || !abandoned.ParkHistory[0].Closed || !abandoned.ParkHistory[0].Tombstoned {
		t.Fatalf("abandoned = %+v", abandoned)
	}
	if _, err := store.FinalizePark(abandoned.Address, abandoned.Revision, identity, 1, time.Unix(61, 0).UTC()); err == nil {
		t.Fatal("tombstoned transaction finalized")
	}
	after, _ := store.GetThread(abandoned.Address)
	if len(after.Incarnations) != 1 || after.VerifiedPark != nil {
		t.Fatalf("historical result released admission: %+v", after)
	}
}

func TestThreadStoreBeginParkRequiresOneExactLiveOrUnknownIncarnation(t *testing.T) {
	for name, mutate := range map[string]func(*ThreadRecord, *ParkIdentity){
		"creating target": func(r *ThreadRecord, _ *ParkIdentity) {
			r.Incarnations[0].State = IncarnationCreating
			r.Incarnations[0].LaunchProfile = nil
		},
		"wrong identity": func(_ *ThreadRecord, id *ParkIdentity) { id.ProcessIdentity = "wrong" },
		"duplicate identity": func(r *ThreadRecord, _ *ParkIdentity) {
			duplicate := r.Incarnations[0]
			r.Incarnations = append(r.Incarnations, duplicate)
		},
	} {
		t.Run(name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			created, identity, _ := createParkableThread(t, store, ns, "park-2222222222222222")
			current, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
				mutate(next, &identity)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.BeginPark(current.Address, current.Revision, identity); err == nil {
				t.Fatal("invalid park target accepted")
			}
			kept, _ := store.GetThread(current.Address)
			if kept.Park != nil || len(kept.Incarnations) == 0 {
				t.Fatalf("invalid begin mutated thread: %+v", kept)
			}
		})
	}
}

func TestThreadStoreMarksOnlyExactLiveIncarnationUnknown(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Incarnations = []ThreadIncarnation{
		{PID: 41, Identity: "older", State: IncarnationLive, StartedAt: record.CreatedAt},
		{PID: 42, Identity: "target", State: IncarnationLive, StartedAt: record.CreatedAt},
	}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	marked, err := store.MarkIncarnationUnknown(record.Address, ProcessIdentity{PID: 42, Identity: "target"})
	if err != nil {
		t.Fatal(err)
	}
	if marked.Revision != created.Revision+1 || marked.Incarnations[0].State != IncarnationLive || marked.Incarnations[1].State != IncarnationUnknown {
		t.Fatalf("exact incarnation mark = %+v", marked)
	}
	if _, err := store.MarkIncarnationUnknown(record.Address, ProcessIdentity{PID: 42, Identity: "reused"}); err == nil {
		t.Fatal("PID-only match changed a different process incarnation")
	}
}
