package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

var errHubAutomationOff = errors.New("hub automation off")

func startupAutomationOn(startup *couchcore.StartupObservation, root string) bool {
	if startup == nil || startup.Kind != couchcore.StartupNormal {
		return false
	}
	hub := startup.Hubs[root]
	return hub.Survived && !hub.Unknown
}

// hubAutomationOwner holds volatile policy only; no callback performs IO while
// its mutex is held. Startup evidence seeds it once, explicit controls change it.
type hubAutomationOwner struct {
	mu      sync.Mutex
	state   couchcore.HubAutomation
	startup couchcore.StartupKind
	missing []string
	epochs  map[string]uint64
}

func newHubAutomationOwner(initial map[string]bool, startup couchcore.StartupKind, missing []string) *hubAutomationOwner {
	state, _ := couchcore.NewHubAutomation(initial)
	return &hubAutomationOwner{state: state, startup: startup, missing: append([]string(nil), missing...), epochs: map[string]uint64{}}
}

func (s *messageService) controlAutomation(hub, action string) (couchcore.HubAutomationStatus, error) {
	owner := s.automation
	if owner == nil {
		return couchcore.HubAutomationStatus{}, errors.New("hub automation unavailable")
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	status, err := couchcore.AutomationStatus(owner.state, hub)
	if err != nil {
		return status, err
	}
	if action == "toggle" {
		if status.Enabled {
			action = "off"
		} else {
			action = "on"
		}
	}
	if action != "status" {
		next, err := couchcore.AdvanceHubAutomation(owner.state, hub, action)
		if err != nil {
			return status, err
		}
		owner.state = next
		updated, _ := couchcore.AutomationStatus(next, hub)
		if updated.Enabled != status.Enabled {
			owner.epochs[hub]++
		}
		status = updated
	}
	status.Startup = string(owner.startup)
	status.MissingHubs = append([]string(nil), owner.missing...)
	return status, nil
}

// hubAutomationOn is shared with the scheduler's reserve and authorization
// boundaries. An unknown root never gains automatic-input authority.
func (s *messageService) hubAutomationOn(hubRoot string) bool {
	on, _ := s.hubAutomationPermit(hubRoot)
	return on
}

// A delayed occurrence retains this epoch. An intervening off/on cannot revive
// it merely because policy is on again at final admission.
func (s *messageService) hubAutomationPermit(hubRoot string) (bool, uint64) {
	owner := s.automation
	if owner == nil {
		return false, 0
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	status, err := couchcore.AutomationStatus(owner.state, hubRoot)
	return err == nil && status.Enabled, owner.epochs[hubRoot]
}

func (s *messageService) handleAutomation(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	refuse := func(err error) couchmessage.Response {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return refuse(err)
	}
	workspace, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return refuse(err)
	}
	if !s.authority.isHub(workspace) {
		return refuse(errors.New("only a hub may control its own automation; ask your hub"))
	}
	root := workspace.WorktreeRoot
	if r.Target != "" {
		target := ""
		if s.authority.hubs != nil {
			target, err = s.authority.hubs.target(r.Target)
		} else if r.Target == root || r.Target == caller.Slot {
			target = root
		} else {
			err = errors.New("unknown hub")
		}
		if err != nil {
			return refuse(err)
		}
		if target != root {
			return refuse(errors.New("a hub may control only its own automation; use the switcher for another hub"))
		}
	}
	if _, err = s.authority.current(ctx, caller); err != nil {
		return refuse(err)
	}
	if !s.isConnected(caller) {
		return refuse(errors.New("automation caller disconnected"))
	}
	if err = ctx.Err(); err != nil {
		return refuse(err)
	}
	status, err := s.controlAutomation(root, r.Body)
	if err != nil {
		return refuse(err)
	}
	return couchmessage.Response{Code: "ok", Automation: &status}
}

func parseAutomationCLI(args []string) (cliInvocation, error) {
	if len(args) < 2 || len(args) > 3 || (args[1] != "on" && args[1] != "off" && args[1] != "status") {
		return cliInvocation{}, errors.New("use couch --automation off|on|status [hub]")
	}
	inv := cliInvocation{kind: cliMessage, messageOp: "automation", messageBody: args[1]}
	if len(args) == 3 {
		if args[2] == "" {
			return cliInvocation{}, errors.New("hub must not be empty")
		}
		inv.ref = args[2]
	}
	return inv, nil
}

func runAutomationCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: automation requires a live Couch hub")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	r := callerIdentity(rt, "automation")
	r.Body = inv.messageBody
	r.Target = inv.ref
	if err = couchmessage.ValidateRequest(r); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err = call(ctx, socket, r, &result); err != nil {
		fmt.Fprintln(stderr, "couch: automation outcome unconfirmed; query status before retrying:", err)
		return 1
	}
	if result.Code != "ok" || result.Automation == nil {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
		return 1
	}
	status := result.Automation
	state := "off"
	if status.Enabled {
		state = "on"
	}
	fmt.Fprintf(stdout, "%s: hub automation %s (startup: %s)\n", status.Hub, state, status.Startup)
	for _, hub := range status.MissingHubs {
		fmt.Fprintln(stdout, "missing hub:", hub)
	}
	return 0
}

// sendAutomaticSystem is the sole broker ingress for generated conversation
// input. Worker-directed orientation and resend cues pass independently.
func (s *messageService) sendAutomaticSystem(ctx context.Context, from, to couchmessage.Binding, id, body string, permit hubInputPermit) (couchmessage.Receipt, error) {
	return s.broker.SendSystem(ctx, from, id, couchmessage.Route{Target: to.Slot}, body, s.automationAdmissionGuard(to, permit))
}

// automationAdmissionGuard pins the exact recipient and the occurrence's original
// hub/epoch through broker IO. Resolve cached role before entering the broker;
// the returned guard performs memory-only work under broker -> automation locks.
// The #737 total order is hub-effect gate -> scheduler authority -> broker ->
// automation. No callback here may acquire an earlier lock or perform IO.
func (s *messageService) automationAdmissionGuard(to couchmessage.Binding, permit hubInputPermit) couchmessage.AdmissionGuard {
	target := s.captureHubInputPermit(to)
	validTarget := target.root == permit.root && (target.root != "" || target.enabled)
	owner := s.automation
	return func(actual couchmessage.Binding) (func(), error) {
		if !permit.enabled || !validTarget || actual != to {
			return nil, errHubAutomationOff
		}
		if permit.root == "" {
			// A known worker has no hub-automation dependency.
			return func() {}, nil
		}
		if owner == nil {
			return nil, errHubAutomationOff
		}
		owner.mu.Lock()
		status, err := couchcore.AutomationStatus(owner.state, permit.root)
		if err != nil || !status.Enabled || owner.epochs[permit.root] != permit.epoch {
			owner.mu.Unlock()
			return nil, errHubAutomationOff
		}
		return owner.mu.Unlock, nil
	}
}
