package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Refusal bookkeeping reuses admission's routing snapshot, even when the
// original lookup times out; it cannot extend the request with another read.
func TestBrokerRefusalReusesAdmissionFamilies(t *testing.T) {
	for _, failure := range []string{"recipient", "lookup-timeout", "lookup-cancelled"} {
		t.Run(failure, func(t *testing.T) {
			now := time.Now()
			b := NewBroker(context.Background(), nil, nil)
			defer b.Close()
			from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
			_ = b.Register(from, newFakeEndpoint(now))
			_ = b.Register(to, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: errors.New("recipient check failed")})
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			var calls atomic.Int32
			b.SetFamilies(func(ctx context.Context) (map[string]string, error) {
				if calls.Add(1) != 1 {
					return map[string]string{"other": "p"}, nil
				}
				if failure == "lookup-cancelled" {
					cancel()
				}
				if failure != "recipient" {
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return map[string]string{"pair": "p"}, nil
			})
			_, err := b.Send(ctx, from, "snapshot", Route{Target: "p:3"}, "work")
			if err == nil {
				t.Fatal("send unexpectedly admitted")
			}
			if got := calls.Load(); got != 1 {
				t.Errorf("families read %d times; refusal must reuse admission's snapshot", got)
			}
			receipt, err := b.Status(from, "snapshot")
			if err != nil || receipt.Status != Refused {
				t.Fatalf("refused receipt: %+v %v", receipt, err)
			}
			if failure == "recipient" && receipt.Message.To.Slot != to.Slot {
				t.Fatalf("refusal discarded original routing snapshot: %+v", receipt)
			}
			if failure == "lookup-cancelled" && !strings.Contains(receipt.Detail, "cancelled") {
				t.Fatalf("cancellation reason lost: %+v", receipt)
			}
		})
	}
}

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
	if got := b.LatestRefusals()["pair:3"]; !strings.Contains(got.Reason, "zellij liveness is stale") {
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
	b.SetFamilies(func(context.Context) (map[string]string, error) {
		return map[string]string{"pair": "p", "ops": ""}, nil
	})
	from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
	from.Nonce = "sender-nonce-credential"
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: &RelaunchRequiredError{Cause: errors.New("wrapper does not own the current binding")}})

	if _, err := b.Send(context.Background(), from, "m1", Route{Target: "p:3"}, "secret body text"); err == nil {
		t.Fatal("send was admitted")
	}
	latest := b.LatestRefusals()
	if got, ok := latest["pair:3"]; !ok || !strings.Contains(got.Reason, "does not own") || got.Class != RefusalRelaunch || len(latest) != 1 {
		t.Fatalf("latest refusals %#v", latest)
	}
	raw, _ := json.Marshal(Response{Code: "ok", Refusals: latest})
	for _, leak := range []string{"sender-nonce-credential", "secret body text", "session-ops:0"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("refusals answer leaks %q: %s", leak, raw)
		}
	}
}

func TestBrokerRefusalRetainsRecoveryClass(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
		class RefusalClass
	}{
		{"ownership reworded", &RelaunchRequiredError{Cause: errors.New("different wrapper owns this launch")}, RefusalRelaunch},
		{"conversation reworded", &RelaunchRequiredError{Cause: errors.New("native conversation identity replaced")}, RefusalRelaunch},
		{"registration reworded", &RelaunchRequiredError{Cause: errors.New("ready evidence unavailable")}, RefusalRelaunch},
		{"listing reworded", &RelaunchRequiredError{Cause: errors.New("terminal absent from verified listing")}, RefusalRelaunch},
		{"checkout reworded", &RelaunchRequiredError{Cause: errors.New("retained working path missing")}, RefusalRelaunch},
		{"unclassified old phrase", errors.New("wrapper does not own the current binding"), ""},
		{"wrapped classification", fmt.Errorf("outer: %w", &RelaunchRequiredError{Cause: errors.New("reworded")}), RefusalRelaunch},
		{"registration deadline", &RelaunchRequiredError{Cause: fmt.Errorf("ready: %w", context.DeadlineExceeded)}, ""},
		{"registration cancellation", &RelaunchRequiredError{Cause: fmt.Errorf("ready: %w", context.Canceled)}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(context.Background(), func() time.Time { return now }, nil)
			defer b.Close()
			from, to := brokerBinding("ops:0"), brokerBinding("pair:3")
			_ = b.Register(from, newFakeEndpoint(now))
			_ = b.Register(to, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), err: tc.cause})
			if _, err := b.Send(context.Background(), from, "refusal-class", Route{Target: "pair:3"}, "private body"); err == nil {
				t.Fatal("send unexpectedly admitted")
			}
			got := b.LatestRefusals()["pair:3"]
			if got.Class != tc.class || got.Reason != refusalReason(fmt.Errorf("recipient %s observe: %w", to.Slot, tc.cause)) {
				t.Fatalf("projection=%+v; want class %q and preserved diagnostic", got, tc.class)
			}
		})
	}
}

func TestRelaunchRequiredErrorPreservesCause(t *testing.T) {
	cause := errors.New("different wording")
	err := fmt.Errorf("authority: %w", &RelaunchRequiredError{Cause: cause})
	var relaunch *RelaunchRequiredError
	if !errors.Is(err, cause) || !errors.As(err, &relaunch) || relaunch.Error() != cause.Error() {
		t.Fatalf("classification lost original error: %v", err)
	}
}
