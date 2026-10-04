package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
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

// TestRecoverPlanAfterRestart is the Done-when's restart acceptance: a world
// seeded "before the restart" read back through the real recover-plan
// dispatch, then re-read after sdlc's state changes.
func TestRecoverPlanAfterRestart(t *testing.T) {
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
	// :1 detached on its claimed branch.
	one := slotRecordFixture(t, env, locals[1])
	env.Artifacts.SetPairSession(one.Address, "pair-slot-one", true)
	env.Artifacts.SetDetachedSession(one.Address, "pair-slot-one")
	fleet.SetBranch(address(1), "000011-x")
	fleet.Claim(address(1), ref(11))
	// :2 parked on its claimed branch; :3 the same, dirty.
	for _, n := range []int{2, 3} {
		slotRecordFixture(t, env, locals[n])
		fleet.SetBranch(address(n), fmt.Sprintf("%06d-x", 10+n))
		fleet.Claim(address(n), ref(10+n))
	}
	fleet.SetDirty(address(3), 2)
	// :4 parked, claimed, sitting on its resting branch.
	slotRecordFixture(t, env, locals[4])
	fleet.Claim(address(4), ref(14))
	// :5 dirty on its resting branch, no claim.
	fleet.SetDirty(address(5), 1)
	// :6 parked on an issue branch nobody claims.
	slotRecordFixture(t, env, locals[6])
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
	// A claim whose slot directory is gone, and one on a scratch worktree.
	fleet.AddDanglingClaim(f.Primary, filepath.Join(root, "worktree", repo+"-slot9", repo), ref(15), address(9))
	fleet.AddOffSlotRow(filepath.Join(root, "scratch"), f.Primary, "")
	fleet.ClaimAt(filepath.Join(root, "scratch"), ref(77), "")

	report := func() RecoverPlan {
		t.Helper()
		result, err := DispatchOperation(OperationExecutors{DirectStore: DirectStoreExecutor(env.Couch)}, OperationCall{Name: "recover-plan", Context: context.Background()})
		if err != nil {
			t.Fatal(err)
		}
		plan, ok := result.(RecoverPlan)
		if !ok {
			t.Fatalf("recover-plan returned %T", result)
		}
		return plan
	}
	expect := func(plan RecoverPlan, n int, class RecoverClass, steps []string, notes ...string) {
		t.Helper()
		row := findRow(t, plan, address(n))
		if row.Class != class || !slices.Equal(stepActions(row), steps) || len(notes) > 0 && !slices.Equal(row.Next.Notes, notes) {
			t.Errorf("%s = %s %v hold %v notes %v (agent %+v, reason %q); want %s %v %v", row.Address, row.Class, stepActions(row), row.Next.Hold, row.Next.Notes, row.Agent, row.Reason, class, steps, notes)
		}
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
