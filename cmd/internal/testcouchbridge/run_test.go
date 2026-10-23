package testcouchbridge

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type runOwner struct {
	mu         sync.Mutex
	calls      []string
	gate       chan struct{}
	entered    chan struct{}
	dead       chan struct{}
	cleanupErr error
}

func (o *runOwner) record(s string) { o.mu.Lock(); defer o.mu.Unlock(); o.calls = append(o.calls, s) }
func (o *runOwner) Prepare(ctx context.Context) error {
	o.record("prepare")
	if o.entered != nil {
		close(o.entered)
	}
	if o.gate != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-o.gate:
		}
	}
	return nil
}
func (o *runOwner) Start(context.Context) error { o.record("start"); return nil }
func (o *runOwner) Readiness(context.Context) (Readiness, error) {
	o.record("check")
	return goodReadiness(), nil
}
func (o *runOwner) Done() <-chan struct{}       { return o.dead }
func (o *runOwner) Close(context.Context) error { o.record("close"); return o.cleanupErr }
func (o *runOwner) AssertCleaned() error        { o.record("assert"); return nil }
func (o *runOwner) RetainedRoot() string        { return "/retained" }

type runPipes struct {
	stop      chan Event
	published chan []byte
	blocked   bool
	joined    chan struct{}
}

func (p *runPipes) WaitStop(ctx context.Context) (Event, error) {
	defer close(p.joined)
	select {
	case e := <-p.stop:
		return e, nil
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}
func (p *runPipes) Publish(ctx context.Context, b []byte) error {
	p.published <- b
	if p.blocked {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}
func (p *runPipes) Close() error { return nil }
func runFixtures() (*runOwner, *runPipes) {
	return &runOwner{dead: make(chan struct{})}, &runPipes{stop: make(chan Event, 1), published: make(chan []byte, 1), joined: make(chan struct{})}
}
func waitRun(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not join")
		return nil
	}
}
func TestRunBridgeReadinessThenEOFJoinsOwner(t *testing.T) {
	o, p := runFixtures()
	done := make(chan error, 1)
	go func() { done <- RunBridge(context.Background(), o, p) }()
	<-p.published
	p.stop <- Event{Kind: EOF}
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.joined:
	default:
		t.Fatal("input waiter not joined")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	want := []string{"prepare", "start", "check", "close", "assert"}
	if len(o.calls) != len(want) {
		t.Fatal(o.calls)
	}
	for i := range want {
		if o.calls[i] != want[i] {
			t.Fatal(o.calls)
		}
	}
}
func TestRunBridgeEarlyStopCancelsPreparation(t *testing.T) {
	o, p := runFixtures()
	o.gate = make(chan struct{})
	o.entered = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- RunBridge(context.Background(), o, p) }()
	<-o.entered
	p.stop <- Event{Kind: TERM}
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, c := range o.calls {
		if c == "start" || c == "check" {
			t.Fatal("late effect", o.calls)
		}
	}
	select {
	case <-p.published:
		t.Fatal("late readiness")
	default:
	}
}
func TestRunBridgeBlockedReadinessStopsWithoutRetry(t *testing.T) {
	o, p := runFixtures()
	p.blocked = true
	done := make(chan error, 1)
	go func() { done <- RunBridge(context.Background(), o, p) }()
	<-p.published
	p.stop <- Event{Kind: EOF}
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.published:
		t.Fatal("publication retried")
	default:
	}
}
func TestRunBridgeProducerDeathAndUncertainCleanupFail(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(map[bool]string{false: "producer", true: "cleanup"}[uncertain], func(t *testing.T) {
			o, p := runFixtures()
			if uncertain {
				o.cleanupErr = errors.New("unknown cleanup")
			}
			done := make(chan error, 1)
			go func() { done <- RunBridge(context.Background(), o, p) }()
			<-p.published
			if uncertain {
				p.stop <- Event{Kind: EOF}
			} else {
				close(o.dead)
			}
			if err := waitRun(t, done); err == nil {
				t.Fatal("failure became success")
			}
		})
	}
}

func TestRunBridgeCallerCancellationIsNotCleanEOF(t *testing.T) {
	o, p := runFixtures()
	o.gate = make(chan struct{})
	o.entered = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunBridge(ctx, o, p) }()
	<-o.entered
	cancel()
	if err := waitRun(t, done); err == nil {
		t.Fatal("caller cancellation reported clean")
	}
}

func TestRunBridgeStartupDeadlineCancelsBlockedPreparation(t *testing.T) {
	o, p := runFixtures()
	o.gate = make(chan struct{})
	o.entered = make(chan struct{})
	startup, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runBridge(context.Background(), startup, o, p, time.Second) }()
	<-o.entered
	cancel()
	if err := waitRun(t, done); err == nil {
		t.Fatal("startup deadline reported success")
	}
	select {
	case <-p.published:
		t.Fatal("readiness after expiry")
	default:
	}
}

func TestRunBridgeStopWinsCanceledStageRepeatedly(t *testing.T) {
	for i := 0; i < 100; i++ {
		o, p := runFixtures()
		o.gate = make(chan struct{})
		o.entered = make(chan struct{})
		done := make(chan error, 1)
		go func() { done <- RunBridge(context.Background(), o, p) }()
		<-o.entered
		p.stop <- Event{Kind: EOF}
		if err := waitRun(t, done); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}
