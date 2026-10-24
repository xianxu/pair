package couchcmd

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func installTestSupervision(w *waitRig, notice func(string)) (*supervisionCoordinator, time.Time) {
	at := time.Now()
	c := newSupervisionCoordinator(w.s, notice)
	c.now = func() time.Time { return at }
	w.s.supervision.Store(c)
	return c, at
}

func supervisionReport(w *waitRig, id, body string) couchmessage.Response {
	req := callerRequest(w.worker, "send")
	req.ID, req.Target, req.Body = id, "hub", body
	return w.s.handle(context.Background(), req)
}

func TestSupervisionIntegrationRefusedReportStillTracked(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, _ := installTestSupervision(w, func(string) {})
	w.s.broker.Disconnect(w.hub)
	response := supervisionReport(w, "lost-report", "Landed #530; PR is merged.")
	if response.Code == "accepted" {
		t.Fatalf("expected disconnected hub refusal: %+v", response)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.reportID(w.worker) != "lost-report" {
		t.Fatal("authenticated refused report was lost")
	}
}

func TestSupervisionIntegrationCompletionAfterMain(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	var notices []string
	_, at := installTestSupervision(w, func(s string) { notices = append(notices, s) })
	response := supervisionReport(w, "landed-report", "Landed #530; PR is merged.")
	if response.Code != "accepted" {
		t.Fatalf("report: %+v", response)
	}
	// Drain the original report before asking the watcher to supervise it.
	if got := w.tickAt(0); len(got) != 1 {
		t.Fatalf("initial deliveries: %+v", got)
	}
	supervisionDrainStartup(t, w)
	w.watcher.tick(context.Background(), at.Add(15*time.Minute))
	got := w.tickAt(16 * time.Minute)
	if len(got) != 2 {
		t.Fatalf("completion on main needs one nudge, got %+v", got)
	}
	if !strings.Contains(got[1].Body, "pair:2") {
		t.Fatalf("nudge omits worker: %q", got[1].Body)
	}
	w.watcher.tick(context.Background(), at.Add(30*time.Minute))
	w.watcher.tick(context.Background(), at.Add(31*time.Minute))
	if len(notices) != 1 || !strings.Contains(notices[0], "pair:2") {
		t.Fatalf("want one operator escalation naming worker: %v", notices)
	}
}

func TestSupervisionIntegrationEarlierWaitCoalescesNudge(t *testing.T) {
	w := newWaitRig(t, true, false, claudePermissionTail)
	_, at := installTestSupervision(w, func(string) {})
	if response := supervisionReport(w, "waiting-report", "Waiting for operator approval."); response.Code != "accepted" {
		t.Fatalf("report: %+v", response)
	}
	if got := w.tickAt(0); len(got) != 1 {
		t.Fatalf("initial report: %+v", got)
	}
	supervisionDrainStartup(t, w)
	if got := w.tickAt(11 * time.Minute); len(got) != 2 {
		t.Fatalf("existing wait notice missing: %+v", got)
	}
	w.watcher.tick(context.Background(), at.Add(15*time.Minute))
	if got := w.tickAt(16 * time.Minute); len(got) != 2 {
		t.Fatalf("duplicate 15-minute nudge after wait notice: %+v", got)
	}
}

// An uncertain endpoint records the one delivered prompt but cannot confirm it.
// Reconnect must not turn that uncertainty into another generated prompt.
type supervisionUncertainEndpoint struct {
	serviceEndpointFake
	mu       sync.Mutex
	messages []couchmessage.Message
}

func (e *supervisionUncertainEndpoint) Deliver(_ context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	e.messages = append(e.messages, m)
	e.mu.Unlock()
	return couchmessage.Receipt{Status: couchmessage.Submitted}, errors.New("acknowledgement lost")
}
func (e *supervisionUncertainEndpoint) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.messages)
}

