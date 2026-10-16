package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestPeerBrokerReleaseDoesNotBlameOperator(t *testing.T) {
	for _, pasted := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-paste", true: "after-paste"}[pasted], func(t *testing.T) {
			now := time.Now()
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output(peerEmptyComposer())
			m := peerTestMessage(now)
			m.To = couchmessage.Binding{Slot: "pair:1", Repository: "/fixture/.git", Scope: "fixture", Tag: "receiver", Session: "fixture-session", Nonce: "launch", Agent: "claude", Version: "fixture", PID: 1, Start: "birth"}
			d := newPeerDelivery(m.To, func() time.Time { return now })
			f.proxy.peer = d
			if err := d.reserve(m.ID, 0); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if pasted {
				f.proxy.dispatchPeer(&out)
				if out.Len() == 0 {
					t.Fatal("no paste")
				}
			}
			before := out.String()
			raw, err := json.Marshal(couchmessage.EndpointRequest{Op: "release", Binding: m.To, ID: m.ID})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.handleEndpoint(context.Background(), raw); err != nil {
				t.Fatal(err)
			}
			f.proxy.dispatchPeer(&out)
			r := d.receipt()
			want := couchmessage.Cancelled
			if pasted {
				want = couchmessage.Indeterminate
			}
			if r.Status != want || !strings.Contains(r.Detail, "supervisor released delivery") || strings.Contains(r.Detail, "operator input") {
				t.Fatalf("release receipt: %+v", r)
			}
			if out.String() != before {
				t.Fatal("input after release")
			}
		})
	}
}
