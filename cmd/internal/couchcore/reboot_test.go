package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// rebootLegacyForTest preserves coverage of the retained direct Reboot API:
// archive-only recovery, explicit profile overrides and its live idle verdict.
// Public dispatch now uses RecoverSelection; its new confirmation and strict
// Fresh readiness contract is exercised by recovery_prepared/acceptance tests.
func rebootLegacyForTest(env *testEnv, args map[string]string) (RebootResult, error) {
	target := RebootTarget{Path: args["path"], Agent: args["agent"], RequireSettled: args[requireSettledArg]}
	if target.Path == "" {
		target.Address = ThreadAddress{RepoScope: args["repo-scope"], Tag: ThreadTag(args["tag"])}
	}
	return env.Couch.Reboot(context.Background(), target)
}

func rebootArgs(address ThreadAddress) map[string]string {
	return map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag)}
}

// parkedNamedPrimary is a parked :0 whose conversation resolves, carrying the
// stored name and description reboot must leave in the archive.
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

// detachedPrimary is a :0 whose agent survives behind a client-less session
// couch is not hosting.
func detachedPrimary(t *testing.T, env *testEnv) ThreadRecord {
	t.Helper()
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	env.Artifacts.SetPairSession(record.Address, "pair-detached", true)
	env.Artifacts.SetDetachedSession(record.Address, "pair-detached")
	if state, reason, err := env.Couch.classifyForAction(context.Background(), record.Address); err != nil || state != ThreadDetached {
		t.Fatalf("fixture classifies %s/%s (%v), want detached", state, reason, err)
	}
	return record
}

func assertArchivedOnce(t *testing.T, store *ThreadStore, address ThreadAddress, name string) {
	t.Helper()
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Address != address || archived[0].Name != name {
		t.Fatalf("archive = %+v, %v; want exactly %v (name %q)", archived, err, address, name)
	}
}

// workingSet is every address the main store's manifest lists. Each fixture
// here holds a single :0, so "only the fresh address" is the whole claim.
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

func TestRebootPrimaryParkedArchivesAndStartsFreshInOneJournal(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	old := parkedNamedPrimary(t, env)
	result, err := rebootLegacyForTest(env, rebootArgs(old.Address))
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == old.Address || result.Archived != old.Address || result.ArchiveOnly {
		t.Fatalf("reboot result %+v", result)
	}
	next, err := env.Couch.Threads.GetThread(started.Record.Thread)
	if err != nil {
		t.Fatal(err)
	}
	if next.StartingPath != old.StartingPath || next.Name != "" || next.Description != "" {
		t.Fatalf("fresh record %+v", next)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	if got := workingSet(t, env.Couch.Threads); len(got) != 1 || got[0] != next.Address {
		t.Fatalf("manifest lists %v, want only %v", got, next.Address)
	}
	if len(env.Runner.Ops) == 0 {
		t.Fatal("no child was spawned")
	}
}

// leakFirstClaim models a claim a dead couch left behind: the first address
// the next start proposes is already held by someone else's artifacts.
type leakFirstClaim struct {
	*FakeThreadArtifactCollisionChecker
	leaked *ThreadAddress
}

func (l *leakFirstClaim) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	if l.leaked == nil {
		l.leaked = &address
		return nil, launcher.ErrThreadAddressClaimed
	}
	return l.FakeThreadArtifactCollisionChecker.Claim(address)
}

