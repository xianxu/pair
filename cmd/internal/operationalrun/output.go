package operationalrun

import "sync"

const OutputLimit = 256 << 10
const outputMarker = "\n[... output truncated ...]\n"

// OutputBuffer keeps a bounded prefix and rolling suffix, accepting all writes.
// It is safe for stdout and stderr to share the same collector.
type OutputBuffer struct {
	mu         sync.Mutex
	head, tail []byte
	total      int64
}

func NewOutputBuffer() *OutputBuffer { return &OutputBuffer{} }
func (b *OutputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	b.total += int64(n)
	headCap := OutputLimit / 2
	take := min(len(p), headCap-len(b.head))
	b.head = append(b.head, p[:take]...)
	p = p[take:]
	tailCap := OutputLimit - headCap
	if len(p) >= tailCap {
		b.tail = append(b.tail[:0], p[len(p)-tailCap:]...)
	} else {
		over := len(b.tail) + len(p) - tailCap
		if over > 0 {
			copy(b.tail, b.tail[over:])
			b.tail = b.tail[:len(b.tail)-over]
		}
		b.tail = append(b.tail, p...)
	}
	return n, nil
}
func (b *OutputBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]byte(nil), b.head...)
	if b.total > OutputLimit {
		out = append(out, outputMarker...)
		return append(out, b.tail[len(outputMarker):]...)
	}
	return append(out, b.tail...)
}
func (b *OutputBuffer) TotalBytes() int64 { b.mu.Lock(); defer b.mu.Unlock(); return b.total }
