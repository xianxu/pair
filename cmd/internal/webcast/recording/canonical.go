package recording

import "encoding/json"

// Canonical is the state as a deterministic JSON document: the shape the
// viewer's canonical() must reproduce from what it rebuilt (the viewer
// conformance check), and the fixtures' golden form.
//
// The DOM is one tree from the top document: an iframe's document hangs
// under its element as "frame", a shadow root as "sr", a template's content
// as "tc". What isn't DOM is folded into the node it belongs to: a form
// control's state as "$input", a scroll offset as "$scroll", an owner
// element's stylesheet text as "$sheet", adopted sheets' texts as "$adopt".
func (s *State) Canonical() []byte {
	c := canonState{Seg: s.Seg, Live: s.Live, URL: s.URL, Title: s.Title, VP: s.VP, Focus: s.Focus}
	if s.Top != nil {
		c.Doc = s.canonNode(s.Top)
	}
	for _, f := range s.Fonts {
		c.Fonts = append(c.Fonts, [2]string{f.Family, f.Src})
	}
	if s.Selection != nil {
		c.Selection = []*Pos{s.Selection.Anchor, s.Selection.Focus}
	}
	b, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		panic(err) // plain data: can't fail
	}
	return b
}

type canonState struct {
	Seg       int         `json:"seg"`
	Live      bool        `json:"live"`
	URL       string      `json:"url"`
	Title     string      `json:"title"`
	VP        Viewport    `json:"vp"`
	Focus     NodeID      `json:"focus,omitempty"`
	Selection []*Pos      `json:"selection,omitempty"`
	Fonts     [][2]string `json:"fonts,omitempty"`
	Doc       *canonNode  `json:"doc"`
}

type canonNode struct {
	ID     NodeID            `json:"id"`
	Y      string            `json:"y"`
	N      string            `json:"n,omitempty"`
	NS     string            `json:"ns,omitempty"`
	URL    string            `json:"url,omitempty"`
	A      map[string]string `json:"a,omitempty"`
	V      *string           `json:"v,omitempty"`
	Mode   string            `json:"mode,omitempty"`
	Input  *canonInput       `json:"$input,omitempty"`
	Scroll *[2]float64       `json:"$scroll,omitempty"`
	Sheet  *string           `json:"$sheet,omitempty"`
	Adopt  []string          `json:"$adopt,omitempty"`
	SR     *canonNode        `json:"sr,omitempty"`
	TC     *canonNode        `json:"tc,omitempty"`
	Frame  *canonNode        `json:"frame,omitempty"`
	C      []*canonNode      `json:"c,omitempty"`
}

type canonInput struct {
	V       *string `json:"v,omitempty"`
	Checked *bool   `json:"checked,omitempty"`
	Sel     []int   `json:"sel,omitempty"`
	Secret  bool    `json:"secret,omitempty"`
}

func (s *State) canonNode(n *DOMNode) *canonNode {
	c := &canonNode{ID: n.ID, Y: n.Y, N: n.N, NS: n.NS, URL: n.URL, A: n.A, Mode: n.Mode}
	if n.Y == YText || n.Y == YComment {
		c.V = &n.V
	}
	if in := s.Inputs[n.ID]; in != nil {
		c.Input = &canonInput{V: in.V, Checked: in.Checked, Sel: in.Sel, Secret: in.Secret}
	}
	if sc, ok := s.Scrolls[n.ID]; ok {
		c.Scroll = &sc
	}
	// The owner's sheet: the newest one, should a page have made two.
	var newest NodeID
	for id, sh := range s.Sheets {
		if sh.Owner == n.ID && id > newest {
			newest, c.Sheet = id, &sh.Text
		}
	}
	for _, id := range s.Adopted[n.ID] {
		if sh := s.Sheets[id]; sh != nil {
			c.Adopt = append(c.Adopt, sh.Text)
		}
	}
	if n.Shadow != nil {
		c.SR = s.canonNode(n.Shadow)
	}
	if n.Content != nil {
		c.TC = s.canonNode(n.Content)
	}
	if n.Frame != nil {
		c.Frame = s.canonNode(n.Frame)
	}
	for _, k := range n.Kids {
		c.C = append(c.C, s.canonNode(k))
	}
	return c
}
