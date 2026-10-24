package couchcore

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func noticeFact(slot string, observation uint64) LifecycleNoticeEvent {
	return LifecycleNoticeEvent{Kind: LifecycleNoticeConfirmed, Fact: LifecycleNoticeFact{Hub: "/hub", Slot: slot, Address: ThreadAddress{RepoScope: "repo", Tag: ThreadTag(slot)}, Incarnation: "one", Observation: observation, Kind: LifecycleStarted, Initiation: "automatic"}}
}
func noticeStep(s *LifecycleNoticeState, e LifecycleNoticeEvent) []LifecycleNoticeEffect {
	var effects []LifecycleNoticeEffect
	*s, effects = AdvanceLifecycleNotices(*s, e)
	return effects
}
func noticeReady(s *LifecycleNoticeState) []LifecycleNoticeEffect {
	return noticeStep(s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "hub-one", Ready: true})
}
func noticePublications(es []LifecycleNoticeEffect) []LifecycleNoticeEffect {
	var out []LifecycleNoticeEffect
	for _, e := range es {
		if e.Kind == LifecycleNoticePublish {
			out = append(out, e)
		}
	}
	return out
}
func TestLifecycleNoticeRestartDrainBarrier(t *testing.T) {
	var s LifecycleNoticeState
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeArm})
	noticeReady(&s)
	for i := 0; i < 20; i++ {
		if got := noticePublications(noticeStep(&s, noticeFact(fmt.Sprint(i), 1))); len(got) != 0 {
			t.Fatal("published before drain")
		}
	}
	es := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDrain, Reason: "cancelled"}))
	if len(es) != 1 || strings.Count(es[0].Body, "started") != 20 || !strings.Contains(es[0].Body, "cancelled") {
		t.Fatalf("drained summary: %+v", es)
	}
	if len(noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDrain}))) != 0 {
		t.Fatal("duplicate drain published")
	}
}
func TestLifecycleNoticeFreezeRetiresOnlyFrozenRevisions(t *testing.T) {
	var s LifecycleNoticeState
	noticeStep(&s, noticeFact("slot", 1))
	first := noticePublications(noticeReady(&s))[0]
	before := s
	noticeStep(&s, noticeFact("slot", 2))
	if !reflect.DeepEqual(before.hubs["/hub"].parts, s.hubs["/hub"].parts) {
		t.Fatal("frozen payload changed")
	}
	next := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: first.Token, Outcome: "submitted"}))
	if len(next) != 1 || strings.Contains(next[0].Body, "changes") {
		t.Fatalf("new revision lost: %+v", next)
	}
	if len(noticePublications(noticeStep(&s, noticeFact("slot", 1)))) != 0 {
		t.Fatal("stale fact published")
	}
}
func TestLifecycleNoticeRefusalSuppressesUntilNewEvidence(t *testing.T) {
	for _, outcome := range []string{"refused", "capacity", "unavailable", "unsupported"} {
		t.Run(outcome, func(t *testing.T) {
			var s LifecycleNoticeState
			noticeStep(&s, noticeFact("slot", 1))
			p := noticePublications(noticeReady(&s))[0]
			if len(noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeAdmission, Hub: "/hub", Token: p.Token, Outcome: outcome}))) != 0 {
				t.Fatal("retry after refusal")
			}
			if len(noticePublications(noticeReady(&s))) != 0 {
				t.Fatal("same readiness retried")
			}
			es := noticePublications(noticeStep(&s, noticeFact("slot", 2)))
			if outcome == "unsupported" {
				if len(es) != 0 {
					t.Fatal("unsupported retried same incarnation")
				}
				es = noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "hub-two", Ready: true}))
			}
			if len(es) != 1 || es[0].Token == p.Token {
				t.Fatal("new evidence did not produce new admission")
			}
		})
	}
}
func TestLifecycleNoticeUncertaintyNeverReplays(t *testing.T) {
	for _, eventKind := range []LifecycleNoticeEventKind{LifecycleNoticeAdmission, LifecycleNoticeReceipt} {
		var s LifecycleNoticeState
		noticeStep(&s, noticeFact("slot", 1))
		p := noticePublications(noticeReady(&s))[0]
		noticeStep(&s, LifecycleNoticeEvent{Kind: eventKind, Hub: "/hub", Token: p.Token, Outcome: "indeterminate"})
		noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Ready: false, Incarnation: "hub-one"})
		if len(noticePublications(noticeReady(&s))) != 0 {
			t.Fatal("uncertain revision replayed")
		}
		if len(noticePublications(noticeStep(&s, noticeFact("slot", 1)))) != 0 {
			t.Fatal("duplicate replayed")
		}
		if len(noticePublications(noticeStep(&s, noticeFact("slot", 2)))) != 1 {
			t.Fatal("new revision suppressed")
		}
	}
}
func TestLifecycleNoticeBoundedMultipartAndInputSanitizing(t *testing.T) {
	var s LifecycleNoticeState
	for i := 0; i < 200; i++ {
		e := noticeFact(fmt.Sprint(i), 1)
		e.Fact.Model = strings.Repeat("é\n\x1b", 500)
		noticeStep(&s, e)
	}
	if len(s.facts) != 128 || s.overflow == 0 {
		t.Fatalf("unbounded facts: %d", len(s.facts))
	}
	ps := noticePublications(noticeReady(&s))
	var bodies []string
	for len(ps) > 0 {
		p := ps[0]
		if len(p.Body) > 8192 {
			t.Fatalf("oversized: %d", len(p.Body))
		}
		for _, r := range p.Body {
			if unicode.IsControl(r) && r != '\n' {
				t.Fatalf("control: %q", r)
			}
		}
		bodies = append(bodies, p.Body)
		ps = noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: p.Token, Outcome: "submitted"}))
	}
	joined := strings.Join(bodies, "")
	if len(bodies) < 2 || !strings.Contains(joined, "resnapshot") || strings.Count(joined, "started") != 128 {
		t.Fatalf("bad multipart: %d parts", len(bodies))
	}
}
func TestLifecycleNoticeGeneratedInvariants(t *testing.T) {
	r := rand.New(rand.NewSource(574))
	var s LifecycleNoticeState
	noticeReady(&s)
	for i := 0; i < 4000; i++ {
		e := noticeFact(fmt.Sprint(r.Intn(160)), uint64(r.Intn(100)+1))
		if r.Intn(3) == 0 {
			for hub, h := range s.hubs {
				if h.token != 0 {
					e = LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: hub, Token: h.token, Outcome: []string{"submitted", "expired", "cancelled", "indeterminate"}[r.Intn(4)]}
					break
				}
			}
		}
		old := s
		es := noticeStep(&s, e)
		if len(s.facts) > 128 || len(s.hubs) > 128 {
			t.Fatal("size bound")
		}
		for _, p := range noticePublications(es) {
			if len(p.Body) > 8192 || p.Token == 0 {
				t.Fatal("invalid publication")
			}
		}
		if reflect.DeepEqual(old, s) {
			continue
		}
	}
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeShutdown})
	if len(s.facts) != 0 || len(s.hubs) != 0 {
		t.Fatal("shutdown retains state")
	}
	if len(noticePublications(noticeStep(&s, noticeFact("later", 999)))) != 0 {
		t.Fatal("shutdown revived")
	}
}

