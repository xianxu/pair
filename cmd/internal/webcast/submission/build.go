// Package submission builds DOM evidence from the frozen historical selection.
package submission

import (
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"math"
	"regexp"
)

var recordingID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

// Build keeps selected order and computes each crop separately at the note's
// recorded moment. Current screenshots are attached later by the cast owner.
func Build(recs []recording.Record, selected board.Selection, id string, target core.Target, path string, replay func(int64) string) (core.Submission, []core.CaptureRegion, error) {
	out := core.Submission{Version: 1, ID: id, Target: target, Delivery: core.Prepared, Source: core.Source{Kind: "dom", RecordingPath: path, TabID: selected.TabID, Document: selected.Document}}
	if len(recs) == 0 {
		return out, nil, fmt.Errorf("recording is empty")
	}
	head, ok := recs[0].(*recording.Head)
	if !ok || !recordingID.MatchString(head.ID) {
		return out, nil, fmt.Errorf("recording has no valid head")
	}
	out.Source.RecordingID = head.ID
	out.HowToRead = core.ReadingGuide(path)
	model := notes.Structural{}
	views, err := query.New(recs, model).Notes(5000)
	if err != nil {
		return out, nil, err
	}
	byID := map[string]query.NoteView{}
	for _, view := range views {
		if _, exists := byID[view.Note.ID]; exists {
			return out, nil, fmt.Errorf("duplicate recorded note %s", view.Note.ID)
		}
		byID[view.Note.ID] = view
	}
	wanted := map[string]bool{}
	for _, n := range selected.Notes {
		if n.ID == "" || wanted[n.ID] {
			return out, nil, fmt.Errorf("invalid duplicate selection")
		}
		wanted[n.ID] = true
	}
	boxes := map[string]recording.Box{}
	regions := map[string]recording.Box{}
	state := recording.NewState()
	for _, r := range recs {
		if err = state.Apply(r); err != nil {
			return out, nil, err
		}
		note, ok := r.(*recording.Note)
		if !ok || !wanted[note.ID] {
			continue
		}
		if state.DocSeg != selected.Document || state.Tab.ID != selected.TabID {
			return out, nil, fmt.Errorf("selected note %s belongs to a different document", note.ID)
		}
		// Structural crop arithmetic consumes both anchor geometry and page bounds.
		// Validate all operands before asking it to pad/union/clip them.
		for _, box := range state.Boxes {
			if err = finiteBox(box); err != nil {
				return out, nil, err
			}
		}
		for _, box := range []*recording.Box{note.At.Box, note.Region} {
			if box != nil {
				if err = finiteBox(*box); err != nil {
					return out, nil, err
				}
			}
		}
		crops, e := model.NoteCrops(state, []*recording.Note{note})
		if e != nil {
			return out, nil, e
		}
		if len(crops) != 1 {
			return out, nil, fmt.Errorf("note %s has no unique crop", note.ID)
		}
		if err = core.ValidateBox(crops[0]); err != nil {
			return out, nil, err
		}
		boxes[note.ID] = crops[0]
		region := note.Region
		if region == nil {
			region = note.At.Box
		}
		if region == nil {
			if b, ok := state.Boxes[note.At.Node]; ok {
				region = &b
			}
		}
		if region == nil {
			return out, nil, fmt.Errorf("note %s has no text region", note.ID)
		}
		if err = core.ValidateBox(*region); err != nil {
			return out, nil, err
		}
		regions[note.ID] = *region
	}
	var ordered []core.CaptureRegion
	for _, selectedNote := range selected.Notes {
		view, exists := byID[selectedNote.ID]
		box, hasBox := boxes[selectedNote.ID]
		if !exists || !hasBox {
			return out, nil, fmt.Errorf("selected note %s is missing from the recording", selectedNote.ID)
		}
		n := view.Note.Note
		note := core.Note{ID: n.ID, Revision: selectedNote.Revision, Author: n.Author, Text: n.Text, T: n.T, At: n.At, Context: view.Context, Element: view.Element, Before: view.BeforeLines, Crop: core.Crop{Unavailable: "not captured"}}
		if note.Element == "" {
			note.Element = "unavailable: recorded element missing"
			note.Context.Text = n.At.Text
		}
		if replay != nil {
			note.Replay = replay(n.T)
		}
		out.Notes = append(out.Notes, note)
		ordered = append(ordered, core.CaptureRegion{Box: box, Region: regions[n.ID]})
	}
	if err = core.Validate(out); err != nil {
		return out, nil, err
	}
	return out, ordered, nil
}
func finiteBox(b recording.Box) error {
	for _, v := range b {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e7 {
			return fmt.Errorf("recorded box outside finite coordinate domain")
		}
	}
	if b[2] < 0 || b[3] < 0 {
		return fmt.Errorf("negative recorded box size")
	}
	return nil
}
