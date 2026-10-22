package wrapcmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// Real broker, Unix endpoint, immutable batch and actual PTY input writer. Only
// the child application's painted composer is a fixture; this is not a claim
// about an agent's native transcript persistence.
func TestRemoteProvenanceBrokerSocketWrapperInputAndReceipts(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			namespace := t.TempDir()
			lifetime, cancel := context.WithCancel(context.Background())
			defer cancel()
			from, to := peerBatchTestBinding("pair:6"), peerBatchTestBinding("ops:0")
			f, _ := peerIntegrationFixture(t)
			var clock atomic.Int64
			clock.Store(time.Now().UnixNano())
			receiver := newPeerDelivery(to, func() time.Time { return time.Unix(0, clock.Load()) })
			f.proxy.peer = receiver
			f.output(peerFilledComposer("operator draft"))
			for _, d := range []*peerDelivery{receiver, newPeerDelivery(from, time.Now)} {
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
			broker := couchmessage.NewBroker(lifetime, time.Now, nil)
			t.Cleanup(func() { _ = broker.Close() })
			if err := broker.Register(from, couchmessage.RemoteEndpoint{Namespace: namespace, Binding: from}); err != nil {
				t.Fatal(err)
			}
			endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: to}
			var registrationErr error
			if batch {
				registrationErr = broker.RegisterWithBatching(to, endpoint, couchmessage.HubAllowance, couchmessage.DefaultPeerBatchMaxChars)
			} else {
				registrationErr = broker.Register(to, endpoint)
			}
			if registrationErr != nil {
				t.Fatal(registrationErr)
			}
			source := couchmessage.RemoteSource{Email: "operator@example.com", IP: "2001:db8::1"}
			docID := "0123456789abcdef0123456789abcdef"
			remote, err := broker.SendRemote(lifetime, from, "remote-submit", couchmessage.Route{Target: to.Slot}, "submitted: /private/session.md", source, true, docID)
			if err != nil {
				t.Fatal(err)
			}
			if !waitFor(t, func() bool { return receiver.receipt().Message.ID == remote.Message.ID }) {
				t.Fatal("remote never prepared")
			}
			messages := []couchmessage.Message{remote.Message}
			if batch {
				local, err := broker.SendLocal(lifetime, from, "local-submit", couchmessage.Route{Target: to.Slot}, "submitted: /private/session.md", true, docID)
				if err != nil {
					t.Fatal(err)
				}
				messages = append(messages, local.Message)
			}
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			if input.Len() != 0 {
				t.Fatal("remote overwrote operator draft")
			}
			f.output(peerEmptyComposer())
			f.proxy.dispatchPeer(&input)
			if batch {
				if !waitFor(t, func() bool {
					receiver.mu.Lock()
					defer receiver.mu.Unlock()
					return receiver.batch != nil && receiver.batch.phase == couchmessage.BatchFrozen
				}) {
					t.Fatal("batch never froze")
				}
				f.proxy.dispatchPeer(&input)
			}
			want := workbenchshortcut.PasteStart + couchmessage.BatchEnvelope(messages) + workbenchshortcut.PasteEnd
			if input.String() != want || strings.Count(input.String(), "[remote operator@example.com 2001:db8::1]") != 1 {
				t.Fatalf("PTY provenance changed: %q want %q", input.String(), want)
			}
			clock.Add(int64(PeerSubmitDelay + time.Second))
			f.proxy.dispatchPeer(&input)
			f.proxy.turnActive.Store(true)
			f.proxy.dispatchPeer(&input)
			if input.String() != want+"\r" {
				t.Fatalf("duplicate or absent submit: %q", input.String())
			}
			for _, original := range messages {
				if !waitFor(t, func() bool {
					r, err := broker.Status(from, original.ID)
					return err == nil && r.Status == couchmessage.Submitted
				}) {
					t.Fatalf("broker confirmation absent %s", original.ID)
				}
				r, err := endpoint.Retained(lifetime, original.ID)
				if err != nil || !r.Message.SameProvenance(original) || r.Message.Body != original.Body {
					t.Fatalf("retained facts changed: %+v %v", r, err)
				}
			}
		})
	}
}

func TestPeerBatchRefusesRemoteProvenanceHeadMutation(t *testing.T) {
	for _, field := range []string{"edited", "source", "document"} {
		t.Run(field, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			d.binding = peerBatchTestBinding("ops:0")
			original := peerTestMessage(time.Now())
			original.From = peerBatchTestBinding("pair:6")
			original.To = d.binding
			original.Origin = couchmessage.OriginRemote
			original.RemoteSource = couchmessage.RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}
			original.RemoteEdited = true
			original.DocumentID = "0123456789abcdef0123456789abcdef"
			if err := d.reserve(original.ID, d.sequence); err != nil {
				t.Fatal(err)
			}
			if err := d.prepareBatch(original); err != nil {
				t.Fatal(err)
			}
			var input bytes.Buffer
			f.output(peerEmptyComposer())
			f.proxy.dispatchPeer(&input)
			changed := original
			switch field {
			case "edited":
				changed.RemoteEdited = false
			case "source":
				changed.RemoteSource.Email = "other@example.com"
			case "document":
				changed.DocumentID = "fedcba9876543210fedcba9876543210"
			}
			if err := d.finalizeBatch(original.ID, []couchmessage.Message{changed}); err == nil {
				t.Fatal("changed trusted head facts")
			}
			f.proxy.dispatchPeer(&input)
			if input.Len() != 0 {
				t.Fatalf("rejected metadata wrote %q", input.String())
			}
		})
	}
}
