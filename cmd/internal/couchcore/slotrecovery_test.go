package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestSlotFreshTransactionPreservesHistoryAndRefusesConcurrentChange(t *testing.T) {
	s := testLocalThreadStore(t)
	old := validThreadRecord(t)
	old.StartingPath = s.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	old.Name = "keep"
	old.Description = "context"
	old, err := s.CreateThread(old)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := s.observeSlotCurrent()
	if err != nil {
		t.Fatal(err)
	}
	next := old
	next.Address.Tag = "couch-1111111111111111"
	next.Incarnations = nil
	next.Revision = 1
	if err := s.replaceSlotCurrent(observed, next); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.archivePath(old.Address)); err != nil {
		t.Fatal(err)
	}
	if err := s.replaceSlotCurrent(observed, next); err == nil {
		t.Fatal("stale observation overwrote current")
	}
}

func TestSlotFreshTransactionPreservesCorruptBytesAndRejectsFuture(t *testing.T) {
	for _, raw := range []string{"broken", `{}`, `{"schema_version":0}`, `{"schema_version":999}`} {
		t.Run(raw, func(t *testing.T) {
			s := testLocalThreadStore(t)
			if err := os.MkdirAll(s.root, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.root, "thread.json"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			observed, err := s.observeSlotCurrent()
			if err != nil {
				t.Fatal(err)
			}
			next := validThreadRecord(t)
			next.StartingPath = s.slot.WorktreeRoot
			next.WorkingPath = next.StartingPath
			err = s.replaceSlotCurrent(observed, next)
			if raw == `{"schema_version":999}` {
				if err == nil {
					t.Fatal("overwrote unsupported version")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			files, err := os.ReadDir(filepath.Join(s.root, "recovery"))
			if err != nil || len(files) != 1 {
				t.Fatalf("backup %v %v", files, err)
			}
			got, err := os.ReadFile(filepath.Join(s.root, "recovery", files[0].Name()))
			if err != nil || string(got) != raw {
				t.Fatal("corrupt evidence lost")
			}
		})
	}
}

func TestObserveSlotSessionsIncludesLostRecordArtifacts(t *testing.T) {
	s := testLocalThreadStore(t)
	env := newTestEnv(t, s.slot.WorktreeRoot)
	env.Couch.Threads = s
	scope, _ := launcher.ResolveRepoScope(s.slot.WorktreeRoot)
	address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-0000000000000001"}
	env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionPresent})
	observed, err := env.Couch.ObserveSlotSessions(context.Background(), *s.slot)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Absent || len(observed.Candidates) != 1 {
		t.Fatalf("lost live candidate %+v", observed)
	}
	env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error { return errors.New("unavailable") }
	if _, err := env.Couch.ObserveSlotSessions(context.Background(), *s.slot); err == nil {
		t.Fatal("unknown observer authorized absence")
	}
}

func slotRecoveryOperationFixture(t *testing.T) (*testEnv, *ThreadStore) {
	t.Helper()
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	env := newTestEnv(t, f.host(1))
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Git.replies[GitCall{Dir: f.host(1), Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
	repository, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.Threads.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(env.Couch.Namespace, repository.Slots[0].Identity)
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) { return slotReadyResult(local), nil })
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	return env, local
}

func TestStartFreshSlotReplacesStoppedCurrentAndKeepsPreferences(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	old := validThreadRecord(t)
	old.Address.RepoScope = scope.Key
	old.StartingPath = local.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	old.Name = "durable name"
	old.Description = "durable description"
	old, err := local.CreateThread(old)
	if err != nil {
		t.Fatal(err)
	}
	saved := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: local.slot.RepoIdentity, PhysicalPath: old.StartingPath, LastAgent: "claude", ArgvByAgent: map[string][]string{"claude": {"--verbose"}, "codex": {"--no-alt-screen"}}, Revision: 1}
	preferenceRaw, _ := json.Marshal(saved)
	if err := os.WriteFile(filepath.Join(local.root, "preferences.json"), preferenceRaw, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := env.Couch.StartFreshSlot(context.Background(), old.StartingPath, "claude")
	if err != nil {
		t.Fatal(err)
	}
	next, err := local.GetThread(result.Record.Thread)
	if err != nil {
		t.Fatal(err)
	}
	preference, found, err := local.GetPathLaunchPreference(local.slot.RepoIdentity, local.slot.WorktreeRoot)
	if err != nil || !found || len(preference.ArgvByAgent["codex"]) != 1 || preference.ArgvByAgent["codex"][0] != "--no-alt-screen" || len(result.Record.Args.ExtraArgs) != 1 || result.Record.Args.ExtraArgs[0] != "--verbose" {
		t.Fatalf("preferences lost %+v %v", preference, err)
	}
	if next.Address == old.Address || next.Name != old.Name || next.Description != old.Description || next.PublishedSummary != "" {
		t.Fatalf("fresh identity %+v", next)
	}
	if _, err := os.Stat(local.archivePath(old.Address)); err != nil {
		t.Fatal(err)
	}
	if result.Handle == nil || len(env.Runner.Ops) == 0 {
		t.Fatal("fresh never launched")
	}
	child := env.Runner.Child(result.Handle.ID())
	if childEnvValue(child.Env, launcher.CouchLaunchProfileEnv) == "" {
		t.Fatal("missing trusted profile")
	}
}

func TestSlotOpenDoesNotFreshOnLostRecordAndFreshRefusesUncertainty(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	if _, err := env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("missing current silently launched")
	}
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	lost := ThreadAddress{RepoScope: scope.Key, Tag: "lost"}
	env.Artifacts.SetSessionPresence(lost, SessionObservation{State: SessionPresent})
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fresh abandoned surviving session")
	}
	env.Artifacts.SetSessionPresence(lost, SessionObservation{State: SessionUnresolved})
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fresh accepted unknown presence")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("refusal spawned child")
	}
}

