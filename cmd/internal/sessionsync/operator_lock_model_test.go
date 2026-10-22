package sessionsync

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var modelTurn = operatorTurn{ID: "0123456789abcdef0123456789abcdef", Token: "fedcba9876543210fedcba9876543210"}

func TestOperatorLockModelProof(t *testing.T) {
	raw := operatorLockText(modelTurn, time.Unix(123, 456))
	if got, ok := parseOperatorProof(raw); !ok || got != modelTurn {
		t.Fatalf("proof did not recover exact turn: %+v %v", got, ok)
	}
	for _, suffix := range []string{"future: retained\n", "nested: opaque future value\n"} {
		got, ok := parseOperatorProof(raw + suffix)
		if !ok || got != modelTurn {
			t.Fatalf("unknown line changed ownership: %+v %v", got, ok)
		}
	}
	for _, bad := range []string{raw + "turn: " + modelTurn.Token + "\n", raw + "holder: agent\n", raw + "document: invalid\n", strings.Replace(raw, "holder: operator", "holder: unknown", 1), strings.Replace(raw, "time:", "unknown:", 1), raw + "\x00", strings.Repeat("x", 4097)} {
		if _, ok := parseOperatorProof(bad); ok {
			t.Fatalf("ambiguous or invalid proof accepted: %q", bad)
		}
	}
	for _, bad := range []operatorTurn{{ID: modelTurn.ID}, {Token: modelTurn.Token}, {ID: strings.ToUpper(modelTurn.ID), Token: modelTurn.Token}, {ID: modelTurn.ID, Token: "../"}} {
		if err := validateOperatorTurn(bad); err == nil {
			t.Fatalf("invalid turn accepted: %+v", bad)
		}
	}
}

func TestOperatorLockModelDecisions(t *testing.T) {
	raw := operatorLockText(modelTurn, time.Unix(123, 0)) + "future: preserved\n"
	own := operatorLockObservation{Path: "one.lock", Text: raw, Exists: true}
	foreign := operatorLockObservation{Path: "foreign.lock", Text: "holder: operator\n", Exists: true}
	for _, tc := range []struct {
		name    string
		action  operatorLockAction
		locks   []operatorLockObservation
		state   OperatorLockState
		refused bool
		remove  []string
	}{
		{name: "new", state: OperatorAcquired},
		{name: "same", locks: []operatorLockObservation{own}, state: OperatorAlreadyHeld},
		{name: "agent is free", locks: []operatorLockObservation{{Path: "legacy", Text: "holder: agent\n", Exists: true}}, state: OperatorAcquired},
		{name: "foreign anywhere", locks: []operatorLockObservation{own, foreign}, refused: true},
		{name: "malformed agent ambiguity", locks: []operatorLockObservation{{Path: "legacy", Text: "holder: agent\nholder: operator\n", Exists: true}}, refused: true},
		{name: "release", action: operatorRelease, locks: []operatorLockObservation{own}, state: OperatorReleased, remove: []string{"one.lock"}},
		{name: "release absent", action: operatorRelease, state: OperatorAlreadyReleased},
		{name: "release foreign", action: operatorRelease, locks: []operatorLockObservation{own, foreign}, refused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decideOperatorLock(modelTurn, tc.action, "one.lock", tc.locks)
			if tc.refused {
				wantRefusal(t, err, 3, "operator-lock")
				if !reflect.DeepEqual(got, operatorLockDecision{}) {
					t.Fatalf("refusal scheduled effects: %+v", got)
				}
				return
			}
			if err != nil || got.State != tc.state || !reflect.DeepEqual(got.Remove, tc.remove) {
				t.Fatalf("decision %+v %v", got, err)
			}
			if got.State == OperatorAlreadyHeld && (got.LockText != raw || got.LockPath != own.Path) {
				t.Fatalf("retry replaced exact proof: %+v", got)
			}
		})
	}
	newTurn := modelTurn
	newTurn.Token = strings.Repeat("a", 32)
	_, err := decideOperatorLock(newTurn, operatorRelease, "one.lock", []operatorLockObservation{own})
	wantRefusal(t, err, 3, "operator-lock")
	alias := own
	alias.Path = "alias.lock"
	canonical := own
	canonical.Text += "canonical: preserve unknown fields\n"
	got, err := decideOperatorLock(modelTurn, operatorAcquire, "one.lock", []operatorLockObservation{alias, canonical})
	if err != nil || got.LockPath != canonical.Path || got.LockText != canonical.Text {
		t.Fatalf("canonical proof did not win: %+v %v", got, err)
	}
}

func FuzzOperatorLockModelProof(f *testing.F) {
	f.Add(operatorLockText(modelTurn, time.Unix(123, 0)))
	f.Add("holder: operator\nturn: x\n")
	f.Fuzz(func(t *testing.T, text string) {
		turn, ok := parseOperatorProof(text)
		if ok && (validateOperatorTurn(turn) != nil || ParseHolder(text, true) != HolderOperator) {
			t.Fatalf("parser granted invalid ownership: %+v", turn)
		}
	})
}
