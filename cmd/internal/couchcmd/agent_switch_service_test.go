package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestSwitchGuardRefusesChangedDirtyUnknownAndBusy(t *testing.T) {
	for _, fault := range []string{"none", "form", "dirty", "unknown-git", "busy", "unknown-settle", "changed", "disconnect", "changed-during-git"} {
		t.Run(fault, func(t *testing.T) {
			world := newMessageWorld()
			b := world.add(2)
			settled := true
			s := &messageService{authority: world.authority(), workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{b: world.slots[b.Thread()].workspace}}
			live := couchmessage.SlotLiveness{Binding: b, Session: 1, Settled: &settled}
			s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
			s.connected.Store(&map[couchmessage.Binding]bool{b: true})
			guard := &switchServiceGuard{}
			guard.service.Store(s)
			token := switchSourceToken(live)
			switch fault {
			case "form":
				token = ""
			case "dirty":
				s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{Dirty: true}, nil
				}
			case "unknown-git":
				s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					return couchcore.SlotGitStatus{}, errors.New("git unavailable")
				}
			case "busy":
				settled = false
			case "unknown-settle":
				live.Settled = nil
				s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
			case "changed":
				token += "other"
			case "disconnect":
				s.connected.Store(&map[couchmessage.Binding]bool{})
			case "changed-during-git":
				s.authority.branch = func(context.Context, string) (couchcore.SlotGitStatus, error) {
					live.Session++
					s.liveness.Store(&map[string]couchmessage.SlotLiveness{b.Slot: live})
					return couchcore.SlotGitStatus{}, nil
				}
			}
			err := guard.ConfirmAgentSwitch(context.Background(), couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}, token)
			if (err == nil) != (fault == "none" || fault == "form") {
				t.Fatalf("guard %s: %v", fault, err)
			}
			if fault == "busy" && world.counts()[2] != 0 {
				t.Fatal("busy source caused git probe")
			}
		})
	}
}

// The queue retains accepted jobs until tests explicitly execute/finish them.
// Like Console it reserves one canonical key and releases it on completion.
type leanSwitchQueue struct {
	mu      sync.Mutex
	jobs    []leanSwitchJob
	pending map[string]bool
}
type leanSwitchJob struct {
	prepare  func(context.Context) (couchcore.OperationCall, error)
	started  func()
	finished func(any, error)
}

func (q *leanSwitchQueue) enqueue(key string, p func(context.Context) (couchcore.OperationCall, error), start func(), finish func(any, error)) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pending == nil {
		q.pending = map[string]bool{}
	}
	if q.pending[key] {
		return couchtty.ErrRemotePending
	}
	q.pending[key] = true
	q.jobs = append(q.jobs, leanSwitchJob{p, start, func(v any, e error) { q.mu.Lock(); delete(q.pending, key); q.mu.Unlock(); finish(v, e) }})
	return nil
}
func (q *leanSwitchQueue) count() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.jobs) }
func (q *leanSwitchQueue) job(i int) leanSwitchJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.jobs[i]
}

