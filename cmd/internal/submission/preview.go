package submission

import (
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"reflect"
	"slices"
)

// PreviewHost identifies one private console projection lifetime.
type PreviewHost struct {
	ID         string
	Generation uint64
}

// PreviewStamp combines independent eligible-board and delivered-version clocks.
type PreviewStamp struct{ Board, Delivered uint64 }

// PreviewItem identifies an immutable accepted sketch prefix, without owning ink.
type PreviewItem struct {
	RecordingID, SketchID, Digest string
	Sequence                      uint64
}

// PreviewSketchCandidate owns scalar original and completion witnesses. Its
// exact delivery key stays independent of page geometry and display indices.
type PreviewSketchCandidate struct {
	Item   PreviewItem
	Source query.SketchSourceFacts
	Finish query.SketchSourceFacts
}

// PreviewNote identifies the exact open note revision shown to the operator.
type PreviewNote struct {
	ID       string `json:"id"`
	Revision uint64 `json:"revision,string"`
}

// PreviewSelection is copied on entry and when read from a publication.
type PreviewSelection struct {
	Stamp         PreviewStamp
	Items         []PreviewItem
	Candidates    []PreviewSketchCandidate
	Notes         []PreviewNote
	Page          *sketchimage.PageScene
	Remaining     int
	NoteRemaining int
	Visibility    *FeedbackVisibility
	ResolvedNodes []recording.NodeID
}

type PreviewPhase uint8

const (
	PreviewDormant PreviewPhase = iota
	PreviewDirty
	PreviewPublishing
	PreviewShown
	PreviewStopped
)

type previewSnapshot struct {
	host      PreviewHost
	selection *PreviewSelection
}
type previewPayload interface{ previewPayload() }
type previewDormant struct{ previewSnapshot }

func (previewDormant) previewPayload() {}

type previewDirty struct{ previewSnapshot }

func (previewDirty) previewPayload() {}

type previewPublishing struct{ publication PreviewPublication }

func (previewPublishing) previewPayload() {}

type previewShown struct{ publication PreviewPublication }

func (previewShown) previewPayload() {}

type previewStopped struct{}

func (previewStopped) previewPayload() {}

// PreviewState is an opaque tagged state. Its zero value is Dormant.
// All transitions are pure; the submission manager serializes their application.
type PreviewState struct {
	payload previewPayload
	next    uint64
}

func (s PreviewState) Phase() PreviewPhase {
	switch s.payload.(type) {
	case previewDirty:
		return PreviewDirty
	case previewPublishing:
		return PreviewPublishing
	case previewShown:
		return PreviewShown
	case previewStopped:
		return PreviewStopped
	default:
		return PreviewDormant
	}
}

// PreviewPublication binds a projection to its host, candidate stamp and operation.
// Its private payload cannot be modified by projection callbacks.
type PreviewPublication struct {
	host                PreviewHost
	selection           PreviewSelection
	revision, operation uint64
}

func (p PreviewPublication) Host() PreviewHost           { return p.host }
func (p PreviewPublication) Stamp() PreviewStamp         { return p.selection.Stamp }
func (p PreviewPublication) Revision() uint64            { return p.revision }
func (p PreviewPublication) Operation() uint64           { return p.operation }
func (p PreviewPublication) Selection() PreviewSelection { return clonePreviewSelection(p.selection) }
func clonePreviewSelection(s PreviewSelection) PreviewSelection {
	s.Items = append([]PreviewItem(nil), s.Items...)
	s.ResolvedNodes = slices.Clone(s.ResolvedNodes)
	s.Candidates = slices.Clone(s.Candidates)
	s.Notes = slices.Clone(s.Notes)
	if s.Page != nil {
		page := *s.Page
		s.Page = &page
	}
	if s.Visibility != nil {
		v := CloneFeedbackVisibility(*s.Visibility)
		s.Visibility = &v
	}
	return s
}

type PreviewEvent interface{ previewEvent() }
type PreviewHostChanged struct{ Host PreviewHost }

func (PreviewHostChanged) previewEvent() {}

type PreviewCandidatesChanged struct{ Selection PreviewSelection }

func (PreviewCandidatesChanged) previewEvent() {}

type PreviewRefresh struct{}

func (PreviewRefresh) previewEvent() {}

type PreviewPublishAccepted struct{ Publication PreviewPublication }

func (PreviewPublishAccepted) previewEvent() {}

type PreviewPublishFailed struct{ Publication PreviewPublication }

func (PreviewPublishFailed) previewEvent() {}

type PreviewDispose struct{}

func (PreviewDispose) previewEvent() {}

