package remotesession

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"testing"
	"time"
)

func TestTailnetNodeEnrollmentAndLookup(t *testing.T) {
	f := newFakeTailnetNode()
	var node tailnetNode = f
	ctx := context.Background()
	if err := node.Start(ctx); err != nil {
		t.Fatal(err)
	}
	status, err := node.Status(ctx)
	if err != nil || status.phase != tailnetNodePending {
		t.Fatalf("pending: %+v %v", status, err)
	}
	f.setStatus(tailnetNodeStatus{phase: tailnetNodeNeedsLogin, loginURL: "https://login.invalid/console-only"})
	status, _ = node.Status(ctx)
	if status.phase != tailnetNodeNeedsLogin || status.loginURL == "" {
		t.Fatal(status)
	}
	f.setStatus(tailnetNodeStatus{phase: tailnetNodeRunning, fqdn: "couch-2.example.ts.net", selfNodeID: "node-self", selfUserID: 9})
	status, _ = node.Status(ctx)
	if status.fqdn != "couch-2.example.ts.net" {
		t.Fatal(status)
	}
	addr := netip.MustParseAddrPort("100.64.0.3:1234")
	obs := tailnetObservation{login: "human", userID: 3, nodeID: "peer", peerIP: addr.Addr(), epoch: 1, observedAt: time.Unix(10, 0), expires: time.Unix(20, 0), state: tailnetObservationCurrent}
	f.setPeer(addr, obs)
	for _, n := range []int{8, 16} {
		before := f.counts().lookups
		for range n {
			got, err := node.WhoIs(ctx, addr)
			if err != nil || got != obs {
				t.Fatalf("lookup: %+v %v", got, err)
			}
		}
		if got := f.counts().lookups - before; got != n {
			t.Fatalf("lookups=%d want %d", got, n)
		}
	}
	if _, err := node.WhoIs(ctx, netip.MustParseAddrPort("100.64.0.4:1234")); err == nil {
		t.Fatal("unknown peer accepted")
	}
	obs.state = tailnetObservationRevoked
	f.setPeer(addr, obs)
	got, _ := node.WhoIs(ctx, addr)
	if got.state != tailnetObservationRevoked {
		t.Fatal("stale peer")
	}
	if err := node.Close(); err != nil {
		t.Fatal(err)
	}
	if c := f.counts(); c.owners != 0 || c.watches != 0 {
		t.Fatal(c)
	}
	if err := node.Start(ctx); err == nil {
		t.Fatal("closed node restarted")
	}
	if _, err := node.WhoIs(ctx, addr); err == nil {
		t.Fatal("closed lookup")
	}
}

func TestTailnetNodeWatchOwnershipAndLoss(t *testing.T) {
	f := newFakeTailnetNode()
	ctx := context.Background()
	_ = f.Start(ctx)
	w, err := f.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Watch(ctx); err == nil {
		t.Fatal("second watch")
	}
	first, err := w.Next()
	if err != nil || first.phase != tailnetNodePending {
		t.Fatal(first, err)
	}
	f.setStatus(tailnetNodeStatus{phase: tailnetNodeRunning, fqdn: "assigned.example.ts.net"})
	got, err := w.Next()
	if err != nil || got.phase != tailnetNodeRunning {
		t.Fatal(got, err)
	}
	loss := errors.New("watch lost")
	f.loseWatch(loss)
	if _, err = w.Next(); !errors.Is(err, loss) {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if c := f.counts(); c.watches != 0 || c.owners != 1 {
		t.Fatal(c)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Next(); err == nil {
		t.Fatal("closed watch read")
	}
}

func TestTailnetNodeHeldStartCancelledThenClosed(t *testing.T) {
	f := newFakeTailnetNode()
	f.startGate = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.Start(ctx) }()
	<-f.startEntered
	if c := f.counts(); c.owners != 1 {
		t.Fatal(c)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	close(f.startGate)
	if c := f.counts(); c.owners != 0 {
		t.Fatal(c)
	}
	status, _ := f.Status(context.Background())
	if status.phase != tailnetNodeClosed {
		t.Fatal(status)
	}
}

func TestTailnetNodeHeldCloseKeepsOwnerOnUnknownOutcome(t *testing.T) {
	f := newFakeTailnetNode()
	_ = f.Start(context.Background())
	w, _ := f.Watch(context.Background())
	_, _ = w.Next()
	f.closeGate = make(chan struct{})
	f.closeErr = errors.New("cleanup unknown")
	done := make(chan error, 1)
	go func() { done <- f.Close() }()
	<-f.closeEntered
	if c := f.counts(); c.owners != 1 {
		t.Fatal(c)
	}
	if err := f.Start(context.Background()); err == nil {
		t.Fatal("second start during close")
	}
	close(f.closeGate)
	if err := <-done; !errors.Is(err, f.closeErr) {
		t.Fatal(err)
	}
	if c := f.counts(); c.owners != 1 || c.watches != 0 {
		t.Fatal(c)
	}
	if _, err := w.Next(); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	// Unknown cleanup remains unknown on retry; tests cannot manufacture proof.
	if err := f.Close(); !errors.Is(err, f.closeErr) {
		t.Fatal(err)
	}
}

func TestTailnetNodeWatchCloseJoinsHeldRead(t *testing.T) {
	f := newFakeTailnetNode()
	_ = f.Start(context.Background())
	w, _ := f.Watch(context.Background())
	_, _ = w.Next()
	fw := w.(*fakeTailnetWatch)
	done := make(chan error, 1)
	go func() { _, err := w.Next(); done <- err }()
	<-fw.readEntered
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if c := f.counts(); c.watches != 0 {
		t.Fatal(c)
	}
	_ = f.Close()
}

func TestTailnetNodeCloseJoinsHeldStart(t *testing.T) {
	f := newFakeTailnetNode()
	f.startGate = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- f.Start(context.Background()) }()
	<-f.startEntered
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.startDone:
	default:
		t.Fatal("Close did not join Start")
	}
	if err := <-done; err == nil {
		t.Fatal("late start succeeded")
	}
	if c := f.counts(); c.owners != 0 {
		t.Fatal(c)
	}
}

