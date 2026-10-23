package recorder

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ResolveOperatorPaths uses the recorder's existing live path resolver. In
// particular, raw browser child indexes must not be walked over the reduced DOM,
// which intentionally excludes private UI and other unrecorded children.
func (r *Recorder) ResolveOperatorPaths(ctx context.Context, session string, document int, tab string, paths [][]int) ([]recording.NodeID, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(paths) > 8*64 {
		return nil, board.ErrShape
	}
	owned := make([][]int, len(paths))
	for i, p := range paths {
		if len(p) == 0 || len(p) > 128 {
			return nil, board.ErrShape
		}
		owned[i] = append([]int(nil), p...)
		for _, step := range p {
			if step < -1 || step > 1<<20 {
				return nil, board.ErrShape
			}
		}
	}
	type result struct {
		ids []recording.NodeID
		err error
	}
	answer := make(chan result, 1)
	err := r.ask(ctx, func() {
		if err := ctx.Err(); err != nil {
			answer <- result{err: err}
			return
		}
		if !r.live || r.top != session || r.m.st.DocSeg != document || r.m.st.Tab.ID != tab || r.m.sessions[session] == nil {
			answer <- result{err: board.ErrStale}
			return
		}
		ids := make([]recording.NodeID, 0, len(owned))
		for _, path := range owned {
			id, _, err := r.m.nodeAt(r.m.sessions[session], r.frameIDs[session], path)
			n := r.m.st.Node(id)
			if err != nil || n == nil || n.Y != "el" {
				answer <- result{err: board.ErrStale}
				return
			}
			ids = append(ids, id)
		}
		answer <- result{ids: ids}
	})
	if err != nil {
		return nil, err
	}
	got := <-answer
	return got.ids, got.err
}
