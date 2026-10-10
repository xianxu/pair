package couchcore

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func slotRow(path string, n int, state ActionableThreadState) ActionableThreadSummary {
	scope, _ := launcher.ResolveRepoScope(path)
	return ActionableThreadSummary{Address: ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(fmt.Sprintf("s%d", n))}, State: state, StartingPath: path,
		Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: SlotIdentity{Repo: "pair", Number: n, WorktreeRoot: path}}}
}

func primaryRow(path, tag string, state ActionableThreadState) ActionableThreadSummary {
	scope, _ := launcher.ResolveRepoScope(path)
	return ActionableThreadSummary{Address: ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(tag)}, State: state, StartingPath: path, Target: ThreadTarget{Kind: ThreadTargetOrdinary}}
}

func slotOperationCode(err error) string {
	var e *SlotOperationError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestSelectSlotRow(t *testing.T) {
	primary, host := "/fleet/pair", "/fleet/worktree/pair-slot1/pair"
	root := primaryRow(primary, "root", ThreadParked)
	sub := primaryRow(primary, "sub", ThreadParked)
	sub.StartingPath = filepath.Join(primary, "cmd")
	one := slotRow(host, 1, ThreadParked)
	rows := []ActionableThreadSummary{sub, root, one}
	if got, err := SelectSlotRow(rows, 1, host); err != nil || got.Address != one.Address {
		t.Fatalf(":1 = %+v %v", got, err)
	}
	if got, err := SelectSlotRow(rows, 0, primary); err != nil || got.Address != root.Address {
		t.Fatalf(":0 = %+v %v; the subdirectory thread is never the slot's row", got, err)
	}
	if _, err := SelectSlotRow([]ActionableThreadSummary{sub}, 0, primary); slotOperationCode(err) != SlotOpNoThread {
		t.Fatalf("subdirectory only: %v", err)
	}
	if _, err := SelectSlotRow(rows, 2, "/fleet/worktree/pair-slot2/pair"); slotOperationCode(err) != SlotOpNoThread {
		t.Fatalf("no row: %v", err)
	}
	twin := one
	twin.Address.Tag = "twin"
	if _, err := SelectSlotRow(append(rows, twin), 1, host); slotOperationCode(err) != SlotOpAmbiguous {
		t.Fatalf("two rows: %v", err)
	}
}

// TestActorOperationArgs pins the three shapes the switcher's resume and
// reboot effects carried before the mapping moved to couchcore (literal
// copies of dispatchMenuRow / dispatchThreadOperation / the warm-only block).
func TestActorOperationArgs(t *testing.T) {
	host := "/fleet/worktree/pair-slot1/pair"
	slot := slotRow(host, 1, ThreadDetached)
	parked := primaryRow("/fleet/pair", "root", ThreadParked)
	detached := primaryRow("/fleet/pair", "root", ThreadDetached)
	for _, c := range []struct {
		name string
		row  ActionableThreadSummary
		op   string
		want map[string]string
	}{
		{"slot resume", slot, "resume", map[string]string{"path": host, "repo-scope": slot.Address.RepoScope}},
		{"slot reboot", slot, "reboot", map[string]string{"path": host, "repo-scope": slot.Address.RepoScope}},
		{"parked primary resume", parked, "resume", map[string]string{"repo-scope": parked.Address.RepoScope, "tag": "root"}},
		{"parked primary reboot", parked, "reboot", map[string]string{"repo-scope": parked.Address.RepoScope, "tag": "root"}},
		{"detached primary resume", detached, "resume", map[string]string{"repo-scope": detached.Address.RepoScope, "tag": "root", "warm-only": "true"}},
		{"detached primary reboot", detached, "reboot", map[string]string{"repo-scope": detached.Address.RepoScope, "tag": "root"}},
	} {
		got := ActorOperationArgs(c.row, c.op)
		if !maps.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
		// The production boundary: the declared operation table accepts the
		// arguments (a slot resume once carried only its path, which the
		// declaration's required repo-scope refused).
		reached := false
		stub := func(OperationCall) (any, error) { reached = true; return nil, nil }
		if _, err := DispatchOperation(OperationExecutors{LiveOwner: stub, DirectStore: stub}, OperationCall{Name: c.op, Args: got, Implicit: true}); err != nil || !reached {
			t.Errorf("%s: the declaration refuses %v: %v", c.name, got, err)
		}
	}
}

func TestSlotOperationCommandText(t *testing.T) {
	if got := SlotOperationCommand("resume", "pair:2"); got != "couch --resume pair:2" {
		t.Fatalf("resume = %q", got)
	}
	if got := SlotOperationCommand("reboot", "pair:2"); got != "couch --reboot pair:2 --confirm" {
		t.Fatalf("reboot = %q", got)
	}
}

func TestPrepareSlotOperation(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	address := WorkspaceReference{Repo: local.slot.Repo, Number: local.slot.Number}.String()
	ctx := context.Background()
	if _, _, err := env.Couch.PrepareSlotOperation(ctx, "resume", "nosuchrepo:1", LiveRestartOptions{}); slotOperationCode(err) != SlotOpUnknownSlot {
		t.Fatalf("unknown repo: %v", err)
	}
	if _, _, err := env.Couch.PrepareSlotOperation(ctx, "park", address, LiveRestartOptions{}); err == nil {
		t.Fatal("park admitted as a slot operation")
	}
	// An enrolled slot with no record is offered reboot (never-started).
	slotRecordFixture(t, env, local)
	call, _, err := env.Couch.PrepareSlotOperation(ctx, "resume", address, LiveRestartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	slotScope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if call.Name != "resume" || !call.Implicit || !maps.Equal(call.Args, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": slotScope.Key}) {
		t.Fatalf("call = %+v", call)
	}
	// Live: nothing to resume, and the refusal says why.
	env, local = slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	live := validThreadRecord(t)
	live.Address.RepoScope = scope.Key
	live.StartingPath, live.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	live.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	live.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "slot-live", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: live.LatestLaunchProfile}}
	if _, err := local.CreateThread(live); err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(5151, "slot-live")
	_, _, err = env.Couch.PrepareSlotOperation(ctx, "resume", address, LiveRestartOptions{})
	var refusal *SlotOperationError
	if !errors.As(err, &refusal) || refusal.Code != SlotOpNotOffered || refusal.Detail != address+" is live" {
		t.Fatalf("live slot: %v", err)
	}
}

