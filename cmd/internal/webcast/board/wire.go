package board

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Request bounds. Bodies are capped by the server before parsing.
const (
	// MaxStrokeBody bounds a stroke request: 64 points of three numbers.
	MaxStrokeBody = 8 << 10
	// MaxNoteBody is the terminal notes' bound: a full note escaped as
	// \uXXXX fits.
	MaxNoteBody = broadcast.MaxNoteBody
	// MaxNameRunes bounds a viewer's name: every cast's rule (pair#495).
	MaxNameRunes = broadcast.MaxViewerNameRunes
	// maxCoord bounds every coordinate, in CSS px: far past any page.
	maxCoord = 1e6
	// maxDT bounds a point's time after its chunk began, in ms.
	maxDT = 10000
)

// Operator is Couch's own author id; no viewer may use it.
const Operator = "operator"

// ErrShape: a request that isn't exactly what the route takes. Its message
// never echoes the input.
var ErrShape = errors.New("board: malformed request")

// idPattern is a viewer's or a stroke's id: the events route's ?viewer=
// shape.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Viewer is who sent a request, as the viewer's own shell says: attribution,
// not authentication. Anyone with the cast link can write as any id; #467
// swaps in seat identity.
type Viewer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StrokeIn is one chunk of a viewer's stroke: points in top-page CSS px,
// each [x, y, ms after the chunk began]. Chunks of one stroke share ID.
type StrokeIn struct {
	Viewer Viewer
	Seg    int
	ID     string
	Color  int
	Pts    [][3]float64
}

// NoteIn is a viewer's new note: its box (Region: dragged, or the default
// box of a tap, #488), the element it anchors on (the innermost at the
// box's centre) with that element's box from the viewer's layout
// (couchViewer.boxOf), and the text.
type NoteIn struct {
	Viewer Viewer
	Seg    int
	Node   recording.NodeID
	Box    recording.Box
	Region recording.Box
	Color  int
	Text   string
}

// StateIn resolves or reopens a note.
type StateIn struct {
	Viewer Viewer
	ID     string
	State  string
}

type wireViewer struct {
	ID   *string `json:"id"`
	Name *string `json:"name"`
}

// ParseStroke reads a stroke request strictly:
// {"viewer":{"id","name"},"seg","id","color","pts":[[x,y,dt],…]}.
func ParseStroke(body []byte) (StrokeIn, error) {
	var raw struct {
		Viewer *wireViewer   `json:"viewer"`
		Seg    *int          `json:"seg"`
		ID     *string       `json:"id"`
		Color  *int          `json:"color"`
		Pts    *[][]*float64 `json:"pts"`
	}
	if len(body) > MaxStrokeBody || !decode(body, &raw) || raw.Seg == nil || raw.ID == nil || raw.Color == nil || raw.Pts == nil {
		return StrokeIn{}, ErrShape
	}
	v, ok := viewer(raw.Viewer)
	if !ok || !idPattern.MatchString(*raw.ID) || !segOK(*raw.Seg) || !colorOK(*raw.Color) {
		return StrokeIn{}, ErrShape
	}
	pts := *raw.Pts
	if len(pts) == 0 || len(pts) > broadcast.MaxPointsPerBatch {
		return StrokeIn{}, ErrShape
	}
	out := StrokeIn{Viewer: v, Seg: *raw.Seg, ID: *raw.ID, Color: *raw.Color, Pts: make([][3]float64, len(pts))}
	for i, p := range pts {
		if len(p) != 3 { // a Go array would drop extras silently
			return StrokeIn{}, ErrShape
		}
		for j, c := range p {
			if c == nil {
				return StrokeIn{}, ErrShape
			}
			out.Pts[i][j] = *c
		}
		if !coord(out.Pts[i][0]) || !coord(out.Pts[i][1]) || !(out.Pts[i][2] >= 0 && out.Pts[i][2] <= maxDT) {
			return StrokeIn{}, ErrShape
		}
	}
	return out, nil
}

