package couchcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLifecycleFailurePreservesUnknownFieldsThroughRewrite(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	raw := []byte(`[{"address":{"repo_scope":"repo","tag":"1-project","future_address": { "n" : 1234567890123456789 }},"nonce":"1","failed_at":"1970-01-01T00:00:01Z","future": { "escaped" : "\u0061", "n":1234567890123456789 }}]`)
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.Path(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(t.Context(), failureFixture(2)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`{ "escaped" : "\u0061", "n":1234567890123456789 }`, `{ "n" : 1234567890123456789 }`} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("unknown raw value changed: %s", got)
		}
	}
}

func TestLifecycleFailureSupersessionFitsTheSameByteBudget(t *testing.T) {
	history := LifecycleFailureHistory{failureFixture(2), failureFixture(1)}
	for i := range history {
		history[i].StartingPath = "/project"
		history[i].Unknown = map[string]json.RawMessage{"padding": json.RawMessage(`""`)}
	}
	raw, err := encodeLifecycleFailureHistory(history)
	if err != nil {
		t.Fatal(err)
	}
	available := launchFailureEncodedLimit - len(raw) - 1
	for i, size := range []int{available / 2, available - available/2} {
		history[i].Unknown["padding"], _ = json.Marshal(strings.Repeat("x", size))
	}
	for name, store := range map[string]LifecycleFailureStorage{"real": NewLifecycleFailureStore(failureNamespace(t)), "fake": NewFakeLifecycleFailureStore()} {
		t.Run(name, func(t *testing.T) {
			for _, failure := range history {
				if err := store.Save(t.Context(), failure); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Supersede(t.Context(), LifecycleFailureTarget{StartingPath: "/project"}, "new-epoch", 1, time.Unix(100, 0)); err != nil {
				t.Fatalf("success annotation refused instead of evicting oldest whole evidence: %v", err)
			}
			got, err := store.Read()
			if err != nil || len(got) != 1 || got[0].Nonce != "2" || got[0].SupersededAt == nil || !bytes.Equal(got[0].Unknown["padding"], history[0].Unknown["padding"]) {
				t.Fatalf("supersession lost newest or opaque evidence: count=%d err=%v", len(got), err)
			}
		})
	}
}
func TestLifecycleFailureSixtyFifthMaximumEntryRetained(t *testing.T) {
	var h LaunchFailureHistory
	for n := 1; n <= 65; n++ {
		f := failureFixture(n)
		f.Output = bytes.Repeat([]byte("x"), LaunchFailureOutputLimit)
		f.LaunchError = strings.Repeat("e", 4096)
		var err error
		h, err = h.Upsert(f)
		if err != nil {
			t.Fatalf("attempt %d: %v", n, err)
		}
	}
	raw, err := encodeLaunchFailureHistory(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 1<<20 || len(h) <= 16 || len(h) > 64 || h[0].Nonce != "65" || h[len(h)-1].Nonce == "1" {
		t.Fatalf("bad retention %d bytes, %d entries", len(raw), len(h))
	}
}
func TestLifecycleFailureKnownTypesAndDuplicatesRemainStrict(t *testing.T) {
	for _, body := range []string{`"nonce":3`, `"nonce":"a","nonce":"b"`, `"nonce":null`, `"nonce":"1","address":{"repo_scope":"repo","tag":"x","future":{"a":1,"a":2}}`} {
		s := NewLaunchFailureStore(failureNamespace(t))
		_ = os.MkdirAll(filepath.Dir(s.Path()), 0700)
		raw := `[{"address":{"repo_scope":"repo","tag":"x"},"failed_at":"1970-01-01T00:00:01Z",` + body + `}]`
		_ = os.WriteFile(s.Path(), []byte(raw), 0600)
		if _, err := s.Read(); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	_ = json.Valid
}

func TestLifecycleFailureSupersessionRetainsNestedUnknowns(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	_ = os.MkdirAll(filepath.Dir(s.Path()), 0700)
	raw := `[{"address":{"repo_scope":"repo","tag":"x","future_address": { "a": 1 }},"nonce":"old","starting_path":"/physical/slot","failed_at":"1970-01-01T00:00:01Z","holders":[{"process":{"PID":4,"Identity":"birth","future_process": { "p": 1 }},"kind":"codex","future_holder": { "h": 1 }}],"future": { "v": 1 }}]`
	if err := os.WriteFile(s.Path(), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Supersede(t.Context(), LifecycleFailureTarget{StartingPath: "/physical/slot"}, "epoch", 2, time.Unix(5, 0)); err != nil {
		t.Fatal(err)
	}
	h, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || h[0].SupersededAt == nil {
		t.Fatal("not superseded")
	}
	late := h[0]
	late.Unknown = nil
	late.addressUnknown = nil
	late.SupersededAt = nil
	late.Holders[0].Unknown = nil
	late.Holders[0].processUnknown = nil
	if err := s.Save(t.Context(), late); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{`{ "a": 1 }`, `{ "p": 1 }`, `{ "h": 1 }`, `{ "v": 1 }`} {
		if !bytes.Contains(got, []byte(v)) {
			t.Fatalf("lost opaque %s in %s", v, got)
		}
	}
	h, err = s.Read()
	if err != nil || h[0].SupersededAt == nil {
		t.Fatalf("late enrichment resurrected: %v", err)
	}
}
func TestLifecycleFailureSlotKeyAndSuccessOrder(t *testing.T) {
	var h LifecycleFailureHistory
	for n := 1; n <= 3; n++ {
		f := failureFixture(n)
		f.StartingPath = "/physical/slot"
		f.AttemptEpoch = "epoch"
		f.AttemptOrder = uint64(n)
		var e error
		h, e = h.Upsert(f)
		if e != nil {
			t.Fatal(e)
		}
	}
	h, changed := h.Supersede(LifecycleFailureTarget{StartingPath: "/physical/slot", Slot: "pair:0"}, "epoch", 2, time.Unix(5, 0))
	if !changed || h[0].SupersededAt != nil || h[1].SupersededAt == nil || h[2].SupersededAt == nil {
		t.Fatal("wrong admission order")
	}
	if key := LifecycleFailureSlotKey(LifecycleFailureTarget{StartingPath: "relative"}); key != "" {
		t.Fatalf("unverified key %q", key)
	}
}
func TestLifecycleFailureOversizedOpaqueEntryRefusesAtomically(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	if err := s.Save(t.Context(), failureFixture(1)); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.Path())
	f := failureFixture(2)
	f.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`"` + strings.Repeat("x", 1<<20) + `"`)}
	if err := s.Save(t.Context(), f); err == nil {
		t.Fatal("oversize single entry accepted")
	}
	after, _ := os.ReadFile(s.Path())
	if !bytes.Equal(before, after) {
		t.Fatal("failed save changed evidence")
	}
}
func TestLifecycleFailureNormalizesHolderWithoutMutatingCaller(t *testing.T) {
	f := failureFixture(1)
	f.Holders = []ProcessIdentityLine{{Process: ProcessIdentity{PID: 4, Identity: "birth"}, Kind: "codex --secret value"}}
	h, err := (LifecycleFailureHistory{}).Upsert(f)
	if err != nil {
		t.Fatal(err)
	}
	if f.Holders[0].Kind != "codex --secret value" {
		t.Fatal("mutated caller evidence")
	}
	if h[0].Holders[0].Kind != "process" || h[0].Outcome != "failed" {
		t.Fatal("unsafe kind or missing default outcome")
	}
}

func TestLifecycleFailureLegacyReadAndKnownNullRules(t *testing.T) {
	base := `{"address":{"repo_scope":"repo","tag":"x"},"nonce":"1","failed_at":"1970-01-01T00:00:01Z"}`
	h, err := decodeLifecycleFailureHistory([]byte("[" + base + "]"))
	if err != nil || len(h) != 1 || h[0].HelperStarted {
		t.Fatalf("legacy helper_started omission: %v", err)
	}
	for _, field := range []string{"helper_started", "starting_path", "phase", "outcome", "holders", "superseded_at", "attempt_order", "attempt_epoch"} {
		r := strings.TrimSuffix(base, "}") + `,"` + field + `":null}]`
		if _, err := decodeLifecycleFailureHistory([]byte("[" + r)); err == nil {
			t.Fatalf("accepted null %s", field)
		}
	}
	for _, field := range []string{`"phase":"invented"`, `"outcome":"success"`, `"attempt_order":-1`, `"holders":[{"process":{"PID":0,"Identity":"x"},"kind":"codex"}]`} {
		r := strings.TrimSuffix(base, "}") + "," + field + "}]"
		if _, err := decodeLifecycleFailureHistory([]byte("[" + r)); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}
func TestLifecycleFailureCloneDoesNotAliasOpaqueEvidence(t *testing.T) {
	raw := []byte(`[{"address":{"repo_scope":"repo","tag":"x","future": { "a": 1 }},"nonce":"n","failed_at":"1970-01-01T00:00:01Z","future": { "a": 1 },"holders":[{"process":{"PID":4,"Identity":"birth","future": { "a": 1 }},"kind":"codex","future": { "a": 1 }}]}]`)
	h, err := decodeLifecycleFailureHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	clone := cloneLaunchFailure(h[0])
	clone.Unknown["future"][0] = 'X'
	clone.addressUnknown["future"][0] = 'X'
	clone.Holders[0].Unknown["future"][0] = 'X'
	clone.Holders[0].processUnknown["future"][0] = 'X'
	if _, err := encodeLifecycleFailureHistory(h); err != nil {
		t.Fatalf("clone mutated original: %v", err)
	}
}
func FuzzLifecycleFailureOpaqueRoundTrip(f *testing.F) {
	for _, v := range []string{`{ "x" : [1, true, "\u0061"] }`, `1234567890123456789`, `null`, `"escaped\nvalue"`} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, opaque string) {
		if !json.Valid([]byte(opaque)) {
			return
		}
		record := failureFixture(1)
		record.Unknown = map[string]json.RawMessage{"future": json.RawMessage(opaque)}
		encoded, err := encodeLifecycleFailureHistory(LifecycleFailureHistory{record})
		if err != nil {
			return
		}
		decoded, err := decodeLifecycleFailureHistory(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded[0].Unknown["future"], bytes.TrimSpace([]byte(opaque))) {
			t.Fatalf("raw changed %q to %q", opaque, decoded[0].Unknown["future"])
		}
	})
}