func TestSlotFreshRefusesMutationDuringObservation(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	old := validThreadRecord(t)
	old.Address.RepoScope = scope.Key
	old.StartingPath = local.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	old, err := local.CreateThread(old)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error {
		_, err := local.updateExistingThread(old.Address, old.Revision, func(r *ThreadRecord) error { r.Name = "changed concurrently"; return nil })
		return err
	}
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fresh overwrote concurrent mutation")
	}
	got, err := local.GetThread(old.Address)
	if err != nil || got.Name != "changed concurrently" {
		t.Fatalf("lost concurrent write %+v %v", got, err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("stale fresh spawned")
	}
}

func TestSlotOpenReconstructsSingleDetachedSurvivor(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
	env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionPresent})
	env.Artifacts.SetPairSession(address, "survivor", true)
	env.Artifacts.SetDetachedSession(address, "survivor")
	result, err := env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if result.Record.Thread != address || result.Handle == nil {
		t.Fatalf("wrong reconstruction %+v", result)
	}
	child := env.Runner.Child(result.Handle.ID())
	if childEnvValue(child.Env, launcher.CouchLaunchProfileEnv) != "" {
		t.Fatal("surviving session got a fresh profile")
	}
}

func TestSlotFreshFailedLaunchRetainsDamagedEvidence(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	if err := os.MkdirAll(local.root, 0700); err != nil {
		t.Fatal(err)
	}
	damaged := []byte("operator evidence")
	if err := os.WriteFile(filepath.Join(local.root, "thread.json"), damaged, 0600); err != nil {
		t.Fatal(err)
	}
	env.Runner.FailNextStart(errors.New("spawn failed"))
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("launch fault missing")
	}
	files, err := os.ReadDir(filepath.Join(local.root, "recovery"))
	if err != nil || len(files) != 1 {
		t.Fatalf("lost raw evidence %v %v", files, err)
	}
	got, err := os.ReadFile(filepath.Join(local.root, "recovery", files[0].Name()))
	if err != nil || string(got) != string(damaged) {
		t.Fatal("raw evidence changed")
	}
}

