package couchcore

import (
	"context"
	"errors"
	"slices"
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
