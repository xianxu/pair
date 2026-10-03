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

// dispatchReboot drives `reboot` through the declared operation and the
// live-owner executor -- the production boundary M2's switcher will use.
func dispatchReboot(env *testEnv, args map[string]string) (RebootResult, error) {
	value, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)},
		OperationCall{Name: "reboot", Args: args, Implicit: true, Context: context.Background()})
	result, _ := value.(RebootResult)
	return result, err
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
	env := newTestEnv(t, "/repo")
	old := parkedNamedPrimary(t, env)
	result, err := dispatchReboot(env, rebootArgs(old.Address))
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
	env := newTestEnv(t, "/repo")
	old := parkedNamedPrimary(t, env)
	env.Couch.Threads.hooks.AfterJournal = func() error {
		raw, err := os.ReadFile(env.Couch.Threads.journalPath())
		if err == nil && bytes.Contains(raw, []byte("archive/")) {
			return errors.New("interrupted")
		}
		return nil
	}
	if _, err := dispatchReboot(env, rebootArgs(old.Address)); err == nil {
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
	result, err := dispatchReboot(env, rebootArgs(fresh.Address))
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
	result, err := dispatchReboot(env, rebootArgs(old.Address))
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
			if _, err := dispatchReboot(env, rebootArgs(address)); err == nil {
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
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	env.Couch.Path.(*FakePathOps).Fail("/repo")
	if _, reason, _ := env.Couch.classifyForAction(context.Background(), record.Address); reason != ReasonPathMissing {
		t.Fatalf("fixture reason %q, want path-missing", reason)
	}
	result, err := dispatchReboot(env, rebootArgs(record.Address))
	if err != nil {
		t.Fatal(err)
	}
	if !result.ArchiveOnly || result.Reason != RebootDirectoryMissing || result.Archived != record.Address {
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
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	corrupt := []byte("{not a record")
	if err := os.WriteFile(env.Couch.Threads.recordPath(record.Address), corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := dispatchReboot(env, rebootArgs(record.Address))
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
	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	result, err := dispatchReboot(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
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
	env, local := slotRecoveryOperationFixture(t)
	old := slotRecordFixture(t, env, local)
	env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
	env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
	result, err := dispatchReboot(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
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
	env := newTestEnv(t, "/repo")
	old := detachedPrimary(t, env)
	args := rebootArgs(old.Address)
	args["agent"] = "no-such-agent"
	if _, err := dispatchReboot(env, args); err == nil {
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
	result, err := dispatchReboot(env, rebootArgs(record.Address))
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

// The Done-when, end to end: resume on a conversation that cannot come back
// says so and names reboot; reboot then yields a live thread with a new tag,
// and the old record is in the archive.
func TestRebootAfterResumeFailedIsUsable(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	_, err := dispatchResume(env, map[string]string{"repo-scope": parked.Address.RepoScope, "tag": string(parked.Address.Tag)})
	if ResumeDiagnosticOf(err) != ResumeBindingUnbound || !strings.Contains(err.Error(), "reboot") {
		t.Fatalf("resume: %v", err)
	}
	result, err := dispatchReboot(env, rebootArgs(parked.Address))
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
	t.Run("primary", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		old := parkedNamedPrimary(t, env)
		env.Runner.FailNextStart(errors.New("spawn failed"))
		if _, err := dispatchReboot(env, rebootArgs(old.Address)); err == nil {
			t.Fatal("launch fault missing")
		}
		_, err := dispatchReboot(env, rebootArgs(old.Address))
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
		if _, err := dispatchReboot(env, args); err == nil {
			t.Fatal("launch fault missing")
		}
		result, err := dispatchReboot(env, args)
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
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	got := retiredResult(address, retirement{Record: ThreadRecord{Address: address}, Unreadable: true, SessionNotStopped: true})
	if got.Archived != address || !got.SessionNotStopped || got.Warning() == "" {
		t.Fatalf("unreadable retirement = %+v", got)
	}
	if got := retiredResult(address, retirement{RolledBack: true}); got.Archived != (ThreadAddress{}) || got.SessionNotStopped {
		t.Fatalf("rolled-back retirement = %+v", got)
	}
}
