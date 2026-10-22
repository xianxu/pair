package board

import (
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
)

func visibleFreezeFixture(t *testing.T) (*Board, Selection, core.FeedbackVisibility) {
	t.Helper()
	b := New(notes.Structural{}, nil)
	for _, r := range submissionSketchRecords(2) {
		b.Put(r)
	}
	for i, id := range []string{"n1", "n2"} {
		r := &recording.Note{ID: id, Text: "saved note", Author: recording.Author{ID: "operator", Color: "#ff0000"}, At: recording.NoteAt{Node: 1, Box: &recording.Box{10, 20, 100, 40}}, Region: &recording.Box{10, 20, 100, 40}}
		r.SetTime(int64(3 + i))
		b.Put(r)
	}
	selection, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := b.OperatorView()
	view := core.FeedbackVisibility{Revision: 1, Scene: snapshot.FeedbackContext, MaskScope: snapshot.FeedbackMask.Scope, MaskRevision: snapshot.FeedbackMask.Revision}
	for _, s := range selection.Sketches {
		view.Sketches = append(view.Sketches, core.VisibleSketch{ID: s.ID, Sequence: s.NextSequence, Digest: s.Digest, Paths: []core.VisiblePath{{ID: "p", Locator: []int{1, 1, 0}, Box: recording.Box{10, 20, 100, 40}, Clip: recording.Box{0, 0, 1280, 800}}}})
	}
	for _, n := range selection.Notes {
		view.Notes = append(view.Notes, core.PreviewNote{ID: n.ID, Revision: n.Revision})
	}
	return b, selection, view
}
func TestVisibleFreezePinsCurrentOrdinalAndAllowsExactSubsets(t *testing.T) {
	b, selection, view := visibleFreezeFixture(t)
	frozen, err := b.FreezeVisibleSubmissionSelection(selection.SketchGeneration, selection.Sketches[:1], selection.Notes[:1], view, []recording.NodeID{1})
	if err != nil {
		t.Fatal(err)
	}
	if frozen.CurrentPage == nil || frozen.Current == nil || len(frozen.Current.Projections) != 1 || len(frozen.Notes) != 1 || len(frozen.Sketches) != 1 {
		t.Fatalf("partial frozen context: %+v", frozen)
	}
	if frozen.CurrentPage.RecordIndex <= selection.Sketches[0].Finish.RecordIndex || frozen.NoteRemaining != 1 {
		t.Fatal("froze finish instead of current scene or lost deferrednotes")
	}
	if frozen.Sketches[0] != selection.Sketches[0] {
		t.Fatal("historical source/finish rewritten")
	}
}

func TestVisibleFreezeRefusesChangedOrUntrustedWitnesses(t *testing.T) {
	for name, change := range map[string]func(*Board, *Selection, *core.FeedbackVisibility, *[]recording.NodeID){
		"generation":    func(_ *Board, s *Selection, _ *core.FeedbackVisibility, _ *[]recording.NodeID) { s.SketchGeneration++ },
		"mask scope":    func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.MaskScope = "other" },
		"mask revision": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.MaskRevision++ },
		"recording": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Scene.RecordingID = "other"
		},
		"document": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.Scene.Doc++ },
		"tab":      func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.Scene.Tab = "other" },
		"route": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Scene.URL += "/changed"
		},
		"scroll":   func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.Scene.Scroll[0]++ },
		"viewport": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.Scene.VP.W++ },
		"version": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[0].Sequence++
		},
		"unknown item": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[1].ID = "unknown"
		},
		"unknown path": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[0].Paths[0].ID = "unknown"
		},
		"missing ink": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[0].Paths = nil
		},
		"duplicate ink": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[0].Paths = append(v.Sketches[0].Paths, v.Sketches[0].Paths[0])
		},
		"invisible chosen": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches = v.Sketches[1:]
		},
		"unknown note": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Notes[0].ID = "unknown"
		},
		"stale note":       func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) { v.Notes[0].Revision++ },
		"missing resolved": func(_ *Board, _ *Selection, _ *core.FeedbackVisibility, r *[]recording.NodeID) { *r = nil },
		"extra resolved":   func(_ *Board, _ *Selection, _ *core.FeedbackVisibility, r *[]recording.NodeID) { *r = append(*r, 1) },
		"missing node":     func(_ *Board, _ *Selection, _ *core.FeedbackVisibility, r *[]recording.NodeID) { (*r)[0] = 99999 },
		"non-element":      func(b *Board, _ *Selection, _ *core.FeedbackVisibility, r *[]recording.NodeID) { (*r)[0] = b.st.Top.ID },
		"clip outside view": func(_ *Board, _ *Selection, v *core.FeedbackVisibility, _ *[]recording.NodeID) {
			v.Sketches[0].Paths[0].Clip[0] = -1
		},
		"recorded box mismatch": func(b *Board, _ *Selection, _ *core.FeedbackVisibility, _ *[]recording.NodeID) {
			b.st.Boxes[1] = recording.Box{13, 20, 100, 40}
		},
	} {
		t.Run(name, func(t *testing.T) {
			b, s, v := visibleFreezeFixture(t)
			resolved := []recording.NodeID{1}
			change(b, &s, &v, &resolved)
			if _, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, s.Sketches[:1], s.Notes[:1], v, resolved); err == nil {
				t.Fatal("untrusted witness froze")
			}
		})
	}
}
func TestVisibleFreezePinsNewestSameViewRecordAndOwnsProjection(t *testing.T) {
	b, s, v := visibleFreezeFixture(t)
	v.Scene.RecordIndex = 0
	v.Scene.T = 0
	r := &recording.Layout{Boxes: map[recording.NodeID]recording.Box{1: {11, 22, 100, 40}}}
	r.SetTime(b.st.T + 1)
	b.Put(r)
	frozen, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, s.Sketches[:1], s.Notes[:1], v, []recording.NodeID{1})
	if err != nil {
		t.Fatal(err)
	}
	if frozen.CurrentPage.RecordIndex != b.feedback.records-1 || frozen.CurrentPage.T != r.Time() {
		t.Fatal("did not pin current ordered record")
	}
	frozen.Current.Projections[0].At.Stable["id"] = "mutated"
	frozen.Current.Projections[0].At.Box[0] = 999
	if b.st.Node(1).A["id"] != "save" || b.st.Boxes[1][0] != 11 {
		t.Fatal("frozen anchor aliases reducer")
	}
	snapshot := b.OperatorView()
	if snapshot.NoteRevisions["n1"] != "0" || snapshot.FeedbackContext.RecordIndex != b.feedback.records-1 {
		t.Fatal("operator scalar context incomplete")
	}
	snapshot.NoteRevisions["n1"] = "bad"
	if b.OperatorView().NoteRevisions["n1"] != "0" {
		t.Fatal("operator revisions alias")
	}
}
func TestVisibleFreezeNoteOnlyAndHiddenSubset(t *testing.T) {
	b, s, v := visibleFreezeFixture(t)
	v.Sketches = nil
	v.Notes = v.Notes[:1]
	frozen, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, nil, s.Notes[:1], v, nil)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Current != nil || frozen.CurrentPage != nil || frozen.NoteRemaining != 1 {
		t.Fatal("noteonly acquired sketch render")
	}
	if _, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, nil, s.Notes, v, nil); err == nil {
		t.Fatal("invisible note froze")
	}
}

