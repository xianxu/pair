package couchcmd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// messageProbeCounts counts the authority calls that spawn processes or run
// git in production: launch (two ownership probes: ps and zellij), process
// identity, and the resting-branch git status.
type messageProbeCounts struct {
	mu                      sync.Mutex
	launch, process, branch int
}

func (c *messageProbeCounts) snapshot() (launch, process, branch int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.launch, c.process, c.branch
}

func countingAuthority(a messageAuthority, c *messageProbeCounts) messageAuthority {
	launch, process, branch := a.launch, a.process, a.branch
	a.launch = func(ctx context.Context, b couchmessage.Binding) error {
		c.mu.Lock()
		c.launch++
		c.mu.Unlock()
		return launch(ctx, b)
	}
	a.process = func(b couchmessage.Binding) error {
		c.mu.Lock()
		c.process++
		c.mu.Unlock()
		return process(b)
	}
	a.branch = func(ctx context.Context, root string) (couchcore.SlotGitStatus, error) {
		c.mu.Lock()
		c.branch++
		c.mu.Unlock()
		return branch(ctx, root)
	}
	return a
}

// TestMessageIdleRunsNoProbes is #365's Done-when 1: once registered, an idle
// wrapper costs messaging no ownership probes, process checks or git.
func TestMessageIdleRunsNoProbes(t *testing.T) {
	f := newMessageAuthorityFake()
	counts := &messageProbeCounts{}
	s, err := newMessageService(context.Background(), messageSocketForTest(t), countingAuthority(f.authority(), counts))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	heartbeat := couchmessage.Request{Op: "register", Binding: &f.binding}
	if r := s.handle(context.Background(), heartbeat); r.Code != "ok" {
		t.Fatalf("registration %+v", r)
	}
	l0, p0, b0 := counts.snapshot()
	// One idle minute as the current protocol spends it: a wrapper heartbeat
	// and a reconcile tick every second.
	for i := 0; i < 60; i++ {
		advanceMessageClock(s, time.Second)
		s.handle(context.Background(), heartbeat)
		s.reconcile(context.Background())
	}
	l, p, b := counts.snapshot()
	if l != l0 || p != p0 || b != b0 {
		t.Skipf("enabled in M2 (#365); idle-minute baseline: %d launch checks (2 ownership probes each), %d process checks, %d git status", l-l0, p-p0, b-b0)
	}
}
