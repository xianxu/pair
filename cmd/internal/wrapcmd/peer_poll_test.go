package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestPeerExpiryReportsBlockedInputState(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Now()
	d.now = func() time.Time { return now }
	d.admitInput([]byte("\x1b"))
	d.inputForwarded(false, false)
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(d.receipt().Detail, "incomplete terminal input") {
		t.Fatal("missing pending reason", d.receipt())
	}
	now = now.Add(couchmessage.DeliveryTimeout)
	f.proxy.dispatchPeer(&out)
	r := d.receipt()
	if r.Status != couchmessage.Expired || !strings.Contains(r.Detail, "incomplete terminal input") {
		t.Fatalf("expiry lost the blocking reason: %+v", r)
	}
	if out.Len() != 0 {
		t.Fatal("diagnostic changed input")
	}
}

func TestPeerPollingExistsOnlyForPendingDelivery(t *testing.T) {
	var poll peerDeliveryPoll
	defer poll.stop()
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	if poll.update(nil) != nil || poll.update(d) != nil || poll.ticker != nil {
		t.Fatal("idle receiver allocated a polling timer")
	}
	for _, end := range []couchmessage.Status{couchmessage.Submitted, couchmessage.Cancelled, couchmessage.Expired, couchmessage.Indeterminate} {
		d.current = couchmessage.Receipt{Message: peerTestMessage(time.Now()), Status: couchmessage.Queued}
		ch := poll.update(d)
		if ch == nil {
			t.Fatal("queued delivery has no polling timer")
		}
		d.current.Status = couchmessage.Delivering
		if poll.update(d) != ch || poll.every != peerPastedPoll {
			t.Fatal("a pasted delivery lost its timer or kept the slow pace")
		}
		d.current.Status = end
		if poll.update(d) != nil || poll.ticker != nil {
			t.Fatalf("%s retained its timer", end)
		}
	}
}

func TestPeerBufferedInputBlocksBeyondSettleInterval(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Now()
	d.now = func() time.Time { return now }
	d.admitInput([]byte("\x1b[200~"))
	d.inputForwarded(true, true)
	peerIntegrationEnqueue(t, d)
	now = now.Add(2 * time.Second)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatal("pasted while operator paste was unfinished")
	}
}
