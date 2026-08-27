package couchcore

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
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
	beforeGeneration, err := store.ManifestGeneration()
	if err != nil {
		t.Fatalf("ManifestGeneration: %v", err)
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
	afterGeneration, err := store.ManifestGeneration()
	if err != nil {
		t.Fatalf("ManifestGeneration(after): %v", err)
	}
	if afterGeneration != beforeGeneration {
		t.Fatalf("single-record update changed manifest generation: %d -> %d", beforeGeneration, afterGeneration)
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
		State: IncarnationCreating,
		Policy: &PolicyResult{
			PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64), RepoIdentity: "repo-identity",
			AdmissionKey: ns.Dir(), Capacity: PolicyCapacity{Kind: CapacityUnbounded},
		},
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
		State: IncarnationCreating,
		Policy: &PolicyResult{
			PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64), RepoIdentity: "repo-identity",
			AdmissionKey: ns.Dir(), Capacity: PolicyCapacity{Kind: CapacityUnbounded},
		},
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