// A slot parked with no resolvable conversation (its agent never took a turn)
// is not offered resume: the socket refuses at admission, nothing launches,
// and reboot is still offered (pair#367 smoke test).
func TestPrepareSlotOperationRefusesResumeOfALostConversation(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	address := WorkspaceReference{Repo: local.slot.Repo, Number: local.slot.Number}.String()
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.Address.RepoScope = scope.Key
	record.StartingPath, record.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	record.LatestLaunchProfile = &profile
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: &profile}}
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "park-lost", Address: created.Address, PID: 42, ProcessIdentity: "pair-helper"}
	begun, err := local.BeginPark(created.Address, created.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := local.FinalizePark(created.Address, begun.Revision, identity, 1, env.Now); err != nil {
		t.Fatal(err)
	}
	// No native binding: the parked conversation cannot be resolved.
	_, _, err = env.Couch.PrepareSlotOperation(context.Background(), "resume", address, LiveRestartOptions{})
	var refusal *SlotOperationError
	if !errors.As(err, &refusal) || refusal.Code != SlotOpNotOffered || !strings.Contains(refusal.Detail, "binding-lost") {
		t.Fatalf("resume of a lost conversation: %v", err)
	}
	if call, _, err := env.Couch.PrepareSlotOperation(context.Background(), "reboot", address, LiveRestartOptions{}); err != nil || call.Name != "reboot" {
		t.Fatalf("reboot of a lost conversation: %+v %v", call, err)
	}
	if n := len(env.Runner.Ops); n != 0 {
		t.Fatalf("admission launched %d children", n)
	}
}

// fakeLiveRestartProbe returns fixed facts and records what it was asked.
type fakeLiveRestartProbe struct {
	facts      LiveRestartFacts
	err        error
	asked      []string
	restarts   []ThreadAddress
	restartErr error
	busyErr    error
	busyChecks []ThreadAddress
	forced     []bool
}

func (f *fakeLiveRestartProbe) ConfirmNotBusy(_ context.Context, a ThreadAddress, forced bool) error {
	f.forced = append(f.forced, forced)
	f.busyChecks = append(f.busyChecks, a)
	return f.busyErr
}

func (f *fakeLiveRestartProbe) RestartConversation(_ context.Context, a ThreadAddress, forced bool) error {
	f.forced = append(f.forced, forced)
	f.restarts = append(f.restarts, a)
	return f.restartErr
}