func TestSupervisionIntegrationStartupOnceAcrossReconnectAndUncertainty(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.world.add(1), r.world.add(2)
	endpoint := &supervisionUncertainEndpoint{}
	r.world.endpoints[worker] = endpoint
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	r.attach(worker, "worker")
	_, stop := r.wrapper(worker)
	r.waitConnected(worker, true)
	c := newSupervisionCoordinator(r.s, func(string) {})
	at := time.Now()
	c.now = func() time.Time { return at }
	r.s.supervision.Store(c)
	watcher := newWaitWatcher(r.s, 10*time.Minute, t.TempDir())
	watcher.tick(context.Background(), at)
	deadline := time.Now().Add(3 * time.Second)
	for endpoint.count() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("startup prompt was not delivered")
		}
		time.Sleep(time.Millisecond)
	}
	stop()
	r.waitConnected(worker, false)
	watcher.tick(context.Background(), at.Add(time.Minute))
	r.wrapper(worker)
	r.waitConnected(worker, true)
	watcher.tick(context.Background(), at.Add(2*time.Minute))
	// Admission is synchronous but delivery is not. Drain any admitted work
	// before asserting the count, so an accidental second Send cannot hide
	// behind goroutine scheduling.
	deadline = time.Now().Add(3 * time.Second)
	for {
		rows, err := r.s.broker.Actors(context.Background(), hub)
		if err != nil {
			t.Fatal(err)
		}
		pending := false
		for _, row := range rows {
			if row.Binding == worker {
				pending = row.Pending
			}
		}
		if !pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("startup delivery did not settle")
		}
		time.Sleep(time.Millisecond)
	}
	c.mu.Lock()
	attempted := c.state.attempted[worker.Slot]
	c.mu.Unlock()
	if !attempted {
		t.Fatal("startup attempt was forgotten across reconnect")
	}
	if got := endpoint.count(); got != 1 {
		t.Fatalf("uncertain startup prompt replayed: %d deliveries", got)
	}
}

// supervisionDrainStartup waits for the worker's startup delivery, which the
// waitRig hub inbox barrier does not cover. Advance the synthetic supervision
// clock only after this real asynchronous broker work reaches its outcome.
func supervisionDrainStartup(t *testing.T, w *waitRig) {
	t.Helper()
	c := w.s.supervision.Load()
	c.mu.Lock()
	var ids []string
	for id, send := range c.sends {
		if send.system && send.from == w.hub {
			ids = append(ids, id)
		}
	}
	c.mu.Unlock()
	if len(ids) != 1 {
		t.Fatalf("want one initial startup prompt, got IDs %v", ids)
	}
	supervisionWaitReceipt(t, w.serviceRig, w.hub, ids[0], couchmessage.Submitted)
}

func supervisionEventually(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(time.Millisecond)
	}
}

func supervisionUnanswered(c *supervisionCoordinator, worker couchmessage.Binding) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state.unanswered(worker)
}

func TestSupervisionIntegrationUnauthenticatedReportIgnored(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, _ := installTestSupervision(w, func(string) {})
	request := callerRequest(w.worker, "send")
	request.Nonce = "forged-launch"
	request.ID, request.Target, request.Body = "forged-report", "hub", "Landed #530."
	if response := w.s.handle(context.Background(), request); response.Code == "accepted" {
		t.Fatalf("forged caller accepted: %+v", response)
	}
	if supervisionUnanswered(c, w.worker) {
		t.Fatal("unauthenticated caller manufactured a report")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.state.reports) != 0 {
		t.Fatal("unauthenticated caller created an episode under another identity")
	}
}

func TestSupervisionIntegrationOperatorSubmissionClearsReport(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, at := installTestSupervision(w, func(string) {})
	if response := supervisionReport(w, "before-human", "Landed #530."); response.Code != "accepted" {
		t.Fatalf("report: %+v", response)
	}
	if !supervisionUnanswered(c, w.worker) {
		t.Fatal("missing initial report")
	}
	w.client.Submit(couchmessage.Observation{LastActivity: at.Add(time.Second), Sequence: 2, Submission: 1})
	supervisionEventually(t, "genuine operator submission never reached broker", func() bool { n, ok := w.s.broker.Submission(w.worker); return ok && n == 1 })
	w.watcher.tick(context.Background(), at.Add(2*time.Second))
	if supervisionUnanswered(c, w.worker) {
		t.Fatal("genuine operator submission did not clear report")
	}
}

