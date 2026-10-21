package couchcmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func setTestAutomation(t *testing.T, s *messageService, root, action string) {
	t.Helper()
	if _, err := s.controlAutomation(root, action); err != nil {
		t.Fatal(err)
	}
}

func TestSupervisionGeneratedHubInputOffLeavesWorkerInputIndependent(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, _ := installTestSupervision(w, func(string) {})
	setTestAutomation(t, w.s, "/repo1", "off")
	const id = "suppressed-system-input"
	if _, err := c.sendGenerated(context.Background(), w.worker, w.hub, id, "automatic hub input", false); !errors.Is(err, errHubAutomationOff) {
		t.Fatalf("off hub admitted automatic input: %v", err)
	}
	if _, known, _ := w.s.broker.ReceiptSnapshot(id); known {
		t.Fatal("suppressed input entered broker")
	}
	c.mu.Lock()
	_, remembered := c.sends[id]
	c.mu.Unlock()
	if remembered {
		t.Fatal("suppressed input retained backlog")
	}
	if response := supervisionReport(w, "peer-report-while-off", "Worker report while automation is off"); response.Code != "accepted" {
		t.Fatalf("ordinary worker report gated: %+v", response)
	}
	supervisionWaitReceipt(t, w.serviceRig, w.worker, "peer-report-while-off", couchmessage.Submitted)

	for _, entry := range []struct {
		id   string
		from couchmessage.Binding
		body string
	}{{"worker-resend", w.hub, startupResendPrompt}, {couchmessage.WorkerStartNoticeID(w.worker, 7), w.worker, couchmessage.WorkerStartNoticeBody}} {
		if _, err := c.sendGenerated(context.Background(), entry.from, w.worker, entry.id, entry.body, false); err != nil {
			t.Fatalf("worker input was gated: %v", err)
		}
		supervisionWaitReceipt(t, w.serviceRig, entry.from, entry.id, couchmessage.Submitted)
	}
}

func TestWaitAutomationOffConsumesEpisodeWithoutReplayAndKeepsAttention(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	setTestAutomation(t, w.s, "/repo1", "off")
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("off wait input: %+v", got)
	}
	ep, ok := w.watcher.seen[w.worker]
	if !ok || !ep.notified {
		t.Fatal("off episode was not retired")
	}
	if _, ok := w.watcher.questions[bindingThread(w.worker)]; !ok {
		t.Fatal("off automation hid question attention")
	}
	setTestAutomation(t, w.s, "/repo1", "on")
	if got := w.tickAt(12 * time.Minute); len(got) != 0 {
		t.Fatal("off-to-on replayed wait episode")
	}
	w.mu.Lock()
	w.screen = []string{"A different question?", "❯"}
	w.mu.Unlock()
	w.client.Update(couchmessage.Observation{LastActivity: time.Now().Add(time.Minute), Sequence: 40})
	deadline := time.Now().Add(3 * time.Second)
	for len(w.tickAt(14*time.Minute)) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("new question never produced its own notice")
		}
	}
}

func TestSessionStaleAutomationOffSkipsAndConsumesAllowance(t *testing.T) {
	r := newSessionStaleRig(t)
	setTestAutomation(t, r.s, "/repo1", "off")
	r.watcher.tick(context.Background())
	if r.checks.Load() != 0 {
		t.Fatal("off stale reminder ran command")
	}
	setTestAutomation(t, r.s, "/repo1", "on")
	r.watcher.tick(context.Background())
	if r.checks.Load() != 0 {
		t.Fatal("on replayed skipped stale reminder")
	}
	r.now = r.now.Add(staleReminderEvery)
	r.watcher.tick(context.Background())
	r.waitMessage(t)
	if r.checks.Load() != 1 {
		t.Fatal("next due stale reminder missing")
	}
}

type automationFlipTail struct {
	tailEndpoint
	flip func()
}

func (e automationFlipTail) Tail(ctx context.Context, n int) (couchmessage.Tail, error) {
	tail, err := e.tailEndpoint.Tail(ctx, n)
	e.flip()
	return tail, err
}

