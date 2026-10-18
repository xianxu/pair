package wrapcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func peerBatchFixture(t *testing.T) (*harnessSessionFake, *peerDelivery, []couchmessage.Message, *time.Time) {
	t.Helper()
	f, d := peerIntegrationFixture(t)
	d.binding = peerBatchTestBinding("pair:1")
	now := time.Now()
	d.now = func() time.Time { return now }
	messages := make([]couchmessage.Message, 5)
	for i := range messages {
		messages[i] = peerTestMessage(now)
		messages[i].From = peerBatchTestBinding("brain:0")
		messages[i].To = d.binding
		messages[i].ID = fmt.Sprintf("batch-%d", i)
		messages[i].Body = fmt.Sprintf("report %d", i)
	}
	if err := d.reserve(messages[0].ID, d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.prepareBatch(messages[0]); err != nil {
		t.Fatal(err)
	}
	return f, d, messages, &now
}

func TestPeerBatchRefusesUnqualifiedCodexAndQoderProfiles(t *testing.T) {
	for _, agent := range []string{"codex", "qoder"} {
		m := peerTestMessage(time.Now())
		m.To = peerBatchTestBinding("ops:0")
		m.To.Agent = agent
		d := newPeerDelivery(m.To, time.Now)
		if err := d.reserve(m.ID, d.sequence); err != nil {
			t.Fatal(err)
		}
		if err := d.prepareBatch(m); err == nil {
			t.Fatalf("accepted unqualified %s batch", agent)
		}
		if d.current.Message.ID != "" || d.batch != nil {
			t.Fatal("profile refusal mutated delivery ownership")
		}
	}
}
func TestPeerBatchOnePasteOneConfirmationAndRetention(t *testing.T) {
	f, d, messages, now := peerBatchFixture(t)
	recorded := 0
	d.recordBatch = func(got []couchmessage.Message) {
		recorded++
		if len(got) != 5 {
			t.Errorf("logged %d members", len(got))
		}
	}
	f.output(peerFilledComposer("operator draft"))
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if d.batchReady() || out.Len() != 0 {
		t.Fatal("busy composer offered preparation or pasted")
	}
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if !d.batchReady() || out.Len() != 0 {
		t.Fatal("ready preparation pasted before frozen membership")
	}
	if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	// The caller cannot mutate accepted membership after finalization.
	messages[1].Body = "changed by caller"
	f.proxy.dispatchPeer(&out)
	expected := make([]string, len(messages))
	for i, m := range messages {
		expected[i] = fmt.Sprintf("[Couch peer from %s; delivery %s]\nreport %d", m.From.Slot, m.ID, i)
	}
	want := workbenchshortcut.PasteStart + strings.Join(expected, "\n\n---\n\n") + workbenchshortcut.PasteEnd
	if out.String() != want {
		t.Fatalf("paste=%q want %q", out.String(), want)
	}
	for _, m := range messages {
		r, ok := d.retained(m.ID)
		if !ok || r.Status != couchmessage.Delivering {
			t.Fatalf("member %s: %+v %v", m.ID, r, ok)
		}
	}
	*now = now.Add(PeerSubmitDelay + time.Second)
	f.proxy.dispatchPeer(&out)
	if out.String() != want+"\r" {
		t.Fatalf("not one submit: %q", out.String())
	}
	f.proxy.turnActive.Store(true)
	f.proxy.dispatchPeer(&out)
	for _, m := range messages {
		r, ok := d.retained(m.ID)
		if !ok || r.Status != couchmessage.Submitted || r.Message.ID != m.ID {
			t.Fatalf("member %s: %+v %v", m.ID, r, ok)
		}
	}
	f.proxy.dispatchPeer(&out)
	if recorded != 1 {
		t.Fatalf("recorded %d combined turns", recorded)
	}
	// Move current to a new delivery; every group member remains individually recoverable.
	d.reservation = ""
	next := peerTestMessage(*now)
	next.ID = "next"
	next.To = d.binding
	next.Deadline = now.Add(time.Second)
	if err := d.reserve(next.ID, d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(next); err != nil {
		t.Fatal(err)
	}
	for _, m := range messages {
		r, ok := d.retained(m.ID)
		if !ok || r.Status != couchmessage.Submitted {
			t.Fatalf("forgot member %s", m.ID)
		}
	}
}
func TestPeerBatchFinalizationRechecksOperatorGuards(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !d.batchReady() {
		t.Fatal("not ready")
	}
	f.output(peerFilledComposer("new human draft"))
	f.proxy.dispatchPeer(&out)
	if !d.batchReady() {
		t.Fatal("assembly permission must remain sticky")
	}
	if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("overwrote draft: %q", out.String())
	}
	if err := d.finalizeBatch(messages[0].ID, messages); err == nil {
		t.Fatal("accepted second finalization")
	}
}
func TestPeerBatchFailedPasteAndCancellationNeverSubmitMembers(t *testing.T) {
	for _, failure := range []string{"paste", "submit", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			f, d, messages, now := peerBatchFixture(t)
			d.recordBatch = func([]couchmessage.Message) { t.Error("logged failed batch") }
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "paste":
				f.proxy.dispatchPeer(shortPeerBatchWriter{})
			case "submit":
				f.proxy.dispatchPeer(&out)
				*now = now.Add(PeerSubmitDelay + time.Second)
				f.proxy.dispatchPeer(shortPeerBatchWriter{})
			case "cancel":
				d.released = true
				f.proxy.dispatchPeer(&out)
			}
			for _, m := range messages {
				r, ok := d.retained(m.ID)
				if !ok || !r.Status.Terminal() || r.Status == couchmessage.Submitted {
					t.Fatalf("member %s: %+v %v", m.ID, r, ok)
				}
			}
		})
	}
}

