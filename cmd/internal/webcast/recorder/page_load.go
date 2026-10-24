package recorder

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// loadOwner is the sole authority for ordered snapshot intents and current UI
// scope. Its publications acquire only the leaf Board mutex; it never holds a
// queue, command, CDP or sink lock. The recorder actor remains the one writer.
type loadOwner struct {
	mu                 sync.Mutex
	state              loadState
	cells              map[string]*loadCell
	serial, cellSerial uint64
	board              *board.Board
	cancel             context.CancelFunc
	cancelAttempt      *loadAttempt
}

func newLoadOwner(b *board.Board) *loadOwner {
	return &loadOwner{cells: map[string]*loadCell{}, board: b}
}
func (o *loadOwner) transition(cell *loadCell, event loadEvent) (attempt *loadAttempt, reservation *loadReservation) {
	value := loadCell{}
	if cell != nil {
		value = *cell
	}
	var effects []loadEffect
	o.state, effects = nextLoadScope(o.state, value, event)
	for _, effect := range effects {
		switch e := effect.(type) {
		case loadCancelSnapshot:
			if o.cancelAttempt == e.attempt && o.cancel != nil {
				o.cancel()
			}
		case loadPublishPending:
			if o.board != nil {
				o.board.BeginPageLoad(e.generation)
			}
		case loadPublishConfirmed:
			if o.board != nil {
				o.board.ConfirmPageLoad(e.generation, e.load)
			}
		case loadPublishUnavailable:
			if o.board != nil {
				o.board.UnavailablePageLoad(e.generation)
			}
		case loadBeginSnapshot:
			attempt = e.attempt
		case loadCommitSnapshot:
			reservation = e.reservation
		case loadRetireIntent:
			e.intent.retired = true
		}
	}
	return
}
func (o *loadOwner) issue(cell *loadCell, activation bool) *snapshotIntent {
	if cell == nil || !cell.attached || o.state.closing {
		return nil
	}
	o.serial++
	if o.serial == 0 {
		o.transition(nil, loadClose{})
		return nil
	}
	intent := &snapshotIntent{serial: o.serial, cell: cell, activationStart: activation, activation: o.state.selectedActivation}
	if activation {
		intent.activation = intent.serial
		o.transition(cell, loadLive{cell.id, intent.serial})
	}
	return intent
}

