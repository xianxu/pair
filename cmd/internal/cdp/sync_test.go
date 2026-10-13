package cdp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSyncEventsWaitsForConsumerAcknowledgment(t *testing.T) {
	c, p := newPair(t)
	reply := goCall(context.Background(), c, "Page.getFrameTree", nil)
	call := p.recv()
	p.send(`{"method":"Page.frameNavigated","params":{}}`)
	p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, call.ID))
	if r := await(t, reply); r.err != nil {
		t.Fatal(r.err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.SyncEvents(ctx) }()
	first := await(t, c.Events())
	if first.Method != "Page.frameNavigated" {
		t.Fatalf("first event: %+v", first)
	}
	fence := await(t, c.Events())
	if fence.Barrier == nil {
		t.Fatal("missing consumer barrier")
	}
	select {
	case err := <-done:
		t.Fatalf("returned before consumer completed: %v", err)
	default:
	}
	close(fence.Barrier)
	if err := await(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestSyncEventsCancellationAndClose(t *testing.T) {
	c, _ := newPair(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.SyncEvents(ctx); err == nil {
		t.Fatal("cancelled barrier succeeded")
	}
	c.Close()
	if err := c.SyncEvents(context.Background()); err == nil {
		t.Fatal("closed barrier succeeded")
	}
}
