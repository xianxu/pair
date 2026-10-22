package couchcmd

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// A live fleet may contain a wrapper that consumes its entire admission budget.
// Exercise the real transport: direct handler tests miss the shorter exchange.
func TestDrainOperationSlowFirstTargetDoesNotLoseExchange(t *testing.T) {
	r := newServiceRig(t)
	steward := r.connect(0)
	r.s.authority.hub = "/repo0"
	slow := &sendBudgetEndpoint{deadline: make(chan time.Time, 8)}
	var reserves, deliveries atomic.Int32
	for i := 1; i <= 29; i++ {
		b := r.world.add(i)
		var endpoint couchmessage.DeliveryEndpoint = countedFlagEndpoint{reserves: &reserves, deliveries: &deliveries}
		if i == 1 {
			endpoint = slow
		}
		r.world.mu.Lock()
		r.world.endpoints[b] = endpoint
		r.world.mu.Unlock()
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("slow-fanout-%d", i)))
		r.wrapper(b)
		r.waitConnected(b, true)
	}
	slow.armed.Store(true)
	request := flagRequest(steward, "start")
	var response couchmessage.Response
	started := time.Now()
	err := couchmessage.Call(context.Background(), r.brokerSock, request, &response)
	t.Logf("29-target exchange elapsed=%s error=%v operation=%+v healthy_reserves=%d healthy_deliveries=%d", time.Since(started), err, response.Operation, reserves.Load(), deliveries.Load())
	if err != nil {
		t.Fatalf("one slow first target lost the drain exchange and child receipt mapping: %v", err)
	}
	if response.Operation == nil || response.Operation.ID != request.ID {
		t.Fatalf("flags changed without a registered operation: %+v", response)
	}
}
