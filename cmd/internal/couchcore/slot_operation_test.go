package couchcore

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
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
	facts LiveRestartFacts
	err   error
	asked []string
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
	want := map[string]string{"repo-scope": scope.Key, "tag": string(created.Address.Tag)}
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