type shortPeerBatchWriter struct{}

func (shortPeerBatchWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }

func TestPeerBatchRejectsChangedHeadAndRetainedFollowerBeforePaste(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	changed := append([]couchmessage.Message(nil), messages...)
	changed[0].Body = "forged head"
	if err := d.finalizeBatch(messages[0].ID, changed); err == nil {
		t.Fatal("accepted changed head")
	}
	d.mu.Lock()
	d.recent.Add(couchmessage.Receipt{Message: messages[2], Status: couchmessage.Submitted})
	d.mu.Unlock()
	if err := d.finalizeBatch(messages[0].ID, messages); err == nil {
		t.Fatal("accepted retained follower")
	}
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("rejected finalization wrote %q", out.String())
	}
	retained, ok := d.retained(messages[2].ID)
	if !ok || retained.Status != couchmessage.Submitted {
		t.Fatal("overwrote retained evidence")
	}
}

func peerBatchTestBinding(slot string) couchmessage.Binding {
	return couchmessage.Binding{Slot: slot, Repository: "/isolated", Scope: "test", Tag: "test", Session: "test", Nonce: "test", Agent: "claude", Version: "test", PID: 1, Start: "test"}
}

func TestPeerBatchPreparedHeadSurvivesInitialExchangeDeadline(t *testing.T) {
	f, d, messages, now := peerBatchFixture(t)
	*now = now.Add(2 * couchmessage.PasteTimeout)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !d.batchReady() || d.receipt().Status.Terminal() {
		t.Fatal("prepared connected head expired")
	}
	if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	f.proxy.dispatchPeer(&out)
	if out.Len() == 0 {
		t.Fatal("expired initial exchange revived during finalization")
	}
}

func TestPeerBatchHeadTimeEqualitySurvivesWireEncoding(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	raw, err := json.Marshal(messages)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []couchmessage.Message
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	// Native monotonic metadata and location pointers are not wire identity.
	decoded[0].Deadline = decoded[0].Deadline.In(time.FixedZone("same instant", 3600))
	if err := d.finalizeBatch(decoded[0].ID, decoded); err != nil {
		t.Fatal(err)
	}
}