func TestLifecycleNoticeReplacementWaitsForOldReceipt(t *testing.T) {
	var s LifecycleNoticeState
	noticeStep(&s, noticeFact("slot", 1))
	old := noticePublications(noticeReady(&s))[0]
	es := noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "hub-two", Ready: true})
	if len(es) != 1 || es[0].Kind != LifecycleNoticeCancel || es[0].Token != old.Token {
		t.Fatalf("replacement: %+v", es)
	}
	e := noticeFact("slot", 2)
	e.Fact.Kind = LifecycleParked
	noticeStep(&s, e)
	if got := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: old.Token + 100, Outcome: "submitted"})); len(got) != 0 {
		t.Fatal("foreign token released job")
	}
	next := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: old.Token, Outcome: "cancelled"}))
	if len(next) != 1 || next[0].Incarnation != "hub-two" || !strings.Contains(next[0].Body, "parked") {
		t.Fatalf("replacement lost current: %+v", next)
	}
}
func TestLifecycleNoticeMultipartFailureNeverResendsSubmittedRows(t *testing.T) {
	var s LifecycleNoticeState
	for i := 0; i < 128; i++ {
		e := noticeFact(fmt.Sprintf("slot%03d", i), 1)
		e.Fact.Branch = strings.Repeat("b", 128)
		noticeStep(&s, e)
	}
	first := noticePublications(noticeReady(&s))[0]
	second := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: first.Token, Outcome: "submitted"}))[0]
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: second.Token, Outcome: "expired"})
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "hub-one", Ready: false})
	retry := noticePublications(noticeReady(&s))[0]
	for _, line := range strings.Split(first.Body, "\n")[1:] {
		if line != "" && strings.Contains(retry.Body, line) {
			t.Fatalf("submitted row replayed: %s", line)
		}
	}
	if retry.Token == second.Token {
		t.Fatal("reused failed token")
	}
}
func TestLifecycleNoticeCoalescingCountsOnlyUnretiredChanges(t *testing.T) {
	var s LifecycleNoticeState
	for i := uint64(1); i <= 3; i++ {
		noticeStep(&s, noticeFact("slot", i))
	}
	first := noticePublications(noticeReady(&s))[0]
	if !strings.Contains(first.Body, "3 changes") {
		t.Fatal(first.Body)
	}
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: first.Token, Outcome: "submitted"})
	next := noticePublications(noticeStep(&s, noticeFact("slot", 4)))[0]
	if strings.Contains(next.Body, "changes") {
		t.Fatalf("retired history counted again: %s", next.Body)
	}
}
func TestLifecycleNoticeTransitionDoesNotMutateInput(t *testing.T) {
	var s LifecycleNoticeState
	noticeStep(&s, noticeFact("slot", 1))
	noticeReady(&s)
	snapshot := cloneLifecycleNotices(s)
	AdvanceLifecycleNotices(s, noticeFact("slot", 2))
	if !reflect.DeepEqual(s, snapshot) {
		t.Fatal("input state mutated")
	}
}

