package wrapcmd

import (
	"errors"
	"time"
	"unicode/utf8"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

// outputBoundary observes framing only. Invalid UTF-8 makes injection unsafe
// permanently; original output is always preserved, including malformed bytes.
type outputBoundary struct {
	parser       *xansi.Parser
	runeBytes    []byte
	disabled     bool
	stringKind   byte
	escape       bool
	stringEscape bool
}

func (b *outputBoundary) init() {
	if b.parser == nil {
		b.parser = xansi.NewParser()
		b.parser.SetDataSize(1)
	}
}
func (b *outputBoundary) safe() bool {
	b.init()
	return !b.disabled && len(b.runeBytes) == 0 && b.stringKind == 0 && !b.escape && b.parser.State() == parser.GroundState
}
func (b *outputBoundary) advance(c byte) {
	b.init()
	if b.disabled {
		return
	}
	if b.stringKind != 0 {
		if c == 0x18 || c == 0x1a || (b.stringKind == ']' && c == 7) || (b.stringEscape && c == '\\') {
			b.stringKind = 0
			b.stringEscape = false
			b.parser.Reset()
			return
		}
		b.stringEscape = c == 0x1b
		return
	}
	if b.escape || b.parser.State() == parser.EscapeState {
		b.escape = false
		switch c {
		case ']', 'P', '_', '^', 'X':
			b.stringKind = c
			b.parser.Reset()
			return
		}
	}
	if c == 0x1b && len(b.runeBytes) == 0 {
		b.escape = true
	}

	if len(b.runeBytes) > 0 {
		if c&0xc0 != 0x80 {
			b.disabled = true
			b.runeBytes = nil
			return
		}
		b.runeBytes = append(b.runeBytes, c)
		if utf8.FullRune(b.runeBytes) {
			r, n := utf8.DecodeRune(b.runeBytes)
			if r == utf8.RuneError && n == 1 {
				b.disabled = true
			} else {
				for _, v := range b.runeBytes {
					b.parser.Advance(v)
				}
			}
			b.runeBytes = nil
		}
		return
	}
	if c >= 0x80 {
		if b.parser.State() != parser.GroundState {
			b.disabled = true
			return
		}
		if c < 0xc2 || c > 0xf4 {
			b.disabled = true
			return
		}
		b.runeBytes = append(b.runeBytes, c)
		return
	}
	b.parser.Advance(c)
}

type pendingNotification struct {
	wire     []byte
	expires  time.Time
	complete func()
}
type notificationReceipt struct {
	end      uint64
	complete func()
}

var errNotificationUnsafe = errors.New("notification insertion disabled: ambiguous output framing")

func (p *stdoutPump) notify(message string, now time.Time, complete func()) error {
	if p.failure != nil {
		return p.failure
	}
	if p.boundary.disabled {
		return errNotificationUnsafe
	}
	p.expire(now)
	if len(p.notifications)+len(p.receipts) >= 32 {
		return errors.New("notification insertion queue full")
	}
	p.notifications = append(p.notifications, pendingNotification{notifyosc.Encode(message), now.Add(2 * time.Second), complete})
	p.insertReady()
	return nil
}
func (p *stdoutPump) appendOutput(data []byte) {
	if len(data) == 0 {
		return
	}
	if p.observe != nil {
		p.observe(data)
	}
	p.batch.append(data)
	p.queued += uint64(len(data))
}
func (p *stdoutPump) insertReady() {
	if !p.boundary.safe() || p.failure != nil {
		return
	}
	for _, n := range p.notifications {
		p.appendOutput(n.wire)
		p.receipts = append(p.receipts, notificationReceipt{p.queued, n.complete})
	}
	p.notifications = nil
}
func (p *stdoutPump) expire(now time.Time) {
	kept := p.notifications[:0]
	for _, n := range p.notifications {
		if now.Before(n.expires) && !p.boundary.disabled {
			kept = append(kept, n)
		} else if p.drop != nil {
			p.drop("notification expired or framing invalid")
		}
	}
	clear(p.notifications[len(kept):])
	p.notifications = kept
}
func (p *stdoutPump) finish() {
	if len(p.notifications) > 0 && p.drop != nil {
		p.drop("notification dropped at incomplete EOF")
	}
	p.notifications = nil
}
