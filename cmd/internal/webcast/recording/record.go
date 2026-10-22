package recording

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// Format and Version are what a head record names. A reader refuses another
// major version; within one, unknown kinds and optional fields are ignored.
const (
	Format  = "couch-webcast"
	Version = 1
)

// NodeID names a node (or a stylesheet) for a whole recording: one counter
// per recording, never reused for a different thing. 0 is "none".
type NodeID int64

// Node types (the y field).
const (
	YDoc     = "doc"
	YDoctype = "doctype"
	YEl      = "el"
	YText    = "text"
	YComment = "comment"
	YShadow  = "shadow"
	YFrag    = "frag"
)

// Node is a wire node: a whole subtree, as a doc snapshot or an insert
// carries it. An iframe's content document is never inline; it comes as its
// own Doc record with Host set.
type Node struct {
	ID NodeID `json:"id"`
	Y  string `json:"y"`
	// N is the tag name (lower case for HTML), or the doctype's name.
	N  string            `json:"n,omitempty"`
	NS string            `json:"ns,omitempty"` // "svg" or "math"; empty is HTML
	A  map[string]string `json:"a,omitempty"`
	// V is a text or comment node's content.
	V    *string `json:"v,omitempty"`
	Mode string  `json:"mode,omitempty"` // a shadow root's: "open" or "closed"
	C    []Node  `json:"c,omitempty"`
	SR   *Node   `json:"sr,omitempty"` // an element's shadow root
	TC   *Node   `json:"tc,omitempty"` // a template's content (a frag)
}

// Box is [x, y, w, h] in CSS px of the top document's page.
type Box [4]float64

// Contains reports whether b holds all of o.
func (b Box) Contains(o Box) bool {
	return o[0] >= b[0] && o[1] >= b[1] && o[0]+o[2] <= b[0]+b[2] && o[1]+o[3] <= b[1]+b[3]
}

// Union is the smallest box holding b and o.
func (b Box) Union(o Box) Box {
	x0, y0 := min(b[0], o[0]), min(b[1], o[1])
	x1, y1 := max(b[0]+b[2], o[0]+o[2]), max(b[1]+b[3], o[1]+o[3])
	return Box{x0, y0, x1 - x0, y1 - y0}
}

// Tab is the cast tab a segment shows.
type Tab struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// Viewport is in CSS px, with the device pixel ratio.
type Viewport struct {
	W   int     `json:"w"`
	H   int     `json:"h"`
	DPR float64 `json:"dpr"`
}

// Author is who made a note or a stroke.
type Author struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
}

// Pos is a selection end: [node id, offset].
type Pos [2]int64

// Record is one line of a recording. Every kind embeds Header.
type Record interface {
	Kind() string
	Time() int64
	SetTime(int64)
	header() *Header
	check() error
}

// Header is the envelope every record carries.
type Header struct {
	T int64  `json:"t"`
	K string `json:"k"`
}

func (h *Header) Time() int64 { return h.T }

// SetTime places a record on the timeline (ms since the recording began).
func (h *Header) SetTime(t int64) { h.T = t }
func (h *Header) header() *Header { return h }

// need is a check's one line: nil when ok, else "missing f".
func need(ok bool, f string) error {
	if ok {
		return nil
	}
	return fmt.Errorf("missing %s", f)
}

// Stream control.
type (
	Head struct {
		Header
		Format  string `json:"format"`
		V       int    `json:"v"`
		ID      string `json:"id"`
		Started string `json:"started"`
		Enc     string `json:"enc,omitempty"`
	}
	// Seg starts a live segment. Why: SegLive, SegResume, SegNavigate, SegTab.
	Seg struct {
		Header
		Seg int      `json:"seg"`
		Why string   `json:"why"`
		Tab Tab      `json:"tab"`
		VP  Viewport `json:"vp"`
	}
	// Stop ends a segment. Why: StopPause, StopEnd.
	Stop struct {
		Header
		Why string `json:"why"`
	}
	// Doc is a full snapshot of one document: the top one (Host 0, right
	// after a Seg) or an iframe's (Host is the iframe element).
	Doc struct {
		Header
		Doc  NodeID `json:"doc"`
		Host NodeID `json:"host,omitempty"`
		URL  string `json:"url"`
		Tree Node   `json:"tree"`
	}
)

// Seg and Stop reasons.
const (
	SegLive     = "live"
	SegResume   = "resume"
	SegNavigate = "navigate"
	SegTab      = "tab"
	StopPause   = "pause"
	StopEnd     = "end"
)

