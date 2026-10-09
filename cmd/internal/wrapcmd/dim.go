package wrapcmd

import (
	"os"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// dimPendingMax bounds the CSI tail held across reads. An SGR is a few dozen
// bytes; anything longer is flushed raw rather than held — the dim is cosmetic
// and must never stall output.
const dimPendingMax = 256

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
	on      bool
	pending []byte
}

func (d *sgrDimmer) Feed(data []byte, on bool) []byte {
	var out []byte
	if on != d.on {
		if d.on {
			out = append(out, d.pending...)
			d.pending = nil
			out = append(out, sgrNormalLevel...)
		} else {
			out = append(out, sgrFaint...)
		}
		d.on = on
	}
	if !on {
		return append(out, data...)
	}
	buf := append(d.pending, data...)
	d.pending = nil
	for i := 0; i < len(buf); {
		if buf[i] != 0x1b {
			j := i + 1
			for j < len(buf) && buf[j] != 0x1b {
				j++
			}
			out = append(out, buf[i:j]...)
			i = j
			continue
		}
		n, status := ansi.Frame(buf[i:])
		switch status {
		case ansi.Incomplete:
			if len(buf)-i <= dimPendingMax {
				d.pending = append([]byte(nil), buf[i:]...)
			} else {
				out = append(out, buf[i:]...)
			}
			return out
		case ansi.Complete:
			out = append(out, buf[i:i+n]...)
			if isSGR(buf[i : i+n]) {
				out = append(out, sgrFaint...)
			}
			i += n
		default:
			out = append(out, buf[i])
			i++
		}
	}
	return out
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

// zellijFocusObserver asks zellij, which owns the mode, through the same
// predicate the cycle plans by (layoutcmd.FocusModeActive). Outside zellij
// there is no right pane, so nothing to observe.
func zellijFocusObserver() func() (bool, error) {
	if os.Getenv("ZELLIJ") == "" {
		return nil
	}
	return func() (bool, error) {
		raw, err := layoutcmd.OSRuntime{}.ListPanesJSON()
		if err != nil {
			return false, err
		}
		ids, _ := layoutcmd.OSRuntime{}.TerminalPaneIDs()
		return layoutcmd.FocusModeActive(zellijpane.Parse(raw), ids), nil
	}
}
