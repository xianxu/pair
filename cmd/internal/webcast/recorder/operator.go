package recorder

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// OperatorReply carries the context-local board result and the opaque handle
// returned by prepare-note. The callback's session/context are trusted CDP data.
type OperatorReply struct {
	board.OperatorReply
	Draft       string    `json:"draft,omitempty"`
	State       string    `json:"state,omitempty"`
	CapturedAt  time.Time `json:"captured_at,omitempty"`
	Unavailable string    `json:"unavailable,omitempty"`
	Seg         int       `json:"seg"`
	Doc         int       `json:"doc"`
}

const maxOperatorQueue = 32

var errOperatorQueue = errors.New("recorder: operator queue full")

type operatorContextKey struct {
	session string
	context int64
}
type operatorDocument struct {
	loader   string
	document string // mirror identity: loader plus root backend node
	doc      int
}

type operatorDraft struct {
	handle, request string
	prepare         board.OperatorPrepare
	node            rec.NodeID
	used            string
}
type operatorSave struct {
	request string
	body    board.OperatorSave
	note    board.NoteIn
}
type operatorContext struct {
	session string
	writing string
	context int64
	doc     int
	board   *board.OperatorSession
	draft   *operatorDraft
	saved   *operatorSave
}
type operatorWrite struct {
	admission board.AnnotationAdmission
	owner     *operatorContext
	record    rec.Record
	doc       int
	settle    func(board.OperatorOutcome, error)
}