// DOM changes.
type (
	// Ins inserts Node under P right after sibling After (0: first).
	Ins struct {
		Header
		P     NodeID `json:"p"`
		After NodeID `json:"after"`
		Node  Node   `json:"node"`
	}
	Rm struct {
		Header
		ID NodeID `json:"id"`
	}
	// Kids replaces all of P's children.
	Kids struct {
		Header
		P NodeID `json:"p"`
		C []Node `json:"c"`
	}
	// Attr sets an attribute; V nil removes it.
	Attr struct {
		Header
		ID   NodeID  `json:"id"`
		Name string  `json:"name"`
		V    *string `json:"v"`
	}
	// Text is a text or comment node's new content.
	Text struct {
		Header
		ID NodeID `json:"id"`
		V  string `json:"v"`
	}
	// Shadow attaches a shadow root to Host after its snapshot.
	Shadow struct {
		Header
		Host NodeID `json:"host"`
		Root Node   `json:"root"`
	}
)

// Styles, fonts, assets.
type (
	// Sheet is a stylesheet's complete text. Owner is the style or link
	// element whose CSSOM it is; 0 for a constructed sheet.
	Sheet struct {
		Header
		ID    NodeID `json:"id"`
		Owner NodeID `json:"owner,omitempty"`
		Text  string `json:"text"`
	}
	// Adopt is a document's or shadow root's adoptedStyleSheets.
	Adopt struct {
		Header
		ID     NodeID   `json:"id"`
		Sheets []NodeID `json:"sheets"`
	}
	Font struct {
		Header
		Family  string `json:"family"`
		Src     string `json:"src"`
		Style   string `json:"style,omitempty"`
		Weight  string `json:"weight,omitempty"`
		Stretch string `json:"stretch,omitempty"`
		Range   string `json:"range,omitempty"`
	}
	// Asset binds an absolute URL to the body stored under Hash.
	Asset struct {
		Header
		URL  string `json:"url"`
		Hash string `json:"hash"`
		MIME string `json:"mime"`
		Size int64  `json:"size"`
		Orig *Orig  `json:"orig,omitempty"`
	}
)

// Orig describes an image before it was resampled.
type Orig struct {
	Size int64 `json:"size"`
	W    int   `json:"w"`
	H    int   `json:"h"`
}

// Observer state.
type (
	// Input is a form control's state (properties, not attributes).
	// Secret says a password control has a value that is never recorded;
	// it never comes with V.
	Input struct {
		Header
		Source
		ID      NodeID  `json:"id"`
		V       *string `json:"v,omitempty"`
		Checked *bool   `json:"checked,omitempty"`
		Sel     []int   `json:"sel,omitempty"`
		Secret  bool    `json:"secret,omitempty"`
	}
	// Scroll is an element's scroll offset, or a document's viewport's.
	Scroll struct {
		Header
		Source
		ID NodeID  `json:"id"`
		X  float64 `json:"x"`
		Y  float64 `json:"y"`
	}
	// Selection is the text selection; neither end clears it.
	Selection struct {
		Header
		Source
		Anchor *Pos `json:"anchor,omitempty"`
		Focus  *Pos `json:"focus,omitempty"`
	}
)