// A crash after the replace journal is durable is completed by replay: the old
// record is archived and the fresh claimed one exists, but no agent was ever
// launched, so the fresh record must never read as live. And a death BETWEEN
// the artifact claim and the journal leaks that claim (the window
// AllocateThreadTag has always had), which must not wedge the path: the next
// reboot moves past the leaked address and still starts.
func TestRebootPrimaryCrashAfterJournalRecovers(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	old := parkedNamedPrimary(t, env)
	env.Couch.Threads.hooks.AfterJournal = func() error {
		raw, err := os.ReadFile(env.Couch.Threads.journalPath())
		if err == nil && bytes.Contains(raw, []byte("archive/")) {
			return errors.New("interrupted")
		}
		return nil
	}
	if _, err := rebootLegacyForTest(env, rebootArgs(old.Address)); err == nil {
		t.Fatal("the interrupted reboot reported success")
	}
	env.Couch.Threads.hooks = threadStoreHooks{}
	if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	scoped := workingSet(t, env.Couch.Threads)
	if len(scoped) != 1 || scoped[0] == old.Address {
		t.Fatalf("replay did not publish exactly the fresh claimed record: %v", scoped)
	}
	fresh, err := env.Couch.Threads.GetThread(scoped[0])
	if err != nil || !startClaimed(fresh) {
		t.Fatalf("fresh record %+v (%v) does not carry its start claim", fresh, err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("runner ops %v: an agent launched through a crash", env.Runner.Ops)
	}
	if state, _, err := env.Couch.classifyForAction(context.Background(), fresh.Address); err != nil || state == ThreadLive {
		t.Fatalf("the never-launched fresh record classifies %s (%v)", state, err)
	}

	// The couch that made the fresh claim died in the crash.
	env.Proc.Kill(fresh.Incarnations[0].Start.OwnerPID)
	leak := &leakFirstClaim{FakeThreadArtifactCollisionChecker: env.Artifacts}
	env.Couch.Artifacts = leak
	result, err := rebootLegacyForTest(env, rebootArgs(fresh.Address))
	if err != nil {
		t.Fatalf("reboot after a crash and a leaked claim: %v", err)
	}
	started, ok := result.Started()
	if !ok || leak.leaked == nil || started.Record.Thread == *leak.leaked {
		t.Fatalf("reboot did not move past the leaked claim: %+v (leaked %v)", result, leak.leaked)
	}
}

// Detached means an agent is running. It is stopped BEFORE the journal that
// retires its record, never after: the other order is a record in the archive
// with a live session behind it.
func TestRebootDetachedStopsTheSessionFirst(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	old := detachedPrimary(t, env)
	quiescedAtReplace := -1
	env.Couch.Threads.hooks.AfterJournal = func() error {
		raw, err := os.ReadFile(env.Couch.Threads.journalPath())
		if err == nil && bytes.Contains(raw, []byte("archive/")) {
			quiescedAtReplace = len(env.Artifacts.Quiesces())
		}
		return nil
	}
	result, err := rebootLegacyForTest(env, rebootArgs(old.Address))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Started(); !ok {
		t.Fatalf("reboot started nothing: %+v", result)
	}
	if quiescedAtReplace < 1 {
		t.Fatalf("the replace journal ran with %d quiesces behind it; the session must stop first", quiescedAtReplace)
	}
	assertArchivedOnce(t, env.Couch.Threads, old.Address, "")
}

func TestRebootRefusesLiveBusyAndStillUnknown(t *testing.T) {
	t.Parallel()

	cases := map[string]func(t *testing.T) (*testEnv, ThreadAddress){
		"live": func(t *testing.T) (*testEnv, ThreadAddress) {
			env, source := switchEnvWithLiveThread(t)
			return env.testEnv, source.Address
		},
		"busy": func(t *testing.T) (*testEnv, ThreadAddress) {
			env := newTestEnv(t, "/repo")
			record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
			if _, err := env.Couch.Threads.CommitStartClaim(record.Address, record.Revision, "/repo/.git", time.Unix(200, 0).UTC(), StartEvent{
				Kind: StartClaimed, Nonce: "start-0123456789abcdef", Shape: StartFreshExisting,
				Owner: SupervisorOwner{PID: 77, Identity: "owner-couch"}, Profile: record.LatestLaunchProfile,
			}); err != nil {
				t.Fatal(err)
			}
			env.Proc.Set(77, "owner-couch")
			return env, record.Address
		},
		"unknown": func(t *testing.T) (*testEnv, ThreadAddress) {
			env, source := switchEnvWithLiveThread(t)
			env.Proc.SetUnknown(42)
			return env.testEnv, source.Address
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			env, address := setup(t)
			ops := len(env.Runner.Ops)
			if _, err := rebootLegacyForTest(env, rebootArgs(address)); err == nil {
				t.Fatal("reboot accepted it")
			}
			if archived, err := env.Couch.Threads.ArchivedThreads(); err != nil || len(archived) != 0 {
				t.Fatalf("a refused reboot archived %+v (%v)", archived, err)
			}
			if len(env.Runner.Ops) != ops || len(env.Artifacts.Quiesces()) != 0 {
				t.Fatalf("a refused reboot acted: ops %v, quiesces %v", env.Runner.Ops[ops:], env.Artifacts.Quiesces())
			}
		})
	}
}

