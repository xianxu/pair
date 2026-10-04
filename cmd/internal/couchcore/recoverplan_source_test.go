package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSDLCFleetSourceArgv(t *testing.T) {
	fake := NewFakeFleetSDLC()
	fake.Fleet("/fleet").AddSlot("pair:0")
	if _, err := (SDLCFleetSource{IO: fake, Timeout: 90 * time.Second}).FleetInventory(context.Background(), "/fleet/pair"); err != nil {
		t.Fatal(err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("calls = %+v, want one", fake.Calls)
	}
	got := fake.Calls[0]
	if got.Dir != "/fleet/pair" || got.Program != "sdlc" || !slices.Equal(got.Args, []string{"fleet", "inventory", "--json", "--path", "/fleet/pair"}) || got.Timeout != 90*time.Second {
		t.Fatalf("call = %+v", got)
	}
	// The zero timeout is the shared bound, never unbounded.
	if _, err := (SDLCFleetSource{IO: fake}).FleetInventory(context.Background(), "/fleet/pair"); err != nil || fake.Calls[1].Timeout != FleetInventoryTimeout {
		t.Fatalf("default timeout call = %+v (%v)", fake.Calls[1], err)
	}
}

func TestFakeFleetSDLCRefusesOtherCommands(t *testing.T) {
	fake := NewFakeFleetSDLC()
	fake.Fleet("/fleet").AddSlot("pair:0")
	for _, c := range []ProvisionCommand{
		{Dir: "/fleet/pair", Program: "git", Args: []string{"fleet", "inventory", "--json", "--path", "/fleet/pair"}},
		{Dir: "/fleet/pair", Program: "sdlc", Args: []string{"workspace", "--json"}},
		{Dir: "/fleet/pair", Program: "sdlc", Args: []string{"fleet", "inventory", "--json", "--path", "/elsewhere"}},
	} {
		if _, err := fake.Run(context.Background(), c); err == nil {
			t.Errorf("fake accepted %+v", c)
		}
	}
}

func fakeSlotOf(t *testing.T, inv FleetInventory, address string) FleetSlot {
	t.Helper()
	for _, s := range inv.Slots {
		if s.Address == address {
			return s
		}
	}
	t.Fatalf("no slot %s in %+v", address, inv.Slots)
	return FleetSlot{}
}

func runFake(t *testing.T, fake *FakeFleetSDLC, vantage string) (FleetInventory, error) {
	t.Helper()
	raw, err := (SDLCFleetSource{IO: fake}).FleetInventory(context.Background(), vantage)
	if err != nil {
		return FleetInventory{}, err
	}
	return DecodeFleetInventory(raw)
}

func TestFakeFleetSDLCIsStateful(t *testing.T) {
	fake := NewFakeFleetSDLC()
	fleet := fake.Fleet("/fleet")
	fleet.AddSlot("pair:0")
	fleet.AddSlot("pair:1")
	step := func(want string, reasons ...string) FleetInventory {
		t.Helper()
		inv, err := runFake(t, fake, "/fleet/pair")
		if err != nil {
			t.Fatal(err)
		}
		slot := fakeSlotOf(t, inv, "pair:1")
		if slot.Verdict != want || !slices.Equal(slot.Members[0].Reasons, reasons) {
			t.Fatalf("pair:1 = %s %v, want %s %v", slot.Verdict, slot.Members[0].Reasons, want, reasons)
		}
		return inv
	}
	step("ready")
	fleet.Claim("pair:1", "pair#7")
	step("holds-work", "claimed:pair#7")
	fleet.SetDirty("pair:1", 1)
	inv := step("needs-recovery", "dirty")
	var claimed bool
	for _, row := range inv.Rows {
		if row.TreePath == "/fleet/worktree/pair-slot1/pair" && len(row.Claims) == 1 && row.Claims[0].Ref == "pair#7" {
			claimed = true
		}
	}
	if !claimed {
		t.Fatal("the claim vanished from the dirty slot's row")
	}
	fleet.RemoveSlot("pair:1")
	inv, err := runFake(t, fake, "/fleet/pair")
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.DanglingClaims) != 1 || inv.DanglingClaims[0].Ref != "pair#7" || inv.DanglingClaims[0].Claimant.Worktree != "/fleet/worktree/pair-slot1/pair" {
		t.Fatalf("dangling = %+v", inv.DanglingClaims)
	}
	for _, s := range inv.Slots {
		if s.Address == "pair:1" {
			t.Fatal("removed slot still listed")
		}
	}
	fake.Schema = 2
	if _, err := runFake(t, fake, "/fleet/pair"); !errors.Is(err, ErrFleetSchemaUnsupported) {
		t.Fatalf("schema 2 decoded: %v", err)
	}
}

