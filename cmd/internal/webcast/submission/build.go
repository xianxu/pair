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
// recorded moment. The cast owner attaches immutable draw-time evidence later.
func Build(recs []recording.Record, selected board.Selection, id string, target core.Target, path string, replay func(int64) string) (core.Submission, []core.CaptureRegion, error) {
	out := core.Submission{Version: 3, PrivacyDecision: core.PrivacyDecision{Privacy: core.Private}, ID: id, Target: target, Delivery: core.Prepared, Source: core.Source{Kind: "dom", RecordingPath: path, TabID: selected.TabID, Document: selected.Document}}
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
	documents := map[string]int{}
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
		documents[note.ID] = state.DocSeg
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
		anchor := n.At
		anchor.Sel = ""
		reason := n.CropUnavailable
		if reason == "" {
			reason = "not captured when drawn"
		}
		note := core.Note{PrivacyDecision: core.PrivacyDecision{Privacy: core.Private}, CaptureID: n.Capture, SourceDocument: documents[n.ID], ID: n.ID, Revision: selectedNote.Revision, Author: n.Author, Text: n.Text, T: n.T, At: anchor, Elements: &view.Elements, Context: view.Context, Element: view.Element, Before: view.BeforeLines, Crop: core.Crop{Unavailable: reason}, BoxText: core.BoxText{Unavailable: reason}}
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

// ContextAt keeps the capture timestamp and narrated DOM evidence coherent.
// The authored note, frozen anchor, revision and original document are unchanged.
func ContextAt(recs []recording.Record, n core.Note, at int64) (core.Note, error) {
	record := &recording.Note{ID: n.ID, Author: n.Author, Text: n.Text, At: n.At}
	if n.BoxText.Region != (recording.Box{}) {
		region := n.BoxText.Region
		record.Region = &region
	}
	record.SetTime(n.T)
	view, err := query.New(recs, notes.Structural{}).ContextForNote(record, at, 5000)
	if err != nil {
		return n, err
	}
	n.T = at
	n.Elements = &view.Elements
	n.Context = view.Context
	n.Element = view.Element
	n.Before = view.BeforeLines
	if n.Element == "" {
		n.Element = "unavailable: recorded element missing at draw time"
		n.Context = notes.Context{Text: "unavailable: recorded element missing at draw time"}
	}
	return n, nil
}
