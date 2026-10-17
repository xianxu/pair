package couchmessage

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func batchFixture(id, body string) Message {
	b := Binding{Slot: "pair:0", Repository: "/repo", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 1, Start: "start"}
	from := b
	from.Slot = "pair:3"
	return Message{ID: id, From: from, To: b, Body: body, Deadline: time.Unix(1, 0), AcceptedAt: time.Unix(2, 0)}
}
func batchWireSize(ms []Message) int {
	wire := append([]Message(nil), ms...)
	for i := range wire {
		wire[i].AcceptedAt = time.Time{}
	}
	raw, err := json.Marshal(EndpointRequest{Op: "commit-batch", Binding: wire[0].To, ID: wire[0].ID, Messages: wire})
	if err != nil {
		panic(err)
	}
	return len(raw)
}
func TestPeerBatchEnvelopeAndSelection(t *testing.T) {
	a, b, c := batchFixture("a", "界🙂e\u0301\nfirst"), batchFixture("b", "second"), batchFixture("c", "third")
	want := "[Couch peer from pair:3; delivery a]\n界🙂e\u0301\nfirst\n\n---\n\n[Couch peer from pair:3; delivery b]\nsecond"
	if got := BatchEnvelope([]Message{a, b}); got != want {
		t.Fatalf("envelope %q", got)
	}
	if got := Envelope(a); got != "[Couch peer from pair:3; delivery a]\n"+a.Body {
		t.Fatalf("singleton %q", got)
	}
	cap := utf8.RuneCountInString(want)
	if got := SelectBatch(a, []Message{b, c}, cap); !reflect.DeepEqual(got, []Message{a, b}) {
		t.Fatalf("FIFO capped prefix: %#v", got)
	}
	if got := SelectBatch(a, []Message{b, c}, 1); !reflect.DeepEqual(got, []Message{a}) {
		t.Fatalf("over-cap singleton: %#v", got)
	}
	b.Body = strings.Repeat("x", 8192)
	if got := SelectBatch(a, []Message{b, c}, cap); len(got) != 1 {
		t.Fatalf("skipped oversized follower: %d", len(got))
	}
	if BatchEnvelope(nil) != "" {
		t.Fatal("empty batch has text")
	}
}
func TestPeerBatchEncodedFrameAndCountBounds(t *testing.T) {
	head := batchFixture("head", strings.Repeat("<", MaxBodyBytes))
	head.From.Repository = strings.Repeat("&", 500)
	head.To.Repository = strings.Repeat("&", 500)
	queued := make([]Message, MaxQueued+4)
	for i := range queued {
		queued[i] = head
		queued[i].ID = fmt.Sprint(i)
	}
	got := SelectBatch(head, queued, 1<<30)
	if len(got) < 2 || len(got) > MaxQueued+1 || batchWireSize(got) > MaxFrameBytes {
		t.Fatalf("unbounded batch count=%d bytes=%d", len(got), batchWireSize(got))
	}
	next := append(append([]Message(nil), got...), queued[len(got)-1])
	if len(got) < MaxQueued+1 && batchWireSize(next) <= MaxFrameBytes {
		t.Fatal("not maximal under frame bound")
	}
	// The artificial cap above isolates transport framing. Production policy
	// is narrower: only its qualified composer range may be finalized.
	qualified := SelectBatch(head, queued, MaxPeerBatchChars)
	if err := ValidateBatch(qualified, head.To, head.ID); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBatch(append([]Message{head}, queued...), head.To, head.ID); err == nil {
		t.Fatal("accepted excessive count")
	}
}
func TestPeerBatchValidation(t *testing.T) {
	a, b := batchFixture("a", "a"), batchFixture("b", "b")
	if err := ValidateBatch([]Message{a, b}, a.To, a.ID); err != nil {
		t.Fatalf("past nonzero deadline should remain valid: %v", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func([]Message) []Message
	}{
		{"empty", func([]Message) []Message { return nil }},
		{"duplicate", func(ms []Message) []Message { ms[1].ID = ms[0].ID; return ms }},
		{"bad id", func(ms []Message) []Message { ms[1].ID = "bad id"; return ms }},
		{"wrong receiver", func(ms []Message) []Message { ms[1].To.Nonce = "other"; return ms }},
		{"bad sender", func(ms []Message) []Message { ms[1].From.PID = 0; return ms }},
		{"empty body", func(ms []Message) []Message { ms[1].Body = ""; return ms }},
		{"control", func(ms []Message) []Message { ms[1].Body = "\x1b"; return ms }},
		{"zero deadline", func(ms []Message) []Message { ms[1].Deadline = time.Time{}; return ms }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateBatch(tc.mutate([]Message{a, b}), a.To, a.ID); err == nil {
				t.Fatal("accepted invalid batch")
			}
		})
	}
	if ValidateBatch([]Message{a, b}, a.To, "other") == nil {
		t.Fatal("accepted wrong head")
	}
	bad := a.To
	bad.PID = 0
	if ValidateBatch([]Message{a, b}, bad, a.ID) == nil {
		t.Fatal("accepted invalid receiver")
	}
	huge := make([]Message, MaxQueued+1)
	for i := range huge {
		huge[i] = batchFixture(fmt.Sprint(i), strings.Repeat("<", MaxBodyBytes))
	}
	if batchWireSize(huge) <= MaxFrameBytes {
		t.Fatal("fixture should exceed frame limit")
	}
	if ValidateBatch(huge, huge[0].To, huge[0].ID) == nil {
		t.Fatal("accepted oversized encoded request")
	}
}
func TestPeerBatchPreparation(t *testing.T) {
	phase := BatchPreparing
	var err error
	for _, event := range []BatchPreparationEvent{BatchUnavailable, BatchAvailable, BatchUnavailable, BatchAvailable, BatchFinalize} {
		phase, err = AdvanceBatchPreparation(phase, event)
		if err != nil {
			t.Fatal(err)
		}
	}
	if phase != BatchFrozen {
		t.Fatal("did not freeze")
	}
	for _, event := range []BatchPreparationEvent{BatchAvailable, BatchUnavailable, BatchFinalize} {
		if got, err := AdvanceBatchPreparation(phase, event); err == nil || got != BatchFrozen {
			t.Fatal("late event changed frozen phase")
		}
	}
	if got, err := AdvanceBatchPreparation(BatchPreparing, BatchFinalize); err == nil || got != BatchPreparing {
		t.Fatal("finalized without readiness")
	}
	if got, err := AdvanceBatchPreparation(BatchReady, BatchUnavailable); err != nil || got != BatchReady {
		t.Fatal("readiness permission was not sticky")
	}
	if _, err := AdvanceBatchPreparation(BatchPreparationPhase(99), BatchAvailable); err == nil {
		t.Fatal("accepted unknown phase")
	}
	if _, err := AdvanceBatchPreparation(BatchPreparing, BatchPreparationEvent(99)); err == nil {
		t.Fatal("accepted unknown event")
	}
}
func FuzzPeerBatchSelection(f *testing.F) {
	f.Add("界🙂e\u0301\n<&\\\"", uint16(210), uint8(5))
	f.Add("simple", uint16(1), uint8(32))
	f.Fuzz(func(t *testing.T, body string, limit uint16, count uint8) {
		if ValidateBody(body) != nil {
			t.Skip()
		}
		n := int(count) % (MaxQueued + 1)
		head := batchFixture("head<&\\\"", body)
		head.From.Repository = string([]rune(body)[:min(len([]rune(body)), 100)])
		queued := make([]Message, n)
		for i := range queued {
			queued[i] = batchFixture(fmt.Sprintf("queued-%d<&\\\"", i), body)
			queued[i].From.Repository = head.From.Repository
		}
		cap := int(limit)%MaxPeerBatchChars + 1
		got := SelectBatch(head, queued, cap)
		all := append([]Message{head}, queued...)
		if len(got) < 1 || len(got) > len(all) || !reflect.DeepEqual(got, all[:len(got)]) {
			t.Fatal("not contiguous prefix")
		}
		expected := make([]string, len(got))
		for i, m := range got {
			expected[i] = "[Couch peer from " + m.From.Slot + "; delivery " + m.ID + "]\n" + m.Body
		}
		if BatchEnvelope(got) != strings.Join(expected, "\n\n---\n\n") {
			t.Fatal("changed headers or bodies")
		}
		if len(got) > 1 && utf8.RuneCountInString(BatchEnvelope(got)) > cap {
			t.Fatal("character cap exceeded")
		}
		if batchWireSize(got) > MaxFrameBytes {
			t.Fatal("encoded frame exceeded")
		}
		if len(got) < len(all) {
			next := all[:len(got)+1]
			if utf8.RuneCountInString(BatchEnvelope(next)) <= cap && batchWireSize(next) <= MaxFrameBytes {
				t.Fatal("not maximal prefix")
			}
		}
		if err := ValidateBatch(got, head.To, head.ID); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzPeerBatchValidation(f *testing.F) {
	f.Add("valid 界🙂\n<&\\\"", uint8(0))
	f.Add("valid", uint8(1))
	f.Fuzz(func(t *testing.T, body string, mutation uint8) {
		if ValidateBody(body) != nil {
			t.Skip()
		}
		a, b := batchFixture("a", body), batchFixture("b", body)
		ms := []Message{a, b}
		wantValid := false
		switch mutation % 8 {
		case 0:
			wantValid = true
		case 1:
			ms[1].ID = ms[0].ID
		case 2:
			ms[1].ID = "bad\nid"
		case 3:
			ms[1].To.Session = "replacement"
		case 4:
			ms[1].From.Repository = ""
		case 5:
			ms[1].Deadline = time.Time{}
		case 6:
			ms[1].Body = body + "\x00"
		case 7:
			ms[0].ID = "wrong-head"
		}
		if got := ValidateBatch(ms, a.To, a.ID) == nil; got != wantValid {
			t.Fatalf("mutation %d accepted=%v", mutation%8, got)
		}
	})
}

func FuzzPeerBatchPreparation(f *testing.F) {
	f.Add([]byte{0, 1, 0, 2, 0})
	f.Fuzz(func(t *testing.T, events []byte) {
		phase := BatchPreparing
		offered, frozen := false, false
		for _, raw := range events {
			event := BatchPreparationEvent(raw % 4)
			next, err := AdvanceBatchPreparation(phase, event)
			if frozen {
				if next != BatchFrozen || err == nil {
					t.Fatal("frozen state changed or accepted late event")
				}
			} else if event > BatchFinalize {
				if next != phase || err == nil {
					t.Fatal("unknown event accepted")
				}
			} else if event == BatchFinalize {
				if offered {
					if next != BatchFrozen || err != nil {
						t.Fatal("ready finalization rejected")
					}
					frozen = true
				} else if next != phase || err == nil {
					t.Fatal("unready finalization accepted")
				}
			} else {
				if event == BatchAvailable {
					offered = true
				}
				want := BatchPreparing
				if offered {
					want = BatchReady
				}
				if next != want || err != nil {
					t.Fatal("readiness was not sticky")
				}
			}
			phase = next
		}
	})
}
