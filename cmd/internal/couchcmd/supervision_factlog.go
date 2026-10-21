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
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// The mutex orders offers and lifecycle transitions. Disk IO is confined to run.
// rows is never closed: closing admission cannot race a channel send.
type supervisionFactLog struct {
	mu                        sync.Mutex
	state                     factLogLifecycle
	seq, dropped              uint64
	elapsed                   int64
	rows                      chan []byte
	cancel                    context.CancelFunc
	done                      chan struct{}
	now                       func() time.Time
	start                     time.Time
	generation                string
	key                       [32]byte
	lastSnapshot              time.Time
	written                   uint64
	lastWriteAt, lastErrorAt  time.Time
	lastError, errorOperation string
}

func newSupervisionFactLog(parent context.Context, now func() time.Time, open func(context.Context) (io.WriteCloser, error)) *supervisionFactLog {
	ctx, cancel := context.WithCancel(parent)
	l := &supervisionFactLog{state: factLogOpening, rows: make(chan []byte, 256), cancel: cancel, done: make(chan struct{}), now: now, start: now()}
	var generation [16]byte
	if _, err := rand.Read(generation[:]); err != nil {
		l.state = factLogUnavailable
		l.recordError("entropy", err)
		close(l.done)
		return l
	}
	l.generation = hex.EncodeToString(generation[:])
	if _, err := rand.Read(l.key[:]); err != nil {
		l.state = factLogUnavailable
		l.recordError("entropy", err)
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
	defer func() {
		if ctx.Err() != nil {
			l.mu.Lock()
			l.state = l.state.advance(factLogClose)
			l.mu.Unlock()
		}
		close(l.done)
	}()
	sink, err := open(ctx)
	l.mu.Lock()
	if err != nil {
		l.state = l.state.advance(factLogOpenFailed)
		l.recordError("open", err)
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
			l.mu.Lock()
			if err != nil || n != len(row) {
				if err == nil {
					err = io.ErrShortWrite
				}
				l.dropped++
				l.state = l.state.advance(factLogWriteFailed)
				l.recordError("write", err)
			} else {
				l.written++
				l.lastWriteAt = l.now()
				l.state = l.state.advance(factLogWriteSucceeded)
			}
			l.mu.Unlock()
		}
	}
}

// recordError is called under mu, or before the owner goroutine starts.
// The sink supplies metadata errors only; never pass a fact row or body here.
func (l *supervisionFactLog) recordError(operation string, err error) {
	raw := err.Error()
	raw = raw[:min(len(raw), 512)]
	safe := rowtext.Sanitize(raw)
	safe = safe[:min(len(safe), 512)]
	for !utf8.ValidString(safe) {
		safe = safe[:len(safe)-1]
	}
	if safe == "" {
		safe = "error text unavailable"
	}
	l.lastError, l.errorOperation, l.lastErrorAt = safe, operation, l.now()
}

// health reads only owner memory, even while the optional sink is blocked.
func (l *supervisionFactLog) health() *couchcore.SupervisionLogHealth {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return &couchcore.SupervisionLogHealth{State: l.state.String(), Offered: l.seq,
		Written: l.written, Dropped: l.dropped, LastWriteAt: l.lastWriteAt,
		LastError: l.lastError, ErrorOperation: l.errorOperation, LastErrorAt: l.lastErrorAt}
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