func TestSlotFreshBackupCapAndJournalRecovery(t *testing.T) {
	t.Run("cap", func(t *testing.T) {
		s := testLocalThreadStore(t)
		if err := os.MkdirAll(filepath.Join(s.root, "recovery"), 0700); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 16; n++ {
			if err := os.WriteFile(filepath.Join(s.root, "recovery", fmt.Sprintf("old-%d", n)), []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(s.root, "thread.json"), []byte("damaged"), 0600); err != nil {
			t.Fatal(err)
		}
		old, err := s.observeSlotCurrent()
		if err != nil {
			t.Fatal(err)
		}
		next := validThreadRecord(t)
		next.StartingPath = s.slot.WorktreeRoot
		next.WorkingPath = next.StartingPath
		if err := s.replaceSlotCurrent(old, next); err == nil {
			t.Fatal("recovery exceeded bound")
		}
		got, _ := os.ReadFile(filepath.Join(s.root, "thread.json"))
		if string(got) != "damaged" {
			t.Fatal("cap lost evidence")
		}
	})
	t.Run("journal", func(t *testing.T) {
		s := testLocalThreadStore(t)
		old := validThreadRecord(t)
		old.StartingPath = s.slot.WorktreeRoot
		old.WorkingPath = old.StartingPath
		old, err := s.CreateThread(old)
		if err != nil {
			t.Fatal(err)
		}
		observed, err := s.observeSlotCurrent()
		if err != nil {
			t.Fatal(err)
		}
		next := old
		next.Address.Tag = "couch-1111111111111111"
		s.hooks.AfterJournal = func() error { return errors.New("interrupted") }
		if err := s.replaceSlotCurrent(observed, next); err == nil {
			t.Fatal("fault not reached")
		}
		s.hooks = threadStoreHooks{}
		if err := s.RecoverStoreJournal(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetThread(next.Address); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(s.archivePath(old.Address)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSlotOpenRefusesCompetingLiveSurvivor(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	first := ThreadAddress{RepoScope: scope.Key, Tag: "first"}
	second := ThreadAddress{RepoScope: scope.Key, Tag: "second"}
	env.Artifacts.SetSessionPresence(first, SessionObservation{State: SessionPresent})
	env.Artifacts.SetPairSession(first, "first-session", true)
	env.Artifacts.SetDetachedSession(first, "first-session")
	env.Artifacts.SetSessionPresence(second, SessionObservation{State: SessionPresent})
	env.Artifacts.SetPairSession(second, "attached-session", true)
	if _, err := env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("reconstruction ignored competing live owner")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("competing recovery launched")
	}
}

func TestSlotFreshRechecksOldOwnershipAfterReadiness(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	old := ThreadAddress{RepoScope: scope.Key, Tag: "surprise"}
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		env.Artifacts.SetSessionPresence(old, SessionObservation{State: SessionPresent})
		return slotReadyResult(local), nil
	})
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fresh launched after prior owner appeared")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("readiness race launched child")
	}
}

func TestSlotFreshRefusesReplacedHost(t *testing.T) {
	s := testLocalThreadStore(t)
	old, err := s.observeSlotCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(s.slot.WorktreeRoot, s.slot.WorktreeRoot+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(s.slot.WorktreeRoot, 0700); err != nil {
		t.Fatal(err)
	}
	next := validThreadRecord(t)
	next.StartingPath = s.slot.WorktreeRoot
	next.WorkingPath = next.StartingPath
	if err := s.replaceSlotCurrent(old, next); err == nil {
		t.Fatal("fresh accepted replaced physical host")
	}
}

func TestSlotFreshRetryAfterFailedLaunchWithLiveSupervisor(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	env.Runner.FailNextStart(errors.New("spawn failed"))
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fault missing")
	}
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err != nil {
		t.Fatalf("failed start trapped slot: %v", err)
	}
}

func TestSlotNativeScopeScannerIncludesClaimsAndHistory(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := "0123456789abcdef"
	paths := launcher.NewScopedPaths(root, launcher.RepoScope{Key: scope}, "orphan")
	if err := os.MkdirAll(filepath.Dir(paths.ThreadClaim()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ThreadClaim(), []byte("retained ownership"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(paths.ThreadClaim()), "ledger-history.jsonl"), []byte("evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	source := NewScopedThreadArtifactCollisionChecker(root)
	got, err := source.SlotSessionCandidates(context.Background(), scope)
	if err != nil || len(got) != 2 {
		t.Fatalf("scope candidates %+v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(paths.ThreadClaim()), "session-names.jsonl"), []byte("broken binding"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.SlotSessionCandidates(context.Background(), scope); err == nil {
		t.Fatal("unreadable binding index treated as empty")
	}
}

func TestSlotFreshPreservesExistingArchiveWhenRecoveredCurrentRetires(t *testing.T) {
	s := testLocalThreadStore(t)
	old := validThreadRecord(t)
	old.StartingPath = s.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	old, err := s.CreateThread(old)
	if err != nil {
		t.Fatal(err)
	}
	previous := []byte("retained earlier archive bytes")
	if err := writeAtomicBytes(s.archivePath(old.Address), previous); err != nil {
		t.Fatal(err)
	}
	observed, err := s.observeSlotCurrent()
	if err != nil {
		t.Fatal(err)
	}
	next := old
	next.Address.Tag = "couch-1111111111111111"
	if err := s.replaceSlotCurrent(observed, next); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(filepath.Join(s.root, "recovery"))
	if err != nil || len(files) != 1 {
		t.Fatalf("previous archive evidence %v %v", files, err)
	}
	got, err := os.ReadFile(filepath.Join(s.root, "recovery", files[0].Name()))
	if err != nil || string(got) != string(previous) {
		t.Fatal("earlier archive lost")
	}
}

func TestSlotFreshRefusesUnreadableDurableRegistry(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	if err := os.WriteFile(env.Couch.Store.registryPath(), []byte("unreadable registry"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("fresh ignored unreadable hosted registry")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("registry uncertainty spawned child")
	}
}

func TestSlotFreshRetainsProcessEvidenceFromDamagedRecord(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	record := validThreadRecord(t)
	record.Address.RepoScope = scope.Key
	record.StartingPath = local.slot.WorktreeRoot
	record.WorkingPath = record.StartingPath
	record, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := env.Proc.Current()
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	if _, err := local.CommitStartClaim(record.Address, record.Revision, local.slot.RepoIdentity, env.Now, StartEvent{Kind: StartClaimed, Nonce: "still-preparing", Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &profile}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(local.root, "thread.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = append([]byte(`{"unknown":true,`), raw[1:]...)
	if err := os.WriteFile(filepath.Join(local.root, "thread.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude"); err == nil {
		t.Fatal("damaged record hid live starting owner")
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("damaged record spawned second agent")
	}
}

func TestSlotFreshSkipsLocalConversationTagWithoutNativeMarker(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	old := validThreadRecord(t)
	old.Address = ThreadAddress{RepoScope: scope.Key, Tag: "couch-0000000000000000"}
	old.StartingPath = local.slot.WorktreeRoot
	old.WorkingPath = old.StartingPath
	if _, err := local.CreateThread(old); err != nil {
		t.Fatal(err)
	}
	entropy := append(make([]byte, 16), bytes.Repeat([]byte{1}, 8)...)
	env.Couch.Entropy = bytes.NewReader(entropy)
	result, err := env.Couch.StartFreshSlot(context.Background(), local.slot.WorktreeRoot, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if result.Record.Thread == old.Address {
		t.Fatal("fresh reused prior native conversation address")
	}
}