// operatorMessage is bounded before allocating the binding's payload. Only the
// known isolated world is admitted by handleOperator, never by claimed identity.
func operatorMessage(e Event) (int64, board.OperatorRequest, bool) {
	if e.Method != "Runtime.bindingCalled" || len(e.Params) > 6*board.MaxOperatorBody+1024 {
		return 0, board.OperatorRequest{}, false
	}
	var p struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
		Context int64  `json:"executionContextId"`
	}
	if json.Unmarshal(e.Params, &p) != nil || p.Name != bindingName || len(p.Payload) > board.MaxOperatorBody+64 {
		return 0, board.OperatorRequest{}, false
	}
	var outer struct {
		T       string          `json:"t"`
		Request json.RawMessage `json:"request"`
	}
	if !broadcast.DecodeStrict([]byte(p.Payload), &outer) || outer.T != "operator" {
		return 0, board.OperatorRequest{}, false
	}
	req, err := board.ParseOperatorRequest(outer.Request)
	if err != nil {
		return 0, board.OperatorRequest{}, false
	}
	return p.Context, req, true
}
func (r *Recorder) operatorReply(session string, ctx int64, reply OperatorReply) {
	if r.opts.OperatorReply != nil {
		r.opts.OperatorReply(session, ctx, reply)
	}
}
func (r *Recorder) operatorRefusal(session string, ctx int64, id string, err error) {
	r.operatorReply(session, ctx, OperatorReply{OperatorReply: board.OperatorReply{Request: id, Outcome: board.OperatorRefused, Error: err.Error()}})
}
func (r *Recorder) refuseOperatorEvent(e Event) {
	if ctx, req, ok := operatorMessage(e); ok {
		r.operatorRefusal(e.Session, ctx, req.Request, errOperatorQueue)
	}
}
func (r *Recorder) operatorIdentity(session string, ctx int64) bool {
	frame := r.contexts[session][ctx]
	stamp := r.operatorDocs[operatorContextKey{session, ctx}]
	return session != "" && session == r.top && frame != "" && frame == r.frameIDs[r.top] && stamp.doc != 0 && stamp.doc == r.m.st.DocSeg
}
func (r *Recorder) operatorAdmission(session string, ctx int64, doc int) error {
	switch {
	case r.opts.Operator == nil:
		return board.ErrNotLive
	case !r.operatorIdentity(session, ctx):
		return board.ErrStale
	case !r.opened || r.m.st.DocSeg == 0:
		return board.ErrNotLive
	case r.m.st.Ended:
		return board.ErrEnded
	case doc != r.m.st.DocSeg:
		return board.ErrStale
	case !r.live || r.m.st.Paused():
		return board.ErrPaused
	case r.m.st.Control.Driver != nil:
		return board.ErrDriving
	}
	return nil
}
func (r *Recorder) expireOperator() {
	r.stampOperatorDocuments()
	r.expireDraftOwner()
	o := r.operator
	if o != nil && (!r.operatorIdentity(o.session, o.context) || o.doc != r.m.st.DocSeg || r.m.st.Ended) {
		o.board.Close()
		// Retain this context identity as a tombstone: a late binding cannot
		// acquire a replacement Chrome document. handleOperator replaces it
		// only for a trusted context in the current recording document scope.
	}
}
func (r *Recorder) operatorBinding(e Event) bool {
	ctx, req, ok := operatorMessage(e)
	if !ok {
		return false
	}
	r.handleOperator(e.Session, ctx, req)
	return true
}
func (r *Recorder) handleOperator(session string, ctx int64, req board.OperatorRequest) {
	r.expireOperator()
	if !r.operatorIdentity(session, ctx) || r.opts.Operator == nil {
		r.operatorRefusal(session, ctx, req.Request, board.ErrStale)
		return
	}
	// Identity is actor-owned: never issue owner authority from a binding
	// label, claimed attribution, or an unverified context at enqueue time.
	// Use the sender's generation so queued/delayed work cannot acquire a
	// newer generation when it finally reaches the actor.
	admission, _ := r.opts.Operator.AdmitOperatorAnnotationGeneration(req.Generation)
	if r.handleDraftOperation(session, ctx, req, admission) {
		return
	}
	o := r.operator
	if o == nil || o.session != session || o.context != ctx || o.doc != r.m.st.DocSeg {
		if o != nil {
			o.board.Close()
		}
		o = &operatorContext{session: session, context: ctx, doc: r.m.st.DocSeg}
		o.board = r.opts.Operator.NewOperator(func(record rec.Record, doc int, admitted board.AnnotationAdmission, settle func(board.OperatorOutcome, error)) error {
			request := o.writing
			var once sync.Once
			done := func(outcome board.OperatorOutcome, err error) {
				once.Do(func() {
					settle(outcome, err)
					// Settlement happens on the recorder loop, including queue refusal
					// during admission. Session.Status now holds the authoritative result.
					if request != "" {
						if outcome == board.OperatorRefused && o.draft != nil && o.draft.used == request {
							o.draft.used = ""
						}
						r.replyOperatorOperation(o, o.board.Status(request))
					}
				})
			}
			// The enqueue refusal is returned to Board; its caller publishes that
			// synchronous result. Queued completions alone publish a later reply.
			if !r.enqueueOperator(operatorWrite{admission: admitted, owner: o, record: record, doc: doc, settle: done}) {
				return errOperatorQueue
			}
			return nil
		})
		r.operator = o
	}
	reply := func(v board.OperatorReply) { r.replyOperatorOperation(o, v) }
	switch req.Op {
	case "status":
		reply(o.board.Status(req.Request))
		return
	case "cancel-draft":
		if o.draft != nil && o.draft.handle == req.Draft {
			o.draft = nil
		}
		reply(board.OperatorReply{Request: req.Request, Outcome: board.OperatorAccepted})
		return
	}
	if err := admission.Check(); err != nil {
		r.operatorRefusal(session, ctx, req.Request, err)
		return
	}
	if err := r.operatorAdmission(session, ctx, o.doc); err != nil {
		r.operatorRefusal(session, ctx, req.Request, err)
		return
	}
	switch req.Op {
	case "prepare-note":
		p := req.Prepare
		if p.Seg < o.doc || p.Seg > r.m.st.Seg {
			r.operatorRefusal(session, ctx, req.Request, board.ErrStale)
			return
		}
		if o.draft != nil && o.draft.request == req.Request {
			a, _ := json.Marshal(o.draft.prepare)
			b, _ := json.Marshal(p)
			if string(a) != string(b) {
				r.operatorRefusal(session, ctx, req.Request, board.ErrShape)
				return
			}
		} else {
			ms := r.m.sessions[session]
			if ms == nil {
				r.operatorRefusal(session, ctx, req.Request, board.ErrStale)
				return
			}
			node, _, err := r.m.nodeAt(ms, r.frameIDs[session], p.Path)
			if err != nil || r.m.st.Node(node) == nil || r.m.st.Node(node).Y != "el" {
				r.operatorRefusal(session, ctx, req.Request, board.ErrStale)
				return
			}
			var token [16]byte
			if _, err := rand.Read(token[:]); err != nil {
				r.operatorRefusal(session, ctx, req.Request, err)
				return
			}
			if err := admission.Check(); err != nil {
				r.operatorRefusal(session, ctx, req.Request, err)
				return
			}
			o.draft = &operatorDraft{handle: hex.EncodeToString(token[:]), request: req.Request, prepare: *p, node: node}
		}
		if err := admission.Check(); err != nil {
			r.operatorRefusal(session, ctx, req.Request, err)
			return
		}
		r.operatorReply(session, ctx, OperatorReply{OperatorReply: board.OperatorReply{Request: req.Request, Outcome: board.OperatorAccepted}, Draft: o.draft.handle, Seg: r.m.st.Seg, Doc: o.doc})
	case "save-note":
		in := req.Save
		if o.saved != nil && o.saved.request == req.Request {
			if o.saved.body != *in {
				r.operatorRefusal(session, ctx, req.Request, board.ErrShape)
				return
			}
			reply(o.board.NoteAdmitted(admission, req.Request, o.saved.note))
			return
		}
		d := o.draft
		if d == nil || d.handle != in.Draft || d.used != "" {
			r.operatorRefusal(session, ctx, req.Request, board.ErrStale)
			return
		}
		n := board.NoteIn{Seg: d.prepare.Seg, Node: d.node, Box: d.prepare.Box, Region: d.prepare.Region, Color: in.Color, Text: in.Text}
		o.writing = req.Request
		result := o.board.NoteAdmitted(admission, req.Request, n)
		if result.Outcome != board.OperatorRefused {
			d.used = req.Request
			o.saved = &operatorSave{request: req.Request, body: *in, note: n}
		}
		reply(result)
	case "stroke":
		o.writing = req.Request
		reply(o.board.StrokeAdmitted(admission, req.Request, *req.Stroke))
	case "edit-note":
		o.writing = req.Request
		reply(o.board.EditAdmitted(admission, req.Request, board.EditIn{Seg: req.Edit.Seg, ID: req.Edit.ID, Text: req.Edit.Text}))
	case "resolve":
		o.writing = req.Request
		reply(o.board.SetStateAdmitted(admission, req.Request, req.Resolve.Seg, board.StateIn{ID: req.Resolve.ID, State: req.Resolve.State}))
	}
}
func (r *Recorder) replyOperatorOperation(o *operatorContext, reply board.OperatorReply) {
	r.operatorReply(o.session, o.context, OperatorReply{OperatorReply: reply, Seg: r.m.st.Seg, Doc: o.doc})
}
func (r *Recorder) enqueueOperator(in operatorWrite) bool {
	at := r.opts.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.operatorQueued >= maxOperatorQueue {
		return false
	}
	r.operatorQueued++
	r.queue = append(r.queue, stamped{in: in, at: at})
	select {
	case r.signal <- struct{}{}:
	default:
	}
	return true
}
func (r *Recorder) writeOperator(at time.Time, in operatorWrite) {
	if r.sink == nil {
		in.settle(board.OperatorRefused, board.ErrClosed)
		return
	}
	if err := r.operatorAdmission(in.owner.session, in.owner.context, in.doc); err != nil {
		in.settle(board.OperatorRefused, err)
		return
	}
	if r.operator != in.owner {
		in.settle(board.OperatorRefused, board.ErrStale)
		return
	}
	if r.err != nil {
		in.settle(board.OperatorRefused, r.err)
		return
	}
	if err := in.admission.Check(); err != nil {
		in.settle(board.OperatorRefused, err)
		return
	}
	if state, ok := in.record.(*rec.NoteState); ok && state.IfRevision != nil && !r.m.State().NoteUnchanged(state.ID, *state.IfRevision) {
		in.settle(board.OperatorRefused, board.ErrStale)
		return
	}
	if !r.m.Emit(r.t(at), in.record) {
		in.settle(board.OperatorRefused, board.ErrShape)
		return
	}
	if err := r.write(); err != nil {
		in.settle(board.OperatorUnknown, err)
		return
	}
	in.settle(board.OperatorAccepted, nil)
}

