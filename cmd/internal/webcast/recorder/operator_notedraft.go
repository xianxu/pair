package recorder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
)

type operatorDraftOwner struct {
	owner   notedraft.Owner
	session string
	context int64
	doc     int
}

// handleDraftOperation freezes the anchor on the actor. The bounded worker does
// capture-service IO without blocking CDP reduction; only Writer's check closure
// returns to the actor to validate the original live context before dispatch.
func (r *Recorder) handleDraftOperation(session string, contextID int64, req board.OperatorRequest, a board.AnnotationAdmission) bool {
	if r.opts.NoteDrafts == nil {
		return false
	}
	switch req.Op {
	case "prepare-note", "draft-status", "save-note", "cancel-draft":
	default:
		return false
	}
	if err := r.operatorAdmission(session, contextID, r.m.st.DocSeg); err != nil {
		r.operatorRefusal(session, contextID, req.Request, err)
		return true
	}
	if err := a.Check(); err != nil {
		r.operatorRefusal(session, contextID, req.Request, err)
		return true
	}
	if req.Op == "prepare-note" && !r.opts.Operator.PageLoadReady() {
		r.operatorRefusal(session, contextID, req.Request, board.ErrStale)
		return true
	}
	doc, seg := r.m.st.DocSeg, r.m.st.Seg
	hash := sha256.Sum256([]byte(fmt.Sprintf("operator\x00%s\x00%d\x00%d", session, contextID, doc)))
	owner := notedraft.Owner{Identity: broadcast.OperatorNoteOwner, Scope: hex.EncodeToString(hash[:]), Generation: req.Generation}
	r.draftOwner = &operatorDraftOwner{owner, session, contextID, doc}
	var p board.PreparedNote
	if req.Op == "prepare-note" {
		in := req.Prepare
		node, _, err := r.m.nodeAt(r.m.sessions[session], r.frameIDs[session], in.Path)
		if err == nil {
			p, err = a.PrepareNote(board.PrepareNoteIn{Seg: in.Seg, Node: node, Box: in.Box, Region: in.Region}, true)
		}
		if err != nil {
			r.operatorRefusal(session, contextID, req.Request, err)
			return true
		}
	}
	if r.draftWork == nil {
		r.draftWork = make(chan struct{}, maxOperatorQueue)
	}
	select {
	case r.draftWork <- struct{}{}:
	default:
		r.operatorRefusal(session, contextID, req.Request, errOperatorQueue)
		return true
	}
	go func() {
		defer func() { <-r.draftWork }()
		ctx, cancel := context.WithTimeout(r.ctx, notedraft.OperationBudget)
		defer cancel()
		var status notedraft.Status
		var err error
		switch req.Op {
		case "prepare-note":
			status, err = r.opts.NoteDrafts.Prepare(ctx, owner, req.Request, a, p, true)
		case "draft-status":
			status, err = r.opts.NoteDrafts.Status(owner, req.Draft)
		case "cancel-draft":
			err = r.opts.NoteDrafts.Cancel(owner, req.Draft)
			status = notedraft.Status{State: "cancelled", Draft: req.Draft}
		case "save-note":
			in := req.Save
			status, err = r.opts.NoteDrafts.Save(ctx, owner, req.Request, in.Draft, a, board.Viewer{ID: board.Operator, Name: "Operator"}, in.Color, in.Text, func() error { return r.operatorAdmission(session, contextID, doc) })
		}
		outcome := board.OperatorAccepted
		if status.State == "pending" {
			outcome = board.OperatorPending
		}
		if err != nil {
			outcome = board.OperatorRefused
			status.Error = err.Error()
		}
		r.operatorReply(session, contextID, OperatorReply{OperatorReply: board.OperatorReply{Request: req.Request, Outcome: outcome, ID: status.NoteID, Error: status.Error}, Draft: status.Draft, Seg: seg, Doc: doc, State: status.State, CapturedAt: status.CapturedAt, Unavailable: status.Unavailable})
	}()
	return true
}
func (r *Recorder) expireDraftOwner() {
	o := r.draftOwner
	if o == nil || r.opts.NoteDrafts == nil {
		return
	}
	if r.operatorIdentity(o.session, o.context) && o.doc == r.m.st.DocSeg && !r.m.st.Ended {
		return
	}
	r.draftOwner = nil
	go r.opts.NoteDrafts.Revoke(o.owner)
}