type PreviewEffect interface{ previewEffect() }
type PreviewPublish struct{ Publication PreviewPublication }

func (PreviewPublish) previewEffect() {}

type PreviewCancel struct{ Publication PreviewPublication }

func (PreviewCancel) previewEffect() {}

type PreviewRefreshNeeded struct{}

func (PreviewRefreshNeeded) previewEffect() {}

// PreviewStop requires the wrapper to cancel and join its sole owned worker.
type PreviewStop struct{}

func (PreviewStop) previewEffect() {}

type PreviewUnavailable struct{}

func (PreviewUnavailable) previewEffect() {}

func NextPreview(state PreviewState, event PreviewEvent) (PreviewState, []PreviewEffect) {
	if state.Phase() == PreviewStopped {
		return state, nil
	}
	snapshot := state.snapshot()
	invalidate := func() []PreviewEffect {
		var effects []PreviewEffect
		if publishing, ok := state.payload.(previewPublishing); ok {
			effects = append(effects, PreviewCancel{Publication: publishing.publication})
		}
		if snapshot.host.ID == "" || snapshot.selection == nil {
			state.payload = previewDormant{snapshot}
		} else {
			state.payload = previewDirty{snapshot}
			effects = append(effects, PreviewRefreshNeeded{})
		}
		return effects
	}
	switch e := event.(type) {
	case PreviewDispose:
		var effects []PreviewEffect
		if p, ok := state.payload.(previewPublishing); ok {
			effects = append(effects, PreviewCancel{Publication: p.publication})
		}
		state.payload = previewStopped{}
		return state, append(effects, PreviewStop{})
	case PreviewHostChanged:
		if snapshot.host == e.Host {
			return state, nil
		}
		snapshot.host = e.Host
		effects := invalidate()
		return state, effects
	case PreviewCandidatesChanged:
		if snapshot.selection != nil && samePreviewSelection(*snapshot.selection, e.Selection) {
			return state, nil
		}
		selection := clonePreviewSelection(e.Selection)
		snapshot.selection = &selection
		effects := invalidate()
		return state, effects
	case PreviewRefresh:
		if state.Phase() != PreviewDirty {
			return state, nil
		}
		// Exhaustion fails closed rather than reusing a retired operation identity.
		if state.next == ^uint64(0) {
			state.payload = previewStopped{}
			return state, []PreviewEffect{PreviewUnavailable{}, PreviewStop{}}
		}
		state.next++
		publication := PreviewPublication{host: snapshot.host, selection: clonePreviewSelection(*snapshot.selection), revision: state.next, operation: state.next}
		state.payload = previewPublishing{publication}
		return state, []PreviewEffect{PreviewPublish{Publication: publication}}
	case PreviewPublishAccepted:
		if p, ok := state.payload.(previewPublishing); ok && samePreviewPublication(p.publication, e.Publication) {
			state.payload = previewShown{p.publication}
		}
	case PreviewPublishFailed:
		if p, ok := state.payload.(previewPublishing); ok && samePreviewPublication(p.publication, e.Publication) {
			state.payload = previewDirty{snapshot}
			// An explicit Refresh or a source event retries; failure never spins.
			return state, []PreviewEffect{PreviewUnavailable{}}
		}
	}
	return state, nil
}

func (s PreviewState) snapshot() previewSnapshot {
	switch p := s.payload.(type) {
	case previewDormant:
		return p.previewSnapshot
	case previewDirty:
		return p.previewSnapshot
	case previewPublishing:
		return previewSnapshot{host: p.publication.host, selection: &p.publication.selection}
	case previewShown:
		return previewSnapshot{host: p.publication.host, selection: &p.publication.selection}
	default:
		return previewSnapshot{}
	}
}
func samePreviewPublication(a, b PreviewPublication) bool {
	return a.operation != 0 && a.operation == b.operation && a.revision == b.revision && a.host == b.host && a.selection.Stamp == b.selection.Stamp
}

// Comparing the bounded candidate values also detects changed contents whose
// producer accidentally reused a generation stamp.
func samePreviewSelection(a, b PreviewSelection) bool {
	return a.Stamp == b.Stamp && a.Remaining == b.Remaining && a.NoteRemaining == b.NoteRemaining && reflect.DeepEqual(a.Visibility, b.Visibility) && slices.Equal(a.Items, b.Items) && slices.Equal(a.Candidates, b.Candidates) && slices.Equal(a.Notes, b.Notes) && (a.Page == nil && b.Page == nil || a.Page != nil && b.Page != nil && *a.Page == *b.Page)
}