func TestRebootDirectoryMissingArchivesOnly(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	env.Couch.Path.(*FakePathOps).Fail("/repo")
	if _, reason, _ := env.Couch.classifyForAction(context.Background(), record.Address); reason != ReasonPathMissing {
		t.Fatalf("fixture reason %q, want path-missing", reason)
	}
	result, err := rebootLegacyForTest(env, rebootArgs(record.Address))
	if err != nil {
		t.Fatal(err)
	}
	if !result.ArchiveOnly || result.Reason != RebootCheckoutMissing || result.Archived != record.Address {
		t.Fatalf("result %+v", result)
	}
	if _, ok := result.Started(); ok || len(env.Runner.Ops) != 0 {
		t.Fatalf("a missing directory got an agent: %v", env.Runner.Ops)
	}
	assertArchivedOnce(t, env.Couch.Threads, record.Address, "")
}

// An unreadable :0 is retired as bytes, and its session is left alone: couch
// does not stop what it cannot identify.
func TestRebootUnreadablePrimaryArchivesOnly(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	corrupt := []byte("{not a record")
	if err := os.WriteFile(env.Couch.Threads.recordPath(record.Address), corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := rebootLegacyForTest(env, rebootArgs(record.Address))
	if err != nil {
		t.Fatal(err)
	}
	if !result.ArchiveOnly || !result.SessionNotStopped || result.Reason == "" {
		t.Fatalf("result %+v", result)
	}
	moved, err := os.ReadFile(env.Couch.Threads.archivePath(record.Address))
	if err != nil || !bytes.Equal(moved, corrupt) {
		t.Fatalf("archived bytes %q (%v), want them as they were", moved, err)
	}
	if len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("an unreadable record was acted on beyond its bytes")
	}
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
	env.Artifacts.SetNativeBinding(old.Address, "claude", sessioninventory.BindingEstablished, "native-slot")
	return old
}

func slotArchiveFiles(t *testing.T, local *ThreadStore) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(local.root, "archive", "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRebootSlotParkedIsTodaysFreshSlot(t *testing.T) {
	t.Parallel()

	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	result, err := rebootLegacyForTest(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == old.Address || result.Archived != old.Address {
		t.Fatalf("result %+v", result)
	}
	next, err := local.GetThread(started.Record.Thread)
	if err != nil || next.Name != "" || next.Description != "" {
		t.Fatalf("fresh slot record %+v (%v)", next, err)
	}
	if _, err := os.Stat(local.archivePath(old.Address)); err != nil {
		t.Fatalf("old slot record not archived: %v", err)
	}
	assertArchivedOnce(t, local, old.Address, "durable name")
}

func TestRebootSlotDetachedQuiescesThenStarts(t *testing.T) {
	t.Parallel()

	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
	env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
	result, err := rebootLegacyForTest(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
	if err != nil {
		t.Fatal(err)
	}
	if got := env.Artifacts.Quiesces(); len(got) != 1 || got[0] != old.Address {
		t.Fatalf("quiesced %v, want exactly the slot's old thread", got)
	}
	if started, ok := result.Started(); !ok || started.Record.Thread == old.Address {
		t.Fatalf("result %+v", result)
	}
	assertArchivedOnce(t, local, old.Address, "durable name")
}

// The fresh profile resolves before anything is stopped, so an agent that
// cannot launch leaves the running one alone.
func TestRebootProfileFailureStopsNothing(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	old := detachedPrimary(t, env)
	args := rebootArgs(old.Address)
	args["agent"] = "no-such-agent"
	if _, err := rebootLegacyForTest(env, args); err == nil {
		t.Fatal("reboot accepted an agent that cannot launch")
	}
	if got := env.Artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("quiesced %v before the profile resolved", got)
	}
	if state, _, err := env.Couch.classifyForAction(context.Background(), old.Address); err != nil || state != ThreadDetached {
		t.Fatalf("the running agent did not survive: %s (%v)", state, err)
	}
	if archived, _ := env.Couch.Threads.ArchivedThreads(); len(archived) != 0 {
		t.Fatalf("archived %+v", archived)
	}
}

// A record holding nothing but a start whose couch died has nothing to
// archive: retirement rolls it back, and reboot starts fresh at its path.
func TestRebootOfARolledBackStartStartsFresh(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	// The husk a launch leaves when it dies before registering: a start claim
	// whose couch is gone, and nothing durable besides (no profile, no name).
	husk := actionableTestThread("couch-00000000000000d1", time.Unix(100, 0).UTC())
	husk.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor"},
	}}
	record, err := env.Couch.Threads.CreateThread(husk)
	if err != nil {
		t.Fatal(err)
	}
	result, err := rebootLegacyForTest(env, rebootArgs(record.Address))
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == record.Address || result.Archived != (ThreadAddress{}) {
		t.Fatalf("result %+v", result)
	}
	next, err := env.Couch.Threads.GetThread(started.Record.Thread)
	if err != nil || next.StartingPath != record.StartingPath {
		t.Fatalf("fresh record %+v (%v)", next, err)
	}
	if archived, _ := env.Couch.Threads.ArchivedThreads(); len(archived) != 0 {
		t.Fatalf("a rolled-back start was archived: %+v", archived)
	}
}

