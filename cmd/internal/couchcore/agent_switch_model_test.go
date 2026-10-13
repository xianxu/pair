package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/orientation"
	"strings"
	"testing"
	"time"
)

func switchSpec() AgentSwitchSpec {
	now := time.Unix(1000, 0)
	return AgentSwitchSpec{RequestID: "request", Source: "authenticated-binding/session", Agent: "codex", Fallback: SwitchContent{Kind: SwitchContentFallback, Body: "Read previous session."}, AdmittedAt: now, BriefDeadline: now.Add(time.Minute), OverallDeadline: now.Add(2 * time.Minute)}
}
func switchEvent(kind AgentSwitchEventKind) AgentSwitchEvent {
	s := switchSpec()
	return AgentSwitchEvent{Kind: kind, At: s.AdmittedAt.Add(time.Second), RequestID: s.RequestID, Source: s.Source, Agent: s.Agent, Body: " exact brief\n", Alive: true, SettledKnown: true, Settled: true, GitKnown: true, Clean: true}
}
func newSwitch(t *testing.T, content SwitchContent) AgentSwitch {
	t.Helper()
	spec := switchSpec()
	spec.Content = content
	s, err := NewAgentSwitch(spec)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestAgentSwitchContent(t *testing.T) {
	for _, c := range []SwitchContent{{Kind: SwitchContentFresh}, {Kind: SwitchContentAuthored, Body: " exact\ttext\n"}, {Kind: SwitchContentFallback, Body: "pointer"}} {
		if err := ValidateSwitchContent(c); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []SwitchContent{{}, {Kind: "unknown", Body: "x"}, {Kind: SwitchContentFresh, Body: "x"}, {Kind: SwitchContentAuthored}, {Kind: SwitchContentAuthored, Body: " \n"}, {Kind: SwitchContentAuthored, Body: "\xff"}, {Kind: SwitchContentAuthored, Body: "\x1b[2J"}, {Kind: SwitchContentFallback, Body: strings.Repeat("x", orientation.MaxBodyBytes+1)}} {
		if ValidateSwitchContent(c) == nil {
			t.Errorf("accepted invalid content %+v", c)
		}
	}
}
func TestAgentSwitchConstructor(t *testing.T) {
	for _, mutate := range []func(*AgentSwitchSpec){func(s *AgentSwitchSpec) { s.RequestID = "" }, func(s *AgentSwitchSpec) { s.Source = "" }, func(s *AgentSwitchSpec) { s.Agent = "" }, func(s *AgentSwitchSpec) { s.Source = strings.Repeat("x", 16385) }, func(s *AgentSwitchSpec) { s.Agent = "a\n" }, func(s *AgentSwitchSpec) { s.BriefDeadline = s.AdmittedAt }, func(s *AgentSwitchSpec) { s.OverallDeadline = s.BriefDeadline }, func(s *AgentSwitchSpec) { s.AdmittedAt = time.Time{} }, func(s *AgentSwitchSpec) { s.Content = SwitchContent{Body: "hidden"} }, func(s *AgentSwitchSpec) { s.Fallback = SwitchContent{Kind: SwitchContentAuthored, Body: "x"} }} {
		spec := switchSpec()
		mutate(&spec)
		if _, err := NewAgentSwitch(spec); err == nil {
			t.Errorf("accepted invalid spec %+v", spec)
		}
	}
}
func TestAgentSwitchCallbackIdentityAndDuplicates(t *testing.T) {
	initial := newSwitch(t, SwitchContent{})
	for _, mutate := range []func(*AgentSwitchEvent){func(e *AgentSwitchEvent) { e.RequestID = "old" }, func(e *AgentSwitchEvent) { e.Source = "old" }, func(e *AgentSwitchEvent) { e.Agent = "old" }} {
		e := switchEvent(AgentSwitchCallback)
		mutate(&e)
		got, effect, err := AdvanceAgentSwitch(initial, e)
		if err == nil || got != initial || effect != AgentSwitchNoEffect {
			t.Fatal("foreign callback changed request")
		}
	}
	e := switchEvent(AgentSwitchCallback)
	s, effect, err := AdvanceAgentSwitch(initial, e)
	if err != nil || effect != AgentSwitchNoEffect || s.Phase() != AgentSwitchWaitingSettled || s.Content().Body != e.Body {
		t.Fatalf("callback: %+v %v %v", s, effect, err)
	}
	duplicate, effect, err := AdvanceAgentSwitch(s, e)
	if duplicate != s || effect != AgentSwitchNoEffect || err != nil {
		t.Fatal("duplicate was not acknowledged")
	}
	e.Body = "changed"
	got, _, err := AdvanceAgentSwitch(s, e)
	if err == nil || got != s {
		t.Fatal("conflicting callback accepted")
	}
}
func TestAgentSwitchObservedSafety(t *testing.T) {
	for _, mutate := range []func(*AgentSwitchEvent){func(e *AgentSwitchEvent) { e.SettledKnown = false }, func(e *AgentSwitchEvent) { e.Settled = false }, func(e *AgentSwitchEvent) { e.GitKnown = false; e.Settled = false }} {
		s := newSwitch(t, SwitchContent{Kind: SwitchContentFresh})
		e := switchEvent(AgentSwitchObserve)
		mutate(&e)
		got, effect, err := AdvanceAgentSwitch(s, e)
		if err != nil || got != s || effect != AgentSwitchNoEffect {
			t.Fatal("unknown/busy observation advanced")
		}
	}
	for _, mutate := range []func(*AgentSwitchEvent){func(e *AgentSwitchEvent) { e.Source = "replacement" }, func(e *AgentSwitchEvent) { e.Alive = false }, func(e *AgentSwitchEvent) { e.Clean = false }, func(e *AgentSwitchEvent) { e.GitKnown = false }} {
		s := newSwitch(t, SwitchContent{Kind: SwitchContentFresh})
		e := switchEvent(AgentSwitchObserve)
		mutate(&e)
		got, effect, _ := AdvanceAgentSwitch(s, e)
		if got.Phase() != AgentSwitchRefused || effect != AgentSwitchPublish {
			t.Fatal("unsafe observation did not refuse")
		}
	}
}
func TestAgentSwitchFallbackAndExpiry(t *testing.T) {
	s := newSwitch(t, SwitchContent{})
	e := switchEvent(AgentSwitchTick)
	e.At = switchSpec().BriefDeadline
	s, effect, err := AdvanceAgentSwitch(s, e)
	if err != nil || s.Content().Kind != SwitchContentFallback || effect != AgentSwitchFallbackNotice {
		t.Fatal("missing fallback")
	}
	callback := switchEvent(AgentSwitchCallback)
	callback.At = e.At
	got, _, err := AdvanceAgentSwitch(s, callback)
	if err == nil || got != s {
		t.Fatal("late callback replaced fallback")
	}
	e.At = switchSpec().OverallDeadline
	s, effect, err = AdvanceAgentSwitch(s, e)
	if err != nil || s.Phase() != AgentSwitchRefused || effect != AgentSwitchPublish {
		t.Fatal("overall timeout")
	}
	// A delayed callback cannot race past an unprocessed timer.
	s = newSwitch(t, SwitchContent{})
	callback.At = switchSpec().BriefDeadline
	s, effect, err = AdvanceAgentSwitch(s, callback)
	if err == nil || s.Content().Kind != SwitchContentFallback || effect != AgentSwitchFallbackNotice {
		t.Fatal("callback bypassed deadline")
	}
}
func TestAgentSwitchCompletionAndTerminalSequences(t *testing.T) {
	for _, content := range []SwitchContent{{Kind: SwitchContentFresh}, {Kind: SwitchContentAuthored, Body: "exact"}} {
		s := newSwitch(t, content)
		s, effect, err := AdvanceAgentSwitch(s, switchEvent(AgentSwitchObserve))
		if err != nil || effect != AgentSwitchEnqueue || s.Phase() != AgentSwitchSwitching {
			t.Fatal("did not enqueue")
		}
		got, effect, _ := AdvanceAgentSwitch(s, switchEvent(AgentSwitchObserve))
		if got != s || effect != AgentSwitchNoEffect {
			t.Fatal("enqueued twice")
		}
		s, _, err = AdvanceAgentSwitch(s, switchEvent(AgentSwitchRegistered))
		if err != nil {
			t.Fatal(err)
		}
		if content.Kind == SwitchContentFresh {
			if s.Phase() != AgentSwitchSucceeded {
				t.Fatal("fresh not completed")
			}
		} else {
			if s.Phase() != AgentSwitchDeliveryPending {
				t.Fatal("authored prematurely completed")
			}
			s, _, _ = AdvanceAgentSwitch(s, switchEvent(AgentSwitchDeliverySubmitted))
		}
		if !s.Terminal() {
			t.Fatal("not terminal")
		}
		for kind := AgentSwitchCallback; kind <= AgentSwitchDeliveryUnconfirmed; kind++ {
			got, effect, _ := AdvanceAgentSwitch(s, switchEvent(kind))
			if got != s || effect != AgentSwitchNoEffect {
				t.Fatal("terminal resurrected")
			}
		}
	}
}
func TestAgentSwitchEventSequences(t *testing.T) {
	// Exhaustively interleave short event streams, including callbacks, deadlines,
	// completion before launch, and cancellation. At most one transaction may run.
	kinds := []AgentSwitchEventKind{AgentSwitchObserve, AgentSwitchCallback, AgentSwitchCancel, AgentSwitchRegistered, AgentSwitchDeliverySubmitted, AgentSwitchFailed, AgentSwitchDeliveryUnconfirmed, AgentSwitchTick}
	var visit func(AgentSwitch, int, int)
	visit = func(s AgentSwitch, depth, enqueues int) {
		if depth == 0 {
			return
		}
		for _, kind := range kinds {
			e := switchEvent(kind)
			if kind == AgentSwitchTick {
				e.At = switchSpec().BriefDeadline
			}
			next, effect, _ := AdvanceAgentSwitch(s, e)
			n := enqueues
			if effect == AgentSwitchEnqueue {
				n++
			}
			if n > 1 {
				t.Fatal("multiple enqueue effects")
			}
			if s.Terminal() && next != s {
				t.Fatal("terminal changed")
			}
			visit(next, depth-1, n)
		}
	}
	visit(newSwitch(t, SwitchContent{}), 5, 0)
}

func TestAgentSwitchExpiredDuplicateCallback(t *testing.T) {
	s := newSwitch(t, SwitchContent{})
	s, _, _ = AdvanceAgentSwitch(s, switchEvent(AgentSwitchCallback))
	s, _, _ = AdvanceAgentSwitch(s, switchEvent(AgentSwitchObserve))
	e := switchEvent(AgentSwitchCallback)
	e.At = switchSpec().OverallDeadline
	got, effect, err := AdvanceAgentSwitch(s, e)
	if err == nil || got != s || effect != AgentSwitchNoEffect {
		t.Fatal("expired callback was acknowledged")
	}
}

func TestAgentSwitchFailureAndUnconfirmed(t *testing.T) {
	s := newSwitch(t, SwitchContent{Kind: SwitchContentAuthored, Body: "exact"})
	// Completion events arriving before a switch are inert.
	for _, kind := range []AgentSwitchEventKind{AgentSwitchRegistered, AgentSwitchFailed, AgentSwitchDeliverySubmitted, AgentSwitchDeliveryUnconfirmed} {
		got, effect, err := AdvanceAgentSwitch(s, switchEvent(kind))
		if err != nil || got != s || effect != AgentSwitchNoEffect {
			t.Fatal("early completion changed state")
		}
	}
	switching, _, _ := AdvanceAgentSwitch(s, switchEvent(AgentSwitchObserve))
	failed, _, _ := AdvanceAgentSwitch(switching, switchEvent(AgentSwitchFailed))
	if failed.Phase() != AgentSwitchFailure || !failed.Terminal() {
		t.Fatal("failure not terminal")
	}
	delivering, _, _ := AdvanceAgentSwitch(switching, switchEvent(AgentSwitchRegistered))
	unconfirmed, _, _ := AdvanceAgentSwitch(delivering, switchEvent(AgentSwitchDeliveryUnconfirmed))
	if unconfirmed.Phase() != AgentSwitchUnconfirmed || !unconfirmed.Terminal() {
		t.Fatal("partial delivery not terminal")
	}
	for _, initial := range []AgentSwitch{s, switching, delivering} {
		cancelled, _, _ := AdvanceAgentSwitch(initial, switchEvent(AgentSwitchCancel))
		if !cancelled.Terminal() {
			t.Fatal("cancel not terminal")
		}
	}
}

func FuzzValidateSwitchContent(f *testing.F) {
	f.Add("authored", "exact\n")
	f.Add("fresh", "")
	f.Add("fallback", "\x1b")
	f.Fuzz(func(t *testing.T, kind, body string) {
		c := SwitchContent{Kind: SwitchContentKind(kind), Body: body}
		before := c
		err := ValidateSwitchContent(c)
		if c != before {
			t.Fatal("rewrote content")
		}
		if err == nil {
			if c.Kind == SwitchContentFresh {
				if c.Body != "" {
					t.Fatal("fresh has body")
				}
			} else if c.Kind != SwitchContentAuthored && c.Kind != SwitchContentFallback {
				t.Fatal("unknown kind accepted")
			} else if orientation.ValidateBody(c.Body) != nil {
				t.Fatal("invalid body accepted")
			}
		}
	})
}
