package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func gateAddr(tag string) ThreadAddress { return ThreadAddress{RepoScope: "s", Tag: ThreadTag(tag)} }

func TestGateDecision(t *testing.T) {
	a, b := gateAddr("a"), gateAddr("b")
	mine, stale := &gateToken{}, &gateToken{}
	held := map[ThreadAddress]gateHold{a: {op: "relaunch", token: mine}}
	cases := []struct {
		name    string
		holds   map[ThreadAddress]*gateToken
		address ThreadAddress
		want    gateVerdict
		running string
	}{
		{"free thread admits", nil, b, gateAdmit, ""},
		{"held thread refuses a stranger", nil, a, gateBusy, "relaunch"},
		{"holder re-enters its own hold", map[ThreadAddress]*gateToken{a: mine}, a, gateReenter, ""},
		{"a released hold in a surviving context admits a free thread", map[ThreadAddress]*gateToken{b: stale}, b, gateAdmit, ""},
		{"a stale token cannot enter a later hold", map[ThreadAddress]*gateToken{a: stale}, a, gateBusy, "relaunch"},
		{"a holder of another thread is still refused", map[ThreadAddress]*gateToken{b: mine}, a, gateBusy, "relaunch"},
	}
	for _, tc := range cases {
		got, running := gateDecision(held, tc.holds, tc.address)
		if got != tc.want || running != tc.running {
			t.Errorf("%s: got %v/%q, want %v/%q", tc.name, got, running, tc.want, tc.running)
		}
	}
}

func TestThreadGateRefusesNamingTheRunningOperationAndReleases(t *testing.T) {
	var g ThreadGate
	ctx, release, err := g.acquire(context.Background(), gateAddr("a"), "relaunch", false)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = g.acquire(context.Background(), gateAddr("a"), "resume", false)
	var busy *ThreadBusyError
	if !errors.As(err, &busy) || busy.Running != "relaunch" || busy.Address != gateAddr("a") {
		t.Fatalf("want ThreadBusyError naming relaunch, got %v", err)
	}
	if !IsThreadBusy(err) {
		t.Fatalf("IsThreadBusy(%v) = false", err)
	}
	// Re-entry through the holder's context neither refuses nor releases.
	_, inner, err := g.acquire(ctx, gateAddr("a"), "resume", false)
	if err != nil {
		t.Fatal(err)
	}
	inner()
	if _, _, err := g.acquire(context.Background(), gateAddr("a"), "resume", false); err == nil {
		t.Fatal("inner release must not free the outer hold")
	}
	// A different thread is unaffected.
	if _, r, err := g.acquire(context.Background(), gateAddr("b"), "resume", false); err != nil {
		t.Fatal(err)
	} else {
		r()
	}
	release()
	release() // idempotent
	if _, r, err := g.acquire(context.Background(), gateAddr("a"), "resume", false); err != nil {
		t.Fatalf("released thread must admit: %v", err)
	} else {
		r()
	}
	// The holder's context outlived its release: it must not bypass a new holder.
	_, other, err := g.acquire(context.Background(), gateAddr("a"), "detach", false)
	if err != nil {
		t.Fatal(err)
	}
	defer other()
	if _, _, err := g.acquire(ctx, gateAddr("a"), "resume", false); !IsThreadBusy(err) {
		t.Fatalf("a stale holder context must be refused, got %v", err)
	}
}

func TestThreadGateWaitAdmitsAfterReleaseAndHonoursContext(t *testing.T) {
	var g ThreadGate
	_, release, _ := g.acquire(context.Background(), gateAddr("a"), "resume", false)
	done := make(chan error, 1)
	go func() {
		_, r, err := g.acquire(context.Background(), gateAddr("a"), "leave", true)
		if err == nil {
			r()
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("wait must block while held, returned %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	_, release, _ = g.acquire(context.Background(), gateAddr("a"), "resume", false)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := g.acquire(ctx, gateAddr("a"), "leave", true); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait must return ctx error, got %v", err)
	}
}

func TestCouchGateIsSharedAcrossCalls(t *testing.T) {
	c := &Couch{}
	_, release, err := c.hold(context.Background(), gateAddr("a"), "relaunch")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, _, err := c.hold(context.Background(), gateAddr("a"), "resume"); !IsThreadBusy(err) {
		t.Fatalf("second hold on one Couch must refuse, got %v", err)
	}
}
