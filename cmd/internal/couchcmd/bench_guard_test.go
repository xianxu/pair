package couchcmd

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func benchRequest(t *testing.T, caller couchmessage.Binding, op string, argv []string) couchmessage.Request {
	t.Helper()
	req := slotRequest(caller, op, "bench", "pair:2")
	req.Confirmed = true
	if op == couchcore.OpBenchStart {
		req.Agent = "claude"
		if argv != nil {
			args, err := couchmessage.NewSwitchArguments(argv)
			if err != nil {
				t.Fatal(err)
			}
			req.AgentArgs = args
		}
	}
	return req
}

// pair#847: bench verbs pass only for the hub whose .hub claims the slot, at
// admission and again at the queued effect; the main hub (production), a
// worker, a hubless Couch and a profile without explicit argv are refused.
func TestBenchSlotOperationsNeedTheClaimingHub(t *testing.T) {
	for _, op := range []string{couchcore.OpBenchStart, couchcore.OpBenchRetire} {
		for _, c := range []struct {
			name    string
			slot    int // the world roots slot n at /repo<n>: 0 main hub, 1 claiming hub
			noHubs  bool
			argv    []string
			refusal string
		}{
			{name: "claiming hub", slot: 1, argv: []string{"--model", "x"}},
			{name: "main hub", slot: 0, argv: []string{}, refusal: "main hub"},
			{name: "worker", slot: 3, argv: []string{}, refusal: "not a hub"},
			{name: "no hub configuration", slot: 1, noHubs: true, argv: []string{}, refusal: "hub configuration"},
			{name: "absent argv", slot: 1, refusal: "explicit argv"},
		} {
			if c.name == "absent argv" && op != couchcore.OpBenchStart {
				continue
			}
			t.Run(op+"/"+c.name, func(t *testing.T) {
				r, runner, _ := slotRig(t, func(a messageAuthority) messageAuthority {
					if !c.noHubs {
						a.hubs = guardHubState(t)
					}
					return a
				})
				caller := r.world.add(c.slot)
				r.world.set(caller, func(s *worldSlot) { n := c.slot; s.workspace.PrimaryRoot = "/repo"; s.workspace.Slot = &n })
				r.attach(caller, "caller")
				r.wrapper(caller)
				r.waitConnected(caller, true)
				r.waitWorkspace(caller)
				r.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
					return []couchcore.RepositoryName{{Key: "/repo", Dir: "pair"}}, nil
				}
				wireGuardTestMemo(r, func(context.Context, string) (couchcore.HubSlot, error) {
					return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
				})
				var check func(context.Context) error
				var opts couchcore.LiveRestartOptions
				r.slotOps.guardedRun = func(key, op, target string, o couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
					check, opts = guard, o
					return runner.run(key, op, target, o, started, progress, finished)
				}
				got := switchCall(t, r, benchRequest(t, caller, op, c.argv))
				if c.refusal != "" {
					if got.Code != "refused" || !strings.Contains(got.Error, c.refusal) || runner.count() != 0 {
						t.Fatalf("want refusal %q, got %+v (runs %d)", c.refusal, got, runner.count())
					}
					return
				}
				if got.Code != "accepted" || check == nil {
					t.Fatalf("claiming hub refused: %+v", got)
				}
				if op == couchcore.OpBenchStart && (opts.Bench == nil || opts.Bench.Agent != "claude" || strings.Join(opts.Bench.Argv, " ") != "--model x") {
					t.Fatalf("profile not carried: %+v", opts.Bench)
				}
				if err := check(context.Background()); err != nil {
					t.Fatal(err)
				}
				// The effect gate re-proves the claim: revoked, it refuses.
				h := r.s.authority.hubs
				h.mu.Lock()
				hubs := h.state.Hubs()
				hubs[1].Claims, _ = couchcore.ParseHubClaims(nil)
				next, err := h.state.WithClaims(hubs, h.state.Revision())
				h.state = next
				h.mu.Unlock()
				if err != nil {
					t.Fatal(err)
				}
				if err := check(context.Background()); err == nil || !strings.Contains(err.Error(), "not claimed") {
					t.Fatalf("revoked claim still authorized: %v", err)
				}
			})
		}
	}
}
