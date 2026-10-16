package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// recoverAcceptanceFixture is slotRecoveryOperationFixture with six slots: a
// real primary and real slot worktrees, enrolled through the OS slot catalog,
// so the report runs the production workspace probe and slot stores.
func recoverAcceptanceFixture(t *testing.T, slots int) (*testEnv, *ProvisionFixture, map[int]*ThreadStore) {
	t.Helper()
	f := newProvisionFixture(t)
	var hosts []string
	for n := 1; n <= slots; n++ {
		f.git(f.Primary, "worktree", "add", "-b", fmt.Sprintf("main-slot%d", n), f.host(n), "main")
		hosts = append(hosts, f.host(n))
	}
	env := newTestEnv(t, hosts...)
	env.Couch.Slots = NewOSSlotCatalog(f)
	for _, host := range hosts {
		env.Git.replies[GitCall{Dir: host, Args: "rev-parse --git-common-dir"}] = filepath.Join(f.Primary, ".git")
	}
	repository, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Couch.Threads.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	locals := map[int]*ThreadStore{}
	for _, candidate := range repository.Slots {
		locals[candidate.Identity.Number] = newSlotThreadStore(env.Couch.Namespace, candidate.Identity)
	}
	return env, f, locals
}

// recoverWorld is Task 1.7's "before the restart" world: six real slots, a
// live :0, a dangling claim and a scratch-worktree claim, read through the
// real recover-plan dispatch.
type recoverWorld struct {
	env     *testEnv
	f       *ProvisionFixture
	locals  map[int]*ThreadStore
	fleet   *FakeFleet
	address func(int) string
	// records are the seeded slot records, by slot number.
	records map[int]ThreadRecord
}

func newRecoverWorld(t *testing.T) *recoverWorld {
	t.Helper()
	env, f, locals := recoverAcceptanceFixture(t, 6)
	repo := filepath.Base(f.Primary)
	root := filepath.Dir(f.Primary)
	sdlc := NewFakeFleetSDLC()
	fleet := sdlc.Fleet(root)
	env.Couch.Fleet = SDLCFleetSource{IO: sdlc}
	address := func(n int) string { return fmt.Sprintf("%s:%d", repo, n) }
	ref := func(n int) string { return fmt.Sprintf("%s#%06d", repo, n) }
	for n := 0; n <= 6; n++ {
		if got := fleet.AddSlot(address(n)); n > 0 && got != f.host(n) {
			t.Fatalf("fake slot path %s, fixture host %s", got, f.host(n))
		}
	}
	records := map[int]ThreadRecord{}
	// :1 detached on its claimed branch.
	one := slotRecordFixture(t, env, locals[1])
	records[1] = one
	env.Artifacts.SetPairSession(one.Address, "pair-slot-one", true)
	env.Artifacts.SetDetachedSession(one.Address, "pair-slot-one")
	fleet.SetBranch(address(1), "000011-x")
	fleet.Claim(address(1), ref(11))
	// :2 parked on its claimed branch; :3 the same, dirty.
	for _, n := range []int{2, 3} {
		records[n] = slotRecordFixture(t, env, locals[n])
		fleet.SetBranch(address(n), fmt.Sprintf("%06d-x", 10+n))
		fleet.Claim(address(n), ref(10+n))
	}
	fleet.SetDirty(address(3), 2)
	// :4 parked, claimed, sitting on its resting branch.
	records[4] = slotRecordFixture(t, env, locals[4])
	fleet.Claim(address(4), ref(14))
	// :5 dirty on its resting branch, no claim.
	fleet.SetDirty(address(5), 1)
	// :6 parked on an issue branch nobody claims.
	records[6] = slotRecordFixture(t, env, locals[6])
	fleet.SetBranch(address(6), "000016-x")
	// :0 live, clean, resting, unclaimed.
	scope, err := launcher.ResolveRepoScope(f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	primary := validThreadRecord(t)
	primary.Address.RepoScope = scope.Key
	primary.StartingPath, primary.WorkingPath = f.Primary, f.Primary
	primary.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	primary.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "pair-primary", State: IncarnationLive, RepoIdentity: filepath.Join(f.Primary, ".git"), LaunchProfile: primary.LatestLaunchProfile}}
	if _, err := env.Couch.Threads.CreateThread(primary); err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(4242, "pair-primary")
	env.Artifacts.SetSessionPresence(primary.Address, SessionObservation{State: SessionPresent})
	// A claim whose slot directory is gone, and one on a scratch worktree.
	fleet.AddDanglingClaim(f.Primary, filepath.Join(root, "worktree", repo+"-slot9", repo), ref(15), address(9))
	fleet.AddOffSlotRow(filepath.Join(root, "scratch"), f.Primary, "")
	fleet.ClaimAt(filepath.Join(root, "scratch"), ref(77), "")
	return &recoverWorld{env: env, f: f, locals: locals, fleet: fleet, address: address, records: records}
}

