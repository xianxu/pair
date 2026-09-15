package terminalqualify

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	vt "github.com/charmbracelet/x/vt"
)

const maxCandidateCells = 262144
const maxCandidateReplies = 1024 * 1024

type replyResult struct {
	text string
	err  error
}

// Candidate is a disposable qualification instrument, not a production terminal.
// All emulator access belongs to Execute. Actions must finish synchronously and
// may only block on emulator IO, which cancellation can interrupt.
type Candidate struct {
	emulator  *vt.Emulator
	reader    io.Reader
	commands  chan struct{}
	stopping  chan struct{}
	drainDone chan struct{}
	replies   chan replyResult
	marker    []byte
	pipe      io.Closer
	stopOnce  sync.Once
	closeOnce sync.Once
	closeErr  error
	effects   Observation
}

func NewCandidate(width, height int) (*Candidate, error) {
	if width <= 0 || height <= 0 || width > maxCandidateCells/height {
		return nil, fmt.Errorf("candidate dimensions %dx%d exceed %d cells", width, height, maxCandidateCells)
	}
	e := vt.NewEmulator(width, height)
	return newCandidate(e, e, e.InputPipe().(io.Closer))
}

// newCandidate accepts a reply transport seam for controlled IO lifecycle tests.
func newCandidate(e *vt.Emulator, reader io.Reader, pipe io.Closer) (*Candidate, error) {
	marker := make([]byte, 32)
	if _, err := rand.Read(marker); err != nil {
		_ = pipe.Close()
		_ = e.Close()
		return nil, err
	}
	e.SetScrollbackSize(1000)
	c := &Candidate{emulator: e, reader: reader, commands: make(chan struct{}, 1), stopping: make(chan struct{}), drainDone: make(chan struct{}), replies: make(chan replyResult, 1), marker: marker, pipe: pipe, effects: Observation{"title": "", "cwd": "", "bells": "0"}}
	bells := 0
	e.SetCallbacks(vt.Callbacks{
		Title: func(s string) { c.effects["title"] = s }, WorkingDirectory: func(s string) { c.effects["cwd"] = s },
		Bell: func() { bells++; c.effects["bells"] = strconv.Itoa(bells) },
	})
	go c.drain()
	return c, nil
}

func (c *Candidate) drain() {
	defer close(c.drainDone)
	defer c.stop()
	buf := make([]byte, 4096)
	var collected []byte
	overflow := false
	for {
		n, err := c.reader.Read(buf)
		if n > 0 {
			// io.Pipe preserves each Write as one or more Reads. The marker is
			// shorter than buf, hence arrives in one Read after preceding replies.
			if bytes.Equal(buf[:n], c.marker) {
				result := replyResult{text: string(collected)}
				if overflow {
					result.err = fmt.Errorf("candidate reply evidence exceeds %d bytes", maxCandidateReplies)
				}
				c.replies <- result
				collected = nil
				overflow = false
			} else {
				room := maxCandidateReplies - len(collected)
				take := n
				if take > room {
					take = room
					overflow = true
				}
				collected = append(collected, buf[:take]...)
			}
		}
		if err != nil {
			return
		}
	}
}

func (c *Candidate) stop() { c.stopOnce.Do(func() { close(c.stopping); _ = c.pipe.Close() }) }

// Execute serializes feed, optional input action and a copied snapshot. A
// two-second deadline applies even when the caller supplies no deadline.
func (c *Candidate) Execute(ctx context.Context, chunks []string, action func(*vt.Emulator)) (Observation, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case c.commands <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.stopping:
		return nil, io.ErrClosedPipe
	}
	defer func() { <-c.commands }()
	select {
	case <-c.stopping:
		return nil, io.ErrClosedPipe
	default:
	}
	type outcome struct {
		observation Observation
		err         error
	}
	done := make(chan outcome, 1)
	go func() {
		for _, chunk := range chunks {
			if err := ctx.Err(); err != nil {
				done <- outcome{err: err}
				return
			}
			if _, err := c.emulator.WriteString(chunk); err != nil {
				done <- outcome{err: err}
				return
			}
		}
		if action != nil {
			action(c.emulator)
		}
		c.emulator.SendText(string(c.marker))
		select {
		case reply := <-c.replies:
			if reply.err != nil {
				done <- outcome{err: reply.err}
				return
			}
			observation, err := c.snapshot()
			if observation != nil {
				observation["replies"] = reply.text
			}
			done <- outcome{observation, err}
		case <-c.stopping:
			done <- outcome{err: io.ErrClosedPipe}
		}
	}()
	select {
	case result := <-done:
		return result.observation, result.err
	case <-ctx.Done():
		c.stop()
		<-done
		return nil, ctx.Err()
	case <-c.stopping:
		<-done
		return nil, io.ErrClosedPipe
	}
}

func (c *Candidate) snapshot() (Observation, error) {
	e := c.emulator
	w, h := e.Width(), e.Height()
	if w <= 0 || h <= 0 || w > maxCandidateCells/h {
		return nil, errors.New("candidate resized beyond cell bound")
	}
	p := e.Cursor()
	out := Observation{"cursor-visible": strconv.FormatBool(!p.Hidden), "cursor-style": fmt.Sprintf("%d,%t", p.Style, !p.Steady), "cursor": fmt.Sprintf("%d,%d", p.X, p.Y), "alt": strconv.FormatBool(e.IsAltScreen()), "width": strconv.Itoa(w), "height": strconv.Itoa(h), "history-lines": strconv.Itoa(e.ScrollbackLen())}
	for k, v := range c.effects {
		out[k] = v
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			suffix := fmt.Sprintf(":%d,%d", x, y)
			cell := e.CellAt(x, y)
			content, width, link, params, fg, bg := " ", 1, "", "", "", ""
			attrs, underline, underlineColor := "0", "0", ""
			if cell != nil {
				content = cell.Content
				if content == "" {
					content = " "
				}
				width = cell.Width
				link = cell.Link.URL
				params = cell.Link.Params
				fg = colorString(cell.Style.Fg)
				bg = colorString(cell.Style.Bg)
				attrs = strconv.Itoa(int(cell.Style.Attrs))
				underline = strconv.Itoa(int(cell.Style.Underline))
				underlineColor = colorString(cell.Style.UnderlineColor)
			}
			out["cell"+suffix] = content
			out["cell-width"+suffix] = strconv.Itoa(width)
			out["link"+suffix] = link
			out["link-params"+suffix] = params
			out["fg"+suffix] = fg
			out["bg"+suffix] = bg
			out["attrs"+suffix] = attrs
			out["underline"+suffix] = underline
			out["underline-color"+suffix] = underlineColor
		}
	}
	return out, nil
}

func colorString(c color.Color) string {
	if c == nil {
		return ""
	}
	switch c := c.(type) {
	case ansi.BasicColor:
		return fmt.Sprintf("ansi:%d", c)
	case ansi.IndexedColor:
		return fmt.Sprintf("ansi:%d", c)
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// Close interrupts pipe IO, joins both execution and drain, then closes emulator
// state. Calling Emulator.Close while Read/Write run would race its closed flag.
func (c *Candidate) Close() error {
	c.closeOnce.Do(func() {
		c.stop()
		c.commands <- struct{}{}
		defer func() { <-c.commands }()
		<-c.drainDone
		c.closeErr = c.emulator.Close()
	})
	return c.closeErr
}