func TestLifecycleNoticeFreshAddressReplacesSlotCurrentState(t *testing.T) {
	var s LifecycleNoticeState
	old := noticeFact("slot", 1)
	noticeStep(&s, old)
	fresh := noticeFact("slot", 2)
	fresh.Fact.Address.Tag = "fresh-address"
	fresh.Fact.Incarnation = "fresh-incarnation"
	fresh.Fact.Kind = LifecycleRecovered
	noticeStep(&s, fresh)
	noticeStep(&s, old)
	parts := noticePublications(noticeReady(&s))
	if len(s.facts) != 1 || len(parts) != 1 || strings.Contains(parts[0].Body, ": started") || !strings.Contains(parts[0].Body, ": recovered") {
		t.Fatalf("old address survived replacement: facts=%d parts=%+v", len(s.facts), parts)
	}
}

func TestLifecycleNoticeDiscardBeforeDrain(t *testing.T) {
	var s LifecycleNoticeState
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeArm})
	noticeReady(&s)
	noticeStep(&s, noticeFact("old-slot", 1))
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/new", Incarnation: "new-one", Ready: true})
	before := s
	if ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/new"}})); len(ps) != 0 {
		t.Fatal("discard bypassed restart barrier")
	}
	if len(s.facts) != 0 || len(before.facts) != 1 {
		t.Fatal("discard lost immutability or retained old facts")
	}
	ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDrain}))
	if len(ps) != 1 || ps[0].Hub != "/new" || ps[0].Body != "Couch hub changed; earlier lifecycle notices were dropped; check Slots" {
		t.Fatalf("move summary: %+v", ps)
	}
}
func TestLifecycleNoticeDiscardFrozenAndSubmitted(t *testing.T) {
	for _, submitted := range []bool{false, true} {
		t.Run(fmt.Sprint(submitted), func(t *testing.T) {
			var s LifecycleNoticeState
			noticeStep(&s, noticeFact("old-slot", 1))
			old := noticePublications(noticeReady(&s))[0]
			if submitted {
				noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: old.Token, Outcome: "submitted"})
			}
			noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/new", Incarnation: "new-one", Ready: true})
			before := s
			ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/new"}}))
			if len(ps) != 1 || ps[0].Token <= old.Token || strings.Contains(ps[0].Body, "old-slot") {
				t.Fatalf("old payload replayed: %+v", ps)
			}
			if !submitted && before.hubs["/hub"].token != old.Token {
				t.Fatal("mutated previous state")
			}
			if got := noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: old.Token, Outcome: "submitted"}); len(got) != 0 {
				t.Fatalf("stale completion changed projection: %+v", got)
			}
			if got := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/new", Token: ps[0].Token, Outcome: "submitted"})); len(got) != 0 {
				t.Fatal("old facts replayed after info receipt")
			}
			e := noticeFact("new-slot", 2)
			e.Fact.Hub = "/new"
			if got := noticePublications(noticeStep(&s, e)); len(got) != 1 || !strings.Contains(got[0].Body, "new-slot") {
				t.Fatalf("new fact suppressed: %+v", got)
			}
		})
	}
}
func TestLifecycleNoticeDiscardBoundsAndReadiness(t *testing.T) {
	var s LifecycleNoticeState
	noticeReady(&s)
	for i := 0; i < 200; i++ {
		ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/hub", "/hub"}}))
		if len(ps) != 1 || len(s.hubs) != 1 || len(s.hubs["/hub"].parts) != 1 || len(s.facts) != 0 {
			t.Fatalf("discard not bounded: %+v", s)
		}
	}
	before := s
	for _, roots := range [][]string{make([]string, 129), {""}, {strings.Repeat("x", 4097)}} {
		noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: roots})
		if !reflect.DeepEqual(s, before) {
			t.Fatal("invalid roots mutated state")
		}
	}
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/unknown"}})
	if s.hubs["/unknown"].token != 0 {
		t.Fatal("unknown hub published before ready")
	}
	if ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/unknown", Incarnation: "ready", Ready: true})); len(ps) != 1 {
		t.Fatal("info did not wait for readiness")
	}
}