func TestLifecycleFailureSuccessWithoutMatchingFailureDoesNotRewrite(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	f := failureFixture(1)
	f.StartingPath = "/physical/other"
	if err := s.Save(t.Context(), f); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Supersede(t.Context(), LifecycleFailureTarget{StartingPath: "/physical/slot"}, "epoch", 2, time.Unix(5, 0)); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("unrelated success rewrote store")
	}
}
func TestLifecycleFailureLateEnrichmentRetainsClassificationAndHolderBound(t *testing.T) {
	f := failureFixture(1)
	f.Outcome = "refused"
	f.Phase = "prove"
	for i := 1; i <= 40; i++ {
		f.Holders = append(f.Holders, ProcessIdentityLine{Process: ProcessIdentity{PID: i, Identity: "birth"}, Kind: "codex"})
	}
	h, err := (LifecycleFailureHistory{}).Upsert(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(h[0].Holders) != 32 || h[0].HoldersTruncated != 8 || h[0].Holders[0].Process.PID != 9 {
		t.Fatal("unbounded holder evidence")
	}
	h, err = h.Upsert(failureFixture(1))
	if err != nil {
		t.Fatal(err)
	}
	if h[0].Outcome != "refused" || h[0].Phase != "prove" || len(h[0].Holders) != 32 {
		t.Fatal("late unclassified enrichment replaced classified evidence")
	}
}

func TestLifecycleFailureByteSelectionEncodesRowsOnce(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			h := make(LifecycleFailureHistory, n)
			for i := range h {
				h[i] = failureFixture(n - i)
			}
			calls := 0
			retained, err := fitLifecycleFailureHistory(h, 2+10+1+10, func(f LifecycleFailure) ([]byte, error) { calls++; return []byte("0123456789"), nil })
			if err != nil || len(retained) != 2 || retained[0].Nonce != fmt.Sprint(n) {
				t.Fatalf("bad retention %d, %v", len(retained), err)
			}
			if calls != n {
				t.Fatalf("selection encoded %d rows for N=%d; want exactly N", calls, n)
			}
		})
	}
}