type freezeAnchorCounter struct {
	notes.Model
	calls int
}

func (c *freezeAnchorCounter) Anchor(st *recording.State, node recording.NodeID) (recording.NoteAt, error) {
	c.calls++
	return c.Model.Anchor(st, node)
}
func TestVisibleFreezeSharesCurrentAnchorReadAcrossPaths(t *testing.T) {
	b, s, v := visibleFreezeFixture(t)
	counter := &freezeAnchorCounter{Model: b.model}
	b.model = counter
	frozen, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, s.Sketches, nil, v, []recording.NodeID{1, 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(frozen.Current.Projections) != 2 || counter.calls != 1 {
		t.Fatalf("repeated current anchor work: calls=%d", counter.calls)
	}
}

func TestVisibleFreezeRefusesDuplicateUnknownAndMaskedSelections(t *testing.T) {
	for _, kind := range []string{"duplicate sketch", "unknown sketch", "changed source", "duplicate note", "unknown note", "changed note", "masked sketch"} {
		t.Run(kind, func(t *testing.T) {
			b, s, v := visibleFreezeFixture(t)
			chosen := append([]SelectedSketch(nil), s.Sketches[:1]...)
			selectedNotes := append([]SelectedNote(nil), s.Notes[:1]...)
			resolved := []recording.NodeID{1}
			switch kind {
			case "duplicate sketch":
				chosen = append(chosen, chosen[0])
				resolved = append(resolved, 1)
			case "unknown sketch":
				chosen[0].ID = "unknown"
			case "changed source":
				chosen[0].Source.Doc++
			case "duplicate note":
				selectedNotes = append(selectedNotes, selectedNotes[0])
			case "unknown note":
				selectedNotes[0].ID = "unknown"
			case "changed note":
				selectedNotes[0].Revision++
			case "masked sketch":
				if err := b.SetDeliveredFeedback(chosen, nil); err != nil {
					t.Fatal(err)
				}
				s, _ = b.Selection()
				mask := b.FeedbackMask()
				v.MaskScope = mask.Scope
				v.MaskRevision = mask.Revision
			}
			if _, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, chosen, selectedNotes, v, resolved); err == nil {
				t.Fatal("invalid selection froze")
			}
		})
	}
}
func TestVisibleFreezeRefusesUnsupportedFrameNode(t *testing.T) {
	b, s, v := visibleFreezeFixture(t)
	parent := b.st.Node(1).Parent.ID
	records := []recording.Record{&recording.Ins{P: parent, Node: recording.Node{ID: 1000, Y: recording.YEl, N: "iframe"}}, &recording.Doc{Doc: 1001, Host: 1000, URL: b.st.URL, Tree: recording.Node{ID: 1001, Y: recording.YDoc, C: []recording.Node{{ID: 1002, Y: recording.YEl, N: "button"}}}}}
	for _, r := range records {
		r.SetTime(b.st.T + 1)
		b.Put(r)
	}
	if b.broken != nil {
		t.Fatal(b.broken)
	}
	if _, err := b.FreezeVisibleSubmissionSelection(s.SketchGeneration, s.Sketches[:1], s.Notes[:1], v, []recording.NodeID{1002}); err == nil {
		t.Fatal("frame-local node treated as top-page locator")
	}
}