func TestLifecycleNoticeDiscardKnownHubBoundAndIncarnation(t *testing.T) {
	var s LifecycleNoticeState
	for i := 0; i < 128; i++ {
		noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: fmt.Sprint("/", i), Incarnation: "old"})
	}
	h := s.hubs["/0"]
	h.unsupported = true
	s.hubs["/0"] = h
	before := s
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/new"}})
	if !reflect.DeepEqual(s, before) {
		t.Fatal("overflowing known hubs mutated state")
	}
	noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/0"}})
	if !s.hubs["/0"].unsupported || s.hubs["/0"].incarnation != "old" || len(s.hubs) != 128 {
		t.Fatal("discard erased recipient metadata")
	}
	if ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/0", Incarnation: "old", Ready: true})); len(ps) != 0 {
		t.Fatal("unsupported incarnation retried")
	}
	if ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/0", Incarnation: "new", Ready: true})); len(ps) != 1 {
		t.Fatal("new incarnation did not receive queued info")
	}
}

func TestLifecycleNoticeDiscardInfoKnownNoWriteSurvivesReadiness(t *testing.T) {
	for _, outcome := range []string{"unavailable", "indeterminate"} {
		t.Run(outcome, func(t *testing.T) {
			var s LifecycleNoticeState
			noticeReady(&s)
			ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/hub"}}))
			if len(ps) != 1 {
				t.Fatal("missing initial info")
			}
			old := ps[0]
			// Readiness changes while the admission's final result is outstanding.
			if got := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "replacement", Ready: true})); len(got) != 0 {
				t.Fatal("replacement published before final admission outcome")
			}
			got := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeAdmission, Hub: "/hub", Token: old.Token, Outcome: outcome}))
			if outcome == "indeterminate" {
				if len(got) != 0 {
					t.Fatal("uncertain info replayed")
				}
				if next := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "third", Ready: true})); len(next) != 0 {
					t.Fatal("uncertain info replayed after readiness changed")
				}
				return
			}
			if len(got) != 1 || got[0].Body != old.Body || got[0].Token <= old.Token {
				t.Fatalf("known-no-write info lost: %+v", got)
			}
			if next := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: "/hub", Token: got[0].Token, Outcome: "submitted"})); len(next) != 0 {
				t.Fatal("submitted info replayed")
			}
		})
	}
}
func TestLifecycleNoticeDiscardInfoNoWriteWaitsForReadinessChange(t *testing.T) {
	var s LifecycleNoticeState
	noticeReady(&s)
	old := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeDiscard, Hubs: []string{"/hub"}}))[0]
	if ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeAdmission, Hub: "/hub", Token: old.Token, Outcome: "unavailable"})); len(ps) != 0 {
		t.Fatal("immediate retry")
	}
	if ps := noticePublications(noticeReady(&s)); len(ps) != 0 {
		t.Fatal("same readiness retried")
	}
	ps := noticePublications(noticeStep(&s, LifecycleNoticeEvent{Kind: LifecycleNoticeHubReady, Hub: "/hub", Incarnation: "replacement", Ready: true}))
	if len(ps) != 1 || ps[0].Body != old.Body {
		t.Fatalf("lost suppressed info: %+v", ps)
	}
}

