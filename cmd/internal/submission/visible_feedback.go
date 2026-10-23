package submission

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

const MaxFeedbackVisibilityBytes = 64 << 20

// VisiblePath describes an actually painted anchor in the current source page.
// Locator uses the recorder's child-index/open-shadow grammar, never a selector.
type VisiblePath struct {
	ID      string        `json:"id"`
	Locator []int         `json:"locator"`
	Box     recording.Box `json:"box"`
	Clip    recording.Box `json:"clip"`
}
type VisibleSketch struct {
	ID       string        `json:"id"`
	Sequence uint64        `json:"next_sequence,string"`
	Digest   string        `json:"digest"`
	Paths    []VisiblePath `json:"paths"`
}

// FeedbackVisibility is a completed console paint, scoped by the private host.
// Its scene ordinal is informational until Board pins the current prefix at send.
type FeedbackVisibility struct {
	Revision     uint64                `json:"revision,string"`
	Scene        sketchimage.PageScene `json:"scene"`
	MaskScope    string                `json:"mask_scope"`
	MaskRevision uint64                `json:"mask_revision,string"`
	Sketches     []VisibleSketch       `json:"sketches"`
	Notes        []PreviewNote         `json:"notes"`
}

func CloneFeedbackVisibility(v FeedbackVisibility) FeedbackVisibility {
	v.Notes = slices.Clone(v.Notes)
	v.Sketches = slices.Clone(v.Sketches)
	for i := range v.Sketches {
		v.Sketches[i].Paths = slices.Clone(v.Sketches[i].Paths)
		for j := range v.Sketches[i].Paths {
			v.Sketches[i].Paths[j].Locator = slices.Clone(v.Sketches[i].Paths[j].Locator)
		}
	}
	return v
}
func visibilityBox(b recording.Box) bool {
	for _, v := range b {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > recording.SketchMaxCoordinate {
			return false
		}
	}
	return b[2] > 0 && b[3] > 0
}
func ValidateFeedbackVisibility(v FeedbackVisibility) error {
	if v.Revision == 0 || v.MaskScope == "" || len(v.Sketches) > 256 {
		return fmt.Errorf("feedback visibility: unavailable frame")
	}
	if _, err := sketchimage.PlanPage(v.Scene); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) > MaxFeedbackVisibilityBytes {
		return fmt.Errorf("feedback visibility: byte budget")
	}
	seen := map[string]bool{}
	for _, s := range v.Sketches {
		if s.ID == "" || seen[s.ID] || s.Sequence == 0 || s.Sequence > recording.MaxSketchInteger || !recording.ValidHash(s.Digest) || len(s.Paths) == 0 || len(s.Paths) > 64 {
			return fmt.Errorf("feedback visibility: invalid sketch")
		}
		seen[s.ID] = true
		paths := map[string]bool{}
		for _, p := range s.Paths {
			if p.ID == "" || paths[p.ID] || len(p.Locator) == 0 || len(p.Locator) > 128 || !visibilityBox(p.Box) || !visibilityBox(p.Clip) {
				return fmt.Errorf("feedback visibility: invalid path")
			}
			paths[p.ID] = true
			for _, step := range p.Locator {
				if step < -1 || step > 1<<20 {
					return fmt.Errorf("feedback visibility: invalid locator")
				}
			}
		}
	}
	seen = map[string]bool{}
	for _, n := range v.Notes {
		if n.ID == "" || seen[n.ID] {
			return fmt.Errorf("feedback visibility: invalid note")
		}
		seen[n.ID] = true
	}
	return nil
}

// SelectVisibleFeedback intersects one completed paint with authoritative saved
// candidates. The returned witness is owned and never infers URL visibility.
func SelectVisibleFeedback(candidates []PreviewSketchCandidate, notes []PreviewNote, delivered DeliveredFeedback, stamp PreviewStamp, view FeedbackVisibility) (PreviewSelection, error) {
	out, _, err := selectVisibleFeedback(candidates, notes, delivered, stamp, view)
	return out, err
}
func selectVisibleFeedback(candidates []PreviewSketchCandidate, notes []PreviewNote, delivered DeliveredFeedback, stamp PreviewStamp, view FeedbackVisibility) (PreviewSelection, int, error) {
	if err := ValidateFeedbackVisibility(view); err != nil {
		return PreviewSelection{}, 0, err
	}
	visits := 0
	visible := map[string]VisibleSketch{}
	for _, v := range view.Sketches {
		visits++
		visible[v.ID] = v
	}
	visibleNotes := map[PreviewNote]bool{}
	for _, n := range view.Notes {
		visits++
		visibleNotes[n] = true
	}
	out := PreviewSelection{Stamp: stamp, Visibility: new(FeedbackVisibility)}
	*out.Visibility = CloneFeedbackVisibility(view)
	for _, c := range candidates {
		visits++
		v, painted := visible[c.Item.SketchID]
		if painted {
			if c.Item.RecordingID != view.Scene.RecordingID || v.Sequence != c.Item.Sequence || v.Digest != c.Item.Digest {
				return PreviewSelection{}, visits, fmt.Errorf("feedback visibility: sketch changed")
			}
			delete(visible, c.Item.SketchID)
		}
		if _, sent := delivered.Sketches[c.Item]; sent {
			continue
		}
		if !painted || len(out.Items) >= MaxSubmissionSketches {
			out.Remaining++
			continue
		}
		out.Items = append(out.Items, c.Item)
		out.Candidates = append(out.Candidates, c)
	}
	if len(visible) > 0 {
		return PreviewSelection{}, visits, fmt.Errorf("feedback visibility: unknown sketch")
	}
	for _, n := range notes {
		visits++
		painted := visibleNotes[n]
		delete(visibleNotes, n)
		if _, sent := delivered.Notes[n]; sent {
			continue
		}
		if painted {
			out.Notes = append(out.Notes, n)
		} else {
			out.NoteRemaining++
		}
	}
	if len(visibleNotes) > 0 {
		return PreviewSelection{}, visits, fmt.Errorf("feedback visibility: note changed")
	}
	if len(out.Items) > 0 {
		scene := view.Scene
		out.Page = &scene
	}
	return out, visits, nil
}