func (w *recoverWorld) report(t *testing.T) RecoverPlan {
	t.Helper()
	result, err := DispatchOperation(OperationExecutors{DirectStore: DirectStoreExecutor(w.env.Couch)}, OperationCall{Name: "recover-plan", Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := result.(RecoverPlan)
	if !ok {
		t.Fatalf("recover-plan returned %T", result)
	}
	return plan
}

func (w *recoverWorld) expect(t *testing.T, plan RecoverPlan, n int, class RecoverClass, steps []string, notes ...string) {
	t.Helper()
	row := findRow(t, plan, w.address(n))
	if row.Class != class || !slices.Equal(stepActions(row), steps) || len(notes) > 0 && !slices.Equal(row.Next.Notes, notes) {
		t.Errorf("%s = %s %v hold %v notes %v (agent %+v, reason %q); want %s %v %v", row.Address, row.Class, stepActions(row), row.Next.Hold, row.Next.Notes, row.Agent, row.Reason, class, steps, notes)
	}
}

// TestRecoverPlanAfterRestart is the Done-when's restart acceptance: a world
// seeded "before the restart" read back through the real recover-plan
// dispatch, then re-read after sdlc's state changes.
func TestRecoverPlanAfterRestart(t *testing.T) {
	t.Parallel()

	w := newRecoverWorld(t)
	address, fleet := w.address, w.fleet
	report := func() RecoverPlan { return w.report(t) }
	expect := func(plan RecoverPlan, n int, class RecoverClass, steps []string, notes ...string) {
		t.Helper()
		w.expect(t, plan, n, class, steps, notes...)
	}
	plan := report()
	if plan.Couch.State != CouchObservationOK || len(plan.Fleets) != 1 || plan.Fleets[0].State != FleetObservationPresent {
		t.Fatalf("sources: couch %+v, fleets %+v", plan.Couch, plan.Fleets)
	}
	for _, n := range []int{1, 2, 3} {
		expect(plan, n, RecoverAgrees, []string{"resume"})
	}
	expect(plan, 4, RecoverRestoreWorkspace, []string{"resume", "ask-agent-restore"})
	expect(plan, 5, RecoverUnidentifiedWork, nil)
	expect(plan, 6, RecoverClaimLikelyLost, []string{"resume"}, "claim-repair")
	expect(plan, 0, RecoverIdle, nil)
	// Each seeded agent state is the one the row's name claims. An enrolled
	// slot with no record is still a Couch row: unusable, never-started.
	for n, want := range map[int]EvidenceAgent{0: AgentLive, 1: AgentDetached, 2: AgentParked, 3: AgentParked, 4: AgentParked, 5: AgentUnusable, 6: AgentParked} {
		if row := findRow(t, plan, address(n)); row.Agent.State != string(want) {
			t.Errorf("%s agent = %+v, want %s", row.Address, row.Agent, want)
		}
	}
	// A never-started reservation is no conversation: :5's only work is dirt.
	if row := findRow(t, plan, address(5)); !slices.Equal(row.Evidence, []string{"dirty"}) {
		t.Errorf(":5 evidence = %v, want [dirty]", row.Evidence)
	}
	expect(plan, 9, RecoverDirectoryMissing, nil)
	if plan.Ignored.OffSlotClaims != 1 {
		t.Errorf("ignored = %+v, want one off-slot claim", plan.Ignored)
	}

	// The plan is recomputed from fresh sdlc output on every call.
	fleet.Release(address(1))
	expect(report(), 1, RecoverClaimLikelyLost, []string{"resume"}, "claim-repair")
}

// TestRecoverPlanStepsConverge is the loop the skill drives: read the report,
// run every automatic row's resume/reboot step through PrepareSlotOperation
// and the live-owner dispatch, mark what started alive, read the report
// again. Rows converge to live with nothing left to run, and a resend is
// refused (pair#367 Task 2.7).
func TestRecoverPlanStepsConverge(t *testing.T) {
	t.Parallel()

	w := newRecoverWorld(t)
	ctx := context.Background()
	untouched, err := os.ReadDir(w.locals[5].root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	// Every slot workspace is ready, as provisioning reports it for a slot
	// that exists (slotRecoveryOperationFixture's readiness, per slot).
	w.env.Couch.Workspaces = slotReadinessFunc(func(_ context.Context, r ProvisionRequest) (ProvisionResult, error) {
		for _, local := range w.locals {
			if local.slot.Number == r.Slot || filepath.Clean(local.slot.WorktreeRoot) == filepath.Clean(r.Path) {
				return slotReadyResult(local), nil
			}
		}
		return ProvisionResult{}, fmt.Errorf("no slot %d at %s", r.Slot, r.Path)
	})
	w.env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	// A parked slot's conversation still resolves (its Pair session is
	// recorded, not running); the cold resume's session coming up births its
	// agent pane, as TestResumeOperationOnASlotPathAdoptsALostPointer models.
	var resuming ThreadAddress
	for _, n := range []int{2, 3, 4, 6} {
		w.env.Artifacts.SetPairSession(w.records[n].Address, fmt.Sprintf("pair-slot-%d", n), false)
	}
	w.env.Runner.AfterAcknowledge = func(id string) error {
		w.env.Artifacts.SetPairSession(resuming, continuationChildSession(t, w.env.Runner, id), true)
		w.env.Artifacts.SetSessionPresence(resuming, SessionObservation{State: SessionPresent})
		return nil
	}
	slotOf := map[string]int{}
	for n := range w.records {
		slotOf[w.address(n)] = n
	}
	executors := OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(w.env.Couch), DirectStore: DirectStoreExecutor(w.env.Couch)}
	ran := map[string][]string{}
	for _, row := range w.report(t).Rows {
		if !row.Automatic {
			continue
		}
		for _, step := range row.Next.Steps {
			if step.Action == "ask-agent-restore" {
				continue // a message to the slot's agent, not a Couch primitive
			}
			call, _, err := w.env.Couch.PrepareSlotOperation(ctx, step.Action, row.Address, LiveRestartOptions{})
			if err != nil {
				t.Fatalf("%s %s: %v", row.Address, step.Action, err)
			}
			resuming = w.records[slotOf[row.Address]].Address
			value, err := DispatchOperation(executors, call)
			if err != nil {
				t.Fatalf("%s %s dispatch: %v", row.Address, step.Action, err)
			}
			child, ok := value.(StartedChild)
			if !ok {
				t.Fatalf("%s %s returned %T", row.Address, step.Action, value)
			}
			if started, hasChild := child.Started(); hasChild {
				w.env.Proc.Set(started.Record.PID, started.Record.Identity)
			}
			ran[row.Address] = append(ran[row.Address], step.Action)
		}
	}
	for _, n := range []int{1, 2, 3, 4, 6} {
		if !slices.Equal(ran[w.address(n)], []string{"resume"}) {
			t.Errorf("%s ran %v, want [resume]", w.address(n), ran[w.address(n)])
		}
	}
	plan := w.report(t)
	for _, n := range []int{1, 2, 3} {
		w.expect(t, plan, n, RecoverAgrees, nil)
	}
	w.expect(t, plan, 6, RecoverClaimLikelyLost, nil, "claim-repair")
	w.expect(t, plan, 4, RecoverRestoreWorkspace, []string{"ask-agent-restore"})
	w.expect(t, plan, 5, RecoverUnidentifiedWork, nil)
	for _, n := range []int{0, 1, 2, 3, 4, 6} {
		if row := findRow(t, plan, w.address(n)); row.Agent.State != string(AgentLive) {
			t.Errorf("%s agent = %+v, want live", row.Address, row.Agent)
		}
	}
	after, err := os.ReadDir(w.locals[5].root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(after) != len(untouched) {
		t.Errorf(":5's slot store changed: %v -> %v", untouched, after)
	}
	// Resume admission remains valid for a live row: execution chooses Continue
	// from the positive listing rather than rejecting the slot or creating a
	// second native conversation. Admission itself has no lifecycle effects.
	before, readErr := w.env.Couch.Threads.PreviewThreadContext(ctx, w.records[1].Address)
	if readErr != nil {
		t.Fatal(readErr)
	}
	call, _, err := w.env.Couch.PrepareSlotOperation(ctx, "resume", w.address(1), LiveRestartOptions{})
	afterRecord, afterErr := w.env.Couch.Threads.PreviewThreadContext(ctx, w.records[1].Address)
	if err != nil || call.Name != "resume" || afterErr != nil || !reflect.DeepEqual(before, afterRecord) {
		t.Fatalf("resend admission: call=%+v err=%v recorderr=%v", call, err, afterErr)
	}
}

// pair#597 close review BR-2: an exact registry collision refuses at both
// recovery gates. :2 resumes and is live; :3 then tries to resume the same
// conversation, recorded from the start (preflight) or only once :3's
// workspace is prepared (the gate before launch). Either way :3 is refused,
// named by its holder, and starts no child.
func TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "preflight", true: "after-prepare"}[late], func(t *testing.T) {
			w := newRecoverWorld(t)
			ctx := context.Background()
			collide := func() {
				w.env.Artifacts.SetNativeBinding(w.records[2].Address, "claude", sessioninventory.BindingEstablished, "native-slot-3")
			}
			prepared3 := 0
			w.env.Couch.Workspaces = slotReadinessFunc(func(_ context.Context, r ProvisionRequest) (ProvisionResult, error) {
				if r.Slot == 3 {
					prepared3++
					if late {
						collide()
					}
				}
				for _, local := range w.locals {
					if local.slot.Number == r.Slot || filepath.Clean(local.slot.WorktreeRoot) == filepath.Clean(r.Path) {
						return slotReadyResult(local), nil
					}
				}
				return ProvisionResult{}, fmt.Errorf("no slot %d at %s", r.Slot, r.Path)
			})
			w.env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			var resuming ThreadAddress
			for _, n := range []int{2, 3} {
				w.env.Artifacts.SetPairSession(w.records[n].Address, fmt.Sprintf("pair-slot-%d", n), false)
			}
			w.env.Runner.AfterAcknowledge = func(id string) error {
				w.env.Artifacts.SetPairSession(resuming, continuationChildSession(t, w.env.Runner, id), true)
				w.env.Artifacts.SetSessionPresence(resuming, SessionObservation{State: SessionPresent})
				return nil
			}
			executors := OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(w.env.Couch), DirectStore: DirectStoreExecutor(w.env.Couch)}
			resume := func(n int) error {
				call, _, err := w.env.Couch.PrepareSlotOperation(ctx, "resume", w.address(n), LiveRestartOptions{})
				if err != nil {
					return err
				}
				resuming = w.records[n].Address
				value, err := DispatchOperation(executors, call)
				if err != nil {
					return err
				}
				if started, ok := value.(StartedChild).Started(); ok {
					w.env.Proc.Set(started.Record.PID, started.Record.Identity)
				}
				return nil
			}
			if err := resume(2); err != nil {
				t.Fatalf(":2 resume: %v", err)
			}
			if !late {
				collide()
			}
			children := len(w.env.Runner.order)
			err := resume(3)
			if err == nil || !strings.Contains(err.Error(), "conversation native-slot-3 is in use by "+w.address(2)) {
				t.Fatalf(":3 resumed a conversation live in :2: %v", err)
			}
			if len(w.env.Runner.order) != children {
				t.Fatalf(":3 started a child after refusal: %v", w.env.Runner.order)
			}
			if !late && prepared3 != 0 {
				t.Fatalf("preflight refusal came after :3's workspace was prepared (%d)", prepared3)
			}
		})
	}
}

