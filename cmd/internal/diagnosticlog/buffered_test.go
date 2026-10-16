package diagnosticlog

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type batchSink struct {
	mu       sync.Mutex
	data     []byte
	calls    int
	outcomes []struct {
		n   int
		err error
	}
	entered chan struct{}
	release chan struct{}
	closed  bool
}

func (s *batchSink) Write(p []byte) (int, error) {
	if s.entered != nil {
		s.entered <- struct{}{}
		<-s.release
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	n, err := len(p), error(nil)
	if len(s.outcomes) > 0 {
		o := s.outcomes[0]
		s.outcomes = s.outcomes[1:]
		n, err = o.n, o.err
	}
	s.data = append(s.data, p[:n]...)
	return n, err
}
func (s *batchSink) Close() error { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true; return nil }
func (s *batchSink) snapshot() (string, int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.data), s.calls, s.closed
}
func newTestBuffered(s *batchSink) (*BufferedWriter, chan time.Time, chan struct{}) {
	tick := make(chan time.Time)
	stopped := make(chan struct{})
	return newBufferedWriter(s, tick, func() { close(stopped) }), tick, stopped
}
func TestBufferedIdleCopiedAdmissionAndManualTick(t *testing.T) {
	s := &batchSink{}
	w, tick, stopped := newTestBuffered(s)
	tick <- time.Now()
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, calls, _ := s.snapshot(); calls != 0 {
		t.Fatal("idle IO", calls)
	}
	record := []byte("record\n")
	if n, e := w.Write(record); n != len(record) || e != nil {
		t.Fatal(n, e)
	}
	record[0] = 'X'
	tick <- time.Now()
	if e := w.Flush(); e != nil {
		t.Fatal(e)
	}
	if data, calls, _ := s.snapshot(); data != "record\n" || calls != 1 {
		t.Fatal(data, calls)
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	<-stopped
	if _, e := w.Write([]byte("late")); !errors.Is(e, io.ErrClosedPipe) {
		t.Fatal(e)
	}
	stats := w.Stats()
	if stats.AcceptedBytes != 7 || stats.WrittenBytes != 7 || stats.DroppedBytes != 4 {
		t.Fatalf("%+v", stats)
	}
}
func TestBufferedThresholdAndCloseDuringBlockedWrite(t *testing.T) {
	s := &batchSink{entered: make(chan struct{}, 2), release: make(chan struct{})}
	w, _, stopped := newTestBuffered(s)
	if _, e := w.Write(bytes.Repeat([]byte("a"), BatchFlushBytes)); e != nil {
		t.Fatal(e)
	}
	<-s.entered
	if _, e := w.Write([]byte("tail")); e != nil {
		t.Fatal(e)
	}
	closed := make(chan error, 1)
	go func() { closed <- w.Close() }()
	// The close signal follows the synchronous admission fence.
	<-w.closing
	if _, e := w.Write([]byte("late")); !errors.Is(e, io.ErrClosedPipe) {
		t.Fatal(e)
	}
	select {
	case <-closed:
		t.Fatal("Close returned before in-flight IO")
	default:
	}
	close(s.release)
	if e := <-closed; e != nil {
		t.Fatal(e)
	}
	<-stopped
	data, calls, isClosed := s.snapshot()
	if len(data) != BatchFlushBytes+4 || calls != 2 || !isClosed {
		t.Fatal(len(data), calls, isClosed)
	}
}
func TestBufferedPartialErrorAndFullErrorNeverReplay(t *testing.T) {
	boom := errors.New("sink fault")
	for _, n := range []int{2, 4} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			s := &batchSink{outcomes: []struct {
				n   int
				err error
			}{{n, boom}}}
			w, _, _ := newTestBuffered(s)
			w.Write([]byte("abcd"))
			if e := w.Flush(); !errors.Is(e, boom) {
				t.Fatal(e)
			}
			if e := w.Flush(); e != nil {
				t.Fatal(e)
			}
			if e := w.Close(); e != nil {
				t.Fatal(e)
			}
			data, calls, _ := s.snapshot()
			wantCalls := 2
			if n == 4 {
				wantCalls = 1
			}
			if data != "abcd" || calls != wantCalls {
				t.Fatal(data, calls)
			}
		})
	}
}
func TestBufferedFailedCloseAccountsUndrainedBytes(t *testing.T) {
	boom := errors.New("sink fault")
	s := &batchSink{outcomes: []struct {
		n   int
		err error
	}{{0, boom}}}
	w, _, stopped := newTestBuffered(s)
	w.Write([]byte("abcd"))
	if e := w.Close(); !errors.Is(e, boom) {
		t.Fatal(e)
	}
	<-stopped
	stats := w.Stats()
	if stats.AcceptedBytes != 4 || stats.DroppedBytes != 4 || stats.PendingBytes != 0 || !errors.Is(stats.LastError, boom) {
		t.Fatalf("%+v", stats)
	}
	if e := w.Close(); !errors.Is(e, boom) {
		t.Fatal(e)
	}
	var nilWriter *BufferedWriter
	if e := nilWriter.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestBufferedAdmissionDoesNotWriteSink(t *testing.T) {
	s := &batchSink{}
	w, _, _ := newTestBuffered(s)
	defer w.Close()
	if n, e := w.Write([]byte("deferred")); n != 8 || e != nil {
		t.Fatal(n, e)
	}
	if data, calls, _ := s.snapshot(); calls != 0 || data != "" {
		t.Fatalf("admission performed IO: calls=%d data=%q", calls, data)
	}
}

func TestBufferedBlockedSinkAdmissionBounds(t *testing.T) {
	for _, limit := range []string{"bytes", "records"} {
		t.Run(limit, func(t *testing.T) {
			s := &batchSink{entered: make(chan struct{}, 8), release: make(chan struct{})}
			w, _, _ := newTestBuffered(s)
			if _, err := w.Write(bytes.Repeat([]byte("a"), BatchMaxRecordBytes)); err != nil {
				t.Fatal(err)
			}
			<-s.entered
			size, count := BatchMaxRecordBytes, 3
			if limit == "records" {
				size, count = 1, BatchMaxRecords-1
			}
			for i := 0; i < count; i++ {
				if _, err := w.Write(bytes.Repeat([]byte("b"), size)); err != nil {
					t.Fatal(i, err)
				}
			}
			if n, err := w.Write([]byte("overflow")); n != 0 || !errors.Is(err, ErrBusy) {
				t.Fatal(n, err)
			}
			if n, err := w.Write(make([]byte, BatchMaxRecordBytes+1)); n != 0 || !errors.Is(err, ErrLimit) {
				t.Fatal(n, err)
			}
			close(s.release)
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
