package couchcmd

import (
	"context"
	"errors"
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
			remember := func(ctx context.Context) {
				d, ok := ctx.Deadline()
				if !ok {
					t.Error("initial admission stage has no deadline")
				}
				firstDeadline.Store(&d)
			}
			if phase == "target" {
				agent := h.s.authority.agent
				var once atomic.Bool
				h.s.authority.agent = func(ctx context.Context, address couchcore.ThreadAddress) (string, error) {
					if once.CompareAndSwap(false, true) {
						remember(ctx)
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
				h.setResolver(func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
					<-ctx.Done()
					return couchcore.HubSlot{}, ctx.Err()
				})
			} else {
				// Use the uncached source so the successful target stage's
				// request deadline can be compared with the queue-name stage.
				h.s.hubTargets = nil
				h.s.authority.hubTarget = func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
					remember(ctx)
					timer := time.NewTimer(700 * time.Millisecond)
					defer timer.Stop()
					select {
					case <-timer.C:
					case <-ctx.Done():
						return couchcore.HubSlot{}, ctx.Err()
					}
					return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
				}
				h.slotOps.names = func(ctx context.Context) ([]couchcore.RepositoryName, error) {
					d, ok := ctx.Deadline()
					if !ok || !d.Equal(*firstDeadline.Load()) {
						t.Errorf("queue names renewed/lost deadline: %v, initial %v", d, *firstDeadline.Load())
					}
					<-ctx.Done()
					return nil, ctx.Err()
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), couchmessage.AdmissionTimeout)
			defer cancel()
			var response couchmessage.Response
			err := couchmessage.Call(ctx, h.brokerSock, slotRequest(h.caller, "resume", "shared-deadline", "pair:5"), &response)
			if err != nil {
				t.Fatalf("broker dropped admission instead of returning a typed %s refusal: %v", phase, err)
			}
			if response.Code != "refused" || !strings.Contains(response.Error, phase) {
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

func TestHubReceiptFilterKeepsTheRequestDeadline(t *testing.T) {
	h := newHubAdmissionRig(t)
	h.s.hubTargets = nil
	ctx, cancel := context.WithTimeout(t.Context(), 2*slotAdmissionBudget)
	defer cancel()
	want, _ := ctx.Deadline()
	h.s.authority.hubTarget = func(got context.Context, _ string) (couchcore.HubSlot, error) {
		deadline, ok := got.Deadline()
		if !ok || !deadline.Equal(want) {
			t.Errorf("receipt filtering renewed request deadline: %v, want %v", deadline, want)
		}
		return couchcore.HubSlot{Repository: "/repo", Number: 5}, nil
	}
	filter := h.s.hubReceiptFilter(ctx, couchcore.WorkspaceIdentity{PrimaryRoot: "/repo", WorktreeRoot: "/repo0"})
	if !filter(couchmessage.OperationReceipt{Target: "pair:5"}) {
		t.Fatal("same-hub receipt was filtered")
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
