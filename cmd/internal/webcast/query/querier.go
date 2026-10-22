package query

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Querier is the real Surface: the reference reducer over one recording's
// records, nothing indexed beside them. Its notes go through a notes.Model
// (the fake until #461's real model lands).
type Querier struct {
	recs  []recording.Record
	model notes.Model
	// PageLines is DOMAt's line budget; zero is DefaultPageLines.
	PageLines int
}

// New is a Querier over a recording's merged records (recording.ReadDir).
func New(recs []recording.Record, model notes.Model) *Querier {
	return &Querier{recs: recs, model: model}
}

// End is the time of the recording's last record.
func (q *Querier) End() int64 {
	if len(q.recs) == 0 {
		return 0
	}
	return q.recs[len(q.recs)-1].Time()
}

// DOMAt is the page at t: a header naming the page and moment, then each
// element the selector matches (the whole top document when it is empty).
func (q *Querier) DOMAt(t int64, selector string) (string, error) {
	st, roots, err := q.matchAt(t, selector)
	if err != nil {
		return "", err
	}
	budget := q.PageLines
	if budget <= 0 {
		budget = DefaultPageLines
	}
	var b strings.Builder
	b.WriteString(Header(st) + "\n")
	if selector != "" {
		fmt.Fprintf(&b, "%d match(es) for %s\n", len(roots), selector)
	}
	for i, r := range roots {
		if budget <= 0 {
			fmt.Fprintf(&b, "… %d more matches; narrow the selector\n", len(roots)-i)
			break
		}
		page := Page(st, r, budget)
		budget -= strings.Count(page, "\n")
		b.WriteString(page)
	}
	return b.String(), nil
}

// matchAt reduces to t and selects; no page or no match is an error.
func (q *Querier) matchAt(t int64, selector string) (*recording.State, []*recording.DOMNode, error) {
	st, err := recording.Reduce(q.recs, t)
	if err != nil {
		return nil, nil, fmt.Errorf("query: the recording doesn't read: %w", err)
	}
	if st.Top == nil {
		return nil, nil, fmt.Errorf("query: no page at t=%d (the recording starts its first page at t=%d)", t, q.firstDoc())
	}
	if selector == "" {
		return st, []*recording.DOMNode{st.Top}, nil
	}
	roots, err := Select(st, selector)
	if err != nil {
		return nil, nil, err
	}
	if len(roots) == 0 {
		return nil, nil, fmt.Errorf("query: nothing matches %q at t=%d", selector, t)
	}
	return st, roots, nil
}

func (q *Querier) firstDoc() int64 {
	for _, r := range q.recs {
		if _, ok := r.(*recording.Doc); ok {
			return r.Time()
		}
	}
	return -1
}

// Header is one line naming the page and moment of a state.
func Header(st *recording.State) string {
	phase := "live"
	switch {
	case st.Ended:
		phase = "ended"
	case !st.Live:
		phase = "paused"
	}
	h := fmt.Sprintf("t=%d  segment %d (%s)  %s  %s", st.T, st.Seg, phase, st.URL, viewport(st.VP))
	if st.Title != "" {
		h += "  title " + fmt.Sprintf("%q", squash(st.Title, 80))
	}
	if d := st.Control.Driving(); d != "" {
		h += "  driven by " + who(st.Control.Driver.ID, st.Control.Driver.Name)
	}
	return h
}

func viewport(v recording.Viewport) string {
	return fmt.Sprintf("viewport %dx%d@%s", v.W, v.H, num(v.DPR))
}

func (q *Querier) Events(t1, t2 int64) ([]recording.Record, error) {
	if err := checkWindow(t1, t2); err != nil {
		return nil, err
	}
	var out []recording.Record
	for _, r := range q.recs {
		if r.Time() > t2 {
			break
		}
		if r.Time() >= t1 {
			out = append(out, r)
		}
	}
	return out, nil
}

// BeforeLines is how many lines of actions a note shows before it (the
// latest ones); the records stay whole in NoteView.Before.
const BeforeLines = 20

// Notes is one pass over the records: each note is read at its own moment
// (its element, the model's focused context, the actions before it) and
// carries its final state, since an agent asks whether it is resolved now.
func (q *Querier) Notes(before int64) ([]NoteView, error) {
	nar := newNarrator()
	st := nar.st
	words := map[int]said{} // the user actions so far, in words, by index
	var out []NoteView
	for i, r := range q.recs {
		if UserAction(r) {
			words[i] = nar.say(r)
		}
		if err := nar.apply(r); err != nil {
			return nil, unreadable(i, r, err)
		}
		_, ok := r.(*recording.Note)
		if !ok {
			continue
		}
		v, err := q.noteContext(st, st.Notes[len(st.Notes)-1], i, before, words)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ContextForNote derives a frozen note's contextual evidence at a separate draw
// moment. It does not add a record or derive contexts for other recorded notes.
func (q *Querier) ContextForNote(note *recording.Note, at, before int64) (NoteView, error) {
	if note == nil || at < 0 || at > note.T {
		return NoteView{}, fmt.Errorf("invalid note context time")
	}
	copy := *note
	copy.SetTime(at)
	nar := newNarrator()
	words := map[int]said{}
	end := 0
	for i, r := range q.recs {
		if r.Time() > at {
			break
		}
		if UserAction(r) {
			words[i] = nar.say(r)
		}
		if err := nar.apply(r); err != nil {
			return NoteView{}, unreadable(i, r, err)
		}
		end = i + 1
	}
	return q.noteContext(nar.st, &recording.NoteView{Note: &copy, Seg: nar.st.Seg}, end, before, words)
}

func (q *Querier) noteContext(st *recording.State, view *recording.NoteView, i int, before int64, words map[int]said) (NoteView, error) {
	n := view.Note
	v := NoteView{Note: view, URL: st.URL, Header: Header(st), Elements: noteElementFacts(st, n)}
	if node, ok := q.model.Resolve(st, n.At); ok {
		at := n.At
		at.Node = node
		v.Element = Page(st, st.Node(node), ElementLines)
		ctx, err := q.model.Context(st, at)
		if err != nil {
			return NoteView{}, err
		}
		v.Context, v.Landmark = ctx, Describe(st.Node(ctx.Landmark))
	}
	var items []said
	for j := i - 1; j >= 0 && q.recs[j].Time() >= n.T-before; j-- {
		if UserAction(q.recs[j]) {
			v.Before = append(v.Before, q.recs[j])
			items = append(items, words[j])
		}
	}
	slices.Reverse(v.Before)
	slices.Reverse(items)
	v.BeforeLines = fold(items)
	if k := len(v.BeforeLines) - BeforeLines; k > 0 {
		v.BeforeLines = append([]string{fmt.Sprintf("… %d earlier actions", k)}, v.BeforeLines[k:]...)
	}
	return v, nil
}

func noteElementFacts(st *recording.State, n *recording.Note) DeclaredElementFacts {
	region := n.Region
	if region == nil {
		region = n.At.Box
	}
	var anchors []recording.NodeID
	if n.At.Node > 0 {
		anchors = []recording.NodeID{n.At.Node}
	}
	return DeclaredElementFactsAt(st, anchors, region)
}
