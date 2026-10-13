package couchcmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestSwitchTimeoutConfiguration(t *testing.T) {
	for _, tc := range []struct {
		brief, settle string
		valid         bool
	}{
		{"", "", true}, {"1s", "2s", true}, {"0s", "2s", false}, {"2s", "1s", false}, {"1m", "31m", false}, {"bad", "", false},
	} {
		b, s, err := switchTimeouts(func(k string) string {
			if k == "COUCH_SWITCH_BRIEF_TIMEOUT" {
				return tc.brief
			}
			return tc.settle
		})
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
		if err == nil && (b <= 0 || s <= b || s > 30*time.Minute) {
			t.Fatalf("invalid accepted durations %v %v", b, s)
		}
	}
}

func TestSwitchObservationBindsWrapperSessionAndAvoidsBusyGitProbe(t *testing.T) {
	world := newMessageWorld()
	b := world.add(2)
	s := &messageService{authority: world.authority(), workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{b: world.slots[b.Thread()].workspace}}
	settled := false
	live := couchmessage.SlotLiveness{Binding: b, Session: 1, Settled: &settled}
	s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
	s.connected.Store(&map[couchmessage.Binding]bool{b: true})
	source := switchSourceToken(live)
	a := switchAdmission{Content: couchcore.SwitchContent{Kind: couchcore.SwitchContentFresh}, Source: source, Address: couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}}
	got := s.observeSwitch(context.Background(), a)
	if !got.Alive || !got.SettledKnown || got.Settled || got.GitKnown || world.counts()[2] != 0 {
		t.Fatalf("busy observation %+v counts %v", got, world.counts())
	}
	settled = true
	got = s.observeSwitch(context.Background(), a)
	if !got.Alive || !got.Clean || !got.GitKnown || !got.Settled {
		t.Fatalf("settled observation %+v", got)
	}
	live.Session = 2
	s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
	if got = s.observeSwitch(context.Background(), a); got.Alive {
		t.Fatalf("replacement accepted: %+v", got)
	}
}

func TestSwitchGuardRefusesChangedDirtyUnknownAndBusy(t *testing.T) {
	world := newMessageWorld()
	b := world.add(2)
	settled := true
	s := &messageService{authority: world.authority(), workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{b: world.slots[b.Thread()].workspace}}
	live := couchmessage.SlotLiveness{Binding: b, Session: 1, Settled: &settled}
	s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
	s.connected.Store(&map[couchmessage.Binding]bool{b: true})
	guard := &switchServiceGuard{}
	guard.service.Store(s)
	address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
	source := switchSourceToken(live)
	ops := newSlotOperations(nil, nil, time.Now)
	co := newSwitchCoordinator(ops, time.Minute, 2*time.Minute)
	s.switches.Store(co)
	a := switchAdmission{Source: source, Target: b.Slot, Address: address, Content: couchcore.SwitchContent{Kind: couchcore.SwitchContentFresh}, Fallback: couchcore.SwitchContent{Kind: couchcore.SwitchContentFallback, Body: "Pointers"}}
	co.observe = s.observeSwitch
	co.enqueue = func(switchAdmission, couchcore.SwitchContent, func(), func(any, error)) error { return nil }
	response := co.admit(context.Background(), b, couchmessage.Request{Op: "switch-agent", ID: "guard", Target: b.Slot, Agent: "claude", Confirmed: true}, a)
	if response.Code != "accepted" {
		t.Fatal(response)
	}
	co.activate("guard")
	co.step(context.Background())
	a.RequestID = "guard"
	source = switchTransactionToken(a)
	if err := guard.ConfirmAgentSwitch(context.Background(), address, source); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []string{"dirty", "unknown", "busy", "changed"} {
		t.Run(tc, func(t *testing.T) {
			old := s.authority.branch
			defer func() { s.authority.branch = old; settled = true }()
			switch tc {
			case "dirty":
				s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{Dirty: true}, nil
				}
			case "unknown":
				s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{}, errors.New("git failed")
				}
			case "busy":
				settled = false
			case "changed":
				source += "different"
			}
			if err := guard.ConfirmAgentSwitch(context.Background(), address, source); err == nil {
				t.Fatal("unsafe source accepted")
			}
		})
	}
}
