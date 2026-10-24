package couchcmd

import (
	"context"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestAutomationOwnerExplicitTransitions(t *testing.T) {
	s := &messageService{automation: newHubAutomationOwner(map[string]bool{"/a": false, "/b": true}, couchcore.StartupUnknown, []string{"/a"})}
	for _, action := range []string{"status", "off", "off"} {
		got, err := s.controlAutomation("/a", action)
		if err != nil || got.Enabled {
			t.Fatalf("%s: %+v %v", action, got, err)
		}
		got.MissingHubs[0] = "mutated"
	}
	if !s.hubAutomationOn("/b") || s.hubAutomationOn("/missing") {
		t.Fatal("cross-hub policy")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.controlAutomation("/a", "toggle"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := s.controlAutomation("/a", "status")
	if err != nil || got.Enabled || got.MissingHubs[0] != "/a" {
		t.Fatalf("atomic toggles %+v %v", got, err)
	}
}

func TestAutomationUnknownStartupDefaultsOff(t *testing.T) {
	s := &messageService{automation: newHubAutomationOwner(map[string]bool{"/hub": false}, couchcore.StartupUnknown, []string{"/hub"})}
	if s.hubAutomationOn("/hub") {
		t.Fatal("unknown startup enabled automation")
	}
	for _, action := range []string{"operator-input", "worker-report", "reconnect"} {
		if _, err := s.controlAutomation("/hub", action); err == nil {
			t.Fatalf("implicit event %s enabled control", action)
		}
	}
	if s.hubAutomationOn("/hub") {
		t.Fatal("input changed policy")
	}
}

func TestAutomationServiceAdmission(t *testing.T) {
	for _, fault := range []string{"none", "worker", "stale-nonce", "replaced", "disconnected", "other-hub", "invalid-action"} {
		t.Run(fault, func(t *testing.T) {
			world := newMessageWorld()
			world.hub = "/repo0"
			r := &serviceRig{t: t, world: world, slotGit: map[string]couchcore.SlotGitStatus{}}
			r.init()
			hub, worker := r.connect(0), r.connect(1)
			if _, err := r.s.controlAutomation("/repo0", "off"); err != nil {
				t.Fatal(err)
			}
			caller := hub
			if fault == "worker" {
				caller = worker
			}
			req := callerRequest(caller, "automation")
			req.Body = "on"
			switch fault {
			case "stale-nonce":
				req.Nonce = "old"
			case "replaced":
				world.set(hub, func(slot *worldSlot) { slot.pidFile++ })
			case "disconnected":
				r.disconnect(hub)
			case "other-hub":
				req.Target = "other:0"
			case "invalid-action":
				req.Body = "toggle"
			}
			got := r.s.handleAutomation(context.Background(), req)
			if (got.Code == "ok") != (fault == "none") {
				t.Fatalf("%+v", got)
			}
			if r.s.hubAutomationOn("/repo0") != (fault == "none") {
				t.Fatal("refused control changed policy")
			}
		})
	}
}

func TestAutomationPermitFencesOffOn(t *testing.T) {
	s := &messageService{automation: newHubAutomationOwner(map[string]bool{"/hub": true}, couchcore.StartupNormal, nil)}
	on, epoch := s.hubAutomationPermit("/hub")
	if !on {
		t.Fatal("initial permit")
	}
	s.controlAutomation("/hub", "on")
	_, same := s.hubAutomationPermit("/hub")
	if same != epoch {
		t.Fatal("idempotent on invalidated permit")
	}
	s.controlAutomation("/hub", "off")
	s.controlAutomation("/hub", "on")
	on, next := s.hubAutomationPermit("/hub")
	if !on || next == epoch {
		t.Fatal("off/on revived old permit")
	}
}

func TestAutomationCannotControlAnotherConfiguredHub(t *testing.T) {
	r, bindings := hubMessageRig(t)
	req := callerRequest(bindings[0], "automation")
	req.Body = "off"
	req.Target = "/repo1"
	got := r.s.handleAutomation(context.Background(), req)
	if got.Code != "refused" || !r.s.hubAutomationOn("/repo0") || !r.s.hubAutomationOn("/repo1") {
		t.Fatalf("foreign configured hub policy changed: %+v", got)
	}
	req.Target = "/repo0"
	got = r.s.handleAutomation(context.Background(), req)
	if got.Code != "ok" || r.s.hubAutomationOn("/repo0") || !r.s.hubAutomationOn("/repo1") {
		t.Fatalf("own hub policy: %+v", got)
	}
}

func TestUnknownStartupCannotEnablePositiveHubHint(t *testing.T) {
	observation := couchcore.StartupObservation{Kind: couchcore.StartupUnknown, Hubs: map[string]couchcore.StartupHubObservation{"/hub": {Survived: true}}}
	if startupAutomationOn(&observation, "/hub") {
		t.Fatal("unknown startup enabled automation from incomplete positive hint")
	}
	observation.Kind = couchcore.StartupNormal
	if !startupAutomationOn(&observation, "/hub") {
		t.Fatal("proved normal survivor stayed off")
	}
	if startupAutomationOn(&observation, "/missing") || startupAutomationOn(nil, "/hub") {
		t.Fatal("missing evidence enabled automation")
	}
}