// ParseNote reads a note request strictly:
// {"viewer","seg","node","box":[x,y,w,h],"region":[x,y,w,h],"color","text"}.
// Every note has a region, its box (#488); node is the element at its
// centre and box that element's, as the viewer drew them.
// The text is the terminal notes' plain text (broadcast.PlainText), not
// blank.
func ParseNote(body []byte) (NoteIn, error) {
	var raw struct {
		Viewer *wireViewer `json:"viewer"`
		Seg    *int        `json:"seg"`
		Node   *int64      `json:"node"`
		Box    *[]*float64 `json:"box"`
		Region *[]*float64 `json:"region"`
		Color  *int        `json:"color"`
		Text   *string     `json:"text"`
	}
	if len(body) > MaxNoteBody || !decode(body, &raw) || raw.Seg == nil || raw.Node == nil || raw.Box == nil || raw.Region == nil || raw.Color == nil || raw.Text == nil {
		return NoteIn{}, ErrShape
	}
	v, ok := viewer(raw.Viewer)
	if !ok || !segOK(*raw.Seg) || *raw.Node < 1 || !colorOK(*raw.Color) {
		return NoteIn{}, ErrShape
	}
	box, ok := boxOf(raw.Box, 0)
	if !ok {
		return NoteIn{}, ErrShape
	}
	region, ok := boxOf(raw.Region, 1)
	if !ok {
		return NoteIn{}, ErrShape
	}
	n := NoteIn{Viewer: v, Seg: *raw.Seg, Node: recording.NodeID(*raw.Node), Box: box, Region: region, Color: *raw.Color, Text: *raw.Text}
	if !broadcast.PlainText(n.Text) || strings.TrimSpace(n.Text) == "" {
		return NoteIn{}, ErrShape
	}
	return n, nil
}

// ParseNoteState reads {"viewer","id","state":"open"|"resolved"}.
func ParseNoteState(body []byte) (StateIn, error) {
	var raw struct {
		Viewer *wireViewer `json:"viewer"`
		ID     *string     `json:"id"`
		State  *string     `json:"state"`
	}
	if len(body) > MaxNoteBody || !decode(body, &raw) || raw.ID == nil || raw.State == nil {
		return StateIn{}, ErrShape
	}
	v, ok := viewer(raw.Viewer)
	if !ok || !idPattern.MatchString(*raw.ID) || (*raw.State != recording.NoteOpen && *raw.State != recording.NoteResolved) {
		return StateIn{}, ErrShape
	}
	return StateIn{Viewer: v, ID: *raw.ID, State: *raw.State}, nil
}

// decode is broadcast.DecodeStrict over valid UTF-8 only: the decoder
// would turn invalid bytes into U+FFFD.
func decode(body []byte, v any) bool {
	return utf8.Valid(body) && broadcast.DecodeStrict(body, v)
}

// viewer checks a request's viewer: an id of the events route's shape that
// isn't the operator's, and a name of plain text, at most MaxNameRunes,
// with surrounding space trimmed (it may be empty).
func viewer(w *wireViewer) (Viewer, bool) {
	if w == nil || w.ID == nil || !idPattern.MatchString(*w.ID) || *w.ID == Operator {
		return Viewer{}, false
	}
	v := Viewer{ID: *w.ID}
	if w.Name != nil {
		name, ok := broadcast.ViewerName(*w.Name)
		if !ok {
			return Viewer{}, false
		}
		v.Name = name
	}
	return v, true
}

func segOK(s int) bool      { return s >= 1 && s <= math.MaxInt32 }
func colorOK(c int) bool    { return c >= 0 && c < broadcast.PointerColors }
func coord(v float64) bool  { return !math.IsNaN(v) && v >= -maxCoord && v <= maxCoord }
func extent(v float64) bool { return !math.IsNaN(v) && v >= 0 && v <= maxCoord }

// boxOf checks [x, y, w, h]: finite, in range, w and h at least min.
func boxOf(raw *[]*float64, min float64) (recording.Box, bool) {
	var b recording.Box
	if len(*raw) != 4 {
		return b, false
	}
	for i, v := range *raw {
		if v == nil {
			return b, false
		}
		b[i] = *v
	}
	return b, coord(b[0]) && coord(b[1]) && extent(b[2]) && extent(b[3]) && b[2] >= min && b[3] >= min
}
