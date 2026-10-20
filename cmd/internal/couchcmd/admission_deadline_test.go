package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// These are transport regressions: two individually bounded admission
// stages must not add up to more than the broker's 2 s exchange.
func TestSlotAdmissionSocketSharesOneDeadlineAcrossStages(t *testing.T) {
	for _, phase := range []string{"target", "queue names"} {
		t.Run(phase, func(t *testing.T) {
			h := newHubAdmissionRig(t)
			var firstDeadline atomic.Pointer[time.Time]
			if phase == "queue names" {
				if _, err := h.s.hubTargets.Fresh(t.Context(), "pair:5"); err != nil {
					t.Fatal(err)
				}
				h.slotOps.names = func(ctx context.Context) ([]couchcore.RepositoryName, error) {
					d, ok := ctx.Deadline()
					if !ok || !d.Equal(*firstDeadline.Load()) {
						t.Errorf("queue names renewed/lost deadline: %v, initial %v", d, *firstDeadline.Load())
					}
					<-ctx.Done()
					return nil, ctx.Err()
				}
			} else {
				h.setResolver(func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
					<-ctx.Done()
					return couchcore.HubSlot{}, ctx.Err()
				})
			}
			agent := h.s.authority.agent
			var once atomic.Bool
			h.s.authority.agent = func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
				if once.CompareAndSwap(false, true) {
					d, ok := ctx.Deadline()
					if !ok {
						t.Error("initial admission stage has no deadline")
					}
					firstDeadline.Store(&d)
					timer := time.NewTimer(700 * time.Millisecond)
					defer timer.Stop()
					select {
					case <-timer.C:
					case <-ctx.Done():
						return "", ctx.Err()
					}
				}
				return agent(ctx, address)
			}
			ctx, cancel := context.WithTimeout(t.Context(), couchmessage.AdmissionTimeout)
			defer cancel()
			var response couchmessage.Response
			err := couchmessage.Call(ctx, h.brokerSock, slotRequest(h.caller, "resume", "shared-deadline", "pair:5"), &response)
			if err != nil {
				t.Fatalf("broker dropped admission instead of returning typed %s refusal: %v", phase, err)
			}
			wantCode := "refused"
			if phase == "target" {
				wantCode = couchmessage.CodeWarming
			}
			if response.Code != wantCode || !strings.Contains(response.Error, phase) {
				t.Fatalf("%s refusal: %+v", phase, response)
			}
			if h.runner.count() != 0 {
				t.Fatal("expired admission enqueued work")
			}
		})
	}
}

func TestExpiredAdmissionNeverEnqueuesEvenIfNamesIgnoreCancellation(t *testing.T) {
	r, runner, _ := slotRig(t)
	caller := r.world.add(0)
	ctx, cancel := context.WithCancel(t.Context())
	r.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		cancel()
		return []couchcore.RepositoryName{{Key: "/src/pair", Dir: "pair"}}, nil
	}
	got := r.slotOps.handle(ctx, caller, slotRequest(caller, "resume", "expired", "pair:5"))
	if got.Code != "refused" || !strings.Contains(got.Error, "queue names") || runner.count() != 0 {
		t.Fatalf("expired admission %+v, queued %d", got, runner.count())
	}
}

func TestLifecycleAdmissionStartsDeadlineBeforeAuthentication(t *testing.T) {
	for _, op := range []string{"resume", "operation-status", "move-slot"} {
		t.Run(op, func(t *testing.T) {
			h := newHubAdmissionRig(t)
			seen := false
			probe := func(ctx context.Context) error {
				seen = true
				d, ok := ctx.Deadline()
				if !ok || time.Until(d) > slotAdmissionBudget {
					t.Errorf("%s initial authentication has no admission deadline: %v", op, d)
				}
				return errors.New("stop after checking initial authentication context")
			}
			if op == "move-slot" {
				h.s.hubMoves.Store(&hubMoveService{})
				h.s.authority.thread = func(ctx context.Context, _ couchmessage.Binding) (string, error) { return "", probe(ctx) }
			} else {
				h.s.authority.agent = func(ctx context.Context, _ couchcore.ThreadAddress) (string, error) { return "", probe(ctx) }
			}
			req := slotRequest(h.caller, op, "initial", "pair:5")
			if op == "move-slot" {
				req.ToHub = "/repo1"
			}
			if op == "operation-status" {
				req.Target = ""
			}
			h.s.handle(context.Background(), req)
			if !seen {
				t.Fatal("initial authentication was not checked")
			}
			if h.runner.count() != 0 {
				t.Fatal("failed authentication enqueued work")
			}
		})
	}
}

