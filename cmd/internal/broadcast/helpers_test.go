package broadcast

import (
	"strings"
	"testing"

	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// textFrame builds a frame from body text (rows separated by \n) plus an
// optional styled last row, the way Couch composes a child and its chrome.
func textFrame(t *testing.T, cols, rows int, body, chrome string) terminal.Frame {
	t.Helper()
	bodyRows := rows
	if chrome != "" {
		bodyRows--
	}
	var cells []terminal.Cell
	if bodyRows > 0 {
		var err error
		if cells, err = terminal.StyledRows(strings.ReplaceAll(body, "\n", "\r\n"), cols, bodyRows); err != nil {
			t.Fatal(err)
		}
	}
	if chrome != "" {
		row, err := terminal.StyledRows(chrome, cols, 1)
		if err != nil {
			t.Fatal(err)
		}
		cells = append(cells, row...)
	}
	f, err := terminal.PanelFrame(terminal.Geometry{Cols: cols, Rows: rows}, cells, terminal.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// liveChrome is a tab bar as Couch draws it while broadcasting.
func liveChrome(rest string) string { return LiveSGR + LiveLabel + "\x1b[0m " + rest }

// screen interprets messages the way a viewer does: resize to each message's
// geometry, then write its bytes.
type screen struct{ e *vt.Emulator }

func (s *screen) apply(t *testing.T, msgs ...Message) {
	t.Helper()
	for _, m := range msgs {
		if s.e == nil {
			s.e = vt.NewEmulator(m.Cols, m.Rows)
		} else if s.e.Width() != m.Cols || s.e.Height() != m.Rows {
			s.e.Resize(m.Cols, m.Rows)
		}
		if _, err := s.e.Write(m.Data); err != nil {
			t.Fatal(err)
		}
	}
}

func (s *screen) text() string {
	if s.e == nil {
		return ""
	}
	return s.e.String()
}

// frameText is what a fresh viewer shows for f.
func frameText(t *testing.T, f terminal.Frame) string {
	t.Helper()
	var st Stream
	m, ok, err := st.Next(f)
	if err != nil || !ok {
		t.Fatalf("render %v %v", ok, err)
	}
	var s screen
	s.apply(t, m)
	return s.text()
}