func TestTailnetNodeWatchOverflowIsUnknownHealth(t *testing.T) {
	f := newFakeTailnetNode()
	_ = f.Start(context.Background())
	w, _ := f.Watch(context.Background())
	for range 17 {
		f.setStatus(tailnetNodeStatus{phase: tailnetNodeRunning})
	}
	if _, err := w.Next(); err == nil {
		t.Fatal("overflow retained running observation")
	}
	_ = f.Close()
}

func TestTailnetNodeRefusesUnhealthyAndCancelledIO(t *testing.T) {
	f := newFakeTailnetNode()
	if c := f.counts(); c.owners != 0 || c.watches != 0 {
		t.Fatal(c)
	}
	ctx := context.Background()
	_ = f.Start(ctx)
	addr := netip.MustParseAddrPort("100.64.0.3:1234")
	for _, phase := range []tailnetNodePhase{tailnetNodePending, tailnetNodeNeedsLogin, tailnetNodeRevoked} {
		f.setStatus(tailnetNodeStatus{phase: phase})
		if _, err := f.WhoIs(ctx, addr); err == nil {
			t.Fatal("unhealthy lookup")
		}
	}
	f.setStatus(tailnetNodeStatus{phase: tailnetNodeRunning})
	f.lookupErr = errors.New("lookup failed")
	if _, err := f.WhoIs(ctx, addr); !errors.Is(err, f.lookupErr) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := f.Status(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := f.WhoIs(cancelled, addr); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := f.Watch(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	watchCtx, cancelWatch := context.WithCancel(ctx)
	w, _ := f.Watch(watchCtx)
	_, _ = w.Next()
	done := make(chan error, 1)
	go func() { _, err := w.Next(); done <- err }()
	<-w.(*fakeTailnetWatch).readEntered
	cancelWatch()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_ = w.Close()
	_ = f.Close()
}

func TestTailnetNodeStartupFailureRetainsOwnerUntilClose(t *testing.T) {
	f := newFakeTailnetNode()
	f.startErr = errors.New("enrollment rejected")
	if err := f.Start(context.Background()); !errors.Is(err, f.startErr) {
		t.Fatal(err)
	}
	if c := f.counts(); c.owners != 1 {
		t.Fatal(c)
	}
	if err := f.Start(context.Background()); err == nil {
		t.Fatal("failed attempt allowed second start")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if c := f.counts(); c.owners != 0 {
		t.Fatal(c)
	}
}

func TestTailnetNodeCloseBeforeStartDoesNotCreateOwner(t *testing.T) {
	f := newFakeTailnetNode()
	if _, err := f.Watch(context.Background()); err == nil {
		t.Fatal("watch before node start")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if c := f.counts(); c.owners != 0 || c.watches != 0 {
		t.Fatal(c)
	}
	if err := f.Start(context.Background()); err == nil {
		t.Fatal("retired node started")
	}
}

func TestTailnetNodeConcurrentRevocationCannotPublishOlderRunning(t *testing.T) {
	f := newFakeTailnetNode()
	_ = f.Start(context.Background())
	w, _ := f.Watch(context.Background())
	_, _ = w.Next()
	revokedDone := make(chan struct{})
	// Force revocation into the publication gap if that gap exists. TryLock
	// makes the adversarial schedule deterministic, without sleeps: if the
	// authoritative lock is still held, the competing setter must wait for
	// Running's publication; otherwise it can publish Revoked first.
	f.beforeStatusPublish = func(s tailnetNodeStatus) {
		if s.phase != tailnetNodeRunning {
			return
		}
		revoke := func() { f.setStatus(tailnetNodeStatus{phase: tailnetNodeRevoked}); close(revokedDone) }
		if f.mu.TryLock() {
			f.mu.Unlock()
			revoke()
		} else {
			go revoke()
		}
	}
	f.setStatus(tailnetNodeStatus{phase: tailnetNodeRunning})
	<-revokedDone
	first, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}
	if first.phase != tailnetNodeRunning || second.phase != tailnetNodeRevoked {
		t.Fatalf("watch delivered %v then %v; older Running must not follow Revoked", first.phase, second.phase)
	}
	status, err := f.Status(context.Background())
	if err != nil || status.phase != second.phase {
		t.Fatalf("status/watch disagree: %+v %+v %v", status, second, err)
	}
	_ = f.Close()
}