func TestHubReceiptFilterHonorsRequestCancellation(t *testing.T) {
	h := newHubAdmissionRig(t)
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	h.setResolver(func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	filter := h.s.hubReceiptFilter(ctx, couchcore.WorkspaceIdentity{PrimaryRoot: "/repo", WorktreeRoot: "/repo0"})
	done := make(chan bool, 1)
	go func() { done <- filter(couchmessage.OperationReceipt{Target: "pair:5"}) }()
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("memo flight did not start")
	}
	cancel()
	select {
	case allowed := <-done:
		if allowed {
			t.Fatal("cancelled request authorized receipt")
		}
	case <-testwait.After(t):
		t.Fatal("receipt filtering waited for the service-owned resolver after request cancellation")
	}
}

func TestBlockedStatusFilterDoesNotHoldTheReceiptMutex(t *testing.T) {
	r, runner, _ := slotRig(t)
	caller := r.world.add(0)
	r.slotOps.mu.Lock()
	r.slotOps.receipts[receiptKey{caller.Scope, caller.Tag, "switch"}] = couchmessage.OperationReceipt{ID: "switch", Op: "switch-agent", Target: "pair:5", Status: couchmessage.ReceiptQueued}
	r.slotOps.mu.Unlock()
	entered, release, filtered := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		close(release)
		select {
		case <-filtered:
		case <-testwait.After(t):
			t.Error("status filter did not finish")
		}
	}()
	go func() {
		defer close(filtered)
		r.slotOps.switchStatus("switch", func(couchmessage.OperationReceipt) bool {
			close(entered)
			<-release
			return true
		})
	}()
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("status filter did not start")
	}
	admitted := make(chan couchmessage.Response, 1)
	go func() {
		admitted <- r.slotOps.handle(t.Context(), caller, slotRequest(caller, "resume", "parallel", "pair:5"))
	}()
	select {
	case got := <-admitted:
		if got.Code != "accepted" || runner.count() != 1 {
			t.Fatalf("admission while status filter blocked: %+v", got)
		}
	case <-testwait.After(t):
		t.Fatal("blocked status filter held the receipt mutex")
	}
}

// A request deadline does not bound OSProvisionIO's joined process-group
// cancellation. Caller admission must refuse without entering that native seam.
func TestMissingCallerWorkspaceSocketNeverRunsSynchronousResolver(t *testing.T) {
	h := newHubAdmissionRig(t)
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "sdlc"), []byte("#!/bin/sh\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	h.world.set(h.caller, func(s *worldSlot) { s.root = root; s.workspace.WorktreeRoot = root })
	h.s.mu.Lock()
	delete(h.s.workspaces, h.caller)
	h.s.mu.Unlock()
	var nativeCalls atomic.Int64
	native := couchcore.NewOSSlotCatalog(couchcore.OSProvisionIO{})
	h.s.authority.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
		nativeCalls.Add(1)
		return native.ResolveWorkspace(ctx, path)
	}
	ctx, cancel := context.WithTimeout(t.Context(), couchmessage.AdmissionTimeout)
	defer cancel()
	var response couchmessage.Response
	if err := couchmessage.Call(ctx, h.brokerSock, slotRequest(h.caller, "resume", "missing-workspace", "pair:5"), &response); err != nil {
		t.Fatalf("uncached workspace lost socket reply in native teardown: %v", err)
	}
	if response.Code != "refused" || !strings.Contains(response.Error, "reconnect") {
		t.Fatalf("missing caller workspace: %+v", response)
	}
	if nativeCalls.Load() != 0 || h.runner.count() != 0 {
		t.Fatalf("native calls %d, queued %d", nativeCalls.Load(), h.runner.count())
	}
}

func TestMissingHubMemoNeverFallsBackToSynchronousTargetResolution(t *testing.T) {
	h := newHubAdmissionRig(t)
	h.s.hubTargets = nil
	var calls atomic.Int64
	h.s.authority.hubTarget = func(context.Context, string) (couchcore.HubSlot, error) {
		calls.Add(1)
		return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
	}
	got := h.admit("no-memo")
	if got.Code != "refused" || !strings.Contains(got.Error, "unavailable") || calls.Load() != 0 || h.runner.count() != 0 {
		t.Fatalf("missing memo %+v, native calls %d, queued %d", got, calls.Load(), h.runner.count())
	}
}