// Context creation precedes the first operator command. Stamp it only when the
// mirror has the loader it was created for; keep that stamp across navigation,
// even for a context which has never submitted a command. This prevents a late
// path from being resolved against a replacement document.
func (r *Recorder) operatorContextCreated(session string, ctx int64) {
	if r.opts.Operator == nil {
		return
	}
	if r.operatorDocs == nil {
		r.operatorDocs = map[operatorContextKey]operatorDocument{}
	}
	key := operatorContextKey{session, ctx}
	r.operatorDocs[key] = operatorDocument{loader: r.loader[session]}
	if o := r.operator; o != nil && o.session == session && o.context == ctx {
		o.board.Close()
		r.operator = nil
	}
}
func (r *Recorder) stampOperatorDocuments() {
	for key, stamp := range r.operatorDocs {
		frame := r.contexts[key.session][key.context]
		if frame == "" {
			delete(r.operatorDocs, key)
			continue
		}
		if key.session != r.top || frame != r.frameIDs[r.top] || r.m.st.DocSeg == 0 {
			continue
		}
		ms := r.m.sessions[key.session]
		if ms == nil || ms.root == nil {
			continue
		}
		if stamp.loader != "" && !strings.HasPrefix(ms.doc, stamp.loader+":") {
			continue
		}
		// Returning to an unchanged Chrome document begins a new recording
		// document segment. Its existing context may acquire that scope, but
		// navigation (even with the same loader string) may not reuse it.
		if stamp.document != "" && stamp.document != ms.doc {
			continue
		}
		stamp.document = ms.doc
		stamp.doc = r.m.st.DocSeg
		r.operatorDocs[key] = stamp
	}
}
