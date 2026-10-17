package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestPeerConnectedBusyReceiverWaitsAndPastesOnce(t *testing.T) {
	for _, blocked := range []string{"unknown", "draft", "picker"} {
		t.Run(blocked, func(t *testing.T) {
			f, d, now := peerHoldFixture(t)
			if blocked == "unknown" {
				f.output("\x1b[2Jthe receiver is running tools")
			}
			f.proxy.pickerActive.Store(blocked == "picker")
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			*now = now.Add(2 * time.Hour)
			f.proxy.dispatchPeer(&out)
			if r := d.receipt(); r.Status.Terminal() || out.Len() != 0 || !strings.Contains(r.Detail, map[string]string{"unknown": "recognized composer", "draft": "empty composer", "picker": "picker"}[blocked]) {
				t.Fatalf("connected wait lost mail or wrote input: %+v bytes=%q", r, out.String())
			}
			f.proxy.pickerActive.Store(false)
			f.output(peerEmptyComposer())
			f.proxy.dispatchPeer(&out)
			if r := d.receipt(); r.Status != couchmessage.Delivering || strings.Count(out.String(), peerEnvelope(r.Message)) != 1 {
				t.Fatalf("composer return did not paste once: %+v bytes=%q", r, out.String())
			}
			before := out.String()
			*now = now.Add(couchmessage.DeliveryTimeout)
			f.proxy.dispatchPeer(&out)
			if r := d.receipt(); r.Status != couchmessage.Indeterminate || out.String() != before {
				t.Fatalf("post-paste deadline/replay protection changed: %+v bytes=%q", r, out.String())
			}
		})
	}
}
