package submission

import (
	"bytes"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"image/png"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxSubmissionSketches = 8

// Sketch is historical evidence, never a recording writer's admission token.
type Sketch struct {
	PrivacyDecision
	ID                 string                     `json:"id"`
	RecordingID        string                     `json:"recording_id"`
	NextSequence       uint64                     `json:"next_sequence"`
	Digest             string                     `json:"digest"`
	Author             recording.Author           `json:"author"`
	Note               string                     `json:"note"`
	HistoryT           int64                      `json:"history_t"`
	HistoryRecordIndex int                        `json:"history_record_index"`
	Phase              string                     `json:"phase"`
	Reason             string                     `json:"reason,omitempty"`
	Source             SketchSource               `json:"source"`
	Elements           query.DeclaredElementFacts `json:"elements"`
	Image              SketchImage                `json:"image"`
}
type SketchSource struct {
	query.SketchSourceFacts
	Provenance string `json:"provenance"`
}
type SketchImage struct {
	Limitations []string               `json:"limitations,omitempty"`
	File        string                 `json:"file"`
	Provenance  string                 `json:"provenance"`
	Plan        sketchimage.RenderPlan `json:"plan"`
}

var sketchName = regexp.MustCompile(`^sketch-(0|[1-9][0-9]{0,5})\.png$`)
var sketchColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ValidateSketch recomputes the transform from original paths, so image
// metadata cannot describe different ink or substitute captured page pixels.
func ValidateSketch(s Sketch) error {
	fail := func(why string) error { return fmt.Errorf("invalid sketch evidence: %s", why) }
	if !sketchID(s.ID) || s.RecordingID == "" || len(s.RecordingID) > 256 || s.NextSequence == 0 || s.NextSequence > recording.MaxSketchInteger || !recording.ValidHash(s.Digest) {
		return fail("identity or accepted prefix")
	}
	if !sketchID(s.Author.ID) || len(s.Author.Name) > recording.SketchMaxIDBytes || s.Author.Name != "" && !broadcast.PlainText(s.Author.Name) || !sketchColor.MatchString(s.Author.Color) {
		return fail("author")
	}
	if len(s.Note) > recording.SketchMaxLabelBytes || strings.TrimSpace(s.Note) == "" || !broadcast.PlainText(s.Note) {
		return fail("required note")
	}
	if s.Phase != "complete" && s.Phase != "interrupted" {
		return fail("unfinished or cleared")
	}
	if s.Phase == "complete" && s.Reason != "" {
		return fail("complete reason")
	}
	if s.Phase == "interrupted" {
		switch s.Reason {
		case recording.SketchInterruptExpiry, recording.SketchInterruptNavigation, recording.SketchInterruptHandoff, recording.SketchInterruptReset, recording.SketchInterruptEnd:
		default:
			return fail("interruption reason")
		}
	}
	if s.Source.Provenance != "recorded-dom" || s.Source.T < 0 || uint64(s.Source.T) > recording.MaxSketchInteger || s.HistoryT < s.Source.T || uint64(s.HistoryT) > recording.MaxSketchInteger || s.Source.RecordIndex < 0 || s.HistoryRecordIndex < s.Source.RecordIndex || s.Source.Doc < 1 || !sketchID(s.Source.Tab) {
		return fail("original source")
	}
	if len(s.Source.URL) > MaxPageURLBytes || len(s.Source.Title) > MaxPageTitleBytes || !utf8.ValidString(s.Source.URL) || !utf8.ValidString(s.Source.Title) {
		return fail("page facts")
	}
	if err := s.PrivacyDecision.Validate(); err != nil {
		return err
	}
	if s.Privacy == "" || s.PrivacyOrigin != OriginOf(s.Source.URL) || s.PrivacyOrigin == "" && s.Privacy != Private {
		return fail("original source privacy")
	}
	if !sketchName.MatchString(s.Image.File) || s.Image.Provenance != "replay-rendered" || s.Image.Plan.Sketch != s.ID {
		return fail("image identity or provenance")
	}
	if len(s.Image.Limitations) > 32 {
		return fail("too many image limitations")
	}
	for _, limitation := range s.Image.Limitations {
		if len(limitation) > 256 {
			return fail("image limitation exceeds bound")
		}
	}
	h := recording.SketchView{Begin: &recording.SketchBegin{ID: s.ID}, Phase: s.Phase}
	for _, p := range s.Image.Plan.Paths {
		if !sketchID(p.ID) || p.At.Sel != "" || p.At.Kind != "" && p.At.Kind != recording.AnchorElement {
			return fail("original path anchor")
		}
		h.Visible = append(h.Visible, p.ID)
		h.Strokes = append(h.Strokes, recording.SketchStrokeView{Begin: &recording.SketchStrokeBegin{Sketch: s.ID, ID: p.ID, At: p.At, Width: p.Width}, Pts: p.Pts, Truncated: p.Truncated})
	}
	expected, err := sketchimage.Plan(h, s.Source.VP, s.Source.Scroll, sketchimage.DefaultLimits())
	if err != nil {
		return err
	}
	actual := s.Image.Plan
	if actual.Region != expected.Region || actual.Scale != expected.Scale || actual.Width != expected.Width || actual.Height != expected.Height || actual.Clipped != expected.Clipped {
		return fail("image transform differs from original ink")
	}
	return validateSketchFacts(s.Elements, actual.Paths)
}
func sketchID(s string) bool {
	if s == "" || len(s) > recording.SketchMaxIDBytes || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func validateSketchFacts(f query.DeclaredElementFacts, paths []sketchimage.Path) error {
	if f.InspectedNodes < 0 || f.InspectedNodes > query.ElementFactsNodeLimit || f.Bytes < 0 || f.Bytes > query.ElementFactsByteLimit || len(f.Anchors)+len(f.Ancestors)+len(f.Intersections) > query.ElementFactsCountLimit {
		return fmt.Errorf("declared facts exceed bounds")
	}
	seen := map[recording.NodeID]bool{}
	anchors := map[recording.NodeID]bool{}
	total := 0
	for group, entries := range [][]query.ElementFact{f.Anchors, f.Ancestors, f.Intersections} {
		for _, e := range entries {
			if e.Node <= 0 || uint64(e.Node) > recording.MaxSketchInteger || e.Parent < 0 || uint64(e.Parent) > recording.MaxSketchInteger || seen[e.Node] || len(e.Tag) > 256 || len(e.ID) > 1024 || len(e.Unavailable) > 256 || len(e.Data) > 16 {
				return fmt.Errorf("invalid declared element fact")
			}
			seen[e.Node] = true
			if group == 0 {
				anchors[e.Node] = true
			}
			total += len(e.Tag) + len(e.ID) + len(e.Unavailable) + 32
			for k, v := range e.Data {
				if !strings.HasPrefix(k, "data-") || len(k) > 256 || len(v) > 1024 {
					return fmt.Errorf("invalid declared data attribute")
				}
				total += len(k) + len(v)
			}
		}
	}
	if total != f.Bytes || total > query.ElementFactsByteLimit {
		return fmt.Errorf("declared facts byte count mismatch")
	}
	for _, p := range paths {
		if !anchors[p.At.Node] {
			return fmt.Errorf("original anchor missing declared facts")
		}
	}
	return nil
}

// EvidenceFiles is the single inventory of understood image references. Unknown
// manifest extensions never grant file authority. Names are confined and unique.
func EvidenceFiles(s Submission) ([]string, error) {
	files := make([]string, 0, len(s.Notes)+len(s.Sketches))
	seen := map[string]bool{}
	add := func(file string, valid bool) error {
		if !valid || seen[file] {
			return fmt.Errorf("invalid or duplicate evidence file %q", file)
		}
		seen[file] = true
		files = append(files, file)
		return nil
	}
	for _, n := range s.Notes {
		if n.Crop.File != "" {
			if err := add(n.Crop.File, cropName.MatchString(n.Crop.File)); err != nil {
				return nil, err
			}
		}
	}
	for _, sketch := range s.Sketches {
		if err := add(sketch.Image.File, sketchName.MatchString(sketch.Image.File)); err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func validateEvidencePNG(sub Submission, file string, data []byte) error {
	if err := ValidatePNG(data); err != nil {
		return err
	}
	for _, sketch := range sub.Sketches {
		if sketch.Image.File == file {
			config, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				return err
			}
			if config.Width != sketch.Image.Plan.Width || config.Height != sketch.Image.Plan.Height {
				return fmt.Errorf("sketch PNG dimensions differ from declared transform")
			}
		}
	}
	return nil
}
