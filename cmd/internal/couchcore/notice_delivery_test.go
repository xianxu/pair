package couchcore

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestNoticeDeliveryBoundsGenerated(t *testing.T) {
	for _, n := range []int{512, 1024} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var state NoticeDeliveryState
			now := time.Unix(100, 0)
			ids := map[string]string{}
			admitted := map[uint64]bool{}
			rng := rand.New(rand.NewSource(844))
			step := func(e NoticeDeliveryEvent) {
				t.Helper()
				before := state
				var effects []NoticeDeliveryEffect
				state, effects = AdvanceNoticeDelivery(state, e)
				if len(state.Entries) > 128 {
					t.Fatalf("unbounded entries: %d", len(state.Entries))
				}
				inflight := 0
				for _, x := range state.Entries {
					if len(x.Occurrence.Body) > 4096 {
						t.Fatal("unbounded body")
					}
					if x.Phase == NoticeDeliverySending {
						inflight++
					}
				}
				if inflight > 4 {
					t.Fatalf("inflight=%d", inflight)
				}
				for _, effect := range effects {
					if effect.Kind == NoticeDeliveryAdmit {
						if admitted[effect.Token] {
							t.Fatal("resent attempt")
						}
						admitted[effect.Token] = true
						if old, ok := ids[effect.Occurrence.ID]; ok && old != effect.AdmissionID {
							t.Fatal("multiple canonical IDs")
						}
						ids[effect.Occurrence.ID] = effect.AdmissionID
						if effect.Owner.State != NoticeOwnerEnabled {
							t.Fatal("admission without enabled owner")
						}
					}
				}
				if len(before.Entries) > 0 && len(before.Entries[0].Occurrence.Body) > 4096 {
					t.Fatal("mutated input")
				}
			}
			for i := 0; i < n; i++ {
				step(NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: fmt.Sprint(i), Class: NoticeChildExited, Body: strings.Repeat("x", 5000)}, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled, Binding: "hub", Epoch: 1}})
				if i%3 == 0 && len(state.Entries) > 0 {
					x := state.Entries[rng.Intn(len(state.Entries))]
					if x.Phase == NoticeDeliverySending {
						step(NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: x.Occurrence.ID, Token: x.Token, Result: NoticeDeliveryUncertain})
						step(NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: x.Occurrence.ID, Token: x.Token, Result: NoticeDeliveryRefused})
					}
				}
			}
			if state.OverflowCount == 0 {
				t.Fatal("expected bounded overflow")
			}
			step(NoticeDeliveryEvent{Kind: NoticeDeliveryTick, Now: now.Add(2 * time.Minute)})
			if len(state.Entries) != 0 {
				t.Fatal("expiry retained entries")
			}
			step(NoticeDeliveryEvent{Kind: NoticeDeliveryShutdown, Now: now})
			if !state.Closed {
				t.Fatal("shutdown")
			}
			step(NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: "after", Class: NoticeChildExited}, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled}})
			if len(state.Entries) != 0 {
				t.Fatal("post-shutdown work")
			}
		})
	}
}

func TestNoticeDeliveryOwnershipAndClear(t *testing.T) {
	now := time.Unix(100, 0)
	add := NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: "question", Condition: "thread-question", Class: NoticeJudgedQuestion}, Owner: NoticeDeliveryOwner{State: NoticeOwnerBooting}}
	state, effects := AdvanceNoticeDelivery(NoticeDeliveryState{}, add)
	if len(effects) != 0 || len(state.Entries) != 1 {
		t.Fatal("boot lost occurrence")
	}
	original := state
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOwnerChanged, Now: now, Owner: NoticeDeliveryOwner{State: NoticeOwnerDisabled}})
	if len(effects) != 0 || len(state.Entries) != 0 {
		t.Fatal("disabled must retire")
	}
	if len(original.Entries) != 1 {
		t.Fatal("mutated input")
	}
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOwnerChanged, Now: now, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled}})
	if len(effects) != 0 {
		t.Fatal("off/on replay")
	}
	state, _ = AdvanceNoticeDelivery(NoticeDeliveryState{}, add)
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOwnerChanged, Now: now, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled, Binding: "hub", Epoch: 3}})
	if len(effects) != 1 || effects[0].Kind != NoticeDeliveryAdmit {
		t.Fatalf("missing admission %+v", effects)
	}
	token := effects[0].Token
	state, _ = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryConditionCleared, Now: now, Occurrence: NoticeDeliveryOccurrence{Condition: "thread-question"}})
	if len(state.Entries) != 0 {
		t.Fatal("clear retained")
	}
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "question", Token: token, Result: NoticeDeliveryUncertain})
	if len(state.Entries) != 0 || len(effects) != 0 {
		t.Fatal("stale completion revived")
	}
}

