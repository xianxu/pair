package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func primaryRebootFixture(t *testing.T) (*testEnv, ThreadRecord) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	old := parkedNamedPrimary(t, env)
	env.Artifacts.SetPairSession(old.Address, "old-session", false)
	readyNonce := ""
	var readyAddress ThreadAddress
	env.Couch.FreshRegistration = func(_ context.Context, address ThreadAddress, _ string, nonce string) (bool, error) {
		return nonce != "" && nonce == readyNonce && address == readyAddress, nil
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		requireLauncherAcceptsRecovery(t, env.Runner, id)
		child := env.Runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
			return err
		}
		readyAddress = ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(profile.Tag)}
		readyNonce = profile.FreshNonce
		env.Artifacts.SetPairSession(readyAddress, continuationChildSession(t, env.Runner, id), true)
		env.Artifacts.SetSessionPresence(readyAddress, SessionObservation{State: SessionPresent})
		return nil
	}
	return env, old
}

func TestRebootPrimaryUsesSharedFreshJournalAndNotice(t *testing.T) {
	env, old := primaryRebootFixture(t)
	result, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || result.Archived != old.Address || started.Record.Thread == old.Address || result.Notice == "" || result.Warning() == "" {
		t.Fatalf("shared result lost: %+v", result)
	}
	next, err := env.Couch.Threads.GetThread(started.Record.Thread)
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "" || next.Description != "" || next.RecoveryIntent == nil || next.WorkingPath != old.WorkingPath {
		t.Fatalf("invalid fresh record: %+v", next)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	if active := workingSet(t, env.Couch.Threads); len(active) != 1 || active[0] != next.Address {
		t.Fatalf("working set %v", active)
	}
}

func TestRebootReadableSourceUsesSharedConfirmation(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Proc.Set(700, "owned-server")
	f.Proc.DiesOn[700] = os.Kill
	retired := 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Stops: []RecoveryStopProcess{{PID: 700, Identity: "owned-server", Role: "terminal server"}}, Digest: "owned", Retire: func(context.Context) error { retired++; return f.Proc.Signal(700, os.Kill) }}, nil
	}
	ctx := WithRecoveryConfirmer(t.Context(), "operator")
	_, err := f.Couch.Reboot(ctx, RebootTarget{Address: f.record.Address})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) || retired != 0 || len(f.Runner.Ops) != 0 || f.preserved != 0 {
		t.Fatalf("facade bypassed shared confirmation: %v", err)
	}
	result, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token)
	if err != nil || result.Start.Handle == nil || retired != 1 {
		t.Fatalf("confirmed shared launch %+v %v", result, err)
	}
	if _, err := f.Couch.ConfirmRecovery(ctx, required.Offer.Token); err == nil {
		t.Fatal("confirmed launch executed twice")
	}
}

func TestRebootPrimaryCrashAfterJournalRecoversSharedClaim(t *testing.T) {
	env, old := primaryRebootFixture(t)
	env.Couch.Threads.hooks.AfterJournal = func() error {
		raw, err := os.ReadFile(env.Couch.Threads.journalPath())
		if err == nil && bytes.Contains(raw, []byte("archive/")) {
			return errors.New("interrupted")
		}
		return nil
	}
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address}); err == nil {
		t.Fatal("interrupted journal reported success")
	}
	env.Couch.Threads.hooks = threadStoreHooks{}
	if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	active := workingSet(t, env.Couch.Threads)
	if len(active) != 1 || active[0] == old.Address {
		t.Fatalf("shared journal working set %v", active)
	}
	next, err := env.Couch.Threads.GetThread(active[0])
	if err != nil || !startClaimed(next) || next.Incarnations[0].Start.RecoveryIntent == nil {
		t.Fatalf("lost shared recovery claim: %+v %v", next, err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatal("launch crossed interrupted journal")
	}
}

func TestRebootArchiveEscapePreservesBytesAndStopsNothing(t *testing.T) {
	for _, unreadable := range []bool{false, true} {
		t.Run(fmt.Sprint(unreadable), func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
			path := env.Couch.Threads.recordPath(record.Address)
			if unreadable {
				if err := os.WriteFile(path, []byte(`{"schema_version":99,"nope":`), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				env.Couch.Path.(*FakePathOps).Fail(record.StartingPath)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			env.Proc.Set(700, "untouched")
			env.Artifacts.SetPairSession(record.Address, "unproved-terminal", true)
			result, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: record.Address})
			if err != nil {
				t.Fatal(err)
			}
			if !result.ArchiveOnly || !result.SessionNotStopped || result.Reason == "" || result.Warning() == "" || result.Start.Handle != nil {
				t.Fatalf("archive-only result %+v", result)
			}
			archived, err := os.ReadFile(env.Couch.Threads.archivePath(record.Address))
			if err != nil {
				t.Fatal(err)
			}
			// Undecodable bytes leave exactly; a readable record proven down
			// carries only its proof time on top (pair#695).
			if unreadable && !bytes.Equal(raw, archived) {
				t.Fatal("archive changed the unreadable bytes")
			} else if !unreadable {
				assertRetiredBytes(t, archived, raw)
			}
			if len(env.Runner.Ops) != 0 || len(env.Proc.Signals) != 0 || len(env.Artifacts.Quiesces()) != 0 || env.Proc.Exists(700) != Live {
				t.Fatal("archive escape acted on source")
			}
		})
	}
}

