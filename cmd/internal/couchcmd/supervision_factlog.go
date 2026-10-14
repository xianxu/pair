package couchcmd

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// The mutex orders offers and lifecycle transitions. Disk IO is confined to run.
// rows is never closed: closing admission cannot race a channel send.
type supervisionFactLog struct {
	mu           sync.Mutex
	state        factLogLifecycle
	seq, dropped uint64
	elapsed      int64
	rows         chan []byte
	cancel       context.CancelFunc
	done         chan struct{}
	now          func() time.Time
	start        time.Time
	generation   string
	key          [32]byte
	lastSnapshot time.Time
}

func newSupervisionFactLog(parent context.Context, now func() time.Time, open func(context.Context) (io.WriteCloser, error)) *supervisionFactLog {
	ctx, cancel := context.WithCancel(parent)
	l := &supervisionFactLog{state: factLogOpening, rows: make(chan []byte, 256), cancel: cancel, done: make(chan struct{}), now: now, start: now()}
	var generation [16]byte
	if _, err := rand.Read(generation[:]); err != nil {
		l.state = factLogUnavailable
		close(l.done)
		return l
	}
	l.generation = hex.EncodeToString(generation[:])
	if _, err := rand.Read(l.key[:]); err != nil {
		l.state = factLogUnavailable
		close(l.done)
		return l
	}
	go l.run(ctx, open)
	return l
}
func (l *supervisionFactLog) hash(kind, value string) string {
	h := hmac.New(sha256.New, l.key[:])
	h.Write([]byte(kind))
	h.Write([]byte{0})
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil)[:16])
}
func (l *supervisionFactLog) subject(b couchmessage.Binding) string { return l.hash("slot", b.Slot) }
func (l *supervisionFactLog) incarnation(b couchmessage.Binding) string {
	raw, _ := json.Marshal(b)
	return l.hash("incarnation", string(raw))
}
func (l *supervisionFactLog) offer(f supervisionFact) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.state == factLogClosed {
		return
	}
	l.seq++
	now := l.now()
	l.elapsed = max(l.elapsed, now.Sub(l.start).Milliseconds())
	f.Version, f.Generation, f.Seq, f.At, f.ElapsedMS, f.Dropped = 1, l.generation, l.seq, now.UTC(), l.elapsed, l.dropped
	if l.state == factLogUnavailable {
		l.dropped++
		return
	}
	row, err := encodeFact(f)
	if err != nil {
		l.dropped++
		return
	}
	select {
	case l.rows <- row:
	default:
		l.dropped++
	}
}
func (l *supervisionFactLog) run(ctx context.Context, open func(context.Context) (io.WriteCloser, error)) {
	defer close(l.done)
	sink, err := open(ctx)
	l.mu.Lock()
	if err != nil {
		l.state = l.state.advance(factLogOpenFailed)
	} else {
		l.state = l.state.advance(factLogOpened)
	}
	closed := l.state == factLogClosed
	l.mu.Unlock()
	if sink != nil {
		defer sink.Close()
	}
	if err != nil || closed {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case row := <-l.rows:
			if ctx.Err() != nil {
				return
			}
			n, err := sink.Write(row)
			if err != nil || n != len(row) {
				l.mu.Lock()
				l.dropped++
				l.state = l.state.advance(factLogWriteFailed)
				l.mu.Unlock()
			}
		}
	}
}
func (l *supervisionFactLog) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.state = l.state.advance(factLogClose)
	l.mu.Unlock()
	l.cancel()
	<-l.done
}