func leanSwitchRig(t *testing.T) (*serviceRig, *leanSwitchQueue, couchmessage.Binding, couchmessage.Binding, *couchmessage.SessionClient) {
	t.Helper()
	w := newMessageWorld()
	w.hub = "/repo0"
	r := &serviceRig{t: t, world: w, slotGit: map[string]couchcore.SlotGitStatus{}}
	r.slotOps = newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo/.git", Dir: "pair", Alias: "p"}}, nil
	}, time.Now)
	r.init()
	hub := r.connect(0)
	worker := r.world.add(2)
	client := (&waitRig{serviceRig: r}).v2wrapper(worker, "worker", true)
	r.waitConnected(worker, true)
	testwait.Until(t, "settled worker", func() bool {
		v, ok := r.s.LivenessForThread(worker.Scope, worker.Tag)
		return ok && v.Settled != nil && *v.Settled
	})
	queue := &leanSwitchQueue{}
	guard := &switchServiceGuard{}
	guard.service.Store(r.s)
	r.s.switches.Store(&agentSwitchService{guard: guard, prepare: func(_ context.Context, a couchcore.ThreadAddress, agent string, _ *[]string) (couchcore.PreparedAgentSwitch, error) {
		return couchcore.PreparedAgentSwitch{Address: a, Profile: couchcore.LaunchProfile{Agent: agent, Argv: []string{"--model", "known"}}, Fingerprint: "accepted-config"}, nil
	}, enqueue: queue.enqueue})
	return r, queue, hub, worker, client
}
func switchRequest(b couchmessage.Binding, id, target string) couchmessage.Request {
	r := callerRequest(b, "switch-agent")
	r.ID = id
	r.Target = target
	r.Agent = "claude"
	r.Confirmed = true
	return r
}
func switchCall(t *testing.T, r *serviceRig, request couchmessage.Request) couchmessage.Response {
	t.Helper()
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, request, &response); err != nil {
		t.Fatal(err)
	}
	return response
}
func TestLeanSwitchSocketAdmissionAndReceipt(t *testing.T) {
	r, q, hub, worker, _ := leanSwitchRig(t)
	for i, req := range []couchmessage.Request{switchRequest(worker, "foreign", hub.Slot), switchRequest(hub, "own", hub.Slot), switchRequest(hub, "self", "self")} {
		if got := switchCall(t, r, req); got.Code == "accepted" {
			t.Fatalf("invalid %d admitted: %+v", i, got)
		}
	}
	if q.count() != 0 {
		t.Fatal("refusal queued an effect")
	}
	req := switchRequest(hub, "one", worker.Slot)
	got := switchCall(t, r, req)
	if got.Code != "accepted" || got.Operation == nil || got.Operation.Agent != "claude" || q.count() != 1 {
		t.Fatalf("admit %+v jobs %d", got, q.count())
	}
	job := q.job(0)
	call, err := job.prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if call.Args["known-config"] != "true" || call.Args["argv"] != "" || call.Args["body"] != "" || call.Args["fingerprint"] != "accepted-config" {
		t.Fatalf("call %#v", call)
	}
	job.started()
	job.finished(couchcore.SwitchAgentResult{}, errors.New("target failed; resume old thread"))
	// Repeat stays readable even after target replacement makes fresh admission impossible.
	r.world.set(worker, func(s *worldSlot) { s.live = false })
	again := switchCall(t, r, req)
	if again.Code != "accepted" || again.Operation.Status != couchmessage.ReceiptFailed || q.count() != 1 {
		t.Fatalf("repeat %+v", again)
	}
	req.Agent = "grok"
	if got := switchCall(t, r, req); got.Code != "id-conflict" {
		t.Fatalf("changed intent %+v", got)
	}
	status := switchCall(t, r, statusRequest(hub, "one"))
	if status.Operation == nil || status.Operation.Status != couchmessage.ReceiptFailed {
		t.Fatalf("status %+v", status)
	}
}
func TestLeanSwitchBusyMissingConfigAndQueuedSourceChange(t *testing.T) {
	for _, fault := range []string{"busy", "config", "queued-busy"} {
		t.Run(fault, func(t *testing.T) {
			r, q, hub, worker, client := leanSwitchRig(t)
			if fault == "config" {
				old := r.s.switches.Load()
				next := *old
				next.prepare = func(context.Context, couchcore.ThreadAddress, string, *[]string) (couchcore.PreparedAgentSwitch, error) {
					return couchcore.PreparedAgentSwitch{}, fmt.Errorf("operator must set known-good config")
				}
				r.s.switches.Store(&next)
			}
			if fault == "busy" {
				client.Settle(false)
				testwait.Until(t, "busy source", func() bool {
					v, ok := r.s.LivenessForThread(worker.Scope, worker.Tag)
					return ok && v.Settled != nil && !*v.Settled
				})
			}
			got := switchCall(t, r, switchRequest(hub, "request", worker.Slot))
			if fault != "queued-busy" {
				if got.Code == "accepted" || q.count() != 0 {
					t.Fatalf("unsafe admit %+v jobs %d", got, q.count())
				}
				return
			}
			if got.Code != "accepted" {
				t.Fatalf("admit %+v", got)
			}
			client.Settle(false)
			testwait.Until(t, "busy queued source", func() bool {
				v, ok := r.s.LivenessForThread(worker.Scope, worker.Tag)
				return ok && v.Settled != nil && !*v.Settled
			})
			if _, err := q.job(0).prepare(context.Background()); err == nil {
				t.Fatal("busy source executed")
			}
		})
	}
}
