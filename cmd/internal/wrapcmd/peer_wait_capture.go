package wrapcmd

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

const peerWaitCaptureAfter = 15 * time.Second

// PeerWaitCaptureState belongs to one delivery head; only its transition mutates it.
type PeerWaitCaptureState struct {
	id       string
	since    time.Time
	waiting  bool
	captured bool
}

func advancePeerWaitCapture(s PeerWaitCaptureState, id string, waiting bool, now time.Time) (PeerWaitCaptureState, bool) {
	if s.id != id {
		s = PeerWaitCaptureState{id: id}
	}
	if id == "" || !waiting {
		s.waiting = false
		return s, false
	}
	if s.captured {
		return s, false
	}
	if !s.waiting || now.Before(s.since) {
		s.since, s.waiting = now, true
	}
	if now.Sub(s.since) < peerWaitCaptureAfter {
		return s, false
	}
	s.captured = true
	return s, true
}

// PeerWaitEvidence freezes the refusal, not a later tail query. Screen cells
// remain private to the local optional diagnostic owner, never message transport.
type PeerWaitEvidence struct {
	At               time.Time            `json:"at"`
	Binding          couchmessage.Binding `json:"binding"`
	ID               string               `json:"id"`
	Condition        string               `json:"condition"`
	WaitMillis       int64                `json:"wait_ms"`
	Status           couchmessage.Status  `json:"status"`
	Generation       uint64               `json:"terminal_generation"`
	RenderEpoch      uint64               `json:"render_epoch"`
	Sequence         uint64               `json:"sequence"`
	InputBuffered    bool                 `json:"input_buffered"`
	ForwardedInput   int                  `json:"forwarded_input"`
	PartialInput     int                  `json:"partial_input_bytes"`
	OutputPending    int                  `json:"output_pending"`
	TurnActive       bool                 `json:"turn_active"`
	Picker           bool                 `json:"picker"`
	Image            bool                 `json:"image"`
	AutomaticPhase   automaticInputPhase  `json:"automatic_phase"`
	ScreenAvailable  bool                 `json:"screen_available"`
	DecisionSnapshot bool                 `json:"decision_snapshot"`
	snapshot         terminalSnapshot
}

type peerWaitRow struct {
	Y    int    `json:"y"`
	ANSI string `json:"ansi"`
}

type peerWaitRecord struct {
	TS            time.Time        `json:"ts"`
	Agent         string           `json:"agent"`
	Label         string           `json:"label"`
	Component     string           `json:"component"`
	Seq           uint64           `json:"seq"`
	Evidence      PeerWaitEvidence `json:"evidence"`
	Width         int              `json:"width"`
	Height        int              `json:"height"`
	CursorX       int              `json:"cursor_x"`
	CursorY       int              `json:"cursor_y"`
	CursorVisible bool             `json:"cursor_visible"`
	Complete      bool             `json:"complete"`
	AltScreen     bool             `json:"alt_screen"`
	Truncated     bool             `json:"truncated"`
	Rows          []peerWaitRow    `json:"rows"`
}

// encodePeerWaitCapture bounds the whole JSONL record, including envelope and
// escaping. Rows are stored bottom-first to retain composer/footer on truncation.
func encodePeerWaitCapture(e PeerWaitEvidence, sequence uint64) []byte {
	s := e.snapshot
	r := peerWaitRecord{TS: e.At, Agent: e.Binding.Agent, Label: "PEER-wait", Component: "pair-wrap", Seq: sequence, Evidence: e,
		Width: s.Width, Height: s.Height, CursorX: s.Cursor.X, CursorY: s.Cursor.Y,
		CursorVisible: s.CursorVisible, Complete: s.Complete, AltScreen: s.AltScreen}
	// All dynamic metadata is bounded before marshalling, independently of screen.
	r.Evidence.Condition = boundedPeerWaitText(e.Condition)
	base, _ := json.Marshal(r)
	left := diagnosticlog.BatchMaxRecordBytes - len(base) - 128
	for y := s.Height - 1; y >= 0; y-- {
		row, complete := peerWaitScreenRow(s, y, left/6-64)
		if !complete {
			r.Truncated = true
			break
		}
		encoded, _ := json.Marshal(row)
		left -= len(encoded) + 1
		if left < 0 {
			r.Truncated = true
			break
		}
		r.Rows = append(r.Rows, row)
	}
	raw, err := json.Marshal(r)
	if err != nil || len(raw)+1 > diagnosticlog.BatchMaxRecordBytes {
		return nil // Invalid metadata cannot turn optional diagnostics into a large write.
	}
	return append(raw, '\n')
}