func TestNoticeDeliveryDisposition(t *testing.T) {
	now := time.Unix(100, 0)
	for _, tc := range []struct {
		name    string
		class   NoticeClass
		origin  NoticeOrigin
		owner   NoticeOwnerState
		handled bool
		effect  NoticeDeliveryEffectKind
		count   int
	}{
		{"operator bypasses startup", NoticeCastStartFailed, NoticeOperatorOrigin, NoticeOwnerBooting, false, NoticeDeliveryBell, 1},
		{"routine never falls back", NoticeRoutine, NoticeSystemOrigin, NoticeOwnerUnavailable, false, NoticeDeliveryBell, 0},
		{"existing send suppresses admission", NoticeLivePicker, NoticeSystemOrigin, NoticeOwnerEnabled, true, NoticeDeliveryAdmit, 0},
		{"existing send allows unavailable fallback", NoticeLivePicker, NoticeSystemOrigin, NoticeOwnerUnavailable, true, NoticeDeliveryBell, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, e := AdvanceNoticeDelivery(NoticeDeliveryState{}, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: "event", Class: tc.class, Origin: tc.origin, HubHandled: tc.handled}, Owner: NoticeDeliveryOwner{State: tc.owner}})
			if len(s.Entries) != 0 || len(e) != tc.count {
				t.Fatalf("state=%+v effects=%+v", s, e)
			}
			if tc.count > 0 && e[0].Kind != tc.effect {
				t.Fatalf("effect %+v", e)
			}
		})
	}
	state, effects := AdvanceNoticeDelivery(NoticeDeliveryState{}, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{Class: NoticeHubRequested}})
	if len(state.Entries) != 0 || len(effects) != 0 {
		t.Fatal("blank ID accepted")
	}
}

func TestNoticeDeliveryDuplicateAndOwnershipFence(t *testing.T) {
	now := time.Unix(100, 0)
	oldOwner := NoticeDeliveryOwner{State: NoticeOwnerEnabled, Binding: "old", Epoch: 1}
	newOwner := NoticeDeliveryOwner{State: NoticeOwnerEnabled, Binding: "new", Epoch: 2}
	var state NoticeDeliveryState
	add := func(id string, owner NoticeDeliveryOwner) []NoticeDeliveryEffect {
		t.Helper()
		var e []NoticeDeliveryEffect
		state, e = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: id, Condition: id, Class: NoticeChildExited}, Owner: owner})
		return e
	}
	var first NoticeDeliveryEffect
	for i := 0; i < 5; i++ {
		effects := add(fmt.Sprint(i), oldOwner)
		if i == 0 {
			first = effects[0]
		}
	}
	if effects := add("0", newOwner); len(effects) != 0 {
		t.Fatal("duplicate admission")
	}
	before := state
	state, _ = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOwnerChanged, Now: now, ID: "4", Owner: newOwner})
	if before.Entries[4].Owner != oldOwner {
		t.Fatal("mutated caller state")
	}
	state, effects := AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "0", Token: first.Token + 99, Result: NoticeDeliveryRefused})
	if len(effects) != 0 || len(state.Entries) != 5 {
		t.Fatal("wrong token changed state")
	}
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "0", Token: first.Token, Result: NoticeDeliveryUncertain})
	if len(effects) != 1 || effects[0].Occurrence.ID != "4" || effects[0].Owner != newOwner {
		t.Fatalf("pending did not re-resolve ownership: %+v", effects)
	}
	for _, result := range []NoticeDeliveryResult{NoticeDeliveryRefused, NoticeDeliveryAdmitted, NoticeDeliveryUncertain} {
		state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "0", Token: first.Token, Result: result})
		if len(effects) != 0 || len(state.Entries) != 5 {
			t.Fatal("duplicate result changed uncertain occurrence")
		}
	}
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOwnerChanged, Now: now, ID: "0", Owner: newOwner})
	if len(effects) != 0 || state.Entries[0].Owner != oldOwner {
		t.Fatal("uncertainty resent to new owner")
	}
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryReceiptObserved, Now: now, ID: "0", Token: first.Token})
	if len(effects) != 0 || len(state.Entries) != 4 {
		t.Fatal("receipt did not retire")
	}
}

