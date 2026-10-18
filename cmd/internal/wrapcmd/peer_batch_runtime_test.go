package wrapcmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// This stitches the real Broker, Unix-socket RemoteEndpoint and wrapper input
// writer together. A held head cannot paste until the broker freezes followers.
func TestPeerBatchRuntimeSocketFIFOAndReceiptRecovery(t *testing.T) {
	const batchingEnabled = true // negative control: false must fail the single-paste oracle
	namespace := t.TempDir()
	lifetime, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := peerBatchTestBinding("ops:0")
	f, _ := peerIntegrationFixture(t)
	receiver := newPeerDelivery(hub, time.Now)
	f.proxy.peer = receiver
	f.output(peerFilledComposer("operator draft"))
	senders := make([]couchmessage.Binding, 5)
	deliveries := []*peerDelivery{receiver}
	for i := range senders {
		senders[i] = peerBatchTestBinding(fmt.Sprintf("pair:%d", i+1))
		senders[i].Tag = fmt.Sprintf("sender-%d", i)
		senders[i].Nonce = fmt.Sprintf("launch-%d", i)
		deliveries = append(deliveries, newPeerDelivery(senders[i], time.Now))
	}
	for _, d := range deliveries {
		socket, err := couchmessage.EndpointSocket(namespace, d.binding)
		if err != nil {
			t.Fatal(err)
		}
		server, err := couchmessage.StartServer(lifetime, socket, d.handleEndpoint)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = server.Close() })
	}
	newBroker := func() *couchmessage.Broker {
		b := couchmessage.NewBroker(lifetime, time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
		t.Cleanup(func() { _ = b.Close() })
		remote := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: hub}
		var err error
		if batchingEnabled {
			err = b.RegisterWithBatching(hub, remote, couchmessage.HubAllowance, couchmessage.DefaultPeerBatchMaxChars)
		} else {
			err = b.Register(hub, remote)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, from := range senders {
			if err := b.Register(from, couchmessage.RemoteEndpoint{Namespace: namespace, Binding: from}); err != nil {
				t.Fatal(err)
			}
		}
		return b
	}
	broker := newBroker()
	originals := make([]couchmessage.Message, 5)
	send := func(i int) {
		r, err := broker.Send(lifetime, senders[i], fmt.Sprintf("runtime-batch-%d", i), couchmessage.Route{Target: hub.Slot}, fmt.Sprintf("FIFO report %d\nUnicode 界 🦉", i))
		if err != nil {
			t.Fatal(err)
		}
		originals[i] = r.Message
	}
	send(0)
	if !waitFor(t, func() bool { return receiver.receipt().Message.ID == originals[0].ID }) {
		t.Fatal("head never prepared through real socket")
	}
	var input bytes.Buffer
	f.proxy.dispatchPeer(&input)
	if input.Len() != 0 {
		t.Fatal("prepared head overwrote human draft")
	}
	for i := 1; i < 5; i++ {
		send(i)
	}
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&input)
	if input.Len() != 0 {
		t.Fatal("readiness wrote before membership freeze")
	}
	if !waitFor(t, func() bool {
		receiver.mu.Lock()
		defer receiver.mu.Unlock()
		return receiver.batch != nil && receiver.batch.phase == couchmessage.BatchFrozen
	}) {
		t.Fatal("remote readiness did not freeze/finalize real broker batch")
	}
	f.proxy.dispatchPeer(&input)
	envelopes := make([]string, len(originals))
	for i, m := range originals {
		envelopes[i] = fmt.Sprintf("[Couch peer from %s; delivery %s]\n%s", m.From.Slot, m.ID, m.Body)
	}
	want := workbenchshortcut.PasteStart + strings.Join(envelopes, "\n\n---\n\n") + workbenchshortcut.PasteEnd
	if input.String() != want {
		t.Fatalf("one FIFO paste mismatch: got %q want %q", input.String(), want)
	}
	for _, m := range originals {
		r, err := broker.Status(m.From, m.ID)
		if err != nil || r.Status == couchmessage.Submitted {
			t.Fatalf("member succeeded before confirmation: %+v %v", r, err)
		}
	}
	f.proxy.dispatchPeer(&input)
	if input.String() != want {
		t.Fatal("submitted before fixed paste delay")
	}
	time.Sleep(PeerSubmitDelay)
	f.proxy.dispatchPeer(&input)
	f.proxy.dispatchPeer(&input)
	if input.String() != want+"\r" {
		t.Fatalf("not exactly one submit: %q", input.String())
	}
	f.proxy.turnActive.Store(true)
	f.proxy.dispatchPeer(&input)
	for _, m := range originals {
		if !waitFor(t, func() bool {
			r, err := broker.Status(m.From, m.ID)
			return err == nil && r.Status == couchmessage.Submitted
		}) {
			t.Fatalf("member %s confirmation never returned", m.ID)
		}
	}
	remote := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: hub}
	for _, m := range originals {
		r, err := remote.Retained(lifetime, m.ID)
		if err != nil || r.Status != couchmessage.Submitted || r.Message.ID != m.ID || r.Message.From != m.From || r.Message.To != m.To || r.Message.Body != m.Body {
			t.Fatalf("retained member %+v: %v", r, err)
		}
	}
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	recovered := newBroker()
	for _, m := range originals[1:] {
		r, err := recovered.StatusContext(lifetime, m.From, m.ID)
		if err != nil || r.Status != couchmessage.Submitted || r.Message.ID != m.ID || r.Message.Body != m.Body {
			t.Fatalf("fresh broker lost follower %s: %+v %v", m.ID, r, err)
		}
	}
	f.proxy.dispatchPeer(&input)
	if input.String() != want+"\r" {
		t.Fatal("receipt recovery replayed the batch")
	}
}
