package terminal

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestInputWriterOrdersWholePacketsAndPartialWrites(t *testing.T) {
	fake := ttyio.NewFake()
	hold := make(chan struct{})
	fake.Enqueue(ttyio.WriteStep{Block: hold, Limit: 2})
	w := NewInputWriter(fake, 8, 128)
	defer w.Close()
	if err := w.Enqueue([]byte("operator")); err != nil {
		t.Fatal(err)
	}
	<-fake.Started()
	for _, packet := range []string{"\x1b[1;1R", "\x1b[200~paste\x1b[201~"} {
		if err := w.Enqueue([]byte(packet)); err != nil {
			t.Fatal(err)
		}
	}
	close(hold)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := w.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if got := string(fake.Bytes()); got != "operator\x1b[1;1R\x1b[200~paste\x1b[201~" {
		t.Fatalf("interleaved %q", got)
	}
}
func TestInputWriterFailureKeepsAcceptedPrefixAndStops(t *testing.T) {
	fake := ttyio.NewFake()
	fake.Enqueue(ttyio.WriteStep{Limit: 2, Err: io.ErrUnexpectedEOF})
	w := NewInputWriter(fake, 8, 128)
	defer w.Close()
	if err := w.Enqueue([]byte("abcd")); err != nil {
		t.Fatal(err)
	}
	err := w.Flush(context.Background())
	var failure *WriteFailure
	if !errors.As(err, &failure) || failure.Accepted != 2 || failure.Total != 4 || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("%+v", err)
	}
	if e := w.Enqueue([]byte("later")); e == nil {
		t.Fatal("admitted after failure")
	}
	if string(fake.Bytes()) != "ab" {
		t.Fatalf("replayed/continued: %q", fake.Bytes())
	}
}
func TestInputWriterBoundsIncludeInflightAndCloseJoins(t *testing.T) {
	fake := ttyio.NewFake()
	hold := make(chan struct{})
	fake.Enqueue(ttyio.WriteStep{Block: hold})
	w := NewInputWriter(fake, 2, 6)
	if err := w.Enqueue([]byte("four")); err != nil {
		t.Fatal(err)
	}
	<-fake.Started()
	if err := w.Enqueue([]byte("two!")); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("%v", err)
	}
	if err := w.Enqueue([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if err := w.Enqueue([]byte("x")); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("%v", err)
	}
	done := make(chan struct{})
	go func() { w.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked writer not joined")
	}
	if len(fake.Bytes()) != 0 {
		t.Fatal("cancelled packet written")
	}
	if err := w.Enqueue([]byte("x")); err == nil {
		t.Fatal("write after close")
	}
}
func TestInputWriterZeroProgressFailsWithoutRetry(t *testing.T) {
	fake := ttyio.NewFake()
	fake.Enqueue(ttyio.WriteStep{ZeroProgress: true})
	w := NewInputWriter(fake, 1, 64)
	defer w.Close()
	if err := w.Enqueue([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(context.Background()); !errors.Is(err, io.ErrNoProgress) {
		t.Fatalf("zero progress error: %v", err)
	}
	if fake.Calls() != 1 || len(fake.Bytes()) != 0 {
		t.Fatal("retried unknown progress")
	}
}
