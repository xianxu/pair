package couchmessage

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestBrokerWrapperReplyDecodeFailureNamesSlotAndNeverRecommits(t *testing.T) {
	for _, reply := range []string{`{"FutureReceiptField":true}`, `{"Error":42}`} {
		t.Run(reply, func(t *testing.T) {
			from, to := brokerBinding("ops:0"), brokerBinding("parley.nvim:4")
			var commits atomic.Int32
			socket := transportSocket(t)
			server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var r EndpointRequest
				if err := strictjson.Decode(raw, &r); err != nil {
					return nil, err
				}
				switch r.Op {
				case "observe":
					return json.Marshal(EndpointResponse{Observation: Observation{LastActivity: time.Now().Add(-time.Minute)}})
				case "commit":
					commits.Add(1)
					return []byte(reply), nil
				case "status":
					return []byte(reply), nil
				default:
					return json.Marshal(EndpointResponse{})
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			b := NewBroker(context.Background(), time.Now, nil)
			defer b.Close()
			if err := b.Register(from, newFakeEndpoint(time.Now().Add(-time.Minute))); err != nil {
				t.Fatal(err)
			}
			if err := b.Register(to, RemoteEndpoint{Binding: to, socket: socket}); err != nil {
				t.Fatal(err)
			}
			if _, err := b.Send(context.Background(), from, "decode-failure", Route{Target: to.Slot}, "hello"); err != nil {
				t.Fatal(err)
			}
			waitStatus(t, b, from, "decode-failure", Indeterminate)
			r, err := b.Status(from, "decode-failure")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(r.Detail, "wrapper parley.nvim:4") || !strings.Contains(r.Detail, "restart Couch") || strings.Contains(r.Detail, "operator input") {
				t.Fatalf("missing actionable protocol failure: %+v", r)
			}
			if strings.Contains(reply, "Future") && !strings.Contains(r.Detail, "newer than the Couch supervisor") {
				t.Fatal(r.Detail)
			}
			if _, err := b.StatusContext(context.Background(), from, "forgotten"); err == nil || !strings.Contains(err.Error(), "wrapper parley.nvim:4") || !strings.Contains(err.Error(), "restart Couch") {
				t.Fatalf("status scan lost protocol failure: %v", err)
			}
			if commits.Load() != 1 {
				t.Fatalf("commit attempts = %d", commits.Load())
			}
		})
	}
}