func (f *fakeLiveRestartProbe) LiveRestartFacts(_ context.Context, op string, row ActionableThreadSummary, path string) (LiveRestartFacts, error) {
	f.asked = append(f.asked, op+" "+string(row.Address.Tag)+" "+path)
	return f.facts, f.err
}

// pair#421: relaunch admits a LIVE slot through DecideLiveRestart instead of
// the offer table, addresses it by thread, and carries the note to the caller.
func TestPrepareSlotOperationRelaunchLive(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	address := WorkspaceReference{Repo: local.slot.Repo, Number: local.slot.Number}.String()
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	live := validThreadRecord(t)
	live.Address.RepoScope = scope.Key
	live.StartingPath, live.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	live.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	live.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "slot-live", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: live.LatestLaunchProfile}}
	created, err := local.CreateThread(live)
	if err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(5151, "slot-live")
	ctx := context.Background()

	// No probe wired: refused as unavailable, never guessed.
	if _, _, err := env.Couch.PrepareSlotOperation(ctx, "relaunch", address, LiveRestartOptions{}); slotOperationCode(err) != LiveRestartUnavailable {
		t.Fatalf("no probe: %v", err)
	}
	idle := LiveRestartFacts{Live: true, Session: true, SettledKnown: true, Settled: true, GitKnown: true,
		Binary: BinaryFacts{RunningSHA: "a", OnDiskSHA: "b", OnDiskRevision: "r1", CheckoutHEAD: "r2", Checkout: "/w/pair"}}
	probe := &fakeLiveRestartProbe{facts: idle}
	env.Couch.LiveRestart = probe
	call, note, err := env.Couch.PrepareSlotOperation(ctx, "relaunch", address, LiveRestartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"repo-scope": scope.Key, "tag": string(created.Address.Tag), "require-settled": "known"}
	if call.Name != "relaunch" || !call.Implicit || !maps.Equal(call.Args, want) {
		t.Fatalf("call = %+v", call)
	}
	if !strings.Contains(note, "make build in /w/pair") {
		t.Fatalf("freshness note lost: %q", note)
	}
	if len(probe.asked) != 1 || probe.asked[0] != "relaunch "+string(created.Address.Tag)+" "+local.slot.WorktreeRoot {
		t.Fatalf("probe asked %v", probe.asked)
	}
	// A refusal reaches the caller as a typed code with its reason.
	busy := idle
	busy.Settled = false
	probe.facts = busy
	_, _, err = env.Couch.PrepareSlotOperation(ctx, "relaunch", address, LiveRestartOptions{})
	var refusal *SlotOperationError
	if !errors.As(err, &refusal) || refusal.Code != LiveRestartBusy || !strings.HasPrefix(refusal.Detail, address+": ") {
		t.Fatalf("busy slot: %v", err)
	}
	// The probe failing is an error, not an admission.
	probe.err = errors.New("probe broke")
	if _, _, err := env.Couch.PrepareSlotOperation(ctx, "relaunch", address, LiveRestartOptions{}); err == nil || slotOperationCode(err) != "" {
		t.Fatalf("probe error admitted or typed: %v", err)
	}
	if n := len(env.Runner.Ops); n != 0 {
		t.Fatalf("admission launched %d children", n)
	}
}

// liveSlotFixture is a slot whose one live thread is running.
func liveSlotFixture(t *testing.T) (*testEnv, string, ThreadRecord) {
	t.Helper()
	env, local := slotRecoveryOperationFixture(t)
	address := WorkspaceReference{Repo: local.slot.Repo, Number: local.slot.Number}.String()
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	live := validThreadRecord(t)
	live.Address.RepoScope = scope.Key
	live.StartingPath, live.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	live.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	live.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "slot-live", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: live.LatestLaunchProfile}}
	created, err := local.CreateThread(live)
	if err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(5151, "slot-live")
	return env, address, created
}