func TestRebootArchiveEscapeJournalReplaysExactBytes(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000002")
	raw := []byte("{truncated bytes")
	if err := os.WriteFile(env.Couch.Threads.recordPath(record.Address), raw, 0600); err != nil {
		t.Fatal(err)
	}
	env.Couch.Threads.hooks.AfterJournal = func() error { return errors.New("interrupted archive") }
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: record.Address}); err == nil {
		t.Fatal("interrupted archive reported success")
	}
	env.Couch.Threads.hooks = threadStoreHooks{}
	if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	archived, err := os.ReadFile(env.Couch.Threads.archivePath(record.Address))
	if err != nil || !bytes.Equal(archived, raw) {
		t.Fatal("journal lost unreadable bytes")
	}
	if _, err := env.Couch.Threads.GetThread(record.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("archive remained current: %v", err)
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("archive replay caused process effects")
	}
}

func TestArchiveObservedBytesRefusesChangedUnreadableRecord(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000003")
	before := []byte("{first corrupt record")
	after := []byte("{different corrupt record")
	path := env.Couch.Threads.recordPath(record.Address)
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, after, 0600); err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.Threads.ArchiveThreadExpectedBytes(record.Address, before, UnreadableRecordEscape{raw: before}); err == nil {
		t.Fatal("stale unreadable snapshot archived replacement bytes")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, after) {
		t.Fatal("CAS refusal changed bytes")
	}
	if _, err := os.Stat(env.Couch.Threads.archivePath(record.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("CAS refusal published archive")
	}
}

func TestRebootProfileFailureLeavesSourceUntouched(t *testing.T) {
	env, old := primaryRebootFixture(t)
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address, Agent: "unsupported"}); err == nil {
		t.Fatal("invalid explicit profile admitted")
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("profile failure stopped source")
	}
	current, err := env.Couch.Threads.GetThread(old.Address)
	if err != nil || current.Revision != old.Revision {
		t.Fatal("profile failure changed source")
	}
}

func TestRebootArchiveEscapeRefusesOutstandingClaim(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := admittedStartRecord(t)
	record, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "pending", Owner: SupervisorOwner{PID: 77, Identity: "owner"}})
	if err != nil {
		t.Fatal(err)
	}
	record, err = env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.Path.(*FakePathOps).Fail(record.StartingPath)
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: record.Address}); err == nil {
		t.Fatal("archive-only escaped outstanding claim guard")
	}
	current, err := env.Couch.Threads.GetThread(record.Address)
	if err != nil || !startClaimed(current) {
		t.Fatal("refusal discarded claim")
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("refusal stopped source")
	}
}

func parkedNamedPrimary(t *testing.T, env *testEnv) ThreadRecord {
	t.Helper()
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-1")
	name, description := "durable name", "durable description"
	named, err := env.Couch.Threads.ApplyThreadMetadata(parked.Address, parked.Revision, ThreadMetadataPatch{Name: &name, Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	return named
}

func assertArchivedOnce(t *testing.T, store *ThreadStore, address ThreadAddress, name string) {
	t.Helper()
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Address != address || archived[0].Name != name {
		t.Fatalf("archive = %+v, %v; want exactly %v (name %q)", archived, err, address, name)
	}
}

func workingSet(t *testing.T, store *ThreadStore) []ThreadAddress {
	t.Helper()
	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var out []ThreadAddress
	for _, record := range snapshot.Records {
		out = append(out, record.Address)
	}
	return out
}

func slotRecordFixture(t *testing.T, env *testEnv, local *ThreadStore) ThreadRecord {
	t.Helper()
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	old := validThreadRecord(t)
	old.Address.RepoScope = scope.Key
	old.StartingPath, old.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	old.Name, old.Description = "durable name", "durable description"
	old.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	old, err := local.CreateThread(old)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(old.Address, "claude", sessioninventory.BindingEstablished, fmt.Sprintf("native-slot-%d", local.slot.Number))
	return old
}

func TestWithRebootHint(t *testing.T) {
	t.Parallel()

	for code, hinted := range map[ResumeDiagnosticCode]bool{ResumeBindingUnbound: true, ResumeBindingProvisional: true, ResumeBindingAmbiguous: false, ResumePathMissing: false} {
		err := withRebootHint(refuseResume(code, "why"))
		if ResumeDiagnosticOf(err) != code || strings.Contains(err.Error(), "couch --reboot") != hinted {
			t.Errorf("%s: %v", code, err)
		}
	}
}

func TestRebootRefusesReadableFutureSchemaWithoutEffects(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-00000000000000fa")
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(fmt.Sprintf(`"schema_version":%d`, ThreadSchemaVersion)), []byte(`"schema_version":999`), 1)
	path := env.Couch.Threads.recordPath(record.Address)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Reboot(context.Background(), RebootTarget{Address: record.Address}); err == nil {
		t.Fatal("future format was archived")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, raw) {
		t.Fatal("future record changed")
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("future record caused external effects")
	}
}

func TestRebootSharedLaunchFailureDoesNotRetireSourceTwice(t *testing.T) {
	env, old := primaryRebootFixture(t)
	env.Runner.FailNextStart(errors.New("spawn failed"))
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address}); err == nil {
		t.Fatal("missing spawn failure")
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	if _, err := env.Couch.Reboot(t.Context(), RebootTarget{Address: old.Address}); err == nil || !strings.Contains(err.Error(), "nothing to reboot") {
		t.Fatalf("archived source retry %v", err)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
}