func TestFakeFleetSDLCFailureModes(t *testing.T) {
	fake := NewFakeFleetSDLC()
	fake.Fleet("/fleet").AddSlot("pair:0")
	fake.Fail = FakeFleetExit
	if _, err := runFake(t, fake, "/fleet/pair"); err == nil {
		t.Fatal("exit failure succeeded")
	}
	fake.Fail = FakeFleetGarbage
	if _, err := runFake(t, fake, "/fleet/pair"); err == nil || errors.Is(err, ErrFleetSchemaUnsupported) {
		t.Fatalf("garbage = %v, want a malformed-input error", err)
	}
	fake.Fail = FakeFleetHang
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := (SDLCFleetSource{IO: fake}).FleetInventory(ctx, "/fleet/pair"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("hang = %v, want the deadline", err)
	}
}

// TestFakeFleetSDLCVerdictsMatchSDLCPrecedence copies ariadne's
// TestJudgeCheckout cases (cmd/sdlc/internal/fleet/slots_test.go at
// 35aa7ebd7725602a88f9fb4603c60e4c86db0a64) that the fake can express.
func TestFakeFleetSDLCVerdictsMatchSDLCPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		set     func(*FakeFleet)
		verdict string
		reasons string
	}{
		{"clean on resting", func(*FakeFleet) {}, "ready", ""},
		{"merged feature branch", func(f *FakeFleet) { f.SetBranch("r:1", "topic") }, "ready", ""},
		{"closed-issue branch", func(f *FakeFleet) { f.SetBranch("r:1", "000007-x"); f.SetIssueStatus("r:1", "done") }, "ready", ""},
		{"unlanded commits", func(f *FakeFleet) { f.SetBranch("r:1", "topic"); f.SetAhead("r:1", 2) }, "holds-work", "unlanded-commits"},
		{"unlanded on resting", func(f *FakeFleet) { f.SetAhead("r:1", 1) }, "holds-work", "unlanded-commits"},
		{"zero-commit open-issue branch", func(f *FakeFleet) { f.SetBranch("r:1", "000007-x") }, "holds-work", "open-issue:r#000007"},
		{"claimed on resting", func(f *FakeFleet) { f.Claim("r:1", "r#000009") }, "holds-work", "claimed:r#000009"},
		{"dirty", func(f *FakeFleet) { f.SetDirty("r:1", 3) }, "needs-recovery", "dirty"},
		{"operation", func(f *FakeFleet) { f.SetOperation("r:1", "rebase-merge") }, "needs-recovery", "operation:rebase-merge"},
		{"detached", func(f *FakeFleet) { f.SetDetached("r:1") }, "needs-recovery", "detached"},
		{"dirty and unlanded", func(f *FakeFleet) { f.SetDirty("r:1", 1); f.SetBranch("r:1", "topic"); f.SetAhead("r:1", 1) }, "needs-recovery", "dirty"},
		{"base unavailable", func(f *FakeFleet) { f.SetBaseUnavailable("r:1") }, "unknown", "probe:base"},
		{"base unavailable but dirty", func(f *FakeFleet) { f.SetBaseUnavailable("r:1"); f.SetDirty("r:1", 1) }, "needs-recovery", "dirty,probe:base"},
		{"claims unread, otherwise ready", func(f *FakeFleet) { f.SetClaimsState("r:1", FleetClaimsUnknown, "x") }, "unknown", "probe:claims"},
		{"claims unread, already holding work", func(f *FakeFleet) {
			f.SetClaimsState("r:1", FleetClaimsUnknown, "x")
			f.SetBranch("r:1", "topic")
			f.SetAhead("r:1", 1)
		}, "holds-work", "unlanded-commits"},
		{"missing dependency folds worst", func(f *FakeFleet) { f.MissingMember("r:1", "dep") }, "missing", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := NewFakeFleetSDLC()
			fleet := fake.Fleet("/fleet")
			fleet.AddSlot("r:1")
			tc.set(fleet)
			slot := fakeSlotOf(t, fleet.Inventory(), "r:1")
			if slot.Verdict != tc.verdict || strings.Join(slot.Members[0].Reasons, ",") != tc.reasons {
				t.Fatalf("got %s %v, want %s %s", slot.Verdict, slot.Members[0].Reasons, tc.verdict, tc.reasons)
			}
		})
	}
}