// ingress reads only synchronized session facts. An intent travels in the
// existing recorder queue input; there is no separate capture queue.
func (o *loadOwner) ingress(in any) any {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state.closing {
		return in
	}
	switch v := in.(type) {
	case Live:
		cell := o.cells[v.Session]
		if cell == nil || cell.selection.Target != v.Tab.ID {
			if cell != nil {
				*cell = nextLoadCell(*cell, loadCellChange{detach: true, invalidate: true})
				o.transition(cell, loadChanged{cell.id})
			}
			o.cellSerial++
			if o.cellSerial == 0 {
				o.transition(nil, loadClose{})
				return v
			}
			cell = &loadCell{id: o.cellSerial, epoch: 1, attached: true, selection: loadSelection{Session: v.Session, Target: v.Tab.ID}}
			o.cells[v.Session] = cell
		}
		if o.state.intendedLive && o.state.selectedCell == cell.id {
			return v
		}
		v.intent = o.issue(cell, true)
		return v
	case Pause:
		o.transition(nil, loadPause{})
	case Detached:
		if cell := o.cells[v.Session]; cell != nil {
			*cell = nextLoadCell(*cell, loadCellChange{detach: true, invalidate: true})
			o.transition(cell, loadChanged{cell.id})
			delete(o.cells, v.Session)
		}
	case Event:
		cell := o.cells[v.Session]
		if cell == nil {
			return v
		}
		change := loadCellChange{invalidate: true}
		switch v.Method {
		case "Page.frameNavigated":
			var p struct {
				Frame struct {
					ID     string `json:"id"`
					Parent string `json:"parentId"`
					Loader string `json:"loaderId"`
				} `json:"frame"`
			}
			if json.Unmarshal(v.Params, &p) != nil || p.Frame.Parent != "" || p.Frame.ID == "" || (cell.selection.Frame != "" && cell.selection.Frame != p.Frame.ID) {
				return v
			}
			if p.Frame.Loader != "" && p.Frame.Loader == cell.loader {
				return v
			}
			change.frame, change.loader = p.Frame.ID, p.Frame.Loader
		case "DOM.documentUpdated":
			if v.Seq != 0 && v.Seq <= cell.replySeq {
				return v
			}
		default:
			return v
		}
		*cell = nextLoadCell(*cell, change)
		o.transition(cell, loadChanged{cell.id})
		v.intent = o.issue(cell, false)
		return v
	}
	return in
}
func (o *loadOwner) refresh(session string) *snapshotIntent {
	o.mu.Lock()
	defer o.mu.Unlock()
	cell := o.cells[session]
	if cell == nil {
		return nil
	}
	// A desync is a new observation even without a top navigation notification.
	*cell = nextLoadCell(*cell, loadCellChange{invalidate: true})
	o.transition(cell, loadChanged{cell.id})
	return o.issue(cell, false)
}
func (o *loadOwner) start(parent context.Context, intent *snapshotIntent) (*loadAttempt, context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if intent == nil {
		return nil, parent
	}
	a, _ := o.transition(intent.cell, loadStart{intent})
	if a == nil {
		return nil, parent
	}
	ctx, cancel := context.WithCancel(parent)
	o.cancel, o.cancelAttempt = cancel, a
	return a, ctx
}
func (o *loadOwner) ready(a *loadAttempt, evidence loadEvidence) *loadReservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancelAttempt == a {
		o.cancel()
		o.cancel = nil
		o.cancelAttempt = nil
	}
	_, r := o.transition(a.intent.cell, loadSnapshotReady{a, evidence})
	return r
}
func (o *loadOwner) failed(a *loadAttempt) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancelAttempt == a {
		o.cancel()
		o.cancel = nil
		o.cancelAttempt = nil
	}
	o.transition(a.intent.cell, loadSnapshotFailed{a})
	return !a.intent.retired && a.intent.cell.attached && a.intent.cell.epoch != a.epoch
}
func (o *loadOwner) retry(a *loadAttempt) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return !a.intent.retired && a.intent.cell.attached && a.intent.cell.epoch != a.epoch
}
func (o *loadOwner) done(r *loadReservation, success bool, seq uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	cell := r.attempt.intent.cell
	// Cache facts are installed only for reserved evidence and never regress a
	// newer ingress loader/frame observation of the same session.
	if success && cell.epoch == r.attempt.epoch {
		*cell = nextLoadCell(*cell, loadCellChange{frame: r.evidence.Selection.Frame, loader: r.evidence.Load.Loader, replySeq: seq})
	}
	o.transition(cell, loadCommitDone{r, success})
}
func (o *loadOwner) retire(intent *snapshotIntent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if intent != nil {
		intent.retired = true
	}
}

// finish releases session cells only after the actor has drained all issued
// intents and joined side work. Closing itself cannot revoke FIFO history.
func (o *loadOwner) finish() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancel != nil {
		o.cancel()
	}
	o.cancel = nil
	o.cancelAttempt = nil
	clear(o.cells)
	o.state.active = nil
	o.state.reservation = nil
}
func (o *loadOwner) close() { o.mu.Lock(); defer o.mu.Unlock(); o.transition(nil, loadClose{}) }

// pageFeedbackReady gates new page effects only. Recovery and unrelated ctl
// records retain their existing authority and ordering paths.
func (r *Recorder) pageFeedbackReady(record rec.Record, doc int) bool {
	if state, ok := record.(*rec.NoteState); ok && doc == 0 && state.IfRevision != nil && state.State == rec.NoteResolved {
		return true
	}
	if _, ok := record.(*rec.SketchInterrupt); ok && doc == 0 {
		return true
	}
	switch record.Kind() {
	case "note", "note-edit", "note-state", "stroke", "sketch-begin", "sketch-stroke-begin", "sketch-points", "sketch-stroke-end", "sketch-undo", "sketch-redo", "sketch-label":
		return r.opts.Operator == nil || r.opts.Operator.PageFeedbackReady(record)
	default:
		return true
	}
}
