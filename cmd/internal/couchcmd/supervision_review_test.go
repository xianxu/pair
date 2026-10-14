package couchcmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestSupervisionReviewWaitNoticeDoesNotCreateFileObligation(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	installTestSupervision(w, func(string) {})
	w.s.recordDeliveries(t.TempDir())
	got := w.tickAt(11 * time.Minute)
	if len(got) != 1 {
		t.Fatalf("want ordinary wait notice: %+v", got)
	}
	entries, err := w.s.deliveries.Load().ledger.Read(w.hub.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("generated wait notice created session-file obligations: %+v", entries)
	}
}

func TestSupervisionReviewFailedAttemptsDoNotExhaustReportTracking(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, _ := installTestSupervision(w, func(string) {})
	// Authenticated sends refused by the endpoint must not consume a lifetime
	// quota on tracking later reports and replies.
	w.mu.Lock()
	w.refuse, w.refusal = couchmessage.MaxReceipts+8, couchmessage.ErrUnavailable
	w.mu.Unlock()
	for i := 0; i < couchmessage.MaxReceipts+8; i++ {
		response := supervisionReport(w, fmt.Sprintf("refused-%d", i), "Landed #530.")
		if response.Code == "accepted" {
			t.Fatalf("unavailable endpoint accepted send: %+v", response)
		}
	}
	supervisionSendCLI(t, w.serviceRig, w.worker, "hub", "latest-report", "Landed #530.")
	supervisionWaitReceipt(t, w.serviceRig, w.worker, "latest-report", couchmessage.Submitted)
	c.mu.Lock()
	latest := c.state.reportID(w.worker)
	c.mu.Unlock()
	if latest != "latest-report" {
		t.Errorf("latest report lost after failed attempts: %q", latest)
	}
	supervisionSendCLI(t, w.serviceRig, w.hub, w.worker.Slot, "latest-reply", "Thanks; stop.")
	supervisionWaitReceipt(t, w.serviceRig, w.hub, "latest-reply", couchmessage.Submitted)
	if supervisionUnanswered(c, w.worker) {
		t.Fatal("confirmed reply lost after failed attempts")
	}
}

type supervisionReviewOutcomeEndpoint struct {
	serviceEndpointFake
	outcome  couchmessage.Status
	messages chan couchmessage.Message
}

func (e *supervisionReviewOutcomeEndpoint) Deliver(_ context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.messages <- m
	return couchmessage.Receipt{Status: e.outcome, Detail: "test terminal outcome"}, nil
}

func TestSupervisionReviewTerminalPromptFailureNoticesOnce(t *testing.T) {
	for _, outcome := range []couchmessage.Status{couchmessage.Indeterminate, couchmessage.Cancelled, couchmessage.Expired} {
		t.Run(string(outcome), func(t *testing.T) {
			r := newServiceRig(t)
			r.world.hub = "/repo1"
			r.restart(nil)
			hub, worker := r.world.add(1), r.world.add(2)
			endpoint := &supervisionReviewOutcomeEndpoint{outcome: outcome, messages: make(chan couchmessage.Message, 8)}
			r.world.endpoints[worker] = endpoint
			r.attach(hub, "hub")
			r.wrapper(hub)
			r.waitConnected(hub, true)
			r.attach(worker, "worker")
			r.wrapper(worker)
			r.waitConnected(worker, true)
			var mu sync.Mutex
			var notices []string
			c := newSupervisionCoordinator(r.s, func(s string) { mu.Lock(); defer mu.Unlock(); notices = append(notices, s) })
			at := time.Now()
			c.now = func() time.Time { return at }
			r.s.supervision.Store(c)
			watcher := newWaitWatcher(r.s, 10*time.Minute, t.TempDir())
			watcher.tick(context.Background(), at)
			var prompt couchmessage.Message
			select {
			case prompt = <-endpoint.messages:
			case <-time.After(3 * time.Second):
				t.Fatal("startup prompt not delivered")
			}
			supervisionWaitReceipt(t, r, hub, prompt.ID, outcome)
			watcher.tick(context.Background(), at.Add(time.Minute))
			watcher.tick(context.Background(), at.Add(2*time.Minute))
			mu.Lock()
			defer mu.Unlock()
			if len(notices) != 1 || !strings.Contains(notices[0], worker.Slot) || !strings.Contains(notices[0], "fresh status") {
				t.Fatalf("want one actionable terminal prompt notice: %v", notices)
			}
			select {
			case m := <-endpoint.messages:
				t.Fatalf("terminal prompt failure resent: %+v", m)
			default:
			}
		})
	}
}

func TestSupervisionReviewQueuedPromptCancellationNoticesOnce(t *testing.T) {
	endpoint := &supervisionControlledEndpoint{busy: true}
	r, hub, worker, c := supervisionControlledRig(t, endpoint)
	var notices []string
	c.notice = func(s string) { notices = append(notices, s) }
	at := time.Now()
	watcher := newWaitWatcher(r.s, 10*time.Minute, t.TempDir())
	watcher.tick(context.Background(), at)
	c.mu.Lock()
	var id string
	for key, send := range c.sends {
		if send.system && send.from == hub {
			id = key
		}
	}
	c.mu.Unlock()
	if id == "" {
		t.Fatal("no startup prompt admitted")
	}
	supervisionWaitReceipt(t, r, hub, id, couchmessage.Queued)
	// A disconnected sender cancels held work without calling Deliver. Receipt
	// reconciliation must still surface the failed prompt to the operator.
	r.s.broker.Disconnect(hub)
	receipt, known := r.s.broker.ReceiptSnapshot(id)
	if !known || receipt.Status != couchmessage.Cancelled {
		t.Fatalf("queued prompt cancellation: %+v, known=%v", receipt, known)
	}
	watcher.tick(context.Background(), at.Add(time.Minute))
	watcher.tick(context.Background(), at.Add(2*time.Minute))
	if len(notices) != 1 || !strings.Contains(notices[0], worker.Slot) || !strings.Contains(notices[0], "fresh status") {
		t.Fatalf("want one actionable queued cancellation notice: %v", notices)
	}
	if endpoint.count() != 0 {
		t.Fatalf("cancelled queued prompt delivered %d times", endpoint.count())
	}
}
