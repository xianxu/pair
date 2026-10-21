package couchschedule

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestManualIntentDigestSurvivesOccurrenceTransitions(t *testing.T) {
	s, at := taskFixture(t, true)
	digest := (ManualIntent{ID: "manual-id", Hub: s.Key.Hub, Task: s.Key.Task, Args: []string{"a"}}).Digest()
	reserved, _ := Advance(s, Event{Kind: ManualRequested, At: at, RunID: "manual-id", ManualIntentDigest: digest})
	if reserved.Active == nil || reserved.Active.ManualIntentDigest != digest {
		t.Fatal("manual reservation lost intent identity")
	}
	running, _ := Advance(reserved, Event{Kind: ReservationDurable, At: at, Attempt: "manual-id", Revision: s.Revision})
	completed, _ := Advance(running, Event{Kind: TaskFinished, At: at.Add(time.Second), Attempt: "manual-id", Revision: s.Revision, Exit: 0})
	if completed.Last == nil || completed.Last.ManualIntentDigest != digest {
		t.Fatal("manual completion lost intent identity")
	}
	raw, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{s.Key.ID(): completed}})
	if err != nil {
		t.Fatal(err)
	}
	round, err := DecodeLedger(raw)
	if err != nil || round.Tasks[s.Key.ID()].Last.ManualIntentDigest != digest {
		t.Fatalf("roundtrip: %v", err)
	}
	scheduled, _ := Advance(s, Event{Kind: ClockObserved, At: at, RunID: "scheduled-id", ManualIntentDigest: digest})
	if scheduled.Active == nil || scheduled.Active.ManualIntentDigest != "" {
		t.Fatal("scheduled occurrence acquired manual identity")
	}
}

func TestManualIntentDigestLedgerCompatibilityAndStrictness(t *testing.T) {
	old := phaseStateFixture(t, "manual", "running")
	legacy, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{old.Key.ID(): old}})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy, []byte("manual_intent_digest")) {
		t.Fatal("legacy empty digest should be omitted")
	}
	if _, err = DecodeLedger(legacy); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"manual_intent_digest":12,`, `"manual_intent_digest":null,`, `"manual_intent_digest":"a","manual_intent_digest":"b",`} {
		changed := bytes.Replace(legacy, []byte(`"active":{`), []byte(`"active":{`+field), 1)
		if _, err = DecodeLedger(changed); err == nil {
			t.Fatalf("accepted malformed digest: %s", field)
		}
	}
	old.Active.ManualIntentDigest = strings.Repeat("a", 64)
	old.Active.Unknown = map[string]json.RawMessage{"other": json.RawMessage(`{ "x": 2 }`)}
	raw, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{old.Key.ID(): old}})
	if err != nil {
		t.Fatal(err)
	}
	round, err := DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := round.Tasks[old.Key.ID()].Active; got.ManualIntentDigest != old.Active.ManualIntentDigest || string(got.Unknown["other"]) != `{ "x": 2 }` {
		t.Fatalf("digest/additive members lost: %+v", got)
	}
}

func TestManualIntentDigestRecoveryMatchesRetainedIdentity(t *testing.T) {
	old := phaseStateFixture(t, "manual", "running")
	old.Active.ManualIntentDigest = strings.Repeat("a", 64)
	canonical := summary(old.Active, old.Active.StartedAt.Add(time.Second), nil, "quiet", "manual")
	got := PlanSettlement(old, canonical, Event{At: canonical.EndedAt})
	if got.State.Active != nil || got.State.Last.ManualIntentDigest != old.Active.ManualIntentDigest {
		t.Fatalf("canonical completion lost intent: %+v", got)
	}
	for _, other := range []string{"", strings.Repeat("b", 64)} {
		invalid := *canonical
		invalid.ManualIntentDigest = other
		refused := PlanSettlement(old, &invalid, Event{At: canonical.EndedAt})
		if refused.Kind != SettlementQuarantined || refused.State.Active == nil || refused.State.Active.ManualIntentDigest != old.Active.ManualIntentDigest {
			t.Fatalf("changed/missing digest adopted: %+v", refused)
		}
	}
	raw, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{old.Key.ID(): got.State}})
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{`"manual_intent_digest":null`, `"manual_intent_digest":19`, `"manual_intent_digest":"a","manual_intent_digest":"b"`} {
		bad := bytes.Replace(raw, []byte(`"manual_intent_digest":"`+canonical.ManualIntentDigest+`"`), []byte(replacement), 1)
		if _, err = DecodeLedger(bad); err == nil {
			t.Fatalf("malformed summary digest accepted: %s", replacement)
		}
	}
	legacy := bytes.Replace(raw, []byte(`"manual_intent_digest":"`+canonical.ManualIntentDigest+`",`), nil, 1)
	if _, err = DecodeLedger(legacy); err != nil {
		t.Fatalf("legacy summary absent digest rejected: %v", err)
	}
}
