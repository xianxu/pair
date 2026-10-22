package webcast

import (
	"context"
	"encoding/json"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const operatorOriginalBatch = 16

type operatorOriginalAttempt struct {
	bound    bool
	revision uint64
	tries    int
}
type operatorOriginals struct {
	document       int
	nodes          map[recording.NodeID]operatorOriginalAttempt
	sketchDocument int
	sketchRevision uint64
	sketchNodes    []recording.NodeID
	sketchPending  bool
}

// deliverOperatorOriginals runs on the existing serialized delivery worker.
// At most sixteen debugger bindings share a 250ms deadline per pass; successful
// bindings are reused until their last note or isolated document disappears.
func (c *Chrome) deliverOperatorOriginals(target operatorTarget, view board.OperatorView, cache *operatorOriginals) bool {
	if c.rec == nil || !target.active || view.Doc <= 0 {
		return false
	}
	ids, more := cache.next(view)
	if len(ids) == 0 {
		return more
	}
	ctx, cancel := context.WithTimeout(c.ctx, 250*time.Millisecond)
	defer cancel()
	refs, err := c.rec.OperatorOriginals(ctx, target.session, target.context, view.Doc, ids)
	if err != nil {
		return false
	} // a later recorder/view update supplies the new scope
	for _, id := range ids {
		if ctx.Err() != nil {
			return true
		}
		attempt := cache.nodes[id]
		if attempt.revision != view.Revision {
			attempt.revision = view.Revision
			attempt.tries = 0
		}
		backend, ok := refs[id]
		if !ok {
			attempt.tries = 3
			cache.nodes[id] = attempt
			continue
		}
		attempt.tries++
		attempt.bound = c.bindOperatorOriginal(ctx, target, view.Doc, id, backend)
		cache.nodes[id] = attempt
		if !attempt.bound && attempt.tries < 3 {
			more = true
		}
	}
	return more
}

func (c *Chrome) bindOperatorOriginal(ctx context.Context, target operatorTarget, document int, node recording.NodeID, backend int) bool {
	// One serialized worker owns this group. Release it even when resolve's
	// response is lost; a late remote object must not leak into the next pass.
	const group = "couchcast-operator-originals"
	defer func() {
		release, cancel := context.WithTimeout(c.ctx, 100*time.Millisecond)
		defer cancel()
		_ = c.conn.Call(release, target.session, "Runtime.releaseObjectGroup", map[string]any{"objectGroup": group}, nil)
	}()
	var resolved struct {
		Object struct {
			ID string `json:"objectId"`
		} `json:"object"`
	}
	if err := c.conn.Call(ctx, target.session, "DOM.resolveNode", map[string]any{"backendNodeId": backend, "executionContextId": target.context, "objectGroup": group}, &resolved); err != nil || resolved.Object.ID == "" {
		return false
	}
	var result struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	err := c.conn.Call(ctx, target.session, "Runtime.callFunctionOn", map[string]any{
		"objectId": resolved.Object.ID, "functionDeclaration": `function(document,node){return !!globalThis.__couchcastOperatorOriginal?.(document,node,this)}`,
		"arguments": []any{map[string]any{"value": document}, map[string]any{"value": node}}, "returnByValue": true,
	}, &result)
	return err == nil && len(result.Exception) == 0 && result.Result.Value
}

// next prunes completed lifetimes and limits retry work per delivered snapshot.
func (p *operatorOriginals) next(view board.OperatorView) ([]recording.NodeID, bool) {
	if p.document != view.Doc {
		p.document = view.Doc
		p.nodes = map[recording.NodeID]operatorOriginalAttempt{}
	}
	wanted := map[recording.NodeID]bool{}
	deferred := map[recording.NodeID]bool{}
	for _, note := range view.Notes {
		if note.At.Node > 0 {
			wanted[note.At.Node] = true
		}
	}
	if p.sketchDocument == view.Doc {
		for _, id := range p.sketchNodes {
			if p.sketchPending && !wanted[id] {
				deferred[id] = true
			}
			wanted[id] = true
		}
	}
	for node := range p.nodes {
		if !wanted[node] {
			delete(p.nodes, node)
		}
	}
	ids := make([]recording.NodeID, 0, operatorOriginalBatch)
	more := false
	for node := range wanted {
		if deferred[node] {
			continue
		}
		attempt := p.nodes[node]
		if attempt.bound || attempt.revision == view.Revision && attempt.tries >= 3 {
			continue
		}
		if len(ids) == operatorOriginalBatch {
			more = true
			continue
		}
		ids = append(ids, node)
	}
	return ids, more
}

// acceptSketchAnchors runs only after the parent has acknowledged this
// anchor projection. Even identical IDs can have crossed a clear/prune while
// delivery was pending, so their old debugger ACKs no longer prove retention.
// Work is bounded by old/new anchor IDs, never stroke point history.
func (p *operatorOriginals) acceptSketchAnchors(document int, revision uint64, nodes []recording.NodeID) {
	p.sketchPending = false
	if p.sketchDocument == document && p.sketchRevision == revision {
		return
	}
	for _, id := range p.sketchNodes {
		delete(p.nodes, id)
	}
	for _, id := range nodes {
		delete(p.nodes, id)
	}
	p.sketchDocument = document
	p.sketchRevision = revision
	p.sketchNodes = nodes
}
