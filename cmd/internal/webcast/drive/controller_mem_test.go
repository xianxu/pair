package drive

import (
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestMemControllerContract(t *testing.T) {
	RunControllerContract(t, func(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) Controller {
		return NewMemController(sink, inj, clock, grace)
	})
}

func TestMemControllerLabels(t *testing.T) {
	for in, want := range map[string]string{"  Ana ": "Ana", "": "Viewer 1", "\x07": "Viewer 1", "Bo\x1b[31m": "Bo[31m", "Ana\u202e": "Ana", "A\u2028B": "AB"} {
		if got := label(in, 1); got != want {
			t.Errorf("label(%q) = %q, want %q", in, got, want)
		}
	}
}

// gateSink refuses every record until open.
type gateSink struct {
	open bool
	got  []recording.Record
}

func (g *gateSink) Put(r recording.Record) error {
	if !g.open {
		return ErrNotLive
	}
	g.got = append(g.got, r)
	return nil
}
func (g *gateSink) Close() error { return nil }

// A change the sink refuses (the recorder hasn't opened) changes nothing:
// not the state, not the subscribers; the viewer is told the cast is paused.
func TestMemControllerSinkRefusalChangesNothing(t *testing.T) {
	sink := &gateSink{}
	c := NewMemController(sink, nil, func() int64 { return 1 }, time.Minute)
	defer c.Close()
	seat, err := c.Join("Ana")
	if err != nil {
		t.Fatal(err)
	}
	states, cancel := c.Subscribe()
	defer cancel()
	<-states
	if err := c.Request(seat.Key); !errors.Is(err, ErrNotLive) {
		t.Fatalf("request before open: %v, want ErrNotLive", err)
	}
	if code, say := Answer(ErrNotLive); code != 409 || say != sayNotLive {
		t.Fatalf("answer %d %q", code, say)
	}
	if got := c.Control(); got.Driver != nil || len(got.Pending) != 0 {
		t.Fatalf("control changed: %+v", got)
	}
	select {
	case s := <-states:
		t.Fatalf("subscribers told %+v", s)
	default:
	}
	sink.open = true
	if err := c.Request(seat.Key); err != nil {
		t.Fatalf("request after open: %v", err)
	}
	if len(sink.got) != 1 || len(c.Control().Pending) != 1 {
		t.Fatalf("after open: %d records, %+v", len(sink.got), c.Control())
	}
}