// enrollRoots writes the enrolled-repository list the way an older test
// fixture does (aliasTestStore), without Git discovery.
func enrollRoots(t *testing.T, store *ThreadStore, roots ...string) {
	t.Helper()
	if err := store.withLock(func() error {
		raw, err := json.Marshal(threadManifest{SchemaVersion: 2, Threads: []ThreadAddress{}, SlotRepositories: roots})
		if err != nil {
			return err
		}
		return os.WriteFile(store.manifestPath(), raw, 0600)
	}); err != nil {
		t.Fatal(err)
	}
}

func recoverEnv(t *testing.T, fleets map[string]string) (*testEnv, *FakeFleetSDLC, *SlotCatalogFake) {
	t.Helper()
	env := newTestEnv(t)
	fake := NewFakeFleetSDLC()
	catalog := &SlotCatalogFake{Workspaces: map[string]WorkspaceIdentity{}, Errors: map[string]error{}}
	var roots []string
	for primary, fleetRoot := range fleets {
		roots = append(roots, primary)
		catalog.Workspaces[primary] = WorkspaceIdentity{FleetRoot: fleetRoot, PrimaryRoot: primary}
	}
	sort.Strings(roots)
	enrollRoots(t, env.Couch.Threads, roots...)
	env.Couch.Slots = catalog
	env.Couch.Fleet = SDLCFleetSource{IO: fake}
	return env, fake, catalog
}

func TestRecoverPlanRunsOneInventoryPerEnrolledFleet(t *testing.T) {
	env, fake, _ := recoverEnv(t, map[string]string{"/fa/alpha": "/fa", "/fa/beta": "/fa", "/fb/gamma": "/fb"})
	fake.Fleet("/fa").AddSlot("alpha:0")
	fake.Fleet("/fa").AddSlot("beta:0")
	fake.Fleet("/fb").AddSlot("gamma:0")
	plan, err := env.Couch.RecoverPlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var vantages []string
	for _, call := range fake.Calls {
		vantages = append(vantages, call.Dir)
	}
	if !slices.Equal(vantages, []string{"/fa/alpha", "/fb/gamma"}) {
		t.Fatalf("sdlc vantages = %v, want one per fleet from its first primary", vantages)
	}
	for _, address := range []string{"alpha:0", "beta:0", "gamma:0"} {
		findRow(t, plan, address)
	}
}

func TestRecoverPlanReadsAFleetOnceFromTwoVantages(t *testing.T) {
	env, fake, catalog := recoverEnv(t, map[string]string{"/fa/alpha": "/fa", "/fa/beta": "/fa"})
	fake.Fleet("/fa").AddSlot("alpha:0")
	fake.Fleet("/fa").AddSlot("beta:0")
	fake.Fleet("/fa").AddSlot("beta:1")
	if _, err := env.Couch.RecoverPlan(context.Background()); err != nil || len(fake.Calls) != 1 {
		t.Fatalf("calls = %d (%v), want one sdlc run for one fleet", len(fake.Calls), err)
	}
	// The dedupe backstop: two vantages resolving different roots but sdlc
	// answering both with the same document still give one row per address.
	catalog.Workspaces["/fa/beta"] = WorkspaceIdentity{FleetRoot: "/fa-other", PrimaryRoot: "/fa/beta"}
	fake.Route = map[string]string{"/fa/beta": "/fa"}
	plan, err := env.Couch.RecoverPlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, row := range plan.Rows {
		seen[row.Address]++
	}
	for _, address := range []string{"alpha:0", "beta:0", "beta:1"} {
		if seen[address] != 1 {
			t.Fatalf("%s appears %d times in %v", address, seen[address], seen)
		}
	}
}