func TestNoticeDeliveryOverflowUsesOneAggregateID(t *testing.T) {
	var state NoticeDeliveryState
	now := time.Unix(100, 0)
	for i := 0; i < 130; i++ {
		var effects []NoticeDeliveryEffect
		state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: fmt.Sprint(i), Class: NoticeChildExited}})
		if i >= 128 {
			if len(effects) != 1 || effects[0].Kind != NoticeDeliveryOverflow || effects[0].Occurrence.ID != "notice-delivery-overflow" || effects[0].Count != uint64(i-127) {
				t.Fatalf("unbounded overflow projection: %+v", effects)
			}
		}
	}
}

func TestNoticeDeliveryBodyPreservesUTF8(t *testing.T) {
	body := strings.Repeat("x", 4095) + "🙂"
	state, _ := AdvanceNoticeDelivery(NoticeDeliveryState{}, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: time.Unix(100, 0), Occurrence: NoticeDeliveryOccurrence{ID: "bounded", Body: body, Class: NoticeChildExited}})
	if got := state.Entries[0].Occurrence.Body; got != strings.Repeat("x", 4095) {
		t.Fatalf("truncated a UTF-8 rune: len=%d suffix=%q", len(got), got[4095:])
	}
}

func TestNoticeDeliveryAdmittedTracksReceipt(t *testing.T) {
	now := time.Unix(100, 0)
	state, effects := AdvanceNoticeDelivery(NoticeDeliveryState{}, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: "admitted", Class: NoticeChildExited}, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled}})
	state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "admitted", Token: effects[0].Token, Result: NoticeDeliveryAdmitted})
	if len(effects) != 0 || len(state.Entries) != 1 || state.Entries[0].Phase != NoticeDeliveryAwaitingReceipt {
		t.Fatal("admitted occurrence lost receipt tracking")
	}
}

func TestNoticeDeliveryRefusedOwnerLossFallback(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result NoticeDeliveryResult
		owner  NoticeDeliveryOwner
		bell   bool
	}{
		{"confirmed-owner-loss", NoticeDeliveryRefused, NoticeDeliveryOwner{State: NoticeOwnerUnavailable, Epoch: 7}, true},
		{"connected-refusal", NoticeDeliveryRefused, NoticeDeliveryOwner{State: NoticeOwnerEnabled, Epoch: 7}, false},
		{"automation-revoked", NoticeDeliveryRefused, NoticeDeliveryOwner{State: NoticeOwnerUnavailable, Epoch: 9}, false},
		{"uncertain-owner-loss", NoticeDeliveryUncertain, NoticeDeliveryOwner{State: NoticeOwnerUnavailable, Epoch: 7}, false},
		{"admitted-owner-loss", NoticeDeliveryAdmitted, NoticeDeliveryOwner{State: NoticeOwnerUnavailable, Epoch: 7}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Unix(100, 0)
			state, effects := AdvanceNoticeDelivery(NoticeDeliveryState{}, NoticeDeliveryEvent{Kind: NoticeDeliveryOccurrenceAdded, Now: now, Occurrence: NoticeDeliveryOccurrence{ID: "owner-loss", Class: NoticeChildExited}, Owner: NoticeDeliveryOwner{State: NoticeOwnerEnabled, Epoch: 7}})
			token := effects[0].Token
			state, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "owner-loss", Token: token, Result: tc.result, Owner: tc.owner})
			want := 0
			if tc.bell {
				want = 1
			}
			if len(effects) != want || want > 0 && effects[0].Kind != NoticeDeliveryBell {
				t.Fatalf("effects=%+v want bell=%t", effects, tc.bell)
			}
			_, effects = AdvanceNoticeDelivery(state, NoticeDeliveryEvent{Kind: NoticeDeliverySendResult, Now: now, ID: "owner-loss", Token: token, Result: NoticeDeliveryRefused, Owner: NoticeDeliveryOwner{State: NoticeOwnerUnavailable, Epoch: 7}})
			if len(effects) != 0 {
				t.Fatal("duplicate/uncertain completion generated fallback")
			}
		})
	}
}