// pair#421 M3: reload-context shares relaunch's admission minus freshness, and
// its effect goes through the probe's verified restart.
func TestReloadContextAdmissionAndEffect(t *testing.T) {
	env, address, created := liveSlotFixture(t)
	ctx := context.Background()
	if _, err := env.Couch.ReloadContext(ctx, created.Address, false); slotOperationCode(err) != LiveRestartUnavailable {
		t.Fatalf("no probe: %v", err)
	}
	// An unchanged binary does not matter to reload-context.
	idle := LiveRestartFacts{Live: true, Session: true, SettledKnown: true, Settled: true, GitKnown: true,
		Binary: BinaryFacts{RunningSHA: "same", OnDiskSHA: "same"}}
	probe := &fakeLiveRestartProbe{facts: idle}
	env.Couch.LiveRestart = probe
	call, _, err := env.Couch.PrepareSlotOperation(ctx, OpReloadContext, address, LiveRestartOptions{})
	if err != nil || call.Name != OpReloadContext || call.Args["tag"] != string(created.Address.Tag) {
		t.Fatalf("admission: %+v %v", call, err)
	}
	res, err := env.Couch.ReloadContext(ctx, created.Address, false)
	if err != nil || res.Address != created.Address || len(probe.restarts) != 1 || probe.restarts[0] != created.Address {
		t.Fatalf("effect: %+v %v restarts=%v", res, err, probe.restarts)
	}
	probe.restartErr = &ReloadUnconfirmed{Detail: "no new session"}
	var unconfirmed *ReloadUnconfirmed
	if _, err := env.Couch.ReloadContext(ctx, created.Address, false); !errors.As(err, &unconfirmed) {
		t.Fatalf("unconfirmed collapsed: %v", err)
	}
	// Dispatch reaches it through the declared operation.
	if confirms, _ := OperationConfirms(OpReloadContext); !confirms {
		t.Fatal("reload-context must require confirmation: it ends a conversation")
	}
}

// BR-14 rule (M3 review, 3rd in the safety-guard-wiring family): EVERY
// live-owner operation is reachable through the live-owner executor. With a
// cancelled context each fails early, but never as "not a live-owner
// operation", which is what a missing dispatch case looks like.
func TestEveryLiveOwnerOperationIsDispatched(t *testing.T) {
	env, _, _ := liveSlotFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exec := CouchLiveOwnerExecutor(env.Couch)
	n := 0
	for _, op := range Operations() {
		if op.Execution != ExecuteLiveOwner {
			continue
		}
		n++
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked with empty args: %v", op.Name, r)
				}
			}()
			_, err := exec(OperationCall{Name: op.Name, Operation: op, Args: map[string]string{}, Context: ctx})
			if err != nil && strings.Contains(err.Error(), "is not a live-owner operation") {
				t.Errorf("%s has no live-owner dispatch case", op.Name)
			}
		}()
	}
	if n == 0 {
		t.Fatal("no live-owner operations enumerated")
	}
}

