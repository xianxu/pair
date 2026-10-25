package couchcmd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// Capture the ordinary broker delivery seam while retaining the fixture's quiet
// composer observations. Both agent kinds still use real registry admission.
type noticeAgentEndpoint struct {
	hubQuietEndpoint
	mu       sync.Mutex
	messages []couchmessage.Message
}

func (e *noticeAgentEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.mu.Lock()
	e.messages = append(e.messages, m)
	e.mu.Unlock()
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

func (e *noticeAgentEndpoint) snapshot() []couchmessage.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]couchmessage.Message(nil), e.messages...)
}

// Replace one admitted wrapper through its actual disconnect/register lifecycle;
// no broker or workspace cache entries are written directly.
func noticeHubAgent(t *testing.T, r *serviceRig, old couchmessage.Binding, kind string) (couchmessage.Binding, *noticeAgentEndpoint) {
	t.Helper()
	r.disconnect(old)
	binding := old
	binding.Agent = kind
	endpoint := &noticeAgentEndpoint{hubQuietEndpoint: hubQuietEndpoint{at: time.Now().Add(-time.Hour)}}
	r.world.set(old, func(s *worldSlot) { s.binding = binding })
	r.world.mu.Lock()
	delete(r.world.endpoints, old)
	r.world.endpoints[binding] = endpoint
	r.world.mu.Unlock()
	r.attach(binding, "notice-agent-hub")
	r.wrapper(binding, couchmessage.Observation{LastActivity: endpoint.at, Sequence: 2})
	r.waitConnected(binding, true)
	r.waitWorkspace(binding)
	return binding, endpoint
}

func TestNoticeDeliveryAgentKinds(t *testing.T) {
	for _, kind := range []string{"claude", "codex"} {
		t.Run(kind, func(t *testing.T) {
			r, slots := hubMessageRig(t)
			hub, endpoint := noticeHubAgent(t, r, slots[1], kind)
			r.waitWorkspace(slots[2])
			console := testConsole()
			delivery := newNoticeDelivery(console)
			t.Cleanup(delivery.Close)
			console.SetNoticeRouter(delivery.Publish)
			delivery.Bind(r.s)
			const body = "worker priority diagnostic for its owning hub"
			console.Warn(couchtty.Notice{Class: couchcore.NoticeAgentPriorityWarning, Thread: bindingThread(slots[2]), Label: slots[2].Slot, Kind: "status", Body: body})
			var message couchmessage.Message
			noticeDeliveryEventually(t, func() bool {
				messages := endpoint.snapshot()
				if len(messages) != 1 {
					return false
				}
				message = messages[0]
				receipt, known, active := r.s.broker.ReceiptSnapshot(message.ID)
				return known && !active && receipt.Status == couchmessage.Submitted
			})
			// Join adapter workers before counting, so late duplicate admission cannot
			// hide behind the first endpoint callback.
			delivery.Close()
			if got := endpoint.snapshot(); len(got) != 1 {
				t.Fatalf("diagnostic delivery count=%d: %+v", len(got), got)
			}
			if message.ID == "" || message.Origin != couchmessage.OriginSystem || message.From != hub || message.To != hub || message.Body != body || message.To.Agent != kind {
				t.Fatalf("wrong canonical system delivery: %+v", message)
			}
			receipt, known, active := r.s.broker.ReceiptSnapshot(message.ID)
			if !known || active || receipt.Message != message || receipt.Status != couchmessage.Submitted {
				t.Fatalf("canonical receipt lost: known=%v active=%v receipt=%+v", known, active, receipt)
			}
		})
	}
}