// A native exit is one more confirmed kind (pair#657): it publishes as
// "exited (code N)" and coalesces per slot like every other kind.
func TestLifecycleNoticeExitedPublishesAndCoalesces(t *testing.T) {
	var s LifecycleNoticeState
	noticeReady(&s)
	exited := noticeFact("pair:2", 1)
	exited.Fact.Kind, exited.Fact.Mode, exited.Fact.Incarnation = LifecycleExited, "code 3", "7:child"
	es := noticePublications(noticeStep(&s, exited))
	if len(es) != 1 || !strings.Contains(es[0].Body, "pair:2: exited (code 3)") {
		t.Fatalf("exited notice: %+v", es)
	}
	invalid := exited
	invalid.Fact.Kind = "agent-gone"
	if validLifecycleNoticeFact(invalid.Fact) {
		t.Fatal("an unknown kind validated")
	}
}

func TestLifecycleNoticeSuppressionRetiresObservationWithoutPublication(t *testing.T) {
	var state LifecycleNoticeState
	fact := noticeFact("slot", 1)
	fact.Outcome = "suppressed"
	if len(noticePublications(noticeStep(&state, fact))) != 0 {
		t.Fatal("suppressed observation published")
	}
	if effects := noticeReady(&state); len(effects) != 0 {
		t.Fatalf("readiness replayed consumed fact: %+v", effects)
	}
	if current := state.facts["slot"]; current.retired != 1 || current.changes != 0 {
		t.Fatalf("suppression not retired: %+v", current)
	}
	if effects := noticeStep(&state, noticeFact("slot", 1)); len(effects) != 0 {
		t.Fatal("same observation replayed")
	}
	if effects := noticePublications(noticeStep(&state, noticeFact("slot", 2))); len(effects) != 1 || strings.Contains(effects[0].Body, "changes") {
		t.Fatalf("fresh fact lost or old changes replayed: %+v", effects)
	}
}

func TestLifecycleNoticeSuppressedAdmissionRetiresFrozenOnlySilently(t *testing.T) {
	var state LifecycleNoticeState
	noticeStep(&state, noticeFact("slot", 1))
	first := noticePublications(noticeReady(&state))[0]
	noticeStep(&state, noticeFact("slot", 2))
	effects := noticeStep(&state, LifecycleNoticeEvent{Kind: LifecycleNoticeAdmission, Hub: first.Hub, Token: first.Token, Outcome: "suppressed"})
	for _, effect := range effects {
		if effect.Kind == LifecycleNoticeDiagnostic {
			t.Fatal("intentional suppression reported a failure")
		}
	}
	next := noticePublications(effects)
	if len(next) != 1 || state.facts["slot"].retired != 1 || strings.Contains(next[0].Body, "changes") {
		t.Fatalf("suppression did not preserve newer fact: %+v", effects)
	}
	// The late canonical receipt belongs to the retired token and cannot consume
	// the newer revision or make a second publication.
	if effects := noticeStep(&state, LifecycleNoticeEvent{Kind: LifecycleNoticeReceipt, Hub: first.Hub, Token: first.Token, Outcome: "submitted"}); len(effects) != 0 {
		t.Fatal("late suppressed receipt changed newer attempt")
	}
}

func TestLifecycleNoticeSuppressHubRetiresPendingPartsAndCancelsToken(t *testing.T) {
	var state LifecycleNoticeState
	noticeStep(&state, noticeFact("slot", 1))
	first := noticePublications(noticeReady(&state))[0]
	effects := noticeStep(&state, LifecycleNoticeEvent{Kind: LifecycleNoticeSuppressHub, Hub: "/hub"})
	if len(effects) != 1 || effects[0].Kind != LifecycleNoticeCancel || effects[0].Token != first.Token {
		t.Fatalf("old attempt not cancelled: %+v", effects)
	}
	if len(noticeReady(&state)) != 0 || state.facts["slot"].retired != 1 || len(state.hubs["/hub"].parts) != 0 {
		t.Fatal("suppression retained backlog")
	}
	if effects := noticePublications(noticeStep(&state, noticeFact("slot", 2))); len(effects) != 1 {
		t.Fatal("future fact could not publish")
	}
}
