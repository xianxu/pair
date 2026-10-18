package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestPeerBatchFinalizationRejectsOriginMutation(t *testing.T) {
	f, d, messages, _ := peerBatchFixture(t)
	var out bytes.Buffer
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	messages[0].Origin = couchmessage.OriginSystem
	if err := d.finalizeBatch(messages[0].ID, messages); err == nil {
		t.Fatal("finalization relabeled prepared peer head as system")
	}
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("rejected finalization wrote %q", out.String())
	}
}

func TestPeerBatchPreservesMixedOriginsInInputAndLog(t *testing.T) {
	f, d, messages, now := peerBatchFixture(t)
	messages[1].Origin = couchmessage.OriginSystem
	want := couchmessage.BatchEnvelope(messages)
	var logged string
	d.recordBatch = func(got []couchmessage.Message) { logged = couchmessage.BatchEnvelope(got) }
	var out bytes.Buffer
	f.output(peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if err := d.finalizeBatch(messages[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), want) {
		t.Fatalf("input lost mixed envelope: %q", out.String())
	}
	*now = now.Add(PeerSubmitDelay + time.Second)
	f.proxy.dispatchPeer(&out)
	f.proxy.turnActive.Store(true)
	f.proxy.dispatchPeer(&out)
	if logged != want {
		t.Fatalf("log lost exact envelope: %q want %q", logged, want)
	}
}