// A rolled-back :0 start is an ordinary fresh primary start, so it passes the
// one-primary guard (#363 M3). A store written before that rule may still hold
// a second, live primary in the same repository; the reboot then refuses with
// the rule's words, creates nothing, and leaves that primary running.
func TestRebootOfARolledBackStartRefusesBesideALegacyPrimary(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.cannedTree("/repo", "/repo/sub")
	// The legacy co-tenant, created past the occupancy guard (no rows) as the
	// older couch that wrote such a store did.
	resolution, err := env.Couch.resolveStartResolution(context.Background(), StartArgs{Worktree: "/repo", Cwd: "/repo/sub"})
	if err != nil {
		t.Fatal(err)
	}
	cotenant, _, err := env.Couch.spawnResolved(context.Background(), resolution, nil)
	if err != nil {
		t.Fatalf("seed the legacy co-tenant: %v", err)
	}
	env.Proc.Set(cotenant.PID, cotenant.Identity)
	husk := actionableTestThread("couch-00000000000000d2", time.Unix(100, 0).UTC())
	husk.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{Nonce: "start-0123456789abcdee", OwnerPID: 4243, OwnerIdentity: "supervisor"},
	}}
	record, err := env.Couch.Threads.CreateThread(husk)
	if err != nil {
		t.Fatal(err)
	}
	before, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := rebootLegacyForTest(env, rebootArgs(record.Address))
	if err == nil || !strings.Contains(err.Error(), "one primary slot per repository") {
		t.Fatalf("reboot beside a legacy primary: %+v, %v; want the one-primary refusal", result, err)
	}
	if _, ok := result.Started(); ok {
		t.Fatalf("a refused reboot started a thread: %+v", result)
	}
	after, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Records) != len(before.Records)-1 {
		t.Fatalf("threads %d -> %d; want only the rolled-back husk gone", len(before.Records), len(after.Records))
	}
	if state, reason, err := env.Couch.classifyForAction(context.Background(), cotenant.Thread); err != nil || state != ThreadLive {
		t.Fatalf("the legacy primary classifies %s/%s (%v), want live and untouched", state, reason, err)
	}
}

// The Done-when, end to end: resume on a conversation that cannot come back
// says so and names reboot; reboot then yields a live thread with a new tag,
// and the old record is in the archive.
func TestRebootAfterResumeFailedIsUsable(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	_, err := env.Couch.ResumeTarget(context.Background(), ResumeTarget{Address: parked.Address})
	if ResumeDiagnosticOf(err) != ResumeBindingUnbound || !strings.Contains(err.Error(), "reboot") {
		t.Fatalf("resume: %v", err)
	}
	result, err := rebootLegacyForTest(env, rebootArgs(parked.Address))
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == parked.Address {
		t.Fatalf("result %+v", result)
	}
	env.Proc.Set(started.Record.PID, started.Record.Identity)
	if state, reason, err := env.Couch.classifyForAction(context.Background(), started.Record.Thread); err != nil || state != ThreadLive {
		t.Fatalf("the rebooted thread classifies %s/%s (%v), want live", state, reason, err)
	}
	assertArchivedOnce(t, env.Couch.Threads, parked.Address, "")
}

