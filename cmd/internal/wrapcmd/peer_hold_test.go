package wrapcmd

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// Operator holds remain visible and protect drafts/scrollback. Accepted
// messages do not expire before paste while the receiver stays connected.

// peerHoldFixture enqueues one message on a controlled clock. The composer
// starts holding the operator's draft ("alpha").
func peerHoldFixture(t *testing.T) (*harnessSessionFake, *peerDelivery, *time.Time) {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output("\x1b[?2004h" + claudeLiveComposerPaint())
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	peerIntegrationEnqueue(t, d)
	return f, d, &now
}

func TestPeerDraftHoldStopsTheDeadline(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Held || r.Detail != "waiting for empty composer" {
		t.Fatalf("draft not reported as a hold: %+v", r)
	}
	*now = now.Add(2 * time.Minute) // well past the initial commit deadline
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Held {
		t.Fatalf("held draft spent the deadline: %+v", r)
	}
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") || d.receipt().Status != couchmessage.Delivering {
		t.Fatalf("emptied composer did not release the delivery: %q %+v", out.String(), d.receipt())
	}
}

func TestPeerScrollbackHoldStopsTheDeadline(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	f.output(peerEmptyComposer())
	var inView atomic.Bool
	d.viewProbe = func() bool { return inView.Load() }
	var out bytes.Buffer
	read := func() {
		t.Helper()
		f.proxy.dispatchPeer(&out) // starts a reading
		if !waitFor(t, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return !d.view.checking }) {
			t.Fatal("no viewport reading")
		}
		f.proxy.dispatchPeer(&out)
	}
	read()
	if r := d.receipt(); r.Status != couchmessage.Held || !strings.Contains(r.Detail, "in view") {
		t.Fatalf("scrollback not reported as a hold: %+v", r)
	}
	*now = now.Add(2 * time.Minute)
	read()
	if r := d.receipt(); r.Status != couchmessage.Held || out.Len() != 0 {
		t.Fatalf("held delivery expired or pasted while scrolled: %q %+v", out.String(), r)
	}
	inView.Store(true)
	*now = now.Add(peerViewFresh)
	read()
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("scrolling back did not release: %+v", d.receipt())
	}
}

// A draft disappearing into an unknown composer still retains the message.
func TestPeerUnrecognizedComposerAfterDraftStillWaits(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out) // the draft holds from here
	*now = now.Add(time.Minute)
	f.output("\x1b[2Jno composer here")
	f.proxy.dispatchPeer(&out) // the draft hold closes: one minute credited
	if r := d.receipt(); r.Status != couchmessage.Queued || !strings.Contains(r.Detail, "waiting for recognized composer:") {
		t.Fatalf("unrecognized composer: %+v", r)
	}
	deadline := d.receipt().Message.Deadline.Add(time.Minute)
	*now = deadline.Add(-time.Second)
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status.Terminal() {
		t.Fatalf("expired before the credited deadline: %+v", r)
	}
	*now = deadline
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Queued || out.Len() != 0 {
		t.Fatalf("connected composer wait discarded mail: %+v", r)
	}
}

func TestPeerLongDraftWaitNamesTheHold(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	*now = now.Add(2 * time.Hour)
	f.proxy.dispatchPeer(&out)
	r := d.receipt()
	if r.Status != couchmessage.Held || !strings.Contains(r.Detail, "empty composer") {
		t.Fatalf("hold limit: %+v", r)
	}
	if out.Len() != 0 {
		t.Fatalf("pasted at the hold limit: %q", out.String())
	}
}

// Typing the draft is part of the operator's hold (BR-1): the waits a
// keystroke causes leave an open hold open.
func TestPeerTypingKeepsTheHoldOpen(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	for i := 0; i < 4; i++ {
		*now = now.Add(time.Minute)
		human, _ := d.admitInput([]byte("x"))
		d.inputForwarded(human, false)
		f.proxy.dispatchPeer(&out)
		if r := d.receipt(); r.Status != couchmessage.Held || r.Detail != "waiting for input to settle" {
			t.Fatalf("typing closed the hold or spent the deadline: %+v", r)
		}
	}
	f.output(peerEmptyComposer())
	*now = now.Add(time.Second)
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("draft cleared but no paste: %+v", d.receipt())
	}
}

// A closed draft hold becomes a recognized-composer wait rather than expiry.
func TestPeerLongWaitAfterTheDraftClosed(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	limit := now.Add(2 * time.Hour)
	*now = limit.Add(-time.Second)
	f.output("\x1b[2Jno composer here")
	f.proxy.dispatchPeer(&out) // the old draft is no longer visible
	*now = limit
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Queued || !strings.Contains(r.Detail, "recognized composer") {
		t.Fatalf("closed-hold limit: %+v", r)
	}
}
