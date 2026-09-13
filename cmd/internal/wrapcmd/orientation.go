package wrapcmd

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// orientationDelivery transfers terminal observations to the existing stdin
// owner. Only that owner advances state or writes automatic input. Overlay and
// child-exit observations are sticky, so coalescing wakeups cannot lose them.
type orientationDelivery struct {
	// Observes timer arbitration in deterministic input-scheduler tests.
	settleReadyHook            func()
	request                    orientation.Request
	profile                    harnessTTYProfile
	mu                         sync.Mutex
	operator                   bool
	codexStartupPending        bool
	agyModelFooter             string
	replies                    orientationReplies
	ready, overlay, exited     bool
	wake                       chan struct{}
	finalized                  chan struct{}
	state                      orientation.DeliveryState
	settleAfter, deadlineAfter time.Duration
	publish                    func(orientation.DeliveryState)
}

func newOrientationDelivery(request orientation.Request) *orientationDelivery {
	profile, _ := profileForHarness(request.Agent, true)
	return &orientationDelivery{request: request, profile: profile, wake: make(chan struct{}, 1), finalized: make(chan struct{}), settleAfter: 100 * time.Millisecond, deadlineAfter: 30 * time.Second}
}
func consumeOrientation(env []string, agent string) (*orientation.Request, []string, error) {
	clean := withoutOrientation(env)
	raw := envValue(env, orientation.Env)
	if raw == "" {
		return nil, clean, nil
	}
	var request orientation.Request
	if err := strictjson.Decode([]byte(raw), &request); err != nil {
		return nil, clean, fmt.Errorf("orientation: %w", err)
	}
	if !request.Matches(envValue(env, "PAIR_TAG"), agent, envValue(env, "PAIR_LAUNCH_NONCE")) {
		return nil, clean, fmt.Errorf("orientation: request does not match this launch")
	}
	if _, ok := harnessTTYProfiles[agent]; !ok {
		return nil, clean, fmt.Errorf("orientation: unsupported harness")
	}
	return &request, clean, nil
}
func withoutOrientation(env []string) []string {
	clean := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, orientation.Env+"=") {
			clean = append(clean, entry)
		}
	}
	return clean
}
func (d *orientationDelivery) observe(ready, overlay, exited bool) {
	d.mu.Lock()
	d.ready = ready
	d.overlay = d.overlay || overlay
	d.exited = d.exited || exited
	d.mu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (p *proxy) observeOrientationTerminal() {
	d := p.orientation
	if d == nil {
		return
	}
	d.mu.Lock()
	d.ready = false
	if p.terminal != nil {
		d.ready = p.orientationComposerActive(p.terminal.Snapshot())
	}
	d.overlay = d.overlay || p.pickerActive.Load()
	d.mu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (d *orientationDelivery) admitOperatorInput(data []byte) {
	d.mu.Lock()
	d.replies.inFlight++
	d.operator = d.operator || d.replies.operatorData(data)
	d.mu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// Observation admission and each automatic write share this lock. This makes
// the boundary exact: input/overlay admitted before submit cancels it; an event
// admitted after the completed write is later input.
func (p *proxy) dispatchOrientationObservation(out io.Writer, settle *time.Timer, settled bool) {
	d := p.orientation
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.operator && (d.replies.inFlight > 0 || len(d.replies.input) > 0) {
		if settled {
			settle.Reset(d.settleAfter)
		}
		return
	}
	event := orientation.DeliveryEvent{Kind: orientation.ComposerObserved, ComposerReady: d.ready}
	if settled {
		event.Kind = orientation.SettleElapsed
		event.ComposerReady = p.terminal != nil && p.orientationComposerActive(p.terminal.Snapshot()) && !p.pickerActive.Load()
	}
	switch {
	case d.exited:
		event.Kind = orientation.ChildExited
	case d.operator:
		event.Kind = orientation.OperatorInput
	case d.overlay:
		event.Kind = orientation.OverlayObserved
	}
	p.advanceOrientation(event, out, settle)
}

// advanceOrientation is called exclusively by translateStdinFrom, beside every
// ordinary input write. A short write is final; retrying could duplicate text.
func (p *proxy) advanceOrientation(event orientation.DeliveryEvent, out io.Writer, settle *time.Timer) {
	d := p.orientation
	if d == nil {
		return
	}
	state, effect := orientation.AdvanceDelivery(d.state, event)
	d.state = state
	switch effect {
	case orientation.PastePrompt:
		data := []byte("\x1b[200~" + d.request.Body + "\x1b[201~")
		n, err := out.Write(data)
		p.advanceOrientation(orientation.DeliveryEvent{Kind: orientation.PasteCompleted, Written: n, Expected: len(data), Failed: err != nil}, out, settle)
	case orientation.StartSettle:
		settle.Reset(d.settleAfter)
	case orientation.SubmitPrompt:
		data := d.profile.keymap.altCR
		n, err := out.Write(data)
		if err == nil && n == len(data) {
			p.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
		}
		p.advanceOrientation(orientation.DeliveryEvent{Kind: orientation.SubmitCompleted, Written: n, Expected: len(data), Failed: err != nil}, out, settle)
	case orientation.PublishStatus:
		if d.publish != nil {
			d.publish(state)
		}
		close(d.finalized)
	}
}

// Return remapping deliberately accepts slash menus and Claude's shell mode:
// newline is safe there. Automatic coding prompts need the narrower surface.
func (p *proxy) orientationComposerActive(snapshot terminalSnapshot) bool {
	recognized := p.orientation.profile.recognize != nil && p.orientation.profile.recognize(snapshot)
	if p.agentBasename == "agy" && !recognized {
		recognized, p.orientation.agyModelFooter = agyUncoloredOrientationComposer(snapshot, p.orientation.agyModelFooter)
	}
	if p.agentBasename == "codex" {
		card, pending := codexOrientationStartupStatus(snapshot)
		if card {
			p.orientation.codexStartupPending = pending
		}
	}
	if !recognized || p.orientation.codexStartupPending {
		return false
	}
	prompt := map[string]string{"claude": "❯", "codex": "›", "agy": ">", "muse": "⟩"}[p.agentBasename]
	for y := snapshot.Cursor.Y; y >= 0; y-- {
		cell := snapshot.CellAt(0, y)
		if cell == nil || strings.TrimSpace(cell.Content) == "" {
			continue
		}
		if cell.Content != prompt {
			return false
		}
		for x := 2; x < snapshot.Width; x++ {
			c := snapshot.CellAt(x, y)
			if c == nil || strings.TrimSpace(c.Content) == "" {
				continue
			}
			return c.Content != "/" && c.Content != "!"
		}
		return true
	}
	return false
}

func (d *orientationDelivery) inputForwarded() bool {
	d.mu.Lock()
	d.replies.inFlight--
	pending := len(d.replies.input) > 0
	d.mu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
	return pending
}
func (p *proxy) cancelIncompleteOrientationReply(out io.Writer, settle *time.Timer) {
	if p.orientation == nil {
		return
	}
	d := p.orientation
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.replies.input) > 0 {
		d.operator = true
		d.replies.input = nil
		p.advanceOrientation(orientation.DeliveryEvent{Kind: orientation.OperatorInput}, out, settle)
	}
}

// Codex paints its real composer while startup authority is still resolving.
// A visible startup card qualifies only after both fields resolve; the later
// trust dialog may otherwise arrive after an automatic submission.
func codexOrientationStartupStatus(snapshot terminalSnapshot) (bool, bool) {
	card, model, directory := false, false, false
	for y := 0; y < snapshot.Cursor.Y; y++ {
		line := strings.TrimSpace(orientationRowText(snapshot, y))
		if strings.Contains(line, "OpenAI Codex (") {
			card = true
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "│"))
		for _, field := range []string{"model:", "directory:"} {
			if strings.HasPrefix(line, field) {
				value := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, field)))
				resolved := len(value) > 0 && value[0] != "loading" && value[0] != "│"
				if field == "model:" {
					model = resolved
				} else {
					directory = resolved
				}
			}
		}
	}
	return card, !model || !directory
}

