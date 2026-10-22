package recorder

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// OperatorOriginals returns debugger identities only for current-document open
// notes and sketch source anchors in this exact isolated context. Paths and page attributes are never
// identity. Pause does not revoke a connected node's identity.
func (r *Recorder) OperatorOriginals(ctx context.Context, session string, executionContext int64, document int, nodes []rec.NodeID) (map[rec.NodeID]int, error) {
	type result struct {
		refs map[rec.NodeID]int
		err  error
	}
	reply := make(chan result, 1)
	// Copy before queueing; a cancelled query may still run on the actor.
	wanted := make(map[rec.NodeID]bool, len(nodes))
	for _, id := range nodes {
		wanted[id] = true
	}
	err := r.ask(ctx, func() {
		if document <= 0 || r.m.st.DocSeg != document || !r.operatorIdentity(session, executionContext) {
			reply <- result{err: board.ErrStale}
			return
		}
		allowed := map[rec.NodeID]bool{}
		for _, n := range r.m.st.Notes {
			if n.State == rec.NoteOpen && n.Seg >= document && wanted[n.At.Node] && r.m.st.Node(n.At.Node) != nil {
				allowed[n.At.Node] = true
			}
		}
		// Query only this bounded requested batch. The recording's source-doc
		// anchor index avoids scanning sketch paths or copying historical geometry.
		for id := range wanted {
			if r.m.st.Sketches.HasAnchor(document, id) && r.m.st.Node(id) != nil {
				allowed[id] = true
			}
		}
		refs := map[rec.NodeID]int{}
		s := r.m.sessions[session]
		if s != nil {
			for backend, id := range s.ids {
				if allowed[id] && s.byBackend[backend] != nil {
					refs[id] = backend
				}
			}
		}
		reply <- result{refs: refs}
	})
	if err != nil {
		return nil, err
	}
	out := <-reply
	return out.refs, out.err
}
