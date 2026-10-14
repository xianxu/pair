package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// said is one record in words at its time. Records with the same non-empty
// key in a row fold into one line: a field typed a keystroke at a time is
// one value, not one line per key. The key carries the viewer, so one
// person's typing never folds into another's. A collapsed caret after a
// typed value is that typing's own cursor (the recorder reports one per
// key), so it folds into the value instead of splitting it.
type said struct {
	t, t0 int64
	key   string
	caret bool // a collapsed caret, by by
	by    string
	text  string
	n     int
}

func (n *narrator) say(r recording.Record) said {
	s := said{t: r.Time(), t0: r.Time(), text: n.line(r), n: 1}
	// A typed value often lands twice per key, as the property and as the
	// framework's value attribute; both are the one value being typed.
	switch r := r.(type) {
	case *recording.Input:
		s.key = fmt.Sprintf("input %d %s", r.ID, r.By)
		if r.V != nil && r.Checked == nil && r.Sel == nil {
			s.key = fmt.Sprintf("value %d %s", r.ID, r.By)
		}
	case *recording.Attr:
		if r.Name == "value" {
			s.key = fmt.Sprintf("value %d ", r.ID)
		}
	case *recording.Selection:
		if r.Anchor != nil && r.Focus != nil && *r.Anchor == *r.Focus {
			s.caret, s.by = true, r.By
		}
	}
	return s
}

// fold merges same-key neighbours (keeping the last's words) and writes
// each as a line with its time, or its time span and count.
func fold(items []said) []string {
	var merged []said
	for _, it := range items {
		if k := len(merged) - 1; k >= 0 && it.caret && typedBy(merged[k].key, it.by) {
			continue
		}
		if k := len(merged) - 1; k >= 0 && it.key != "" && merged[k].key == it.key {
			it.t0, it.n = merged[k].t0, merged[k].n+1
			merged[k] = it
			continue
		}
		merged = append(merged, it)
	}
	out := make([]string, len(merged))
	for i, m := range merged {
		if m.n > 1 {
			out[i] = fmt.Sprintf("%d-%d %s (%d changes)", m.t0, m.t, m.text, m.n)
		} else {
			out[i] = fmt.Sprintf("%d %s", m.t, m.text)
		}
	}
	return out
}

// typedBy says whether key is a typed value's, by the person by is.
func typedBy(key, by string) bool {
	return strings.HasPrefix(key, "value ") && strings.HasSuffix(key, " "+by)
}

// unreadable is the error for a record the reducer refuses: the recording
// can't answer past it.
func unreadable(i int, r recording.Record, err error) error {
	return fmt.Errorf("query: the recording doesn't read: record %d (%s at t=%d): %w", i, r.Kind(), r.Time(), err)
}

// checkWindow refuses a window that ends before it starts.
func checkWindow(t1, t2 int64) error {
	if t2 < t1 {
		return fmt.Errorf("query: window %d..%d is empty", t1, t2)
	}
	return nil
}

// shortRun is the longest run of DOM changes written out one per line; a
// longer one is folded into a single summary line.
const shortRun = 3

// narrator reads records in order and says what each one did, naming its
// target as it was at that moment: a record is described against the state
// before it applies (so a removed node is still named), then applied.
type narrator struct {
	st *recording.State
	// names are viewer names by id, from ctl records, so "by" reads as a
	// person and not an id.
	names map[string]string
}

func newNarrator() *narrator {
	return &narrator{st: recording.NewState(), names: map[string]string{}}
}

func (n *narrator) apply(r recording.Record) error {
	if c, ok := r.(*recording.Ctl); ok && c.Who.Name != "" {
		n.names[c.Who.ID] = c.Who.Name
	}
	return n.st.Apply(r)
}

// domChange reports the records a run of DOM changes is folded from.
func domChange(r recording.Record) bool {
	switch r.(type) {
	case *recording.Ins, *recording.Rm, *recording.Kids, *recording.Attr, *recording.Text, *recording.Shadow,
		*recording.Sheet, *recording.Adopt, *recording.Font, *recording.Asset, *recording.Layout:
		return true
	}
	return false
}

func (n *narrator) el(id recording.NodeID) string { return Describe(n.st.Node(id)) }

func (n *narrator) by(s recording.Source) string {
	if s.By == "" {
		return ""
	}
	return " by " + who(s.By, n.names[s.By])
}

func who(id, name string) string {
	if name == "" {
		return id
	}
	return fmt.Sprintf("%s (%s)", name, id)
}