// pair#607: a reboot leaves a zellij session that another live slot's record
// also names, says so, and still stops the rebooted slot's wrapper tree.
func TestRebootLeavesSessionSharedWithAnotherLiveSlot(t *testing.T) {
	w := newRecoverWorld(t)
	ctx := context.Background()
	w.env.Couch.Workspaces = slotReadinessFunc(func(_ context.Context, r ProvisionRequest) (ProvisionResult, error) {
		for _, local := range w.locals {
			if local.slot.Number == r.Slot || filepath.Clean(local.slot.WorktreeRoot) == filepath.Clean(r.Path) {
				return slotReadyResult(local), nil
			}
		}
		return ProvisionResult{}, fmt.Errorf("no slot %d at %s", r.Slot, r.Path)
	})
	w.env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	share := func(n int, live bool) {
		updated, err := w.locals[n].updateExistingThread(w.records[n].Address, w.records[n].Revision, func(next *ThreadRecord) error {
			next.SessionBinding = &couchidentity.SessionBinding{C: 1, M: 5, Name: "📁1-5", ScopeKey: next.Address.RepoScope, Tag: string(next.Address.Tag), StartNonce: "start-0102030405060708"}
			if live {
				next.Incarnations = []ThreadIncarnation{{PID: 900, Identity: "helper-3", State: IncarnationLive, LaunchProfile: next.LatestLaunchProfile}}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		w.records[n] = updated
	}
	share(2, false)
	share(3, true)
	w.env.Proc.Set(900, "helper-3")
	w.env.Artifacts.SetPairSession(w.records[2].Address, "📁1-5", true)
	w.env.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{}, errors.New("legacy readiness has no child birth")
	}
	w.env.Proc.Set(500, "wrapper")
	w.env.Proc.Set(501, "codex")
	w.env.Proc.DiesOn = map[int]os.Signal{500: TermSignal, 501: TermSignal}
	w.env.Couch.SlotWrapper = func(address ThreadAddress) (ProcessIdentity, bool) {
		return ProcessIdentity{PID: 500, Identity: "wrapper"}, address == w.records[2].Address
	}
	w.env.Couch.ProcessTable = func() ([]procutil.Process, error) {
		return []procutil.Process{{PID: 500, PPID: 1, Identity: "wrapper", Name: "pair"}, {PID: 501, PPID: 500, Identity: "codex", Name: "codex"}}, nil
	}
	w.env.Couch.sleep = func(time.Duration) {}
	w.env.Runner.AfterAcknowledge = func(id string) error {
		child := w.env.Runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
			return err
		}
		fresh := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(profile.Tag)}
		w.env.Artifacts.SetPairSession(fresh, continuationChildSession(t, w.env.Runner, id), true)
		w.env.Artifacts.SetSessionPresence(fresh, SessionObservation{State: SessionPresent})
		return nil
	}
	_, err := w.env.Couch.RecoverSelection(ctx, RecoverySelection{Target: ResumeTarget{Address: w.records[2].Address}, Kind: RecoveryFresh})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) {
		t.Fatalf("reboot asked no confirmation: %v", err)
	}
	for _, stop := range required.Offer.Stops {
		if strings.HasPrefix(stop.Role, "zellij session") {
			t.Fatalf("shared session listed to stop: %+v", required.Offer.Stops)
		}
	}
	result, err := w.env.Couch.ConfirmRecovery(ctx, required.Offer.Token)
	if err != nil {
		t.Fatalf("reboot: %v", err)
	}
	if !strings.Contains(result.Notice, "also named by "+w.address(3)) {
		t.Fatalf("notice %q does not say the shared session was left", result.Notice)
	}
	for _, pid := range []int{500, 501} {
		if w.env.Proc.Exists(pid) != Dead {
			t.Fatalf("pid %d survived the reboot", pid)
		}
	}
	for _, address := range w.env.Artifacts.Quiesces() {
		if address == w.records[2].Address {
			t.Fatal("the shared session was killed")
		}
	}
}