type supervisionControlledEndpoint struct {
	serviceEndpointFake
	mu        sync.Mutex
	busy      bool
	uncertain bool
	messages  []couchmessage.Message
}

func (e *supervisionControlledEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.busy {
		return couchmessage.ErrRecipientBusy
	}
	return ctx.Err()
}
func (e *supervisionControlledEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.messages = append(e.messages, m)
	if e.uncertain {
		return couchmessage.Receipt{Status: couchmessage.Submitted}, errors.New("acknowledgement lost")
	}
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}
func (e *supervisionControlledEndpoint) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.messages)
}
func (e *supervisionControlledEndpoint) releaseBusy() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.busy = false
}

func supervisionControlledRig(t *testing.T, endpoint *supervisionControlledEndpoint) (*serviceRig, couchmessage.Binding, couchmessage.Binding, *supervisionCoordinator) {
	t.Helper()
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	hub, worker := r.world.add(1), r.world.add(2)
	r.world.endpoints[worker] = endpoint
	r.attach(hub, "hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	r.attach(worker, "worker")
	r.wrapper(worker)
	r.waitConnected(worker, true)
	c := newSupervisionCoordinator(r.s, func(string) {})
	at := time.Now()
	c.now = func() time.Time { return at }
	r.s.supervision.Store(c)
	return r, hub, worker, c
}

func supervisionSendCLI(t *testing.T, r *serviceRig, from couchmessage.Binding, target, id, body string) couchmessage.Receipt {
	t.Helper()
	req := callerRequest(from, "send")
	req.ID, req.Target, req.Body = id, target, body
	response := r.s.handle(context.Background(), req)
	if response.Code != "accepted" || response.Receipt == nil {
		t.Fatalf("send %s: %+v", id, response)
	}
	return *response.Receipt
}
func supervisionWaitReceipt(t *testing.T, r *serviceRig, caller couchmessage.Binding, id string, status couchmessage.Status) {
	t.Helper()
	supervisionEventually(t, "receipt "+id+" did not become "+string(status), func() bool {
		receipt, err := r.s.broker.Status(caller, id)
		return err == nil && receipt.Status == status
	})
}

func TestSupervisionIntegrationHubReplyMustBeConfirmed(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		name := "confirmed"
		if uncertain {
			name = "uncertain"
		}
		t.Run(name, func(t *testing.T) {
			endpoint := &supervisionControlledEndpoint{busy: true, uncertain: uncertain}
			r, hub, worker, c := supervisionControlledRig(t, endpoint)
			supervisionSendCLI(t, r, worker, "hub", "report", "Landed #530.")
			supervisionWaitReceipt(t, r, worker, "report", couchmessage.Submitted)
			reply := supervisionSendCLI(t, r, hub, worker.Slot, "reply", "Thanks; stop.")
			if reply.Status != couchmessage.Queued {
				t.Fatalf("busy endpoint did not queue reply: %+v", reply)
			}
			if !supervisionUnanswered(c, worker) {
				t.Fatal("queued reply cleared report before delivery")
			}
			if endpoint.count() != 0 {
				t.Fatal("busy worker received reply")
			}
			endpoint.releaseBusy()
			status := couchmessage.Submitted
			if uncertain {
				status = couchmessage.Indeterminate
			}
			supervisionWaitReceipt(t, r, hub, "reply", status)
			if got := supervisionUnanswered(c, worker); got != uncertain {
				t.Fatalf("unanswered=%v after %s reply", got, name)
			}
		})
	}
}