// line is one record in words, without its time.
func (n *narrator) line(r recording.Record) string {
	switch r := r.(type) {
	case *recording.Head:
		return fmt.Sprintf("recording %s started %s", r.ID, r.Started)
	case *recording.Seg:
		return fmt.Sprintf("segment %d starts (%s) on %s, %s", r.Seg, r.Why, r.Tab.URL, viewport(r.VP))
	case *recording.Stop:
		return "stop (" + r.Why + ")"
	case *recording.Doc:
		if r.Host != 0 {
			return fmt.Sprintf("frame document %s in %s", r.URL, n.el(r.Host))
		}
		return "document " + r.URL
	case *recording.Click:
		return fmt.Sprintf("click %s at %s,%s%s", n.el(r.ID), num(r.X), num(r.Y), n.by(r.Source))
	case *recording.Input:
		var parts []string
		if r.V != nil {
			parts = append(parts, "value "+strconv.Quote(trim(*r.V, textChars)))
		}
		if r.Checked != nil {
			parts = append(parts, map[bool]string{true: "checked", false: "unchecked"}[*r.Checked])
		}
		if r.Sel != nil {
			parts = append(parts, fmt.Sprintf("selected %v", r.Sel))
		}
		return fmt.Sprintf("input %s %s%s", n.el(r.ID), strings.Join(parts, ", "), n.by(r.Source))
	case *recording.Focus:
		if r.ID == 0 {
			return "focus none" + n.by(r.Source)
		}
		return "focus " + n.el(r.ID) + n.by(r.Source)
	case *recording.Scroll:
		return fmt.Sprintf("scroll %s to %s,%s%s", n.el(r.ID), num(r.X), num(r.Y), n.by(r.Source))
	case *recording.Selection:
		if r.Anchor == nil && r.Focus == nil {
			return "selection cleared" + n.by(r.Source)
		}
		return fmt.Sprintf("selection %s … %s%s", n.pos(r.Anchor), n.pos(r.Focus), n.by(r.Source))
	case *recording.Nav:
		return "navigate (same document) to " + r.URL
	case *recording.Title:
		return "title " + strconv.Quote(squash(r.V, 80))
	case *recording.VP:
		return viewport(r.Viewport)
	case *recording.Note:
		return fmt.Sprintf("note %s by %s on %s: %s", r.ID, who(r.Author.ID, r.Author.Name), n.el(r.At.Node), strconv.Quote(squash(r.Text, textChars)))
	case *recording.NoteEdit:
		return fmt.Sprintf("note %s edited: %s", r.ID, strconv.Quote(squash(r.Text, textChars)))
	case *recording.NoteState:
		if r.State == recording.NoteResolved && r.Who != nil {
			return fmt.Sprintf("note %s resolved by %s", r.ID, who(r.Who.ID, r.Who.Name))
		}
		return fmt.Sprintf("note %s %s", r.ID, r.State)
	case *recording.Stroke:
		return fmt.Sprintf("pointer stroke by %s (%d points)", who(r.Author.ID, r.Author.Name), len(r.Pts))
	case *recording.Ctl:
		after := "the operator drives"
		if r.Driver != nil {
			after = who(r.Driver.ID, r.Driver.Name) + " drives"
		}
		if len(r.Pending) > 0 {
			var ps []string
			for _, p := range r.Pending {
				ps = append(ps, who(p.ID, p.Name))
			}
			after += "; asking: " + strings.Join(ps, ", ")
		}
		return fmt.Sprintf("control %s %s → %s", r.Op, who(r.Who.ID, r.Who.Name), after)
	case *recording.Attr:
		if r.V == nil {
			return fmt.Sprintf("attr %s %s removed", n.el(r.ID), r.Name)
		}
		return fmt.Sprintf("attr %s %s=%s", n.el(r.ID), r.Name, strconv.Quote(trim(*r.V, attrChars)))
	case *recording.Text:
		return fmt.Sprintf("text in %s → %s", n.el(r.ID), strconv.Quote(squash(r.V, describeChars)))
	case *recording.Ins:
		return fmt.Sprintf("insert %s under %s", nodeShape(&r.Node), n.el(r.P))
	case *recording.Rm:
		return "remove " + n.el(r.ID)
	case *recording.Kids:
		return fmt.Sprintf("replace the children of %s (%d nodes)", n.el(r.P), len(r.C))
	case *recording.Shadow:
		return fmt.Sprintf("attach a %s shadow root to %s", r.Root.Mode, n.el(r.Host))
	case *recording.Sheet:
		if r.Owner != 0 {
			return fmt.Sprintf("stylesheet of %s (%d chars)", n.el(r.Owner), len(r.Text))
		}
		return fmt.Sprintf("constructed stylesheet @%d (%d chars)", r.ID, len(r.Text))
	case *recording.Adopt:
		return fmt.Sprintf("adopted stylesheets of %s: %d", n.el(r.ID), len(r.Sheets))
	case *recording.Font:
		return "font " + r.Family
	case *recording.Asset:
		return fmt.Sprintf("asset %s (%s, %d bytes)", r.URL, r.MIME, r.Size)
	case *recording.Layout:
		return fmt.Sprintf("layout boxes for %d nodes", len(r.Boxes))
	}
	return "? " + r.Kind()
}

