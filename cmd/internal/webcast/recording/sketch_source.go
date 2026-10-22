package recording

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const (
	SketchSourcePrefixLimit = 16 << 20
	SketchSourceStateLimit  = 8 << 20
	SketchSourceChunkSize   = 32 << 10
)

var (
	ErrSketchSourcePrefixLimit = errors.New("sketch source prefix exceeds limit")
	ErrSketchSourceStateLimit  = errors.New("sketch source state exceeds limit")
)

// SketchSource owns a frozen replay prefix and its reduced scene. State follows
// State's read-only consumer contract; Read and Begin return owned values.
type SketchSource struct {
	prefix []byte
	state  *State
	begin  SketchBegin
	stats  SketchSourceStats
	assets map[string]struct{}
}

type SketchSourceStats struct{ PrefixBytes, StateBytes, Nodes, Sheets int }

func (s *SketchSource) State() *State { return s.state }
func (s *SketchSource) Begin() SketchBegin {
	b, _ := cloneSketchBegin(&s.begin)
	b.K = s.begin.K
	return *b
}
func (s *SketchSource) Stats() SketchSourceStats     { return s.stats }
func (s *SketchSource) Length() int                  { return len(s.prefix) }
func (s *SketchSource) AllowsAsset(hash string) bool { _, ok := s.assets[hash]; return ok }
func (s *SketchSource) Read(offset, length int) ([]byte, error) {
	if offset < 0 || offset > len(s.prefix) || length <= 0 || length > SketchSourceChunkSize {
		return nil, errors.New("invalid sketch source chunk")
	}
	return bytes.Clone(s.prefix[offset : offset+min(length, len(s.prefix)-offset)]), nil
}

// CandidateSketchSource is a preflight on the ordered writer, before the exact
// begin is emitted. It never reserves authority, advances the hub, or writes.
func (h *Hub) CandidateSketchSource(begin *SketchBegin, maxPrefixBytes, maxStateBytes int) (*SketchSource, error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrClosed
	}
	log := append([]Record(nil), h.log...)
	h.mu.Unlock()
	if begin == nil || len(log) == 0 || begin.Time() <= log[len(log)-1].Time() {
		return nil, errors.New("sketch begin needs an ordered source fence")
	}
	owned, err := cloneSketchBegin(begin)
	if err != nil {
		return nil, err
	}
	owned.K = begin.K
	recs := append(CatchUp(log), owned)
	var prefix bytes.Buffer
	for _, r := range recs {
		p, err := Encode(r)
		if err != nil {
			return nil, err
		}
		if len(p)+1 > maxPrefixBytes-prefix.Len() {
			return nil, ErrSketchSourcePrefixLimit
		}
		prefix.Write(p)
		prefix.WriteByte('\n')
	}
	state, err := ReduceCatchUp(recs, math.MaxInt64)
	if err != nil {
		return nil, fmt.Errorf("prepare sketch source: %w", err)
	}
	stats, err := sketchSceneSize(state, maxStateBytes)
	if err != nil {
		return nil, err
	}
	stats.PrefixBytes = prefix.Len()
	assets := make(map[string]struct{}, len(state.Assets))
	for _, a := range state.Assets {
		assets[a.Hash] = struct{}{}
	}
	return &SketchSource{prefix: prefix.Bytes(), state: state, begin: *owned, stats: stats, assets: assets}, nil
}

type sceneCounter struct{ n, limit int }

func (w *sceneCounter) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.n {
		return 0, ErrSketchSourceStateLimit
	}
	w.n += len(p)
	return len(p), nil
}

// Account each node and sheet once. Canonical's recursive sheet lookup is
// deliberately avoided here; source preparation is linear in scene size.
func sketchSceneSize(s *State, limit int) (SketchSourceStats, error) {
	w := &sceneCounter{limit: limit}
	e := json.NewEncoder(w)
	stats := SketchSourceStats{}
	id := func(n *DOMNode) NodeID {
		if n == nil {
			return 0
		}
		return n.ID
	}
	for _, n := range s.nodes {
		kids := make([]NodeID, len(n.Kids))
		for i, k := range n.Kids {
			kids[i] = k.ID
		}
		v := struct {
			ID                     NodeID
			Y, N, NS, V, Mode, URL string
			A                      map[string]string
			Kids                   []NodeID
			Shadow, Content, Frame NodeID
		}{n.ID, n.Y, n.N, n.NS, n.V, n.Mode, n.URL, n.A, kids, id(n.Shadow), id(n.Content), id(n.Frame)}
		if err := e.Encode(v); err != nil {
			return stats, err
		}
		stats.Nodes++
	}
	for key, sheet := range s.Sheets {
		if err := e.Encode(struct {
			ID    NodeID
			Sheet *SheetState
		}{key, sheet}); err != nil {
			return stats, err
		}
		stats.Sheets++
	}
	// Other scene inputs have no backreferences; history and annotation state
	// are independently bounded by the encoded prefix and sketch budgets.
	for _, v := range []any{s.VP, s.Tab, s.URL, s.Title, s.Adopted, s.Fonts, s.Assets, s.Inputs, s.Scrolls, s.Selection, s.Focus, s.LastClick, s.Hover, s.Ptr, s.Boxes, s.Control} {
		if err := e.Encode(v); err != nil {
			return stats, err
		}
	}
	stats.StateBytes = w.n
	return stats, nil
}