// BR-15: the busy guard is re-checked at the effect. A slot that turns busy
// between admission and the queue running is refused, and nothing is done.
func TestLiveRestartGuardRecheckedAtEffect(t *testing.T) {
	env, address, created := liveSlotFixture(t)
	ctx := context.Background()
	probe := &fakeLiveRestartProbe{facts: LiveRestartFacts{Live: true, Session: true, SettledKnown: true, Settled: true, GitKnown: true,
		Binary: BinaryFacts{RunningSHA: "a", OnDiskSHA: "b"}}}
	env.Couch.LiveRestart = probe
	call, _, err := env.Couch.PrepareSlotOperation(ctx, OpRelaunch, address, LiveRestartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	probe.busyErr = &SlotOperationError{Code: LiveRestartBusy, Detail: "became busy"}
	op, _ := operationByName(OpRelaunch)
	call.Operation, call.Context = op, ctx
	if _, err := CouchLiveOwnerExecutor(env.Couch)(call); slotOperationCode(err) != LiveRestartBusy {
		t.Fatalf("relaunch after turning busy: %v", err)
	}
	if len(probe.busyChecks) != 1 || probe.busyChecks[0] != created.Address {
		t.Fatalf("busy re-check not asked: %v", probe.busyChecks)
	}
	if thread, err := env.Couch.Threads.GetThread(created.Address); err != nil || thread.Park != nil {
		t.Fatalf("a refused relaunch parked the thread: %+v %v", thread.Park, err)
	}
	// The console's own Alt+n (no require-settled) is not gated.
	probe.busyChecks = nil
	delete(call.Args, requireSettledArg)
	_, _ = CouchLiveOwnerExecutor(env.Couch)(call)
	if len(probe.busyChecks) != 0 {
		t.Fatal("console relaunch consulted the remote busy guard")
	}
}

func TestReloadContextRefusesWhenNoLongerLive(t *testing.T) {
	env, _, created := liveSlotFixture(t)
	probe := &fakeLiveRestartProbe{}
	env.Couch.LiveRestart = probe
	env.Proc.Kill(5151)
	if _, err := env.Couch.Threads.RetireProvedDeadIncarnations(created.Address, created.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.ReloadContext(context.Background(), created.Address, false); slotOperationCode(err) != LiveRestartNotLive || len(probe.restarts) != 0 {
		t.Fatalf("not live: %v restarts=%v", err, probe.restarts)
	}
}

// M3 advisory: the effect reads admission's decision. A forced admission (no
// Settled claim) is marked "forced"; a known-settled one "known".
func TestLiveRestartAdmissionRecordsItsEvidence(t *testing.T) {
	env, address, _ := liveSlotFixture(t)
	ctx := context.Background()
	unknown := LiveRestartFacts{Live: true, Session: true, GitKnown: true, Binary: BinaryFacts{RunningSHA: "a", OnDiskSHA: "b"}}
	env.Couch.LiveRestart = &fakeLiveRestartProbe{facts: unknown}
	for _, op := range []string{OpRelaunch, OpReloadContext} {
		call, _, err := env.Couch.PrepareSlotOperation(ctx, op, address, LiveRestartOptions{ForceUnknown: true})
		if err != nil || call.Args[requireSettledArg] != settledForcedValue {
			t.Fatalf("%s forced: %+v %v", op, call.Args, err)
		}
	}
	known := unknown
	known.SettledKnown, known.Settled = true, true
	env.Couch.LiveRestart = &fakeLiveRestartProbe{facts: known}
	for _, op := range []string{OpRelaunch, OpReloadContext} {
		// --force-unknown on a slot that WAS known stays "known": it changed nothing.
		call, _, err := env.Couch.PrepareSlotOperation(ctx, op, address, LiveRestartOptions{ForceUnknown: true})
		if err != nil || call.Args[requireSettledArg] != settledKnownValue {
			t.Fatalf("%s known: %+v %v", op, call.Args, err)
		}
	}
}

// BR-22 (M4 review): require-settled is tested from producer to consumer.
// PrepareSlotOperation writes it, CouchLiveOwnerExecutor translates it, the
// probe receives the forced flag. Verb x {known, forced, absent}.
func TestRequireSettledProducerToConsumer(t *testing.T) {
	known := LiveRestartFacts{Live: true, Session: true, SettledKnown: true, Settled: true, GitKnown: true, Binary: BinaryFacts{RunningSHA: "a", OnDiskSHA: "b"}}
	unknown := known
	unknown.SettledKnown, unknown.Settled = false, false
	stop := &SlotOperationError{Code: LiveRestartBusy, Detail: "stop before any effect"}
	for _, op := range []string{OpRelaunch, OpReloadContext} {
		for _, tc := range []struct {
			name      string
			facts     LiveRestartFacts
			opts      LiveRestartOptions
			absent    bool
			wantCalls []bool // forced flags the probe saw
		}{
			{"known", known, LiveRestartOptions{}, false, []bool{false}},
			{"forced", unknown, LiveRestartOptions{ForceUnknown: true}, false, []bool{true}},
			// Absent: the console's own Alt+n never gates relaunch; a
			// reload-context without evidence is treated as not forced.
			{"absent", known, LiveRestartOptions{}, true, map[string][]bool{OpRelaunch: nil, OpReloadContext: {false}}[op]},
		} {
			env, address, _ := liveSlotFixture(t)
			probe := &fakeLiveRestartProbe{facts: tc.facts, busyErr: stop, restartErr: stop}
			env.Couch.LiveRestart = probe
			call, _, err := env.Couch.PrepareSlotOperation(context.Background(), op, address, tc.opts)
			if err != nil {
				t.Fatalf("%s/%s admission: %v", op, tc.name, err)
			}
			if tc.absent {
				delete(call.Args, requireSettledArg)
			}
			declared, _ := operationByName(op)
			call.Operation, call.Context = declared, context.Background()
			_, _ = CouchLiveOwnerExecutor(env.Couch)(call)
			if !reflect.DeepEqual(probe.forced, tc.wantCalls) {
				t.Errorf("%s/%s: probe saw forced=%v, want %v", op, tc.name, probe.forced, tc.wantCalls)
			}
		}
	}
}
