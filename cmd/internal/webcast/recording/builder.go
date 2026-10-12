package recording

// Builder writes a recording by hand: ids from one counter, records at the
// current time. For tests and hand-made fixtures; a recorder emits records
// itself.
type Builder struct {
	T    int64
	Recs []Record
	next NodeID
	seg  int
}

// NewBuilder starts a recording with its head.
func NewBuilder(id string) *Builder {
	b := &Builder{}
	b.Add(&Head{Format: Format, V: Version, ID: id, Started: "2026-10-11T00:00:00Z"})
	return b
}

// ID allocates the next id.
func (b *Builder) ID() NodeID { b.next++; return b.next }

// At moves the clock to t.
func (b *Builder) At(t int64) *Builder { b.T = t; return b }

// Add appends a record at the current time.
func (b *Builder) Add(r Record) *Builder {
	r.header().T = b.T
	r.header().K = r.Kind()
	b.Recs = append(b.Recs, r)
	return b
}

// El is a new element; attrs alternate name, value.
func (b *Builder) El(tag string, attrs []string, kids ...Node) Node {
	n := Node{ID: b.ID(), Y: YEl, N: tag, C: kids}
	if len(attrs) > 0 {
		n.A = map[string]string{}
		for i := 0; i+1 < len(attrs); i += 2 {
			n.A[attrs[i]] = attrs[i+1]
		}
	}
	return n
}

// Txt is a new text node.
func (b *Builder) Txt(s string) Node { return Node{ID: b.ID(), Y: YText, V: Str(s)} }

// Page starts a segment showing a new top document whose body holds body,
// and returns the document's node (its html, head and body are C[1],
// C[1].C[0], C[1].C[1]).
func (b *Builder) Page(why, url string, body ...Node) Node {
	b.seg++
	b.Add(&Seg{Seg: b.seg, Why: why, Tab: Tab{ID: "tab1", URL: url}, VP: Viewport{W: 1280, H: 800, DPR: 1}})
	return b.Snapshot(url, b.El("html", nil, b.El("head", nil), b.El("body", nil, body...)))
}

// Snapshot starts a segment-less top doc snapshot (after a Seg the caller
// added) from an html element.
func (b *Builder) Snapshot(url string, html Node) Node {
	d := Node{ID: b.ID(), Y: YDoc, C: []Node{{ID: b.ID(), Y: YDoctype, N: "html"}, html}}
	b.Add(&Doc{Doc: d.ID, URL: url, Tree: d})
	return d
}

// NextSeg is the number the next Seg should carry.
func (b *Builder) NextSeg() int { b.seg++; return b.seg }