// The retry after a reboot whose launch failed: the old record is already
// archived. A :0 tag is gone, so reboot says there is nothing to reboot; a slot
// persists, so it starts clean. Either way the old record is archived once.
func TestRebootAfterFailedLaunchDoesNotArchiveTwice(t *testing.T) {
	t.Parallel()

	t.Run("primary", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		old := parkedNamedPrimary(t, env)
		env.Runner.FailNextStart(errors.New("spawn failed"))
		if _, err := rebootLegacyForTest(env, rebootArgs(old.Address)); err == nil {
			t.Fatal("launch fault missing")
		}
		_, err := rebootLegacyForTest(env, rebootArgs(old.Address))
		if err == nil || !strings.Contains(err.Error(), "nothing to reboot") {
			t.Fatalf("retry with the archived tag: %v", err)
		}
		assertArchivedOnce(t, env.Couch.Threads, old.Address, "durable name")
	})
	t.Run("slot", func(t *testing.T) {
		env, local := slotRecoveryOperationFixture(t)
		old := slotRecordFixture(t, env, local)
		args := map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope}
		env.Runner.FailNextStart(errors.New("spawn failed"))
		if _, err := rebootLegacyForTest(env, args); err == nil {
			t.Fatal("launch fault missing")
		}
		result, err := rebootLegacyForTest(env, args)
		if err != nil {
			t.Fatalf("slot retry: %v", err)
		}
		if _, ok := result.Started(); !ok {
			t.Fatalf("slot retry started nothing: %+v", result)
		}
		if files := slotArchiveFiles(t, local); len(files) != 1 {
			t.Fatalf("archive files %v, want exactly one", files)
		}
	})
}

// Both kinds report the retirement through retiredResult (pair#363 M1 review:
// rebootSlot dropped SessionNotStopped). A slot record can turn unreadable
// only in the window between the preflight's read and retirement's, which no
// fake seam reaches, so the shared construction is what is pinned.
func TestRetiredResultCarriesWhatRetirementDidNotDo(t *testing.T) {
	t.Parallel()

	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	got := retiredResult(address, retirement{Record: ThreadRecord{Address: address}, Unreadable: true, SessionNotStopped: true})
	if got.Archived != address || !got.SessionNotStopped || got.Warning() == "" {
		t.Fatalf("unreadable retirement = %+v", got)
	}
	if got := retiredResult(address, retirement{RolledBack: true}); got.Archived != (ThreadAddress{}) || got.SessionNotStopped {
		t.Fatalf("rolled-back retirement = %+v", got)
	}
}

// pair#456: a live thread whose conversation never bound -- relaunch refuses
// it and names reboot; reboot with admission's verdict parks it and starts a
// fresh agent; a busy re-check refuses before anything stops.
func TestRebootLiveUnboundThread(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*relaunchEnv, ThreadRecord, *fakeLiveRestartProbe) {
		env, live := switchEnvWithLiveThread(t)
		env.Artifacts.SetNativeBinding(live.Address, "claude", sessioninventory.BindingUnbound, "")
		probe := &fakeLiveRestartProbe{}
		env.Couch.LiveRestart = probe
		return env, live, probe
	}
	env, live, probe := setup(t)
	if _, err := env.Couch.Relaunch(context.Background(), live.Address); ResumeDiagnosticOf(err) != ResumeBindingUnbound || !strings.Contains(err.Error(), "reboot") {
		t.Fatalf("relaunch refusal does not name reboot: %v", err)
	}
	args := rebootArgs(live.Address)
	args[requireSettledArg] = settledForcedValue
	result, err := rebootLegacyForTest(env.testEnv, args)
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == live.Address || result.Archived != live.Address {
		t.Fatalf("result %+v", result)
	}
	if len(probe.busyChecks) != 1 || probe.busyChecks[0] != live.Address || !probe.forced[0] {
		t.Fatalf("busy re-check %v forced %v", probe.busyChecks, probe.forced)
	}
	if env.Proc.Exists(42) == Live {
		t.Fatal("the old Pair is still running")
	}
	assertArchivedOnce(t, env.Couch.Threads, live.Address, "")

	env, live, probe = setup(t)
	probe.busyErr = &SlotOperationError{Code: LiveRestartBusy, Detail: "turn open"}
	args = rebootArgs(live.Address)
	args[requireSettledArg] = settledKnownValue
	if _, err := rebootLegacyForTest(env.testEnv, args); slotOperationCode(err) != LiveRestartBusy {
		t.Fatalf("busy: %v", err)
	}
	if current, err := env.Couch.Threads.GetThread(live.Address); err != nil || current.Park != nil || current.VerifiedPark != nil || len(env.Runner.Ops) != 0 {
		t.Fatalf("a refused reboot acted: %+v %v ops %v", current, err, env.Runner.Ops)
	}
}