// Context.
type (
	Nav struct {
		Header
		URL string `json:"url"`
	}
	Title struct {
		Header
		V string `json:"v"`
	}
	// VP is a viewport change mid-segment.
	VP struct {
		Header
		Viewport
	}
	Click struct {
		Header
		Source
		ID NodeID  `json:"id"`
		X  float64 `json:"x"`
		Y  float64 `json:"y"`
	}
	Focus struct {
		Header
		Source
		ID NodeID `json:"id"`
	}
	// Hover is the element under the operator's pointer; 0 is none.
	Hover struct {
		Header
		Source
		ID NodeID `json:"id"`
	}
	// Ptr is a sampled operator pointer position, in top-page CSS px.
	Ptr struct {
		Header
		Source
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	// Layout replaces the known layout boxes.
	Layout struct {
		Header
		Boxes map[NodeID]Box `json:"boxes"`
	}
)

// Notes, pointer, frames.
type (
	Note struct {
		Header
		// Owner is a public opaque digest, never an editing credential. Empty is legacy.
		Owner  string `json:"owner,omitempty"`
		Edited bool   `json:"edited,omitempty"`
		ID     string `json:"id"`
		Author Author `json:"author"`
		Text   string `json:"text"`
		At     NoteAt `json:"at"`
		Region *Box   `json:"region,omitempty"`
		// Capture is an opaque local reference; it never exposes a file path or pixels.
		Capture         string `json:"capture,omitempty"`
		CropUnavailable string `json:"crop_unavailable,omitempty"`
	}
	// NoteEdit changes text in place; the creation record and anchor remain immutable.
	NoteEdit struct {
		Header
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	NoteState struct {
		// Who is absent in legacy recordings; absence means unknown, not operator.
		Who *Author `json:"who,omitempty"`
		// IfRevision is an in-process writer guard, never a wire field.
		IfRevision *uint64 `json:"-"`
		Header
		ID    string `json:"id"`
		State string `json:"state"`
	}
	// Stroke is one pointer stroke; each point is [x, y, ms after T].
	Stroke struct {
		Header
		ID     string       `json:"id"`
		Author Author       `json:"author"`
		Color  string       `json:"color,omitempty"`
		Pts    [][3]float64 `json:"pts"`
	}
	// Frame is a screenshot of the real tab, for the fidelity check only.
	Frame struct {
		Header
		File  string  `json:"file"`
		W     int     `json:"w"`
		H     int     `json:"h"`
		Scale float64 `json:"scale"`
	}
)

// NoteAt is where a note points: the element, a stable selector, its text
// (at most 200 characters) and its box, at the note's time. Kind and Path
// are the anchor's shape shared with the terminal's notes (#488, which
// gives those a layout anchor of the same shape): kind "element", and the path of meaningful
// containers down to the element (e.g. main › article › p#first).
type NoteAt struct {
	Tag     string            `json:"tag,omitempty"`
	Classes []string          `json:"classes,omitempty"`
	Stable  map[string]string `json:"stable,omitempty"`
	Route   string            `json:"route,omitempty"`
	Node    NodeID            `json:"node"`
	Kind    string            `json:"kind,omitempty"`
	Path    []string          `json:"path,omitempty"`
	Sel     string            `json:"sel,omitempty"`
	Text    string            `json:"text,omitempty"`
	Box     *Box              `json:"box,omitempty"`
}

// AnchorElement is an element anchor's Kind.
const AnchorElement = "element"

// Note states.
const (
	NoteOpen     = "open"
	NoteResolved = "resolved"
)

// Unknown is a record of a kind this version doesn't know, kept verbatim.
type Unknown struct {
	Header
	Raw []byte `json:"-"`
}

// Record files: which file of a recording a kind lives in.
const (
	FileEvents  = "events.jsonl"
	FileNotes   = "notes.jsonl"
	FilePointer = "pointer.jsonl"
	FileFrames  = "frames/index.jsonl"
)

// kinds is the one list of record kinds: name, constructor, file. Decode,
// FileOf and the validator derive from it.
var kinds = map[string]struct {
	new  func() Record
	file string
}{
	"sketch-begin":        {func() Record { return &SketchBegin{} }, FileEvents},
	"sketch-stroke-begin": {func() Record { return &SketchStrokeBegin{} }, FileEvents},
	"sketch-points":       {func() Record { return &SketchPoints{} }, FileEvents},
	"sketch-stroke-end":   {func() Record { return &SketchStrokeEnd{} }, FileEvents},
	"sketch-undo":         {func() Record { return &SketchUndo{} }, FileEvents},
	"sketch-redo":         {func() Record { return &SketchRedo{} }, FileEvents},
	"sketch-label":        {func() Record { return &SketchLabel{} }, FileEvents},
	"sketch-interrupt":    {func() Record { return &SketchInterrupt{} }, FileEvents},
	"annotation-clear":    {func() Record { return &AnnotationClear{} }, FileEvents},
	"head":                {func() Record { return &Head{} }, FileEvents},
	"seg":                 {func() Record { return &Seg{} }, FileEvents},
	"stop":                {func() Record { return &Stop{} }, FileEvents},
	"doc":                 {func() Record { return &Doc{} }, FileEvents},
	"ins":                 {func() Record { return &Ins{} }, FileEvents},
	"rm":                  {func() Record { return &Rm{} }, FileEvents},
	"kids":                {func() Record { return &Kids{} }, FileEvents},
	"attr":                {func() Record { return &Attr{} }, FileEvents},
	"text":                {func() Record { return &Text{} }, FileEvents},
	"shadow":              {func() Record { return &Shadow{} }, FileEvents},
	"sheet":               {func() Record { return &Sheet{} }, FileEvents},
	"adopt":               {func() Record { return &Adopt{} }, FileEvents},
	"font":                {func() Record { return &Font{} }, FileEvents},
	"asset":               {func() Record { return &Asset{} }, FileEvents},
	"input":               {func() Record { return &Input{} }, FileEvents},
	"scroll":              {func() Record { return &Scroll{} }, FileEvents},
	"selection":           {func() Record { return &Selection{} }, FileEvents},
	"nav":                 {func() Record { return &Nav{} }, FileEvents},
	"title":               {func() Record { return &Title{} }, FileEvents},
	"vp":                  {func() Record { return &VP{} }, FileEvents},
	"click":               {func() Record { return &Click{} }, FileEvents},
	"focus":               {func() Record { return &Focus{} }, FileEvents},
	"layout":              {func() Record { return &Layout{} }, FileEvents},
	"hover":               {func() Record { return &Hover{} }, FileEvents},
	"ptr":                 {func() Record { return &Ptr{} }, FileEvents},
	"ctl":                 {func() Record { return &Ctl{} }, FileEvents},
	"note":                {func() Record { return &Note{} }, FileNotes},
	"note-edit":           {func() Record { return &NoteEdit{} }, FileNotes},
	"note-state":          {func() Record { return &NoteState{} }, FileNotes},
	"stroke":              {func() Record { return &Stroke{} }, FilePointer},
	"frame":               {func() Record { return &Frame{} }, FileFrames},
}

func (*Head) Kind() string      { return "head" }
func (*Seg) Kind() string       { return "seg" }
func (*Stop) Kind() string      { return "stop" }
func (*Doc) Kind() string       { return "doc" }
func (*Ins) Kind() string       { return "ins" }
func (*Rm) Kind() string        { return "rm" }
func (*Kids) Kind() string      { return "kids" }
func (*Attr) Kind() string      { return "attr" }
func (*Text) Kind() string      { return "text" }
func (*Shadow) Kind() string    { return "shadow" }
func (*Sheet) Kind() string     { return "sheet" }
func (*Adopt) Kind() string     { return "adopt" }
func (*Font) Kind() string      { return "font" }
func (*Asset) Kind() string     { return "asset" }
func (*Input) Kind() string     { return "input" }
func (*Scroll) Kind() string    { return "scroll" }
func (*Selection) Kind() string { return "selection" }
func (*Nav) Kind() string       { return "nav" }
func (*Title) Kind() string     { return "title" }
func (*VP) Kind() string        { return "vp" }
func (*Click) Kind() string     { return "click" }
func (*Focus) Kind() string     { return "focus" }
func (*Layout) Kind() string    { return "layout" }
func (*Hover) Kind() string     { return "hover" }
func (*Ptr) Kind() string       { return "ptr" }
func (*Note) Kind() string      { return "note" }
func (*NoteEdit) Kind() string  { return "note-edit" }
func (*NoteState) Kind() string { return "note-state" }
func (*Stroke) Kind() string    { return "stroke" }
func (*Frame) Kind() string     { return "frame" }
func (u *Unknown) Kind() string { return u.K }

// check: each kind's required fields (references are the reducer's).
func (r *Head) check() error {
	return errors.Join(need(r.Format == Format, "format "+Format), need(r.V > 0, "v"), need(r.ID != "", "id"), need(r.Started != "", "started"))
}
func (r *Seg) check() error {
	return errors.Join(need(r.Seg > 0, "seg"), need(oneOf(r.Why, SegLive, SegResume, SegNavigate, SegTab), "why"),
		need(r.VP.W > 0 && r.VP.H > 0, "vp"))
}
func (r *Stop) check() error { return need(oneOf(r.Why, StopPause, StopEnd), "why") }
func (r *Doc) check() error {
	return errors.Join(need(r.Doc > 0, "doc"), need(r.Tree.Y == YDoc && r.Tree.ID == r.Doc, "tree (a doc node with id doc)"), checkTree(&r.Tree))
}
func (r *Ins) check() error {
	return errors.Join(need(r.P > 0, "p"), need(r.Node.ID > 0, "node"), checkTree(&r.Node))
}
func (r *Rm) check() error { return need(r.ID > 0, "id") }
func (r *Kids) check() error {
	errs := []error{need(r.P > 0, "p")}
	for i := range r.C {
		errs = append(errs, checkTree(&r.C[i]))
	}
	return errors.Join(errs...)
}
func (r *Attr) check() error { return errors.Join(need(r.ID > 0, "id"), need(r.Name != "", "name")) }
func (r *Text) check() error { return need(r.ID > 0, "id") }
func (r *Shadow) check() error {
	return errors.Join(need(r.Host > 0, "host"), need(r.Root.Y == YShadow, "root (a shadow node)"), checkTree(&r.Root))
}
func (r *Sheet) check() error { return need(r.ID > 0, "id") }
func (r *Adopt) check() error { return need(r.ID > 0, "id") }
func (r *Font) check() error {
	return errors.Join(need(r.Family != "", "family"), need(r.Src != "", "src"))
}
func (r *Asset) check() error {
	return errors.Join(need(r.URL != "", "url"), need(ValidHash(r.Hash), "hash (64 lowercase hex)"), need(r.MIME != "", "mime"))
}
func (r *Input) check() error {
	err := errors.Join(need(r.ID > 0, "id"), need(r.V != nil || r.Checked != nil || r.Sel != nil || r.Secret, "v, checked, sel or secret"))
	if r.Secret && r.V != nil {
		err = errors.Join(err, errors.New("secret comes without v"))
	}
	return err
}
func (r *Scroll) check() error    { return need(r.ID > 0, "id") }
func (r *Selection) check() error { return nil }
func (r *Nav) check() error       { return need(r.URL != "", "url") }
func (r *Title) check() error     { return nil }
func (r *VP) check() error        { return need(r.W > 0 && r.H > 0, "w, h") }
func (r *Click) check() error     { return need(r.ID > 0, "id") }
func (r *Focus) check() error     { return nil }
func (r *Layout) check() error    { return nil }
func (r *Hover) check() error     { return need(r.ID >= 0, "id") }
func (r *Ptr) check() error       { return nil }
func (r *Note) check() error {
	return errors.Join(need(r.ID != "", "id"), need(r.Author.ID != "", "author.id"), need(r.At.Node > 0 || r.Region != nil, "at.node or region"))
}
func (r *NoteEdit) check() error {
	return errors.Join(need(r.ID != "", "id"), need(strings.TrimSpace(r.Text) != "" && broadcast.PlainText(r.Text), "bounded nonblank plain text"))
}
func (r *NoteState) check() error {
	return errors.Join(need(r.ID != "", "id"), need(oneOf(r.State, NoteOpen, NoteResolved), "state"))
}
func (r *Stroke) check() error {
	return errors.Join(need(r.ID != "", "id"), need(r.Author.ID != "", "author.id"), need(len(r.Pts) > 0, "pts"))
}
func (r *Frame) check() error {
	return errors.Join(need(r.File != "", "file"), need(r.W > 0 && r.H > 0, "w, h"))
}
func (r *Unknown) check() error { return nil }

func oneOf(s string, set ...string) bool {
	for _, v := range set {
		if s == v {
			return true
		}
	}
	return false
}

// checkTree checks a wire subtree's shape: every node has an id and a known
// type with that type's fields.
func checkTree(n *Node) error {
	if n.ID <= 0 {
		return fmt.Errorf("node without id")
	}
	switch n.Y {
	case YEl:
		if n.N == "" {
			return fmt.Errorf("node %d: element without n", n.ID)
		}
	case YText, YComment:
		if n.V == nil {
			return fmt.Errorf("node %d: %s without v", n.ID, n.Y)
		}
	case YShadow:
		if n.Mode != "open" && n.Mode != "closed" {
			return fmt.Errorf("node %d: shadow root without mode", n.ID)
		}
	case YDoc, YDoctype, YFrag:
	default:
		return fmt.Errorf("node %d: unknown type %q", n.ID, n.Y)
	}
	if n.SR != nil && n.SR.Y != YShadow {
		return fmt.Errorf("node %d: sr is not a shadow node", n.ID)
	}
	if n.TC != nil && n.TC.Y != YFrag {
		return fmt.Errorf("node %d: tc is not a frag node", n.ID)
	}
	for _, sub := range []*Node{n.SR, n.TC} {
		if sub != nil {
			if err := checkTree(sub); err != nil {
				return err
			}
		}
	}
	for i := range n.C {
		if err := checkTree(&n.C[i]); err != nil {
			return err
		}
	}
	return nil
}

// ValidHash: an asset hash is lowercase hex SHA-256.
func ValidHash(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// Str is a helper for the *string fields.
func Str(s string) *string { return &s }