func TestWaitAutomationOffOnDuringTailCannotReviveOccurrence(t *testing.T) {
	w := newWaitRig(t, true, true, claudeQuestionTail)
	setTestAutomation(t, w.s, "/repo1", "on")
	w.world.mu.Lock()
	w.world.endpoints[w.worker] = automationFlipTail{tailEndpoint: tailEndpoint{lines: &w.screen, mu: &w.mu}, flip: func() { setTestAutomation(t, w.s, "/repo1", "off"); setTestAutomation(t, w.s, "/repo1", "on") }}
	w.world.mu.Unlock()
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("delayed wait revived after off/on: %+v", got)
	}
	if got := w.tickAt(12 * time.Minute); len(got) != 0 {
		t.Fatal("retired delayed wait replayed")
	}
	if _, ok := w.watcher.questions[bindingThread(w.worker)]; !ok {
		t.Fatal("retired wait lost passive question attention")
	}
}

func TestSessionStaleAutomationOffOnDuringCheckCannotReviveOccurrence(t *testing.T) {
	r := newSessionStaleRig(t)
	setTestAutomation(t, r.s, "/repo1", "on")
	r.watcher.check = func(context.Context, string) (string, error) {
		r.checks.Add(1)
		setTestAutomation(t, r.s, "/repo1", "off")
		setTestAutomation(t, r.s, "/repo1", "on")
		return r.output, nil
	}
	r.watcher.tick(context.Background())
	c := r.s.supervision.Load()
	c.mu.Lock()
	admitted := len(c.sends)
	c.mu.Unlock()
	if admitted != 0 {
		t.Fatal("delayed stale check admitted after off/on")
	}
	select {
	case m := <-r.endpoint.deliveries:
		t.Fatalf("delayed stale check revived after off/on: %+v", m)
	default:
	}
	if r.checks.Load() != 1 {
		t.Fatal("check missing")
	}
	r.watcher.tick(context.Background())
	if r.checks.Load() != 1 {
		t.Fatal("retired delayed stale check replayed")
	}
}

func TestSupervisionUnknownTargetWorkspaceCannotBypassAutomation(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	c, _ := installTestSupervision(w, func(string) {})
	w.s.mu.Lock()
	identity := w.s.workspaces[w.hub]
	delete(w.s.workspaces, w.hub)
	w.s.mu.Unlock()
	defer func() { w.s.mu.Lock(); w.s.workspaces[w.hub] = identity; w.s.mu.Unlock() }()
	if !w.s.isConnected(w.hub) {
		t.Fatal("fixture needs connected actor before workspace publication")
	}
	const id = "unknown-workspace-automatic"
	if _, err := c.sendGenerated(context.Background(), w.worker, w.hub, id, "automatic input during registration", false); err == nil {
		t.Fatal("unknown workspace bypassed automation")
	}
	if _, known, _ := w.s.broker.ReceiptSnapshot(id); known {
		t.Fatal("unknown role entered broker")
	}
}

func TestSupervisionAutomationOffRetiresNudgeAndKeepsConsoleEscalation(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	var notices []string
	c, at := installTestSupervision(w, func(text string) { notices = append(notices, text) })
	setTestAutomation(t, w.s, "/repo1", "off")
	c.mu.Lock()
	c.state.report(w.worker, "completed-report", at, 0, true)
	c.mu.Unlock()
	c.tick(context.Background(), at.Add(31*time.Minute), nil)
	supervisionDrainStartup(t, w)
	if !c.state.notified(w.worker) || len(notices) != 1 {
		t.Fatalf("off suppression/escalation: notified=%t notices=%v", c.state.notified(w.worker), notices)
	}
	setTestAutomation(t, w.s, "/repo1", "on")
	c.tick(context.Background(), at.Add(32*time.Minute), nil)
	w.mu.Lock()
	got := len(w.delivered)
	w.mu.Unlock()
	if got != 0 || len(notices) != 1 {
		t.Fatalf("on replayed nudge/escalation: deliveries=%d notices=%v", got, notices)
	}
}