// NO_COLOR removes Agy's bright-blue prompt cue. In that mode require the
// complete ruled composer and its adjacent shortcuts/model footer, not a bare >.
// The exact model is learned from the empty composer: Agy hides the shortcuts
// hint after paste, while retaining that model beside the expanded input box.
func agyUncoloredOrientationComposer(snapshot terminalSnapshot, modelFooter string) (bool, string) {
	rule := func(s terminalSnapshot, y int) bool {
		for x := 0; x < 5; x++ {
			c := s.CellAt(x, y)
			if c == nil || c.Content != "─" {
				return false
			}
		}
		return true
	}
	if !ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool { return c.Content == ">" && c.Style.Fg == nil },
		ruleAt:   rule, minCursorX: 2, maxRows: 25,
	}) {
		return false, modelFooter
	}
	for y := snapshot.Cursor.Y + 1; y+1 < snapshot.Height; y++ {
		if rule(snapshot, y) {
			footer := strings.TrimSpace(orientationRowText(snapshot, y+1))
			if strings.HasPrefix(footer, "? for shortcuts") {
				model := strings.TrimSpace(strings.TrimPrefix(footer, "? for shortcuts"))
				return model != "", model
			}
			return modelFooter != "" && footer == modelFooter, modelFooter
		}
	}
	return false, modelFooter
}
func orientationRowText(snapshot terminalSnapshot, y int) string {
	var b strings.Builder
	for x := 0; x < snapshot.Width; x++ {
		if c := snapshot.CellAt(x, y); c != nil {
			b.WriteString(c.Content)
		}
	}
	return b.String()
}
