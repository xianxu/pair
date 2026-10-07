package terminal

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestEndpointObservationsPreserveParserAndResizeOrder(t *testing.T) {
	var observations []Observation
	observe := func(o Observation) {
		o.Data = append([]byte(nil), o.Data...)
		observations = append(observations, o)
	}
	out := ttyio.NewFake()
	e, err := NewEndpoint("observed", Geometry{8, 4}, out, observe)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	now := time.Now()
	chunks := [][]byte{[]byte("\x1b[2;"), []byte("3H"), []byte("X\x1b[6n")}
	for _, chunk := range chunks {
		if _, err := e.Feed(chunk, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(out.Bytes()); got != "\x1b[2;4R" {
		t.Fatalf("query reply changed: %q", got)
	}
	frame, err := e.Snapshot(now)
	if err != nil || frame.Cells[10].Content != "X" {
		t.Fatalf("parser output changed: frame=%+v err=%v", frame, err)
	}
	failure := errors.New("ioctl failed")
	if err := e.Resize(Geometry{10, 5}, func(Geometry) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("failed resize: %v", err)
	}
	if len(observations) != 4 {
		t.Fatalf("failed resize was recorded as applied: %+v", observations)
	}
	acknowledged := false
	if err := e.Resize(Geometry{10, 5}, func(Geometry) error {
		if len(observations) != 4 {
			t.Fatal("resize observed before acknowledgment")
		}
		acknowledged = true
		return nil
	}); err != nil || !acknowledged {
		t.Fatalf("resize: %v", err)
	}
	if _, err := e.Feed([]byte("Y"), now); err != nil {
		t.Fatal(err)
	}
	e.EndInput()
	e.EndInput()
	e.Close()
	want := []Observation{{Kind: "endpoint-open", EndpointID: "observed", Geometry: Geometry{8, 4}}}
	for _, chunk := range chunks {
		want = append(want, Observation{Kind: "endpoint-feed", EndpointID: "observed", Geometry: Geometry{8, 4}, Data: chunk})
	}
	want = append(want,
		Observation{Kind: "endpoint-resize", EndpointID: "observed", Geometry: Geometry{10, 5}},
		Observation{Kind: "endpoint-feed", EndpointID: "observed", Geometry: Geometry{10, 5}, Data: []byte("Y")},
		Observation{Kind: "endpoint-end", EndpointID: "observed", Geometry: Geometry{10, 5}},
	)
	if !reflect.DeepEqual(observations, want) {
		t.Fatalf("observations\ngot  %+v\nwant %+v", observations, want)
	}
}

func TestEndpointObservationEndsOnCloseWithoutEOF(t *testing.T) {
	var kinds []string
	e, err := NewEndpoint("closed", Geometry{8, 4}, ttyio.NewFake(), func(o Observation) { kinds = append(kinds, o.Kind) })
	if err != nil {
		t.Fatal(err)
	}
	e.Close()
	e.Close()
	e.EndInput()
	if !reflect.DeepEqual(kinds, []string{"endpoint-open", "endpoint-end"}) {
		t.Fatalf("close observations: %v", kinds)
	}
}

func TestEndpointObservationRecordsAppliedResizeEvenWhenRepliesFail(t *testing.T) {
	var observations []Observation
	e, err := NewEndpoint("resize", Geometry{8, 4}, ttyio.NewFake(), func(o Observation) { observations = append(observations, o) })
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	replyFailure := errors.New("pending reply failed")
	err = e.Resize(Geometry{10, 5}, func(Geometry) error {
		// A reply failure discovered after the ioctl does not undo geometry.
		e.replies.failure = replyFailure
		return nil
	})
	if !errors.Is(err, replyFailure) {
		t.Fatalf("reply failure not reported: %v", err)
	}
	if len(observations) != 2 || observations[1].Kind != "endpoint-resize" || observations[1].Geometry != (Geometry{10, 5}) {
		t.Fatalf("applied geometry missing despite failed return: %+v", observations)
	}
}

func TestEndpointObservationRunsBeforeParsingUnderEndpointLock(t *testing.T) {
	var e *Endpoint
	var err error
	observed := false
	e, err = NewEndpoint("ordered", Geometry{8, 4}, ttyio.NewFake(), func(o Observation) {
		if o.Kind != "endpoint-feed" {
			return
		}
		observed = true
		if e.mu.TryLock() {
			e.mu.Unlock()
			t.Error("observer did not share the parser/resize lock")
		}
		if cell := e.backend.CellAt(0, 0); cell != nil && cell.Content == "X" {
			t.Error("bytes were parsed before observation")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if _, err := e.Feed([]byte("X"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if !observed {
		t.Fatal("missing pre-parser observation")
	}
}
