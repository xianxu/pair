package wrapcmd

import (
	"bytes"
	"context"
	"os"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// dimPendingMax bounds the unfinished-sequence tail remembered across reads.
// 64 KiB for ptychild's reason: an OSC 52 clipboard write is kilobytes and
// crosses reads, and a transition landing inside it would abort the copy.
// Past the bound the dimmer stops tracking (the bytes were already emitted).
const dimPendingMax = 64 * 1024

var (
	sgrFaint       = []byte("\x1b[2m")
	sgrNormalLevel = []byte("\x1b[22m")
)

// sgrDimmer dims the agent pane while the right pane is in focus mode (#417).
// Terminals have no "dim this region", so it re-asserts faint (SGR 2) after
// every SGR the agent emits: colors, bold and resets survive, and nothing has
// to parse parameters. Entering focus resizes the agent pane, and the agent's
// full redraw flows through here dimmed; leaving it, the redraw comes back
// plain. Only the user-visible stream passes through; the scrollback log and
// the terminal model keep the raw bytes.
type sgrDimmer struct {
	on bool
	// tail is an unfinished escape sequence that has already been emitted,
	// kept only so the next read can find where it ends.
	tail []byte
}

// Feed never delays a byte: the notification pump downstream owns holding
// output at sequence boundaries, and a second holder would hide a split
// sequence from it. Instead the dimmer remembers an unfinished sequence's
// already-emitted tail and frames it with the next read, so it can place a
// mode transition at the first sequence boundary rather than inside the
// agent's own CSI (which would end that CSI early and print its remainder as
// text). A sequence finishes under the mode it started in; the transition
// follows it.
func (d *sgrDimmer) Feed(data []byte, on bool) []byte {
	carried := len(d.tail) // bytes already emitted by the previous Feed
	buf := append(d.tail, data...)
	d.tail = nil
	var out []byte
	emit := func(from, to int) {
		if from < carried {
			from = carried
		}
		if from < to {
			out = append(out, buf[from:to]...)
		}
	}
	applied := false
	transition := func() {
		applied = true
		if on == d.on {
			return
		}
		if on {
			out = append(out, sgrFaint...)
		} else {
			out = append(out, sgrNormalLevel...)
		}
		d.on = on
	}
	i := 0
	for i < len(buf) {
		if !applied && i >= carried {
			transition()
		}
		if buf[i] != 0x1b {
			j := i + 1
			for j < len(buf) && buf[j] != 0x1b {
				j++
			}
			emit(i, j)
			i = j
			continue
		}
		n, status := dimFrame(buf[i:])
		switch status {
		case ansi.Incomplete:
			emit(i, len(buf))
			if len(buf)-i <= dimPendingMax {
				d.tail = append([]byte(nil), buf[i:]...)
			}
			return out
		case ansi.Complete:
			emit(i, i+n)
			if d.on && isSGR(buf[i:i+n]) {
				out = append(out, sgrFaint...)
			}
			i += n
		default:
			emit(i, i+1)
			i++
		}
	}
	if !applied {
		transition()
	}
	return out
}

// dimFrame is ansi.Frame with string sequences (OSC, DCS, SOS, PM, APC) held
// open until BEL or ST. ansi.Frame reads a bare `ESC ]` at a read boundary as
// a complete two-byte escape — right for the regex it mirrors, wrong here,
// where it would let a transition land inside the string.
func dimFrame(buf []byte) (int, ansi.Status) {
	if len(buf) >= 2 && buf[0] == 0x1b && bytes.IndexByte([]byte("]P_^X"), buf[1]) >= 0 {
		if n, ok := ansi.OSCEnd(buf, ansi.Strict); ok {
			return n, ansi.Complete
		}
		// Strict stops at a bare ESC. One at the very end may be ST's first
		// byte; anywhere else the string is malformed, so frame it as Frame does.
		if bare := bytes.IndexByte(buf[2:], 0x1b); bare < 0 || 2+bare == len(buf)-1 {
			return 0, ansi.Incomplete
		}
	}
	return ansi.Frame(buf)
}

// isSGR: CSI, parameter bytes only (digits, ';', ':'), final 'm'. A private
// marker ('<'..'?') makes it a different command — `CSI > 4 ; 2 m` is xterm's
// modifyOtherKeys — so it is left alone.
func isSGR(seq []byte) bool {
	if len(seq) < 3 || seq[1] != '[' || seq[len(seq)-1] != 'm' {
		return false
	}
	for _, c := range seq[2 : len(seq)-1] {
		if c < '0' || c > ';' {
			return false
		}
	}
	return true
}

// refreshDim re-reads focus mode and reports whether the dim flipped. An
// observation error keeps the last known state: a transient zellij failure
// should not flash the agent pane.
func (p *proxy) refreshDim() bool {
	if p.observeFocus == nil {
		return false
	}
	on, err := p.observeFocus()
	if err != nil {
		p.debug("DIM-observe-fail", err.Error())
		return false
	}
	return p.dim.Swap(on) != on
}

// focusObserveTimeout bounds the dim's zellij query. It runs on wrap's signal
// goroutine, which also delivers the capture and restart signals; a hung zellij
// must cost a cosmetic dim, never those.
const focusObserveTimeout = time.Second

// zellijFocusObserver asks zellij, which owns the mode, through the same
// predicate the cycle plans by (layoutcmd.FocusModeActive). Outside zellij
// there is no right pane, so nothing to observe.
func zellijFocusObserver() func() (bool, error) {
	if os.Getenv("ZELLIJ") == "" {
		return nil
	}
	return func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), focusObserveTimeout)
		defer cancel()
		raw, err := layoutcmd.OSRuntime{}.ListPanesJSONContext(ctx)
		if err != nil {
			return false, err
		}
		ids, _ := layoutcmd.OSRuntime{}.TerminalPaneIDs()
		return layoutcmd.FocusModeActive(zellijpane.Parse(raw), ids), nil
	}
}

// handleWinch reads the mode BEFORE the resize reaches the child, so the redraw
// it triggers already passes through the right dim. A flip without a size
// change (the layoutcmd nudge, or a split whose agent pane kept its size) gets
// an explicit redraw signal.
func (p *proxy) handleWinch() {
	changed := p.refreshDim()
	p.setWinsize()
	if changed {
		p.signalChildWinch()
	}
}

func (p *proxy) signalChildWinch() {
	if p.signalChild != nil {
		p.signalChild(syscall.SIGWINCH)
		return
	}
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}
	// The child leads its own process group; signal the group as the kernel
	// does on a real resize.
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGWINCH); err != nil {
		_ = p.cmd.Process.Signal(syscall.SIGWINCH)
	}
}