func (n *narrator) pos(p *recording.Pos) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprintf("%s:%d", n.el(recording.NodeID(p[0])), p[1])
}

func nodeShape(n *recording.Node) string {
	switch n.Y {
	case recording.YEl:
		s := fmt.Sprintf("@%d <%s", n.ID, n.N)
		if id := n.A["id"]; id != "" {
			s += "#" + id
		}
		return s + ">"
	case recording.YText:
		v := ""
		if n.V != nil {
			v = *n.V
		}
		return fmt.Sprintf("@%d text %s", n.ID, strconv.Quote(squash(v, describeChars)))
	}
	return fmt.Sprintf("@%d %s", n.ID, n.Y)
}

// target is the element a DOM change is counted against in a summary.
func (n *narrator) target(r recording.Record) (recording.NodeID, bool) {
	switch r := r.(type) {
	case *recording.Ins:
		return r.P, true
	case *recording.Kids:
		return r.P, true
	case *recording.Rm:
		if e := n.st.Node(r.ID); e != nil {
			if p := e.ParentElement(); p != nil {
				return p.ID, true
			}
		}
		return r.ID, true
	case *recording.Attr:
		return r.ID, true
	case *recording.Text:
		if e := n.st.Node(r.ID); e != nil {
			if p := e.ParentElement(); p != nil {
				return p.ID, true
			}
		}
		return r.ID, true
	case *recording.Shadow:
		return r.Host, true
	case *recording.Sheet:
		return r.Owner, r.Owner != 0
	case *recording.Adopt:
		return r.ID, true
	}
	return 0, false
}

// run is a fold of consecutive DOM changes.
type run struct {
	t1, t2  int64
	lines   []said // written out while short
	kinds   map[string]int
	targets map[recording.NodeID]int
	names   map[recording.NodeID]string
	n       int
}

func (u *run) summary() string {
	kinds := make([]string, 0, len(u.kinds))
	for k := range u.kinds {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool {
		if u.kinds[kinds[i]] != u.kinds[kinds[j]] {
			return u.kinds[kinds[i]] > u.kinds[kinds[j]]
		}
		return kinds[i] < kinds[j]
	})
	var ks []string
	for _, k := range kinds {
		ks = append(ks, fmt.Sprintf("%d %s", u.kinds[k], k))
	}
	ids := make([]recording.NodeID, 0, len(u.targets))
	for id := range u.targets {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if u.targets[ids[i]] != u.targets[ids[j]] {
			return u.targets[ids[i]] > u.targets[ids[j]]
		}
		return ids[i] < ids[j]
	})
	s := fmt.Sprintf("dom changes until %d: %s", u.t2, strings.Join(ks, ", "))
	if len(ids) > 0 {
		var ts []string
		for i, id := range ids {
			if i == 3 {
				ts = append(ts, fmt.Sprintf("%d more elements", len(ids)-3))
				break
			}
			ts = append(ts, fmt.Sprintf("%s ×%d", u.names[id], u.targets[id]))
		}
		s += "; on " + strings.Join(ts, ", ")
	}
	return s
}

// EventLines tells what happened from t1 to t2 (both included), one line per
// record with its time, except that each run of more than shortRun
// consecutive DOM, style, asset or layout records is one summary line. The
// records before t1 are read too, so every target is named as it was.
func EventLines(recs []recording.Record, t1, t2 int64) ([]string, error) {
	if err := checkWindow(t1, t2); err != nil {
		return nil, err
	}
	n := newNarrator()
	var out []said
	var cur *run
	flush := func() {
		if cur == nil {
			return
		}
		if cur.n <= shortRun {
			out = append(out, cur.lines...)
		} else {
			out = append(out, said{t: cur.t1, t0: cur.t1, text: cur.summary(), n: 1})
		}
		cur = nil
	}
	for i, r := range recs {
		if r.Time() > t2 {
			break
		}
		in := r.Time() >= t1
		if in && domChange(r) {
			if cur == nil {
				cur = &run{t1: r.Time(), kinds: map[string]int{}, targets: map[recording.NodeID]int{}, names: map[recording.NodeID]string{}}
			}
			cur.t2, cur.n = r.Time(), cur.n+1
			cur.kinds[r.Kind()]++
			if id, ok := n.target(r); ok {
				if _, seen := cur.names[id]; !seen {
					cur.names[id] = n.el(id)
				}
				cur.targets[id]++
			}
			if cur.n <= shortRun {
				cur.lines = append(cur.lines, n.say(r))
			}
		} else if in {
			flush()
			out = append(out, n.say(r))
		}
		if err := n.apply(r); err != nil {
			return nil, unreadable(i, r, err)
		}
	}
	flush()
	return fold(out), nil
}