func TestRecoverPlanFailedWorkspaceProbeIsUnavailableNotEmpty(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	primary := filepath.Join(root, "delta")
	if err := os.MkdirAll(primary, 0o700); err != nil {
		t.Fatal(err)
	}
	env, fake, catalog := recoverEnv(t, map[string]string{primary: root})
	catalog.Errors[primary] = errors.New("sdlc workspace: exit 1")
	plan, err := env.Couch.RecoverPlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.Calls) != 0 || len(plan.Fleets) != 1 || plan.Fleets[0].State != FleetObservationUnavailable || !strings.Contains(plan.Fleets[0].Error, "exit 1") {
		t.Fatalf("calls %d, fleets %+v", len(fake.Calls), plan.Fleets)
	}
	if row := findRow(t, plan, "delta:0"); row.Class != RecoverEvidenceUnavailable || row.Git.Source != "unknown" {
		t.Fatalf("delta:0 = %+v", row)
	}
}

func TestRecoverPlanDegradesPerSource(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	healthy, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	zeta := filepath.Join(root, "zeta")
	slot := filepath.Join(root, "worktree", "zeta-slot1", "zeta")
	alpha := filepath.Join(healthy, "alpha")
	// The healthy fleet's checkouts exist too, so probing them would be seen.
	for _, dir := range []string{zeta, slot, alpha, filepath.Join(healthy, "worktree", "alpha-slot1", "alpha")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// /gone/omega does not exist: Couch's store cannot enumerate it, which is
	// the store error this test needs (the inventory read fails, names do not).
	env, fake, _ := recoverEnv(t, map[string]string{alpha: healthy, "/gone/omega": "/gone", zeta: root})
	fake.Fleet("/gone").AddSlot("omega:0")
	fake.Fleet(healthy).AddSlot("alpha:0")
	fake.Fleet(healthy).AddSlot("alpha:1")
	fake.Fleet(healthy).SetBranch("alpha:1", "000011-x")
	fake.Fleet(healthy).Claim("alpha:1", "alpha#000011")
	fake.Fleet(root).AddSlot("zeta:0")
	fake.Fleet(root).SetSchema(2)
	status := strings.Join(slotGitStatusArgs, " ")
	env.Git.replies[GitCall{Dir: zeta, Args: status}] = "# branch.head main\n"
	env.Git.replies[GitCall{Dir: slot, Args: status}] = "# branch.head 000012-x\n1 .M N... 100644 100644 100644 a a f.go\n"
	plan, err := env.Couch.RecoverPlan(context.Background())
	if err != nil {
		t.Fatalf("a degraded source was returned as an error: %v", err)
	}
	states := map[string]string{}
	for _, f := range plan.Fleets {
		states[f.Root] = f.State
	}
	if states[healthy] != FleetObservationPresent || states["/gone"] != FleetObservationPresent || states[root] != FleetObservationUnsupported {
		t.Fatalf("fleets = %+v", plan.Fleets)
	}
	if plan.Couch.State != CouchObservationUnavailable || plan.Couch.Error == "" {
		t.Fatalf("couch = %+v", plan.Couch)
	}
	if row := findRow(t, plan, "alpha:1"); row.Git.Source != "sdlc" || row.Claims.Active != "alpha#000011" || row.Class != RecoverAgentUnknown {
		t.Fatalf("alpha:1 = %+v", row)
	}
	z0, z1 := findRow(t, plan, "zeta:0"), findRow(t, plan, "zeta:1")
	if z0.Git.Source != "local-probe" || z0.Git.Branch != "main" || z0.Claims.Quality != "unsupported" {
		t.Fatalf("zeta:0 = %+v", z0)
	}
	if z1.Git.Source != "local-probe" || z1.Git.Dirty != "yes" || z1.Git.Unlanded != "unknown" || z1.Git.Issue != "zeta#000012" {
		t.Fatalf("zeta:1 = %+v", z1)
	}
	var probed []string
	for _, op := range env.Git.Ops {
		if strings.Contains(op, "status --porcelain=v2") {
			probed = append(probed, op)
		}
	}
	if len(probed) != 2 {
		t.Fatalf("git probes = %v, want one per slot of the unsupported fleet only", probed)
	}
}
