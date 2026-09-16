package ptychild

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPublicationBoundsIncludeInflightAndCloseJoins(t *testing.T) {
	p := newPublication()
	started := make(chan struct{})
	exited := make(chan struct{})
	sink := func(ctx context.Context, b OutputBatch) error {
		close(started)
		<-ctx.Done()
		close(exited)
		return ctx.Err()
	}
	if err := p.enqueue(OutputBatch{Raw: []byte("a")}, sink); err != nil {
		t.Fatal(err)
	}
	<-started
	for i := 1; i < publicationPackets; i++ {
		if err := p.enqueue(OutputBatch{Raw: []byte("b")}, sink); err != nil {
			t.Fatal(err)
		}
	}
	p.mu.Lock()
	if len(p.queue) != publicationPackets || p.bytes != publicationPackets {
		t.Fatalf("queue=%d bytes=%d", len(p.queue), p.bytes)
	}
	p.mu.Unlock()
	blocked := make(chan error, 1)
	go func() { blocked <- p.enqueue(OutputBatch{Raw: []byte("overflow")}, sink) }()
	p.close()
	select {
	case <-exited:
	default:
		t.Fatal("close did not join sink")
	}
	select {
	case err := <-blocked:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("enqueue stuck")
	}
}
func TestPublicationFailureIsSticky(t *testing.T) {
	p := newPublication()
	defer p.close()
	boom := errors.New("consumer failed")
	if err := p.enqueue(OutputBatch{Raw: []byte("x")}, func(context.Context, OutputBatch) error { return boom }); err != nil {
		t.Fatal(err)
	}
	if err := p.flush(context.Background()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if err := p.enqueue(OutputBatch{}, func(context.Context, OutputBatch) error { return nil }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
}
func TestChildExitWaitsForFinalPublication(t *testing.T) {
	c := NewFakeChild(nil)
	defer c.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	c.SetSink(func(ctx context.Context, b OutputBatch) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	c.Feed([]byte("final"))
	<-started
	c.Exit(0)
	if c.Done() {
		t.Fatal("exit overtook final output")
	}
	if !c.Endpoint().InputEnded() {
		t.Fatal("ended input still available")
	}
	close(release)
	if c.Wait() != 0 {
		t.Fatal("exit code")
	}
}

func TestConsumerFailureStopsSilentRealChild(t *testing.T) {
	boom := errors.New("UI publication failed")
	c, err := Start(Options{Argv: []string{"sh", "-c", "printf ready; exec sleep 60"}, Size: Size{Rows: 4, Cols: 20}, Sink: func(context.Context, OutputBatch) error { return boom }})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	select {
	case <-c.Exited():
	case <-time.After(time.Second):
		t.Fatal("consumer failure left silent child pump alive")
	}
	if err := c.FlushOutput(context.Background()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
}

func TestEarlyQueryProgressesWhileUIPublicationBlocked(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	c, err := Start(Options{Argv: []string{"sh", "-c", `stty raw -echo; printf '\033[6n'; dd bs=1 count=6 2>/dev/null | od -An -tx1; printf QUERY_DONE`}, Size: Size{Rows: 4, Cols: 20}, Sink: func(ctx context.Context, b OutputBatch) error {
		once.Do(func() { close(started) })
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	<-started
	waitFor(t, "query response before UI callback release", func() bool { return bytes.Contains(c.Snapshot(), []byte("QUERY_DONE")) })
	if !strings.Contains(strings.Join(strings.Fields(string(c.Snapshot())), " "), "1b 5b 31 3b 31 52") {
		t.Fatalf("wrong originating endpoint reply:%q", c.Snapshot())
	}
	close(release)
	if code := c.Wait(); code != 0 {
		t.Fatal(code)
	}
}
