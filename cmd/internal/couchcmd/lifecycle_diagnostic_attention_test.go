package couchcmd

import (
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestLifecycleMissingMetadataDiagnosticRaisesAttention(t *testing.T) {
	console, host, input := warningConsole(t)
	r := newServiceRig(t)
	r.restart(nil)
	published := make(chan struct{}, 1)
	p := startLifecyclePublisher(r.s, func(text string) { console.Notify(couchtty.Notice{Body: text}) }, func(text string) {
		console.Warn(couchtty.Notice{Body: text})
		published <- struct{}{}
	})
	p.observe(couchtty.LifecycleObservation{Kind: "restart-armed"})
	p.observe(couchtty.LifecycleObservation{Kind: "recovered", Path: "/unverified-slot", Initiation: "operator", Incarnation: "source", Observation: 1})
	p.observe(couchtty.LifecycleObservation{Kind: "restart-drained"})
	// observe queues a diagnostic; it does not join the publisher worker.
	// Acknowledge this producer's actual Warn call before sending panel input.
	select {
	case <-published:
	case <-testwait.After(t):
		t.Fatal("missing-metadata warning was not published")
	}
	openWarningAttention(t, input, host, "metadata unverified")
}

func TestLifecycleCapacityWarningIsOncePerPublisherLifetime(t *testing.T) {
	r := newServiceRig(t)
	r.world.hub = "/repo1"
	r.restart(nil)
	p := &lifecyclePublisher{service: r.s, roots: map[string]bool{}, identities: map[string]couchcore.WorkspaceIdentity{}, diagnostics: map[string]lifecycleDiagnostic{}}
	for i := range 128 {
		p.roots[fmt.Sprintf("/retained-hub-%d", i)] = true
	}
	warnings := 0
	for i := range 100 {
		p.mu.Lock()
		p.observeFactLocked(lifecycleSlotObservation(2, uint64(i+1)), p.automationSnapshot())
		for _, diagnostic := range p.diagnostics {
			if diagnostic.actionable {
				warnings++
			}
		}
		// This models the existing worker draining diagnostics each pass: the
		// one-shot guard must outlive that transient output projection.
		p.diagnostics = map[string]lifecycleDiagnostic{}
		p.mu.Unlock()
	}
	if warnings != 1 || len(p.roots) != 128 {
		t.Fatalf("capacity fault emitted %d warnings with %d roots, want one/128", warnings, len(p.roots))
	}
}

func TestLifecycleDiagnosticAttentionClassifiesTypedOutcome(t *testing.T) {
	for _, outcome := range []string{"failed", "refused", "unsupported", "cancelled", "unavailable", "uncertain"} {
		t.Run(outcome, func(t *testing.T) {
			p := &lifecyclePublisher{ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, diagnostics: map[string]lifecycleDiagnostic{}}
			p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeConfirmed, Fact: couchcore.LifecycleNoticeFact{Hub: "/hub", Slot: "pair:2", Incarnation: "worker", Observation: 1, Kind: couchcore.LifecycleStarted}})
			p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "hub", Ready: true})
			if len(p.ready) != 1 {
				t.Fatal("fixture has no lifecycle publication")
			}
			for token := range p.ready {
				p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: "/hub", Token: token, Outcome: outcome})
			}
			diagnostic, ok := p.diagnostics["/hub"]
			want := outcome == "failed" || outcome == "refused" || outcome == "unsupported"
			if !ok || diagnostic.actionable != want {
				t.Fatalf("typed %s outcome warning=%+v, want actionable=%v", outcome, diagnostic, want)
			}
		})
	}
}
