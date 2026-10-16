package wrapcmd

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

const captureBufferBytes = 64 << 10

// captureBuffer keeps raw output and its byte-offset sidecar ordered while
// coalescing small writes. The pump owns the flush schedule and file lifetime;
// this buffer neither starts goroutines nor closes its underlying writers.
type captureBuffer struct {
	mu          sync.Mutex
	raw, events *bufio.Writer
	offset      int64
	err         error
	closed      bool
}

func newCaptureBuffer(raw, events io.Writer) *captureBuffer {
	return newCaptureBufferSize(raw, events, captureBufferBytes)
}

func newCaptureBufferSize(raw, events io.Writer, size int) *captureBuffer {
	return &captureBuffer{raw: bufio.NewWriterSize(raw, size), events: bufio.NewWriterSize(events, size)}
}

// WriteRaw counts bytes accepted by the buffer, including a partial write.
// A storage failure is sticky: later sidecar records must not advertise newer
// offsets when their corresponding raw bytes could not be published.
func (b *captureBuffer) WriteRaw(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, io.ErrClosedPipe
	}
	if b.err != nil {
		return 0, b.err
	}
	n, err := b.raw.Write(p)
	b.offset += int64(n)
	b.err = err
	return n, err
}

func (b *captureBuffer) Offset() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.offset
}

func (b *captureBuffer) WriteEvent(typ string, fields map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return io.ErrClosedPipe
	}
	if b.err != nil {
		return b.err
	}
	payload := map[string]any{"type": typ, "offset": b.offset}
	for k, v := range fields {
		payload[k] = v
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	// bufio can flush or bypass its sidecar buffer inside Write. First make
	// every raw byte covered by that record visible, including oversized events.
	if len(line) > b.events.Available() {
		if b.err = b.raw.Flush(); b.err != nil {
			return b.err
		}
	}
	_, b.err = b.events.Write(line)
	return b.err
}

func (b *captureBuffer) flushLocked() error {
	if b.err != nil {
		return b.err
	}
	if b.err = b.raw.Flush(); b.err != nil {
		return b.err
	}
	b.err = b.events.Flush()
	return b.err
}

func (b *captureBuffer) Flush() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked()
}

// Close drains accepted bytes and excludes subsequent admissions. Descriptors
// stay open so the wrapper can report flush errors before closing them.
func (b *captureBuffer) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		return b.flushLocked()
	}
	return b.err
}