// liveSlotRebootEnv is liveSlotFixture's live slot with a park that completes
// and a never-bound conversation (pair#456's case).
func liveSlotRebootEnv(t *testing.T) (*relaunchEnv, ThreadRecord, *fakeLiveRestartProbe) {
	t.Helper()
	env, _, live := liveSlotFixture(t)
	renv := withParkLifecycle(t, env, live, 5151, "slot-live")
	env.Artifacts.SetNativeBinding(live.Address, "claude", sessioninventory.BindingUnbound, "")
	probe := &fakeLiveRestartProbe{}
	env.Couch.LiveRestart = probe
	return renv, live, probe
}

// pair#456 BR-2: the slot path (rebootSlot) admits a live slot under its
// hold, parks it, and starts fresh through startFreshSlot's re-observation.
func TestRebootLiveSlotByPath(t *testing.T) {
	t.Parallel()

	env, live, probe := liveSlotRebootEnv(t)
	args := map[string]string{"path": live.StartingPath, "repo-scope": live.Address.RepoScope, requireSettledArg: settledKnownValue}
	result, err := rebootLegacyForTest(env.testEnv, args)
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread == live.Address || result.Archived != live.Address {
		t.Fatalf("result %+v", result)
	}
	if len(probe.busyChecks) != 1 || probe.forced[0] {
		t.Fatalf("busy re-check %v forced %v", probe.busyChecks, probe.forced)
	}
	if env.Proc.Exists(5151) == Live {
		t.Fatal("the old Pair is still running")
	}
	// Without a verdict, the same live slot is refused (recover's route).
	env2, live2, _ := liveSlotRebootEnv(t)
	if _, err := rebootLegacyForTest(env2.testEnv, map[string]string{"path": live2.StartingPath, "repo-scope": live2.Address.RepoScope}); err == nil || env2.Proc.Exists(5151) != Live {
		t.Fatalf("a live slot without a verdict: %v", err)
	}
}

// pair#456 BR-1 (ARCH-ORDER): a live reboot whose fresh profile cannot
// resolve refuses with the agent still running and nothing parked, on both
// kinds.
func TestLiveRebootProfileFailureStopsNothing(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, env *relaunchEnv, live ThreadRecord, args map[string]string, pid int) {
		args["agent"] = "no-such-agent"
		args[requireSettledArg] = settledForcedValue
		if _, err := rebootLegacyForTest(env.testEnv, args); err == nil || !strings.Contains(err.Error(), "nothing was stopped") {
			t.Fatalf("reboot of an unlaunchable agent: %v", err)
		}
		current, err := env.Couch.Threads.GetThread(live.Address)
		if err != nil || current.Park != nil || current.VerifiedPark != nil || env.Proc.Exists(pid) != Live {
			t.Fatalf("the live agent was stopped: %+v %v", current, err)
		}
	}
	t.Run("primary", func(t *testing.T) {
		env, live := switchEnvWithLiveThread(t)
		env.Couch.LiveRestart = &fakeLiveRestartProbe{}
		check(t, env, live, rebootArgs(live.Address), 42)
	})
	t.Run("slot", func(t *testing.T) {
		env, live, _ := liveSlotRebootEnv(t)
		check(t, env, live, map[string]string{"path": live.StartingPath, "repo-scope": live.Address.RepoScope}, 5151)
	})
}

// pair#456: both binding refusals a fresh session gets name reboot; others
// pass through unchanged.
func TestWithRebootHint(t *testing.T) {
	t.Parallel()

	for code, hinted := range map[ResumeDiagnosticCode]bool{ResumeBindingUnbound: true, ResumeBindingProvisional: true, ResumeBindingAmbiguous: false, ResumePathMissing: false} {
		err := withRebootHint(refuseResume(code, "why"))
		if ResumeDiagnosticOf(err) != code || strings.Contains(err.Error(), "couch --reboot") != hinted {
			t.Errorf("%s: %v", code, err)
		}
	}
}
