package recorder

import (
	"encoding/json"
	"fmt"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// An observer report names nodes by path: child indexes from its frame's
// document (childNodes order, which is CDP's with whitespace included), with
// shadowStep entering an element's open shadow root. The mirror resolves a
// path against its own tree, with no CDP round trip, so a click whose
// handler navigates resolves before the navigation's events apply.
const shadowStep = -1

// Item is one observer report (observer.js).
type Item struct {
	K string `json:"k"`
	// P is the node's path; nil for none (focus or hover leaving), empty
	// for the document itself.
	P *[]int `json:"p"`
	// X, Y are viewport coordinates in the reporting frame (click, ptr) or a
	// scroll offset (scroll).
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// Form control state; Secret for a password, which never reports V.
	V       *string `json:"v"`
	Checked *bool   `json:"checked"`
	Sel     []int   `json:"sel"`
	Secret  bool    `json:"secret"`
	// Selection ends: [path, offset]; both nil clears it.
	A *pathOffset `json:"a"`
	F *pathOffset `json:"f"`
	// Viewport.
	W   int     `json:"w"`
	H   int     `json:"h"`
	DPR float64 `json:"dpr"`
	// Text is a CSSOM-only stylesheet's rules (k "sheet", P its <style>).
	Text string `json:"text"`
}

type pathOffset struct {
	P   []int
	Off int64
}

func (p *pathOffset) UnmarshalJSON(b []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &p.P); err != nil {
		return err
	}
	return json.Unmarshal(raw[1], &p.Off)
}

// frameDoc is the document node of a frame in a session: the session's own,
// or an in-process iframe's content document.
func (s *msession) frameDoc(frameID string) *cnode {
	if s.root == nil {
		return nil
	}
	if s.frameID == frameID || frameID == "" {
		return s.root
	}
	for _, n := range s.nodes {
		if n.NodeType == 1 && n.FrameID == frameID && n.ContentDocument != nil {
			return n.ContentDocument
		}
	}
	return nil
}

// resolve walks a path from a document. Any step that doesn't exist
// resolves to nil: a stale or malformed path names nothing.
func resolve(doc *cnode, path []int) *cnode {
	n := doc
	for _, step := range path {
		if n == nil {
			return nil
		}
		if step == shadowStep {
			var open *cnode
			for _, r := range n.ShadowRoots {
				if r.ShadowRootType == "open" {
					open = r
				}
			}
			n = open
			continue
		}
		if step < 0 || step >= len(n.Children) {
			return nil
		}
		n = n.Children[step]
	}
	return n
}

// errStale: a report names a node the mirror doesn't have (it changed
// since). Counted, never guessed.
var errStale = fmt.Errorf("recorder: observer path names no recorded node")

// nodeAt resolves a path to a recorded node's id.
func (m *mirror) nodeAt(s *msession, frameID string, path []int) (rec.NodeID, *cnode, error) {
	n := resolve(s.frameDoc(frameID), path)
	if n == nil || !m.visible(s, n) {
		return 0, nil, errStale
	}
	id, ok := s.ids[n.BackendNodeID]
	if !ok || m.st.Node(id) == nil {
		return 0, nil, errStale
	}
	return id, n, nil
}

// TopScroll is the top document's scroll offset: a top-frame viewport
// point plus this is a top-page point.
func (m *mirror) TopScroll() [2]float64 {
	if m.st.Top == nil {
		return [2]float64{}
	}
	return m.st.Scrolls[m.st.Top.ID]
}

// Observe turns one observer report from a frame of a session into a
// record. off maps the frame's viewport into top-page coordinates (the
// shell computes it for a subframe; TopScroll for the top frame). Stale
// reports are counted in Desync and dropped.
func (m *mirror) Observe(t int64, session, frameID string, it Item, off [2]float64) error {
	s := m.sessions[session]
	if s == nil || s.root == nil {
		return nil
	}
	at := func() (rec.NodeID, *cnode, error) {
		if it.P == nil {
			return 0, nil, errStale
		}
		return m.nodeAt(s, frameID, *it.P)
	}
	stale := func() error {
		m.Desync++
		m.note(fmt.Sprintf("t=%d observer %s: stale path", t, it.K))
		return errStale
	}
	switch it.K {
	case "click":
		id, _, err := at()
		if err != nil {
			return stale()
		}
		m.emit(t, &rec.Click{ID: id, X: it.X + off[0], Y: it.Y + off[1]})
	case "input":
		id, n, err := at()
		if err != nil {
			return stale()
		}
		in := &rec.Input{ID: id, Checked: it.Checked, Sel: it.Sel}
		switch {
		case isPassword(n):
			// Whatever the report says, a password's value is never kept.
			in.Checked, in.Sel = nil, nil
			if it.Secret || (it.V != nil && *it.V != "") {
				in.Secret = true
			} else {
				in.V = rec.Str("")
			}
		case it.Secret:
			// The observer saw a password the mirror doesn't yet: keep it secret.
			in.Secret = true
		default:
			in.V = it.V
		}
		m.emit(t, in)
	case "scroll":
		id, n, err := at()
		if err != nil {
			return stale()
		}
		// The document element's scroll is the viewport's: recorded on the
		// document.
		if n.NodeType == 1 && n.LocalName == "html" && n.parent != nil && n.parent.NodeType == 9 {
			if did, ok := s.ids[n.parent.BackendNodeID]; ok {
				id = did
			}
		}
		m.emit(t, &rec.Scroll{ID: id, X: it.X, Y: it.Y})
	case "sel":
		r := &rec.Selection{}
		for _, end := range []struct {
			po  *pathOffset
			dst **rec.Pos
		}{{it.A, &r.Anchor}, {it.F, &r.Focus}} {
			if end.po == nil {
				continue
			}
			id, _, err := m.nodeAt(s, frameID, end.po.P)
			if err != nil {
				return stale()
			}
			*end.dst = &rec.Pos{int64(id), end.po.Off}
		}
		m.emit(t, r)
	case "focus", "hover":
		var id rec.NodeID
		if it.P != nil {
			var err error
			if id, _, err = at(); err != nil {
				return stale()
			}
		}
		if it.K == "focus" {
			m.emit(t, &rec.Focus{ID: id})
		} else if id != m.st.Hover {
			m.emit(t, &rec.Hover{ID: id})
		}
	case "ptr":
		m.emit(t, &rec.Ptr{X: it.X + off[0], Y: it.Y + off[1]})
	case "vp":
		if s.top && s.frameDoc(frameID) == s.root && it.W > 0 && it.H > 0 {
			vp := rec.Viewport{W: it.W, H: it.H, DPR: it.DPR}
			if vp != m.st.VP {
				m.emit(t, &rec.VP{Viewport: vp})
			}
		}
	case "sheet":
		_, n, err := at()
		if err != nil {
			return stale()
		}
		m.OwnedSheet(t, session, n.BackendNodeID, it.Text)
	}
	return nil
}
