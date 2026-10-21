package recording

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

const (
	MaxSketchInteger          = uint64(1<<53 - 1)
	SketchMaxIDBytes          = 128
	SketchMaxLabelBytes       = 1024
	SketchMaxPointsPerChunk   = 64
	SketchMaxPointsPerStroke  = 4096
	SketchMaxPointsPerSketch  = 16384
	SketchMaxStrokes          = 64
	SketchMaxCount            = 256
	SketchMaxCoordinate       = 1e7
	SketchInterruptExpiry     = "expiry"
	SketchInterruptNavigation = "navigation"
	SketchInterruptHandoff    = "handoff"
	SketchInterruptReset      = "reset"
	SketchInterruptEnd        = "end"
)

// SketchPoint holds original top-page CSS x/y and normalized pen pressure.
// It deliberately differs from the pointer Stroke's [x,y,elapsed] tuple.
type SketchPoint [3]float64

// SketchOperation names a bounded, retryable semantic request. Writer time
// and additive unknown fields do not participate in its canonical digest.
type SketchOperation struct {
	Operation string `json:"operation"`
	Sequence  uint64 `json:"sequence"`
	Digest    string `json:"digest"`
	wire      *sketchWire
}

func (o *SketchOperation) sketchOperation() *SketchOperation { return o }
func sketchMetadata(r Record) *SketchOperation {
	if s, ok := r.(interface{ sketchOperation() *SketchOperation }); ok {
		return s.sketchOperation()
	}
	return nil
}

type SketchBegin struct {
	Header
	SketchOperation
	ID     string   `json:"id"`
	Author Author   `json:"author"`
	Owner  string   `json:"owner"`
	Doc    int      `json:"doc"`
	Tab    string   `json:"tab"`
	VP     Viewport `json:"vp"`
}
type SketchStrokeBegin struct {
	Header
	SketchOperation
	Sketch string  `json:"sketch"`
	ID     string  `json:"id"`
	At     NoteAt  `json:"at"`
	Width  float64 `json:"width"`
}
type SketchPoints struct {
	Header
	SketchOperation
	Sketch string        `json:"sketch"`
	ID     string        `json:"id"`
	Chunk  uint64        `json:"chunk"`
	Pts    []SketchPoint `json:"pts"`
}
type SketchStrokeEnd struct {
	Header
	SketchOperation
	Sketch string `json:"sketch"`
	ID     string `json:"id"`
}
type SketchUndo struct {
	Header
	SketchOperation
	Sketch string `json:"sketch"`
	ID     string `json:"id"`
}
type SketchRedo struct {
	Header
	SketchOperation
	Sketch string `json:"sketch"`
	ID     string `json:"id"`
}
type SketchLabel struct {
	Header
	SketchOperation
	Sketch string `json:"sketch"`
	Label  string `json:"label"`
}
type SketchInterrupt struct {
	Header
	SketchOperation
	Sketch string `json:"sketch"`
	Reason string `json:"reason"`
}

func (*SketchBegin) Kind() string       { return "sketch-begin" }
func (*SketchStrokeBegin) Kind() string { return "sketch-stroke-begin" }
func (*SketchPoints) Kind() string      { return "sketch-points" }
func (*SketchStrokeEnd) Kind() string   { return "sketch-stroke-end" }
func (*SketchUndo) Kind() string        { return "sketch-undo" }
func (*SketchRedo) Kind() string        { return "sketch-redo" }
func (*SketchLabel) Kind() string       { return "sketch-label" }
func (*SketchInterrupt) Kind() string   { return "sketch-interrupt" }

func validSketchID(s string) bool {
	if s == "" || len(s) > SketchMaxIDBytes || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return false
		}
	}
	return true
}
func sketchFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) <= SketchMaxCoordinate
}
func sketchEnvelope(r Record) error {
	o := sketchMetadata(r)
	return errors.Join(need(r.Time() >= 0 && uint64(r.Time()) <= MaxSketchInteger, "safe sketch time"), need(validSketchID(o.Operation), "operation"), need(o.Sequence <= MaxSketchInteger, "safe sequence"), need(ValidHash(o.Digest), "digest"))
}
func validSketchColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
func (r *SketchBegin) check() error {
	return errors.Join(sketchEnvelope(r), need(validSketchID(r.ID), "id"), need(validSketchID(r.Author.ID), "author.id"), need(r.Author.Name == "" || len(r.Author.Name) <= SketchMaxIDBytes && broadcast.PlainText(r.Author.Name), "author.name"), need(validSketchColor(r.Author.Color), "author.color"), need(r.Owner == OperatorID || ValidHash(r.Owner), "owner"), need(r.Doc > 0 && uint64(r.Doc) <= MaxSketchInteger, "doc"), need(validSketchID(r.Tab), "tab"), need(r.VP.W > 0 && uint64(r.VP.W) <= MaxSketchInteger && r.VP.H > 0 && uint64(r.VP.H) <= MaxSketchInteger && sketchFinite(r.VP.DPR) && r.VP.DPR > 0, "vp"))
}
func checkSketchStroke(r Record, sketch, id string) error {
	return errors.Join(sketchEnvelope(r), need(validSketchID(sketch), "sketch"), need(validSketchID(id), "id"))
}
func (r *SketchStrokeBegin) check() error {
	err := checkSketchStroke(r, r.Sketch, r.ID)
	err = errors.Join(err, need(sketchFinite(r.Width) && r.Width > 0, "width"), need(r.At.Node > 0 && uint64(r.At.Node) <= MaxSketchInteger, "at.node"), need(r.At.Kind == "" || r.At.Kind == AnchorElement, "at.kind"))
	if r.At.Box == nil {
		return errors.Join(err, fmt.Errorf("missing at.box"))
	}
	for _, v := range *r.At.Box {
		err = errors.Join(err, need(sketchFinite(v), "at.box coordinate"))
	}
	return errors.Join(err, need(r.At.Box[2] > 0 && r.At.Box[3] > 0, "positive at.box"))
}
func (r *SketchPoints) check() error {
	err := errors.Join(checkSketchStroke(r, r.Sketch, r.ID), need(r.Chunk <= MaxSketchInteger, "safe chunk"), need(len(r.Pts) > 0 && len(r.Pts) <= SketchMaxPointsPerChunk, "pts"))
	for _, p := range r.Pts {
		if !sketchFinite(p[0]) || !sketchFinite(p[1]) || math.IsNaN(p[2]) || p[2] < 0 || p[2] > 1 {
			return errors.Join(err, fmt.Errorf("invalid pressure point"))
		}
	}
	return err
}
func (r *SketchStrokeEnd) check() error { return checkSketchStroke(r, r.Sketch, r.ID) }
func (r *SketchUndo) check() error      { return checkSketchStroke(r, r.Sketch, r.ID) }
func (r *SketchRedo) check() error      { return checkSketchStroke(r, r.Sketch, r.ID) }
func (r *SketchLabel) check() error {
	return errors.Join(sketchEnvelope(r), need(validSketchID(r.Sketch), "sketch"), need(len(r.Label) <= SketchMaxLabelBytes && strings.TrimSpace(r.Label) != "" && broadcast.PlainText(r.Label), "label"))
}
func (r *SketchInterrupt) check() error {
	return errors.Join(sketchEnvelope(r), need(validSketchID(r.Sketch), "sketch"), need(oneOf(r.Reason, SketchInterruptExpiry, SketchInterruptNavigation, SketchInterruptHandoff, SketchInterruptReset, SketchInterruptEnd), "reason"))
}