func boundedPeerWaitText(s string) string {
	if len(s) > 1024 {
		return strings.ToValidUTF8(s[:1024], "") + "<truncated>"
	}
	return s
}

func peerWaitScreenRow(s terminalSnapshot, y, limit int) (peerWaitRow, bool) {
	var b strings.Builder
	var style uv.Style
	for x := 0; x < s.Width; {
		c := s.CellAt(x, y)
		if c == nil {
			return peerWaitRow{}, false
		}
		prefix := ""
		if !style.Equal(&c.Style) {
			prefix = "\x1b[0m" + c.Style.String()
		}
		text := c.Content
		if text == "" {
			text = " "
		}
		if len(prefix)+len(text)+b.Len()+4 > limit {
			return peerWaitRow{}, false
		}
		b.WriteString(prefix)
		b.WriteString(text)
		style = c.Style
		x += max(c.Width, 1)
	}
	b.WriteString("\x1b[0m")
	return peerWaitRow{Y: y, ANSI: b.String()}, true
}

// selectPeerWaitCaptureLocked runs only on the input owner, under both locks.
// It neither serializes nor enqueues diagnostic bytes.
func (p *proxy) selectPeerWaitCaptureLocked(d *peerDelivery, snapshot *terminalSnapshot, generation uint64) *PeerWaitEvidence {
	id := d.current.Message.ID
	prePaste := d.current.Status == couchmessage.Queued || d.current.Status == couchmessage.Held
	if !prePaste || d.released || d.exited {
		id = ""
	}
	now := d.now()
	var capture bool
	d.waitCapture, capture = advancePeerWaitCapture(d.waitCapture, id, id != "" && d.current.Detail != "", now)
	if !capture || p.wrapEventsFD == nil {
		return nil
	}
	e := &PeerWaitEvidence{At: now, Binding: d.binding, ID: id, Condition: d.current.Detail,
		WaitMillis: now.Sub(d.waitCapture.since).Milliseconds(), Status: d.current.Status, Sequence: d.sequence,
		RenderEpoch: p.automaticRenderEpoch.Load(), InputBuffered: d.inputBuffered,
		ForwardedInput: d.replies.inFlight, PartialInput: len(d.replies.input), OutputPending: d.outputPending,
		TurnActive: p.turnActive.Load(), Picker: p.pickerActive.Load(), Image: d.image, AutomaticPhase: p.automaticInput.phase}
	if snapshot != nil {
		e.snapshot, e.Generation, e.ScreenAvailable, e.DecisionSnapshot = *snapshot, generation, true, true
	} else if p.terminal != nil {
		e.snapshot, e.Generation = p.terminal.SnapshotGeneration()
		e.ScreenAvailable = true
	}
	return e
}

func (p *proxy) emitPeerWaitCapture(e *PeerWaitEvidence) {
	if e == nil || p.wrapEventsFD == nil {
		return
	}
	p.traceMu.Lock()
	p.traceSeq++
	raw := encodePeerWaitCapture(*e, p.traceSeq)
	n, err := p.wrapEventsFD.Write(raw)
	p.traceMu.Unlock()
	if err != nil || n != len(raw) || len(raw) == 0 {
		p.debug("PEER-wait-capture-failed", io.ErrShortWrite.Error())
	}
}
