package diagnosticlog

import "io"

const (
	BatchMaxRecordBytes  = 64 << 10
	BatchFlushBytes      = 64 << 10
	BatchMaxPendingBytes = 256 << 10
	BatchMaxRecords      = 1024
)

type BatchEventKind uint8

const (
	BatchEnqueue BatchEventKind = iota
	BatchBegin
	BatchResult
	BatchClosing
	BatchClosed
)

// BatchState is the pure admission/progress authority. Pending includes the one
// in-flight batch, so a blocked sink cannot enlarge outstanding admission.
// Queue snapshots copy at most1024 slice headers; record payload is copied once.
// Treat returned states as owned values; transitions never mutate input slices.
type BatchState struct {
	queue, flight                             [][]byte
	PendingBytes, Records                     int
	AcceptedBytes, WrittenBytes, DroppedBytes int64
	phase                                     batchPhase
	LastError                                 error
}
type batchPhase uint8

const (
	batchOpen batchPhase = iota
	batchBlocked
	batchClosing
	batchClosed
)

func (s BatchState) isClosing() bool { return s.phase == batchClosing || s.phase == batchClosed }
func (s BatchState) isBlocked() bool { return s.phase == batchBlocked }

type BatchEvent struct {
	Kind  BatchEventKind
	Data  []byte
	N     int
	Err   error
	Retry bool
}
type BatchEffect struct {
	Data []byte
	Err  error
}

func BatchTransition(s BatchState, e BatchEvent) (BatchState, BatchEffect) {
	var out BatchEffect
	switch e.Kind {
	case BatchEnqueue:
		switch {
		case s.isClosing():
			out.Err = io.ErrClosedPipe
		case len(e.Data) > BatchMaxRecordBytes:
			out.Err = ErrLimit
		case len(e.Data) > BatchMaxPendingBytes-s.PendingBytes || (len(e.Data) > 0 && s.Records >= BatchMaxRecords):
			out.Err = ErrBusy
		}
		if out.Err != nil {
			s.DroppedBytes += int64(len(e.Data))
			break
		}
		if len(e.Data) == 0 {
			break
		}
		record := append([]byte(nil), e.Data...)
		s.queue = append(append([][]byte(nil), s.queue...), record)
		s.PendingBytes += len(record)
		s.Records++
		s.AcceptedBytes += int64(len(record))
	case BatchBegin:
		if s.phase == batchClosed || len(s.flight) > 0 || (s.isBlocked() && !e.Retry) {
			break
		}
		if s.phase == batchBlocked {
			s.phase = batchOpen
		}
		n, count := 0, 0
		for _, r := range s.queue {
			if n+len(r) > BatchFlushBytes {
				break
			}
			n += len(r)
			count++
		}
		if count == 0 {
			break
		}
		s.flight = append([][]byte(nil), s.queue[:count]...)
		s.queue = append([][]byte(nil), s.queue[count:]...)
		out.Data = make([]byte, 0, n)
		for _, r := range s.flight {
			out.Data = append(out.Data, r...)
		}
	case BatchResult:
		if len(s.flight) == 0 {
			break
		}
		total := 0
		for _, r := range s.flight {
			total += len(r)
		}
		n := max(0, min(e.N, total))
		if n != e.N {
			e.Err = io.ErrShortWrite
		}
		if n < total && e.Err == nil {
			e.Err = io.ErrShortWrite
		}
		s.WrittenBytes += int64(n)
		s.PendingBytes -= n
		left := n
		remaining := make([][]byte, 0, len(s.flight))
		for _, r := range s.flight {
			if left >= len(r) {
				left -= len(r)
				s.Records--
				continue
			}
			remaining = append(remaining, r[left:])
			left = 0
		}
		s.queue = append(remaining, s.queue...)
		s.flight = nil
		s.LastError = e.Err
		if !s.isClosing() {
			if e.Err != nil {
				s.phase = batchBlocked
			} else {
				s.phase = batchOpen
			}
		}
		out.Err = e.Err
	case BatchClosing:
		if s.phase != batchClosed {
			s.phase = batchClosing
		}
	case BatchClosed:
		s.DroppedBytes += int64(s.PendingBytes)
		s.PendingBytes = 0
		s.Records = 0
		s.queue = nil
		s.flight = nil
		s.phase = batchClosed
		if e.Err != nil {
			s.LastError = e.Err
		}
	}
	return s, out
}
