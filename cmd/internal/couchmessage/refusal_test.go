package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type failingObserveEndpoint struct {
	*FakeDeliveryEndpoint
	err error
}

func (f *failingObserveEndpoint) Observe(ctx context.Context) (Observation, error) {
	if f.err == context.DeadlineExceeded {
		<-ctx.Done()
		return Observation{}, ctx.Err()
	}
	return Observation{}, f.err
}

// A send that is not admitted leaves a refused receipt naming why, so the
// status query no longer says "no connected recipient retains" (pair#618).
func TestBrokerRecordsWhyASendWasNotAdmitted(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
	_ = b.Register(from, newFakeEndpoint(now))
	recipient := &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: errors.New("zellij liveness is stale (last listing 7s ago)")}
	_ = b.Register(to, recipient)

	_, err := b.Send(context.Background(), from, "m1", Route{Target: "pair:3"}, "dispatch")
	if err == nil || !strings.Contains(err.Error(), "zellij liveness is stale") {
		t.Fatalf("send error %v", err)
	}
	r, err := b.StatusContext(context.Background(), from, "m1")
	if err != nil || r.Status != Refused || !strings.Contains(r.Detail, "zellij liveness is stale") || r.Message.To.Slot != "pair:3" {
		t.Fatalf("status %+v %v", r, err)
	}
	if got := b.LatestRefusals()["pair:3"]; !strings.Contains(got, "zellij liveness is stale") {
		t.Fatalf("latest refusals %+v", b.LatestRefusals())
	}

	// An admission that runs out of budget is a definite refusal, not "uncertain".
	recipient.err = context.DeadlineExceeded
	_, err = b.Send(context.Background(), from, "m2", Route{Target: "pair:3"}, "dispatch")
	var refused *RefusedError
	if !errors.As(err, &refused) || errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timed-out send %v", err)
	}
	if resp := protocolError(err, false); resp.Code != "refused" {
		t.Fatalf("protocol code %q", resp.Code)
	}

	// A resend with a refused ID is a fresh attempt, and an admission to the
	// slot clears it from the list.
	recipient.err = nil
	_ = b.Register(to, newFakeEndpoint(now))
	b.Disconnect(to)
	healthy := newFakeEndpoint(now)
	_ = b.Register(to, healthy)
	if _, err := b.Send(context.Background(), from, "m2", Route{Target: "pair:3"}, "dispatch"); err != nil {
		t.Fatalf("resend after refusal: %v", err)
	}
	if r, err := b.Status(from, "m2"); err != nil || r.Status == Refused {
		t.Fatalf("resent status %+v %v", r, err)
	}
	if _, listed := b.LatestRefusals()["pair:3"]; listed {
		t.Fatal("a later admission must supersede the slot's refusal")
	}
}

func TestBrokerRefusalsAreBoundedAndExpire(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: errors.New("wrapper does not own the current binding")})
	for i := range MaxRefusals + 5 {
		now = now.Add(time.Millisecond)
		_, _ = b.Send(context.Background(), from, fmt.Sprintf("r%d", i), Route{Target: "pair:3"}, "x")
	}
	b.mu.Lock()
	n, receipts := len(b.refusals), len(b.receipts)
	b.mu.Unlock()
	if n != MaxRefusals || receipts != 0 {
		t.Fatalf("refusals %d receipts %d", n, receipts)
	}
	if _, err := b.Status(from, "r0"); err == nil {
		t.Fatal("the oldest refusal was not evicted")
	}
	now = now.Add(RefusalRetention)
	if _, err := b.Status(from, fmt.Sprintf("r%d", MaxRefusals+4)); err == nil {
		t.Fatal("a refusal outlived its retention")
	}
}

func TestRefusalsRequestTakesNoFields(t *testing.T) {
	if err := ValidateRequest(Request{Op: "refusals"}); err != nil {
		t.Fatalf("bare refusals: %v", err)
	}
	if err := ValidateRequest(Request{Op: "refusals", Target: "pair:3"}); err == nil {
		t.Fatal("refusals accepted a field")
	}
}

// The refusals op is unauthenticated, so its answer is the canonical slot and
// the reason only (pair#618 BR-10), and an alias-spelled send is listed under
// the slot's own name, as --list labels it (BR-13).
func TestLatestRefusalsAreCanonicalReasonsOnly(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	b.SetFamilies(func(context.Context) (map[string]string, error) { return map[string]string{"pair": "p", "ops": ""}, nil })
	from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
	from.Nonce = "sender-nonce-credential"
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: errors.New("wrapper does not own the current binding")})

	if _, err := b.Send(context.Background(), from, "m1", Route{Target: "p:3"}, "secret body text"); err == nil {
		t.Fatal("send was admitted")
	}
	latest := b.LatestRefusals()
	if got, ok := latest["pair:3"]; !ok || !strings.Contains(got, "does not own") || len(latest) != 1 {
		t.Fatalf("latest refusals %#v", latest)
	}
	raw, _ := json.Marshal(Response{Code: "ok", Refusals: latest})
	for _, leak := range []string{"sender-nonce-credential", "secret body text", "session-ops:0"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("refusals answer leaks %q: %s", leak, raw)
		}
	}
}
