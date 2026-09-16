package vt

import (
	"errors"
	"io"
	"unsafe"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Limits bounds retained state, independent of transport chunk boundaries.
// All fields must be positive. StringBytes excludes the overflow sentinel.
type Limits struct{ MaxCells, StringBytes, GraphemeBytes, HistoryLines, HistoryCells, HistoryBytes, KeyboardStack, MetadataBytes, LinkBytes int }

// DefaultLimits returns the production retention limits.
func DefaultLimits() Limits {
	return Limits{MaxCells: 262144, StringBytes: 65536, GraphemeBytes: 256, HistoryLines: 1000, HistoryCells: 65536, HistoryBytes: 4 << 20, KeyboardStack: 16, MetadataBytes: 4096, LinkBytes: 2048}
}
func (l Limits) valid() bool {
	return l.MaxCells > 0 && l.StringBytes > 0 && l.StringBytes < 1<<24 && l.GraphemeBytes > 0 && l.GraphemeBytes <= 65536 && l.HistoryLines > 0 && l.HistoryCells > 0 && l.HistoryBytes > 0 && l.KeyboardStack > 0 && l.KeyboardStack <= 1024 && l.MetadataBytes > 0 && l.LinkBytes > 0
}
func (l Limits) geometry(w, h int) bool { return w > 0 && h > 0 && w <= l.MaxCells/h }
func NewEmulatorWithLimits(w, h int, l Limits) (*Emulator, error) {
	if !l.valid() || !l.geometry(w, h) {
		return nil, errors.New("vt: invalid limits or geometry")
	}
	return newEmulator(w, h, l), nil
}

// ResizeChecked validates before mutating either screen.
func (e *Emulator) ResizeChecked(w, h int) error {
	if !e.limits.geometry(w, h) {
		return errors.New("vt: geometry exceeds cell limit")
	}
	e.resize(w, h)
	return nil
}

// SetReplyWriter replaces the legacy pipe destination. It must not reenter e.
// The caller owns the writer and its bounds; callbacks and writes are synchronous.
func (e *Emulator) SetReplyWriter(w io.Writer) {
	if w == nil {
		w = e.pw
	}
	e.replyWriter = w
}
func (e *Emulator) TakeReplyError() error             { err := e.replyError; e.replyError = nil; return err }
func (e *Emulator) Mode(m ansi.Mode) ansi.ModeSetting { return e.modes[m] }
func (e *Emulator) KeyboardFlags() uint32             { return e.keyboard[e.screenIndex()].flags }
func (e *Emulator) screenIndex() int {
	if e.scr == &e.scrs[1] {
		return 1
	}
	return 0
}

type replySink struct{ e *Emulator }

func (s replySink) Write(b []byte) (int, error) {
	if s.e.replyError != nil {
		return 0, s.e.replyError
	}
	n, err := s.e.replyWriter.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		s.e.replyError = err
	}
	return n, err
}
func (e *Emulator) replies() io.Writer { return replySink{e} }

// Usage is a conservative engineering estimate, excluding endpoint queues.
// It includes allocator allowances, not a mathematical heap upper bound across
// Go implementations. HistoryBytes remains the logical payload used by limits.
// String payloads are counted per reference even when Go shares their storage.
type Usage struct{ ScreenCells, HistoryCells, ParserBytes, GraphemeBytes, MetadataBytes, HistoryBytes, RetainedBytes int }

func (e *Emulator) Usage() Usage {
	u := Usage{ParserBytes: cap(e.parser.Data()), GraphemeBytes: len(e.cluster), MetadataBytes: len(e.title) + len(e.iconName) + len(e.cwd)}
	size := int(unsafe.Sizeof(uv.Cell{}))
	u.RetainedBytes = int(unsafe.Sizeof(*e)) + u.ParserBytes + u.GraphemeBytes + u.MetadataBytes
	// Fixed parser parameters, maps/handler closures and allocator overhead allowance.
	u.RetainedBytes += 16 * 1024
	for _, v := range []string{e.title, e.iconName, e.cwd, e.cluster} {
		u.RetainedBytes += stringAllowance(v)
	}
	for i := range e.clusterOld {
		u.RetainedBytes += cellPayload(&e.clusterOld[i]) + cellAllowance(&e.clusterOld[i])
	}
	for i := range e.scrs {
		s := &e.scrs[i]
		u.ScreenCells += s.Width() * s.Height()
		u.RetainedBytes += cap(s.rows) * int(unsafe.Sizeof(RowMetadata{}))
		for _, row := range s.rows {
			if row.clipped != nil {
				u.RetainedBytes += size + cellPayload(row.clipped) + cellAllowance(row.clipped)
			}
		}
		u.RetainedBytes += cap(s.buf.Lines)*int(unsafe.Sizeof(uv.Line{})) + cap(s.buf.Touched)*int(unsafe.Sizeof((*uv.LineData)(nil))) + len(s.buf.Touched)*int(unsafe.Sizeof(uv.LineData{}))
		for y := 0; y < s.Height(); y++ {
			for x := 0; x < s.Width(); x++ {
				if c := s.CellAt(x, y); c != nil {
					u.RetainedBytes += cellPayload(c) + cellAllowance(c)
				}
			}
		}
		for _, c := range []Cursor{s.cur, s.saved} {
			u.RetainedBytes += len(c.Link.URL) + len(c.Link.Params) + cellAllowance(&uv.Cell{Style: c.Pen, Link: c.Link})
		}
		if b := s.scrollback; b != nil {
			u.HistoryCells += b.cells
			u.RetainedBytes += cap(b.ids)*8 + cap(b.metadata)*int(unsafe.Sizeof(RowMetadata{}))
			u.HistoryBytes += b.bytes
			for _, line := range b.lines {
				for j := range line {
					u.RetainedBytes += cellAllowance(&line[j])
				}
			}
			u.RetainedBytes += cap(b.lines) * int(unsafe.Sizeof(uv.Line{}))
		}
		u.RetainedBytes += cap(e.keyboard[i].stack) * 4
	}
	u.RetainedBytes += u.ScreenCells*size + u.HistoryBytes
	return u
}
func cellPayload(c *uv.Cell) int { return len(c.Content) + len(c.Link.URL) + len(c.Link.Params) }

// Nonempty strings reserve 25% plus one minimum-size allocation; interface
// color values reserve one small box each. Shared payloads are deliberately
// counted per reference. Logical history limits use cellPayload, not this slack.
func stringAllowance(s string) int {
	if s == "" {
		return 0
	}
	return len(s)/4 + 16
}
func cellAllowance(c *uv.Cell) int {
	n := stringAllowance(c.Content) + stringAllowance(c.Link.URL) + stringAllowance(c.Link.Params)
	if c.Style.Fg != nil {
		n += 16
	}
	if c.Style.Bg != nil {
		n += 16
	}
	if c.Style.UnderlineColor != nil {
		n += 16
	}
	return n
}
