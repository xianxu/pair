package board

import "github.com/xianxu/pair/cmd/internal/webcast/recording"

type pageLoadScope struct {
	generation uint64
	managed    bool
	pending    bool
	load       *recording.PageLoad
}

// BeginPageLoad makes a newer recorder observation pending before its snapshot
// IO. This leaf-lock operation never calls recorder or draft-service callbacks.
func (b *Board) BeginPageLoad(generation uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || generation == 0 || generation <= b.pageLoad.generation {
		return false
	}
	b.pageLoad = pageLoadScope{generation: generation, managed: true, pending: true}
	b.publishPageLoadLocked()
	return true
}

// ConfirmPageLoad publishes only the exact current generation's accepted
// recording identity. A later pending observation cannot be overwritten.
func (b *Board) ConfirmPageLoad(generation uint64, load *recording.PageLoad) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.broken != nil || !b.pageLoad.pending || generation != b.pageLoad.generation || b.st.Top == nil || !recording.SamePageLoad(load, b.st.CurrentPageLoad()) {
		return false
	}
	b.pageLoad.pending = false
	b.pageLoad.load = &recording.PageLoad{Target: load.Target, Loader: load.Loader, Root: load.Root}
	b.publishPageLoadLocked()
	return true
}

// UnavailablePageLoad retires a failed or closing observation without restoring
// the last recorded page as current authority. Older failures have no effects.
func (b *Board) UnavailablePageLoad(generation uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || generation < b.pageLoad.generation || b.pageLoad.managed && generation == b.pageLoad.generation && !b.pageLoad.pending && b.pageLoad.load == nil {
		return false
	}
	b.pageLoad = pageLoadScope{generation: generation, managed: true}
	b.publishPageLoadLocked()
	return true
}

// PageLoadReady checks fresh page work independently of capability admission.
// Exact receipt and previously admitted delivery recovery do not use this gate.
func (b *Board) PageLoadReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.closed && b.broken == nil && b.st.Live && !b.st.Ended && b.st.Top != nil && b.currentPageLoadLocked() != nil
}

// PageFeedbackReady admits a fresh page mutation, including its item's load.
// Callers reconcile exact receipts and conditional historical resolutions first.
// Finishing a label on the same known page remains possible while paused.
func (b *Board) PageFeedbackReady(record recording.Record) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pageFeedbackReadyLocked(record)
}

func (b *Board) pageFeedbackReadyLocked(record recording.Record) bool {
	if b.closed || b.broken != nil || b.st.Ended || b.st.Top == nil {
		return false
	}
	load := b.currentPageLoadLocked()
	if load == nil {
		return false
	}
	switch r := record.(type) {
	case *recording.SketchLabel:
		m, ok := b.st.Sketches.Metadata(r.Sketch)
		return ok && b.st.OnPageLoad(m.Doc, load)
	case *recording.NoteEdit:
		return b.st.Live && b.noteOnPageLoadLocked(r.ID, load)
	case *recording.NoteState:
		return b.st.Live && b.noteOnPageLoadLocked(r.ID, load)
	case *recording.Note, *recording.Stroke:
		return b.st.Live
	case *recording.SketchBegin:
		return b.st.Live && b.st.OnPageLoad(r.Doc, load)
	default:
		if !b.st.Live {
			return false
		}
		id := recording.SketchID(record)
		m, ok := b.st.Sketches.Metadata(id)
		return ok && b.st.OnPageLoad(m.Doc, load)
	}
}

func (b *Board) noteOnPageLoadLocked(id string, load *recording.PageLoad) bool {
	for _, note := range b.st.Notes {
		if note.ID == id {
			return b.st.OnPageLoad(note.Seg, load)
		}
	}
	return false
}

func (b *Board) sketchOnCurrentPageLocked(id string) bool {
	load := b.currentPageLoadLocked()
	if load == nil {
		return false
	}
	if id == "" {
		return true
	}
	m, ok := b.st.Sketches.Metadata(id)
	return ok && b.st.OnPageLoad(m.Doc, load)
}

// currentPageLoadLocked is the live feedback scope, separate from DocSeg's
// activation/draft fence. Runtime pending observation is installed here by the
// recorder; without that owner, accepted recording metadata supplies the scope.
func (b *Board) currentPageLoadLocked() *recording.PageLoad {
	if b.pageLoad.managed {
		if b.pageLoad.pending || !recording.SamePageLoad(b.pageLoad.load, b.st.CurrentPageLoad()) {
			return nil
		}
		load := *b.pageLoad.load
		return &load
	}
	return b.st.CurrentPageLoad()
}

// publishPageLoadLocked retires display continuity before publishing the new
// scope. It changes only projections; canonical history and receipts remain.
func (b *Board) publishPageLoadLocked() {
	b.resetSketchDeliveryLocked()
	b.refreshLocked()
	b.refreshOperatorLocked(nil)
	b.submissionChangedLocked()
}

func sameOptionalPageLoad(a, b *recording.PageLoad) bool {
	if a == nil || b == nil {
		return a == b
	}
	return recording.SamePageLoad(a, b)
}

// CurrentPageLoad returns owned, confirmed scope, including while paused.
// Pending, failed, ended or closed scopes cannot provide anchor authority.
func (b *Board) CurrentPageLoad() *recording.PageLoad {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.broken != nil || b.st.Ended {
		return nil
	}
	return b.currentPageLoadLocked()
}
