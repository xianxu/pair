package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// FakeSurface is the stateful fake Surface, backed by the reference reducer.
// Its selector is #id only, and its text is a plain indented outline.
type FakeSurface struct {
	Records []recording.Record
	Model   notes.Model
}

func (f FakeSurface) DOMAt(t int64, selector string) (string, error) {
	st, err := recording.Reduce(f.Records, t)
	if err != nil {
		return "", err
	}
	if st.Top == nil {
		return "", fmt.Errorf("query: no page at t=%d", t)
	}
	root := st.Top
	if selector != "" {
		id, ok := strings.CutPrefix(selector, "#")
		if !ok {
			return "", fmt.Errorf("query: the fake supports #id selectors only, not %q", selector)
		}
		if root = byID(st.Top, id); root == nil {
			return "", fmt.Errorf("query: nothing matches %q at t=%d", selector, t)
		}
	}
	return Outline(root), nil
}

func (f FakeSurface) Events(t1, t2 int64) ([]recording.Record, error) {
	if err := checkWindow(t1, t2); err != nil {
		return nil, err
	}
	var out []recording.Record
	for _, r := range f.Records {
		if r.Time() >= t1 && r.Time() <= t2 {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f FakeSurface) Notes(before int64) ([]NoteView, error) {
	if len(f.Records) == 0 {
		return nil, nil
	}
	st := recording.NewState()
	var out []NoteView
	for i, r := range f.Records {
		if err := st.Apply(r); err != nil {
			return nil, err
		}
		n, ok := r.(*recording.Note)
		if !ok {
			continue
		}
		v := NoteView{PageLoad: PageLoadFactsOf(st.PageLoadAt(st.Notes[len(st.Notes)-1].Seg)), Note: st.Notes[len(st.Notes)-1], Elements: noteElementFacts(st, n)}
		if el := st.Node(n.At.Node); el != nil {
			v.Element = Outline(el)
			ctx, err := f.Model.Context(st, n.At)
			if err != nil {
				return nil, err
			}
			v.Context = ctx
		}
		for _, p := range f.Records[:i] {
			if p.Time() >= n.T-before && UserAction(p) {
				v.Before = append(v.Before, p)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func byID(n *recording.DOMNode, id string) *recording.DOMNode {
	if n.Y == recording.YEl && n.A["id"] == id {
		return n
	}
	for _, k := range n.Kids {
		if f := byID(k, id); f != nil {
			return f
		}
	}
	return nil
}

// Outline is a node as an indented outline: one line per element with its
// attributes, text on its own line.
func Outline(n *recording.DOMNode) string {
	var b strings.Builder
	var walk func(n *recording.DOMNode, depth int)
	walk = func(n *recording.DOMNode, depth int) {
		pad := strings.Repeat("  ", depth)
		switch n.Y {
		case recording.YEl:
			names := make([]string, 0, len(n.A))
			for k := range n.A {
				names = append(names, k)
			}
			sort.Strings(names)
			b.WriteString(pad + "<" + n.N)
			for _, k := range names {
				fmt.Fprintf(&b, " %s=%q", k, n.A[k])
			}
			b.WriteString(">\n")
		case recording.YText:
			if t := strings.TrimSpace(n.V); t != "" {
				b.WriteString(pad + t + "\n")
			}
			return
		case recording.YDoc, recording.YShadow, recording.YFrag:
		default:
			return
		}
		for _, k := range n.Kids {
			walk(k, depth+1)
		}
	}
	walk(n, 0)
	return b.String()
}

// SketchesAt shares the pure history/fact projection; fake IO does not replace
// the recording's transition owner with a permissive sketch model.
func (f FakeSurface) SketchesAt(at int64, selected ...SketchSelection) ([]SketchView, error) {
	return New(f.Records, f.Model).SketchesAt(at, selected...)
}