func TestSupervisionIntegrationStartupQueuedOnceWithoutOwedRow(t *testing.T) {
	endpoint := &supervisionControlledEndpoint{busy: true}
	r, hub, worker, c := supervisionControlledRig(t, endpoint)
	r.s.recordDeliveries(t.TempDir())
	recorder := r.s.deliveries.Load()
	if recorder == nil {
		t.Fatal("delivery recorder not installed")
	}
	watcher := newWaitWatcher(r.s, 10*time.Minute, t.TempDir())
	at := time.Now()
	watcher.tick(context.Background(), at)
	c.mu.Lock()
	var promptID string
	for id, send := range c.sends {
		if send.system && send.from == hub {
			promptID = id
		}
	}
	c.mu.Unlock()
	if promptID == "" {
		t.Fatal("startup prompt was not attempted")
	}
	supervisionWaitReceipt(t, r, hub, promptID, couchmessage.Queued)
	watcher.tick(context.Background(), at.Add(time.Minute))
	if endpoint.count() != 0 {
		t.Fatal("busy worker received startup prompt")
	}
	endpoint.releaseBusy()
	supervisionWaitReceipt(t, r, hub, promptID, couchmessage.Submitted)
	watcher.tick(context.Background(), at.Add(2*time.Minute))
	if endpoint.count() != 1 {
		t.Fatalf("startup prompt delivered %d times", endpoint.count())
	}
	entries, err := recorder.ledger.Read(worker.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("startup prompt created session-file obligations: %+v", entries)
	}
	// The same body from an actual hub CLI send is ordinary conversation. The
	// exemption is trusted provenance, not text an agent can impersonate.
	supervisionSendCLI(t, r, hub, worker.Slot, "ordinary-copy", startupResendPrompt)
	supervisionWaitReceipt(t, r, hub, "ordinary-copy", couchmessage.Submitted)
	entries, err = recorder.ledger.Read(worker.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "ordinary-copy" {
		t.Fatalf("ordinary hub delivery was omitted: %+v", entries)
	}
}

// The existing inbox becomes the extra hub; main is a separate connected slot.
func multiHubWaitRig(t *testing.T) (*waitRig, couchmessage.Binding) {
	t.Helper()
	claims, err := couchcore.ParseHubClaims([]byte("repo:2\n"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: "/repo3", Workspace: "/", Slot: couchcore.HubSlot{Repository: "/repo", Number: 3}},
		{Root: "/repo1", Workspace: "/", Slot: couchcore.HubSlot{Repository: "/repo", Number: 1}, Claims: claims},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := newWaitRig(t, true, false, claudePermissionTail, func(a messageAuthority) messageAuthority {
		a.hub, a.hubs = "/repo3", &hubRuntime{state: state}
		return a
	})
	main := w.world.add(3)
	w.v2wrapper(main, "h3", false)
	for b, n := range map[couchmessage.Binding]int{w.hub: 1, w.worker: 2, main: 3} {
		number := n
		w.world.set(b, func(s *worldSlot) {
			s.workspace.PrimaryRoot = "/repo"
			s.workspace.Slot = &number
			s.workspace.FleetRoot = "/"
		})
		w.s.mu.Lock()
		identity := w.s.workspaces[b]
		identity.PrimaryRoot = "/repo"
		identity.Slot = &number
		identity.FleetRoot = "/"
		w.s.workspaces[b] = identity
		w.s.mu.Unlock()
	}
	return w, main
}

func TestMultiHubWaitRigSeedsOwnershipBeforeServiceStart(t *testing.T) {
	w, _ := multiHubWaitRig(t)
	for _, phase := range []string{"initial", "restart"} {
		if phase == "restart" {
			w.restart(nil)
		}
		for _, root := range []string{"/repo3", "/repo1"} {
			status, err := w.s.controlAutomation(root, "status")
			if err != nil || !status.Enabled {
				t.Fatalf("%s: configured hub %s lacks startup automation: %+v %v", phase, root, status, err)
			}
		}
	}
}

func TestHubSupervisionWaitUsesOwnHub(t *testing.T) {
	w, _ := multiHubWaitRig(t)
	got := w.tickAt(11 * time.Minute)
	if len(got) != 1 || got[0].To != w.hub {
		t.Fatalf("own-hub wait delivery: %+v", got)
	}
}

func TestHubSupervisionReplyEvidenceAndResend(t *testing.T) {
	w, main := multiHubWaitRig(t)
	c, at := installTestSupervision(w, func(string) {})
	c.state.report(w.worker, "report", at, 0, true)
	c.sends["foreign"] = supervisionSend{from: main, at: at.Add(time.Second)}
	c.delivered(couchmessage.Message{ID: "foreign", From: main, To: w.worker}, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	if !c.state.unanswered(w.worker) {
		t.Fatal("foreign hub cleared own-hub report")
	}
	c.now = func() time.Time { return at.Add(time.Second) }
	c.sending(w.hub, couchmessage.Request{ID: "own", Target: w.worker.Slot})
	c.delivered(couchmessage.Message{ID: "own", From: w.hub, To: w.worker}, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	if c.state.unanswered(w.worker) {
		t.Fatal("own hub reply did not clear report")
	}
	c.tick(context.Background(), at, nil)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, send := range c.sends {
		if send.prompt != "" && (send.prompt != w.worker.Slot || send.from != w.hub) {
			t.Fatalf("wrong startup prompt: %+v", send)
		}
	}
	if !c.state.attempted[w.worker.Slot] || c.state.attempted[main.Slot] || c.state.attempted[w.hub.Slot] {
		t.Fatalf("prompt targets: %+v", c.state.attempted)
	}
}

func TestHubSupervisionReassignmentChangesBinding(t *testing.T) {
	w, main := multiHubWaitRig(t)
	if got, ok := w.s.hubBindingFor(w.worker); !ok || got != w.hub {
		t.Fatalf("initial owner: %+v %v", got, ok)
	}
	h := w.s.authority.hubs
	h.mu.Lock()
	next, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo3", Revision: h.state.Revision()})
	if err == nil {
		h.state = next
	}
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := w.s.hubBindingFor(w.worker); !ok || got != main {
		t.Fatalf("moved owner: %+v %v", got, ok)
	}
	c, at := installTestSupervision(w, func(string) {})
	c.state.report(w.worker, "report", at, 0, true)
	c.sends["old"] = supervisionSend{from: w.hub, at: at.Add(time.Second)}
	c.delivered(couchmessage.Message{ID: "old", From: w.hub, To: w.worker}, couchmessage.Receipt{Status: couchmessage.Submitted}, nil)
	if !c.state.unanswered(w.worker) {
		t.Fatal("old owner cleared report after move")
	}
}

func TestHubSupervisionDelayedReplyPreservesAdmissionOwnership(t *testing.T) {
	for _, scenario := range []string{"foreign-to-own", "own-to-foreign", "same-hub"} {
		t.Run(scenario, func(t *testing.T) {
			endpoint := &supervisionControlledEndpoint{busy: true}
			r, b := hubMessageRigWithWorkerEndpoint(t, endpoint)
			c := newSupervisionCoordinator(r.s, func(string) {})
			at := time.Now()
			c.now = func() time.Time { return at.Add(time.Second) }
			c.state.report(b[2], "report", at, 0, true)
			r.s.supervision.Store(c)
			sender := b[1]
			if scenario == "foreign-to-own" {
				sender = b[0]
			}
			receipt := supervisionSendCLI(t, r, sender, b[2].Slot, "delayed-reply", "Thanks; stop.")
			if receipt.Status != couchmessage.Queued {
				t.Fatalf("want queued: %+v", receipt)
			}
			if scenario != "same-hub" {
				h := r.s.authority.hubs
				h.mu.Lock()
				next, err := h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
				if err == nil {
					h.state = next
				}
				h.mu.Unlock()
				if err != nil {
					t.Fatal(err)
				}
			}
			endpoint.releaseBusy()
			supervisionWaitReceipt(t, r, sender, "delayed-reply", couchmessage.Submitted)
			if got, want := supervisionUnanswered(c, b[2]), scenario != "same-hub"; got != want {
				t.Fatalf("unanswered=%v, want %v after delayed reply", got, want)
			}
		})
	}
}
