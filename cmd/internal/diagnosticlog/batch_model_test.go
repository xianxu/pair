package diagnosticlog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

func TestBatchTransitionBoundsAndConfirmedProgress(t *testing.T) {
	var s BatchState
	apply := func(e BatchEvent) BatchEffect { var out BatchEffect; s, out = BatchTransition(s, e); return out }
	for i := 0; i < BatchMaxRecords; i++ {
		if e := apply(BatchEvent{Kind: BatchEnqueue, Data: []byte("a")}); e.Err != nil {
			t.Fatal(e.Err)
		}
	}
	if e := apply(BatchEvent{Kind: BatchEnqueue, Data: []byte("b")}); !errors.Is(e.Err, ErrBusy) {
		t.Fatal(e.Err)
	}
	if e := apply(BatchEvent{Kind: BatchBegin}); len(e.Data) != BatchMaxRecords {
		t.Fatal(len(e.Data))
	}
	apply(BatchEvent{Kind: BatchResult, N: 1, Err: io.ErrUnexpectedEOF})
	if s.WrittenBytes != 1 || s.PendingBytes != BatchMaxRecords-1 {
		t.Fatalf("%+v", s)
	}
	if e := apply(BatchEvent{Kind: BatchBegin}); len(e.Data) != 0 {
		t.Fatal("immediate retry")
	}
	e := apply(BatchEvent{Kind: BatchBegin, Retry: true})
	if len(e.Data) != BatchMaxRecords-1 {
		t.Fatal(len(e.Data))
	}
	apply(BatchEvent{Kind: BatchResult, N: len(e.Data), Err: io.ErrUnexpectedEOF})
	if s.PendingBytes != 0 || s.WrittenBytes != BatchMaxRecords {
		t.Fatalf("%+v", s)
	}
	if e := apply(BatchEvent{Kind: BatchBegin, Retry: true}); len(e.Data) != 0 {
		t.Fatal("full-with-error replay")
	}
}

func TestBatchTransitionByteCapsAndClose(t *testing.T) {
	var s BatchState
	enqueue := func(n int) error {
		var out BatchEffect
		s, out = BatchTransition(s, BatchEvent{Kind: BatchEnqueue, Data: make([]byte, n)})
		return out.Err
	}
	if !errors.Is(enqueue(BatchMaxRecordBytes+1), ErrLimit) {
		t.Fatal("oversize accepted")
	}
	for i := 0; i < BatchMaxPendingBytes/BatchMaxRecordBytes; i++ {
		if err := enqueue(BatchMaxRecordBytes); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(enqueue(1), ErrBusy) {
		t.Fatal("overflow accepted")
	}
	var effect BatchEffect
	s, effect = BatchTransition(s, BatchEvent{Kind: BatchBegin})
	if len(effect.Data) != BatchMaxRecordBytes {
		t.Fatal(len(effect.Data))
	}
	if !errors.Is(enqueue(1), ErrBusy) {
		t.Fatal("in-flight bytes not counted")
	}
	s, _ = BatchTransition(s, BatchEvent{Kind: BatchClosing})
	if !errors.Is(enqueue(1), io.ErrClosedPipe) {
		t.Fatal("late admission")
	}
	s, _ = BatchTransition(s, BatchEvent{Kind: BatchResult, N: len(effect.Data)})
	s, _ = BatchTransition(s, BatchEvent{Kind: BatchClosed})
	if s.PendingBytes != 0 || s.Records != 0 || s.DroppedBytes != BatchMaxRecordBytes+1+1+1+1+3*BatchMaxRecordBytes {
		t.Fatalf("%+v", s)
	}
}

func TestBatchTransitionGeneratedFIFOAndBounds(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			var s BatchState
			var expected, confirmed []byte
			var flight []byte
			for step := 0; step < 2000; step++ {
				var event BatchEvent
				switch rng.Intn(4) {
				case 0, 1:
					data := bytes.Repeat([]byte{byte(step)}, 1+rng.Intn(BatchMaxRecordBytes+100))
					event = BatchEvent{Kind: BatchEnqueue, Data: data}
				case 2:
					event = BatchEvent{Kind: BatchBegin, Retry: true}
				case 3:
					if len(flight) == 0 {
						continue
					}
					n := rng.Intn(len(flight) + 1)
					event = BatchEvent{Kind: BatchResult, N: n, Err: io.ErrUnexpectedEOF}
				}
				next, effect := BatchTransition(s, event)
				switch event.Kind {
				case BatchEnqueue:
					if effect.Err == nil {
						expected = append(expected, event.Data...)
					}
				case BatchBegin:
					if len(effect.Data) > 0 {
						if !bytes.Equal(effect.Data, expected[len(confirmed):len(confirmed)+len(effect.Data)]) {
							t.Fatal("FIFO violated")
						}
						flight = effect.Data
					}
				case BatchResult:
					confirmed = append(confirmed, flight[:event.N]...)
					flight = nil
				}
				s = next
				if s.PendingBytes < 0 || s.PendingBytes > BatchMaxPendingBytes || s.Records < 0 || s.Records > BatchMaxRecords || int64(len(confirmed)) != s.WrittenBytes || int64(len(expected)) != s.AcceptedBytes || len(expected)-len(confirmed) != s.PendingBytes {
					t.Fatalf("step=%d state=%+v", step, s)
				}
			}
		})
	}
}

func TestBatchTransitionBatchesDoNotSplitRecords(t *testing.T) {
	var s BatchState
	for _, record := range []string{string(bytes.Repeat([]byte("a"), 40<<10)), string(bytes.Repeat([]byte("b"), 40<<10))} {
		s, _ = BatchTransition(s, BatchEvent{Kind: BatchEnqueue, Data: []byte(record)})
	}
	s, effect := BatchTransition(s, BatchEvent{Kind: BatchBegin})
	if len(effect.Data) != 40<<10 || effect.Data[len(effect.Data)-1] != 'a' {
		t.Fatal("record split", len(effect.Data))
	}
	s, _ = BatchTransition(s, BatchEvent{Kind: BatchResult, N: len(effect.Data)})
	_, effect = BatchTransition(s, BatchEvent{Kind: BatchBegin})
	if len(effect.Data) != 40<<10 || effect.Data[0] != 'b' {
		t.Fatal("record order", len(effect.Data))
	}
}
