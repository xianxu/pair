package wrapcmd

import (
	"bytes"
	"fmt"
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

func TestPeerConnectedWaitDoesNotAccumulatePickerHistory(t *testing.T) {
	f, d, now := peerHoldFixture(t)
	var out bytes.Buffer
	for i := 1; i <= 200; i++ {
		title := fmt.Sprintf("Do you want to run command %d?", i)
		peerRender(f, d, "\x1b[?2026h\x1b[2J\x1b[H\r\n\r\n"+title+"\r\n❯ 1. Yes\r\n  2. No\x1b[?2026l")
		f.proxy.pickerActive.Store(true)
		f.proxy.pickerEpisode.Store(uint64(i))
		*now = now.Add(time.Minute)
		f.proxy.dispatchPeer(&out)
		if r := d.receipt(); r.Status != couchmessage.Held || !strings.Contains(r.Detail, title) || out.Len() != 0 {
			t.Fatalf("picker %d lost its standing wait: %+v bytes=%q episodes=%d/%d tail=%v", i, r, out.String(), f.proxy.pickerEpisode.Load(), d.pickerTitleEpisode, f.proxy.agentTail(60).Lines)
		}
	}
	d.mu.Lock()
	count := len(d.waits)
	d.mu.Unlock()
	if count != 0 {
		t.Fatalf("unbounded pre-paste wait retained %d obsolete picker reasons", count)
	}
	f.proxy.pickerActive.Store(false)
	peerRender(f, d, peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Delivering || strings.Count(out.String(), peerEnvelope(r.Message)) != 1 {
		t.Fatalf("picker return did not paste once: %+v bytes=%q", r, out.String())
	}
}
