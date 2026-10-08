package couchtty

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// The console reports its broadcast phase and the session's description
// (pair#413): nothing while off, the state while starting, and mode and
// viewers once live.
func TestConsoleBroadcastStatusFollowsThePhase(t *testing.T) {
	c := &Console{}
	if _, running := c.BroadcastStatus(); running {
		t.Fatal("an idle console reported a broadcast")
	}
	c.bcast = broadcastState{phase: broadcastStarting}
	if status, running := c.BroadcastStatus(); !running || status.State != "starting" || status.Viewers != 0 {
		t.Fatalf("starting: %+v %v", status, running)
	}
	session, err := broadcast.Start(context.Background(), broadcast.Config{Tunnel: &broadcast.FakeTunnel{}, Ping: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Stop(nil); <-session.Done() })
	c.bcast = broadcastState{phase: broadcastLive, session: session}
	status, running := c.BroadcastStatus()
	if !running || status.State != "live" || status.Mode != broadcast.ModeTunnel || status.StartedAt.IsZero() {
		t.Fatalf("live: %+v %v", status, running)
	}
}
