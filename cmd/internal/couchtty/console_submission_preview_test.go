package couchtty

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type submissionPreviewFake struct {
	*submissionCastFake
	muPreview      sync.Mutex
	selection      board.Selection
	host           core.PreviewHost
	life           chan struct{}
	updates        chan struct{}
	candidates     chan struct{}
	published      chan core.PreviewPublication
	gate           chan struct{}
	freeze         func()
	visibility     core.FeedbackVisibility
	readVisibility func()
	resolvePreview func()
	publishErr     error
	stale          chan string
	unsaved        string
	readUnsaved    func()
}

func previewFakeSketch(id string) board.SelectedSketch {
	source := query.SketchSourceFacts{T: 1, RecordIndex: 1, Doc: 1, Tab: "tab", URL: "https://example.test/", VP: recording.Viewport{W: 800, H: 600, DPR: 1}}
	finish := source
	finish.T = 2
	finish.RecordIndex = 2
	return board.SelectedSketch{ID: id, NextSequence: 5, Digest: strings.Repeat("a", 64), Source: source, Finish: finish}
}

func newSubmissionPreviewFake(f *submissionCastFake) *submissionPreviewFake {
	return &submissionPreviewFake{submissionCastFake: f, selection: board.Selection{Document: 1, TabID: "tab", SketchGeneration: 1, Sketches: []board.SelectedSketch{previewFakeSketch("shape")}}, host: core.PreviewHost{ID: "host", Generation: 1}, life: make(chan struct{}), updates: make(chan struct{}, 1), candidates: make(chan struct{}, 1), published: make(chan core.PreviewPublication, 8)}
}

// setVisibleFrame is explicit fixture paint completion, independent of selection reads.
func (f *submissionPreviewFake) setVisibleFrame() {
	source := previewFakeSketch("frame").Finish
	f.visibility = core.FeedbackVisibility{Revision: 1, MaskScope: "scope", Scene: sketchimage.PageScene{RecordingID: filepath.Base(f.RecordingPath()), SketchSourceFacts: source}}
	for _, s := range f.selection.Sketches {
		f.visibility.Sketches = append(f.visibility.Sketches, core.VisibleSketch{ID: s.ID, Sequence: s.NextSequence, Digest: s.Digest, Paths: []core.VisiblePath{{ID: s.ID + "-path", Locator: []int{0}, Box: recording.Box{0, 0, 100, 100}, Clip: recording.Box{0, 0, 800, 600}}}})
	}
	for _, n := range f.selection.Notes {
		f.visibility.Notes = append(f.visibility.Notes, core.PreviewNote{ID: n.ID, Revision: n.Revision})
	}
}
func (f *submissionPreviewFake) ReadSubmissionVisibility(ctx context.Context, host core.PreviewHost) (core.FeedbackVisibility, error) {
	if f.readVisibility != nil {
		f.readVisibility()
	}
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	if host != f.host {
		return core.FeedbackVisibility{}, errors.New("retired host")
	}
	return core.CloneFeedbackVisibility(f.visibility), ctx.Err()
}
func (f *submissionPreviewFake) ResolveSubmissionPreview(ctx context.Context, host core.PreviewHost, selection core.PreviewSelection) (core.PreviewSelection, error) {
	if f.resolvePreview != nil {
		f.resolvePreview()
	}
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	if host != f.host {
		return core.PreviewSelection{}, errors.New("retired host")
	}
	for _, s := range selection.Visibility.Sketches {
		chosen := false
		for _, c := range selection.Items {
			if c.SketchID == s.ID {
				chosen = true
			}
		}
		if !chosen {
			continue
		}
		for range s.Paths {
			selection.ResolvedNodes = append(selection.ResolvedNodes, 7)
		}
	}
	return selection, ctx.Err()
}
func (f *submissionPreviewFake) SubmissionUnsavedNote(context.Context) (string, error) {
	if f.readUnsaved != nil {
		f.readUnsaved()
	}
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	return f.unsaved, nil
}
func (f *submissionPreviewFake) SubmissionSelection() (board.Selection, error) {
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	s := f.selection
	s.Sketches = append([]board.SelectedSketch(nil), s.Sketches...)
	s.Notes = slices.Clone(s.Notes)
	return s, nil
}
func (f *submissionPreviewFake) FreezeSubmissionPreview(host core.PreviewHost, selection core.PreviewSelection) (board.Selection, error) {
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	if f.freeze != nil {
		f.freeze()
	}
	chosen := make([]board.SelectedSketch, 0, len(selection.Candidates))
	for _, c := range selection.Candidates {
		chosen = append(chosen, board.SelectedSketch{ID: c.Item.SketchID, NextSequence: c.Item.Sequence, Digest: c.Item.Digest, Source: c.Source, Finish: c.Finish})
	}
	notes := make([]board.SelectedNote, 0, len(selection.Notes))
	for _, n := range selection.Notes {
		notes = append(notes, board.SelectedNote{ID: n.ID, Revision: n.Revision})
	}
	validNotes := true
	for _, n := range notes {
		if !slices.Contains(f.selection.Notes, n) {
			validNotes = false
		}
	}
	if host != f.host || selection.Stamp.Board != f.selection.SketchGeneration || !validNotes {
		return board.Selection{}, errors.New("changed")
	}
	s := f.selection
	s.Sketches = append([]board.SelectedSketch(nil), chosen...)
	s.Notes = slices.Clone(notes)
	return s, nil
}
func (f *submissionPreviewFake) SubmissionCandidateUpdates() (<-chan struct{}, func()) {
	return f.candidates, func() {}
}
func (f *submissionPreviewFake) SubmissionPreviewUpdates() (<-chan struct{}, func()) {
	return f.updates, func() {}
}
func (f *submissionPreviewFake) SubmissionPreviewHost(context.Context) (core.PreviewHost, <-chan struct{}, error) {
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	return f.host, f.life, nil
}
func (f *submissionPreviewFake) MarkSubmissionPreviewStale(_ context.Context, _ core.PreviewHost, reason string) error {
	if f.stale != nil {
		f.stale <- reason
	}
	return nil
}
func (f *submissionPreviewFake) PublishSubmissionPreview(ctx context.Context, p core.PreviewPublication) error {
	f.published <- p
	f.muPreview.Lock()
	err := f.publishErr
	f.muPreview.Unlock()
	if err != nil {
		return err
	}
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// pair#849: a press takes the board's feedback as it is; no preview needs to
// have been shown, and nothing the page does refuses it.
func TestSubmissionPressNeedsNoPreview(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.selection.Notes = []board.SelectedNote{{ID: "note", Revision: 1}}
	m.current = func() (submissionCast, error) { return f, nil }
	status, err := m.request(context.Background(), "prepare", "unseen", target)
	if err != nil || status.State != "preparing" || status.Sketches != 1 {
		t.Fatalf("press refused without a preview: %+v %v", status, err)
	}
	submissionWait(t, m, "unseen", target, "prepared")
}

func waitSubmissionPreviewShown(t *testing.T, m *submissionManager) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		m.mu.Lock()
		shown := m.previewState.Phase() == core.PreviewShown
		m.mu.Unlock()
		if shown {
			return
		}
		select {
		case <-deadline:
			t.Fatal("preview not shown")
		case <-time.After(time.Millisecond):
		}
	}
}
func TestSubmissionPreviewHostReplacementAndDispose(t *testing.T) {
	m, base, _ := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	m.current = func() (submissionCast, error) { return f, nil }
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	<-f.published
	waitSubmissionPreviewShown(t, m)
	cancel()
	<-done
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.previewRun != nil || m.previewState.Phase() != core.PreviewStopped {
		t.Fatal("disposed worker retained readiness")
	}
}
func TestSubmissionPreviewCancellationJoinsActivePublication(t *testing.T) {
	m, base, _ := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.gate = make(chan struct{})
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	first := <-f.published
	f.muPreview.Lock()
	f.selection.SketchGeneration++
	f.muPreview.Unlock()
	f.candidates <- struct{}{}
	second := <-f.published
	if second.Stamp().Board == first.Stamp().Board || second.Operation() == first.Operation() {
		t.Fatal("candidate change reused retired publication")
	}
	cancel()
	<-done
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.previewState.Phase() != core.PreviewStopped {
		t.Fatal("late result revived disposed preview")
	}
}
func TestSubmissionPreviewFreezesEightOldestAndRemaining(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.selection.Sketches = nil
	for i := 0; i < 10; i++ {
		f.selection.Sketches = append(f.selection.Sketches, previewFakeSketch(string(rune('a'+i))))
	}
	m.current = func() (submissionCast, error) { return f, nil }
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	p := <-f.published
	waitSubmissionPreviewShown(t, m)
	if len(p.Selection().Items) != 8 || p.Selection().Remaining != 2 || p.Selection().Items[0].SketchID != "a" || p.Selection().Items[7].SketchID != "h" {
		t.Fatalf("preview %+v", p.Selection())
	}
	status, err := m.request(context.Background(), "prepare", "eight", target)
	if err != nil || status.Sketches != 8 || status.Remaining != 2 {
		t.Fatalf("status %+v %v", status, err)
	}
	submissionWait(t, m, "eight", target, "prepared")
	cancel()
	<-done
}

func watchSubmissionPreview(m *submissionManager, ctx context.Context, cast submissionPreviewCast) {
	if f, ok := cast.(interface{ setVisibleFrame() }); ok {
		f.setVisibleFrame()
	}
	ctx, cancel := context.WithCancel(ctx)
	m.runSubmissionPreview(ctx, m.newSubmissionPreviewRun(cast, cancel))
}

// A canceled publisher deliberately returns a late successful ACK only after
// Release. The production manager must join it and retire its model operation
// before starting another projection; scheduler timing cannot decide the test.
type controlledPreviewPublication struct {
	publication core.PreviewPublication
	canceled    chan struct{}
	release     chan struct{}
}
type controlledSubmissionPreview struct {
	*submissionPreviewFake
	calls chan *controlledPreviewPublication
}

func (f *controlledSubmissionPreview) PublishSubmissionPreview(ctx context.Context, p core.PreviewPublication) error {
	call := &controlledPreviewPublication{publication: p, canceled: make(chan struct{}), release: make(chan struct{})}
	f.calls <- call
	select {
	case <-ctx.Done():
		close(call.canceled)
		<-call.release
	case <-call.release:
	}
	return nil
}
func controlledPreviewCall(t *testing.T, f *controlledSubmissionPreview) *controlledPreviewPublication {
	t.Helper()
	select {
	case call := <-f.calls:
		return call
	default:
		t.Fatal("no replacement publication; canceled operation remained Publishing")
		return nil
	}
}
func TestSubmissionPreviewSettlesCanceledOperationBeforeUnchangedRefresh(t *testing.T) {
	for _, cause := range []string{"initial-host-wake", "duplicate-host", "receiver-catchup", "unchanged-candidates", "explicit-refresh"} {
		t.Run(cause, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				m, base, _ := newSubmissionManagerTest(t)
				f := &controlledSubmissionPreview{submissionPreviewFake: newSubmissionPreviewFake(base), calls: make(chan *controlledPreviewPublication, 4)}
				ctx, cancel := context.WithCancel(m.ctx)
				defer cancel()
				if cause == "initial-host-wake" {
					f.updates <- struct{}{}
				}
				done := make(chan struct{})
				go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
				synctest.Wait()
				first := controlledPreviewCall(t, f)
				if cause != "initial-host-wake" {
					switch cause {
					case "duplicate-host", "receiver-catchup":
						f.updates <- struct{}{}
					case "unchanged-candidates":
						f.candidates <- struct{}{}
					case "explicit-refresh":
						m.mu.Lock()
						m.previewRun.wake <- struct{}{}
						m.mu.Unlock()
					}
					synctest.Wait()
				}
				select {
				case <-first.canceled:
				default:
					t.Fatal("refresh did not cancel its publication")
				}
				if len(f.calls) != 0 {
					t.Fatal("replacement started before canceled IO joined")
				}
				close(first.release) // Return a late successful ACK from canceled IO.
				synctest.Wait()
				second := controlledPreviewCall(t, f)
				if second.publication.Operation() <= first.publication.Operation() || second.publication.Host() != first.publication.Host() || second.publication.Stamp() != first.publication.Stamp() {
					t.Fatal("unchanged refresh reused identity or changed candidates")
				}
				m.mu.Lock()
				phase := m.previewState.Phase()
				m.mu.Unlock()
				if phase != core.PreviewPublishing {
					t.Fatal("late canceled ACK showed a retired publication")
				}
				// Acknowledge only the exact replacement operation.
				close(second.release)
				synctest.Wait()
				m.mu.Lock()
				phase = m.previewState.Phase()
				m.mu.Unlock()
				if phase != core.PreviewShown {
					t.Fatalf("exact replacement ACK not Shown: %v", phase)
				}
				f.updates <- struct{}{}
				f.candidates <- struct{}{}
				synctest.Wait()
				m.mu.Lock()
				phase = m.previewState.Phase()
				m.mu.Unlock()
				if phase != core.PreviewShown || len(f.calls) != 0 {
					t.Fatal("unchanged notification republished after Shown")
				}
				cancel()
				synctest.Wait()
				<-done
			})
		})
	}
}
func TestSubmissionPreviewTerminalCancellationJoinsBeforeDispose(t *testing.T) {
	for _, cause := range []string{"context", "hosts-closed", "candidates-closed"} {
		t.Run(cause, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				m, base, _ := newSubmissionManagerTest(t)
				f := &controlledSubmissionPreview{submissionPreviewFake: newSubmissionPreviewFake(base), calls: make(chan *controlledPreviewPublication, 2)}
				ctx, cancel := context.WithCancel(m.ctx)
				defer cancel()
				done := make(chan struct{})
				go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
				synctest.Wait()
				call := controlledPreviewCall(t, f)
				switch cause {
				case "context":
					cancel()
				case "hosts-closed":
					close(f.updates)
				case "candidates-closed":
					close(f.candidates)
				}
				synctest.Wait()
				select {
				case <-call.canceled:
				default:
					t.Fatal("terminal event did not cancel IO")
				}
				select {
				case <-done:
					t.Fatal("worker returned before publication joined")
				default:
				}
				close(call.release)
				synctest.Wait()
				<-done
				m.mu.Lock()
				phase := m.previewState.Phase()
				run := m.previewRun
				m.mu.Unlock()
				if phase != core.PreviewStopped || run != nil || len(f.calls) != 0 {
					t.Fatal("late successful ACK revived disposed worker")
				}
			})
		})
	}
}

func TestSubmissionPreviewExpiredPublicationCannotBecomeShown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, base, _ := newSubmissionManagerTest(t)
		f := &controlledSubmissionPreview{submissionPreviewFake: newSubmissionPreviewFake(base), calls: make(chan *controlledPreviewPublication, 2)}
		ctx, cancel := context.WithCancel(m.ctx)
		defer cancel()
		done := make(chan struct{})
		go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
		synctest.Wait()
		expired := controlledPreviewCall(t, f)
		// Blocking on the owned cancellation event lets the synctest clock
		// advance to the production deadline without guessing its duration.
		<-expired.canceled
		synctest.Wait()
		close(expired.release) // The transport replies success after its deadline.
		synctest.Wait()
		m.mu.Lock()
		phase := m.previewState.Phase()
		m.mu.Unlock()
		if phase != core.PreviewDirty || len(f.calls) != 0 {
			t.Fatalf("expired ACK became authoritative or retried without a trigger: %v", phase)
		}
		f.updates <- struct{}{}
		synctest.Wait()
		replacement := controlledPreviewCall(t, f)
		close(replacement.release)
		synctest.Wait()
		m.mu.Lock()
		phase = m.previewState.Phase()
		m.mu.Unlock()
		if phase != core.PreviewShown {
			t.Fatal("explicit refresh did not recover expired publication")
		}
		cancel()
		synctest.Wait()
		<-done
	})
}

func TestSubmissionPreviewLifetimeRetirementJoinsBeforeReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, base, _ := newSubmissionManagerTest(t)
		f := &controlledSubmissionPreview{submissionPreviewFake: newSubmissionPreviewFake(base), calls: make(chan *controlledPreviewPublication, 2)}
		ctx, cancel := context.WithCancel(m.ctx)
		defer cancel()
		done := make(chan struct{})
		go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
		synctest.Wait()
		first := controlledPreviewCall(t, f)
		f.muPreview.Lock()
		oldLife := f.life
		f.life = make(chan struct{})
		f.host.Generation++
		newHost := f.host
		f.muPreview.Unlock()
		close(oldLife)
		synctest.Wait()
		select {
		case <-first.canceled:
		default:
			t.Fatal("retired host did not cancel publication")
		}
		if len(f.calls) != 0 {
			t.Fatal("replacement host published before old IO joined")
		}
		close(first.release)
		synctest.Wait()
		replacement := controlledPreviewCall(t, f)
		if replacement.publication.Host() != newHost || replacement.publication.Operation() <= first.publication.Operation() {
			t.Fatal("replacement reused retired host/operation")
		}
		close(replacement.release)
		synctest.Wait()
		m.mu.Lock()
		phase := m.previewState.Phase()
		m.mu.Unlock()
		if phase != core.PreviewShown {
			t.Fatal("new host ACK not Shown")
		}
		cancel()
		synctest.Wait()
		<-done
	})
}

func TestSubmissionRetryOfVisibleAckInvalidatesDeliveredPreview(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, base)
	m.spawn = func(fn func()) { fn() }
	if _, err := m.request(context.Background(), "prepare", "visible-ack", target); err != nil {
		t.Fatal(err)
	}
	submissionWait(t, m, "visible-ack", target, "prepared")
	if _, err := m.request(context.Background(), "begin", "visible-ack", target); err != nil {
		t.Fatal(err)
	}
	// Model the visible on-disk disposition after a previous ACK whose directory
	// durability confirmation was lost. The manager never observed a successful
	// transition, so its delivered-version cache still has the old clock.
	if _, err := m.store.Transition("visible-ack", target, core.Ack); err != nil {
		t.Fatal(err)
	}
	wake := make(chan struct{}, 1)
	m.mu.Lock()
	m.previewRun = &submissionPreviewRun{wake: wake}
	m.mu.Unlock()
	if _, err := m.request(context.Background(), "ack", "visible-ack", target); err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
	default:
		t.Fatal("confirmed retry retained stale delivered-version preview")
	}
}

// pair#849: the press captures each note's revision; an edit after it is a
// new revision the submission does not carry (resolving is by that revision).
func TestSubmissionPressCapturesNoteRevisions(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.selection.Sketches = nil
	f.selection.Notes = []board.SelectedNote{{ID: "note", Revision: 1}}
	m.current = func() (submissionCast, error) { return f, nil }
	var got board.Selection
	base.onSelection = func(s board.Selection) {
		f.muPreview.Lock()
		f.selection.Notes[0].Revision++ // edited after the press
		f.muPreview.Unlock()
		got = s
	}
	if _, err := m.request(context.Background(), "prepare", "captured", target); err != nil {
		t.Fatal(err)
	}
	submissionWait(t, m, "captured", target, "prepared")
	if len(got.Notes) != 1 || got.Notes[0] != (board.SelectedNote{ID: "note", Revision: 1}) {
		t.Fatalf("prepared notes %+v", got.Notes)
	}
}

// pair#849: the one refusal is a note the operator is still writing.
func TestSubmissionRefusesAnUnsavedNote(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.unsaved = `note "fix the header" has unsaved edits`
	m.current = func() (submissionCast, error) { return f, nil }
	_, err := m.request(context.Background(), "prepare", "unsaved", target)
	if err == nil || err.Error() != `note "fix the header" has unsaved edits; save or cancel it, then submit again` {
		t.Fatalf("unsaved note: %v", err)
	}
	base.mu.Lock()
	defer base.mu.Unlock()
	if base.prepared != 0 {
		t.Fatal("unsaved note started preparation")
	}
}

func TestSubmissionPreviewUsesCurrentVisibleSubset(t *testing.T) {
	m, base, _ := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.selection.Sketches = append(f.selection.Sketches, previewFakeSketch("hidden"))
	f.setVisibleFrame()
	f.visibility.Sketches = f.visibility.Sketches[:1]
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, stop := context.WithCancel(ctx)
		m.runSubmissionPreview(ctx, m.newSubmissionPreviewRun(f, stop))
	}()
	p := <-f.published
	cancel()
	<-done
	if len(p.Selection().Items) != 1 || p.Selection().Items[0].SketchID != "shape" {
		t.Fatalf("published unpainted feedback: %+v", p.Selection())
	}
}

// pair#849: the press reads outside the manager lock, and a page whose paint
// keeps changing -- geometry, revision, ordinals -- never refuses it.
func TestSubmissionPressReadsOutsideLockAndIgnoresPageChanges(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	m.current = func() (submissionCast, error) { return f, nil }
	f.readUnsaved = func() {
		if !m.mu.TryLock() {
			t.Error("browser IO ran under manager lock")
			return
		}
		m.mu.Unlock()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	defer func() { cancel(); <-done }()
	<-f.published
	waitSubmissionPreviewShown(t, m)
	f.muPreview.Lock()
	f.visibility.Revision++
	f.visibility.Scene.T++
	f.visibility.Scene.RecordIndex++
	f.visibility.Sketches[0].Paths[0].Box[0]++
	f.muPreview.Unlock()
	status, err := m.request(context.Background(), "prepare", "moved", target)
	if err != nil || status.State != "preparing" {
		t.Fatalf("changed page refused: %+v %v", status, err)
	}
	submissionWait(t, m, "moved", target, "prepared")
}

func TestSubmissionPreviewUnavailablePaintDoesNotUseSavedCandidates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, base, _ := newSubmissionManagerTest(t)
		f := newSubmissionPreviewFake(base)
		// Saved candidates exist, but there has been no completed paint.
		ctx, cancel := context.WithCancel(m.ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			ctx, stop := context.WithCancel(ctx)
			m.runSubmissionPreview(ctx, m.newSubmissionPreviewRun(f, stop))
		}()
		synctest.Wait()
		select {
		case <-f.published:
			t.Error("missing paint fell back to saved candidates")
		default:
		}
		cancel()
		synctest.Wait()
		<-done
	})
}

func TestSubmissionPressRechecksDeliveryGeneration(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	acknowledgeSubmissionFeedback(t, m, base)
	base.feedback.readUnsaved = func() { m.mu.Lock(); m.submissionDeliveredLocked(); m.mu.Unlock() }
	if _, err := m.request(context.Background(), "prepare", "delivery-race", target); err == nil {
		t.Fatal("delivery changed during the press's read but prepare proceeded")
	}
	base.mu.Lock()
	defer base.mu.Unlock()
	if base.prepared != 0 {
		t.Fatal("stale read started capture")
	}
}

// pair#849: the operator's shape. Notes were shown, then a labeled sketch the
// panel could not show: the panel says it is out of date, and the press
// submits both anyway; the panel is information, never a gate.
func TestSubmissionPanelOutOfDateNeverGates(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	f.selection.Sketches = nil
	f.selection.Notes = []board.SelectedNote{{ID: "note", Revision: 1}}
	f.stale = make(chan string, 4)
	m.current = func() (submissionCast, error) { return f, nil }
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	defer func() { cancel(); <-done }()
	<-f.published
	waitSubmissionPreviewShown(t, m)

	f.muPreview.Lock()
	f.selection.Sketches = []board.SelectedSketch{previewFakeSketch("shape")}
	f.selection.SketchGeneration++
	f.publishErr = errors.New("Selected sketch preview is unavailable.")
	f.setVisibleFrame()
	f.muPreview.Unlock()
	f.candidates <- struct{}{}
	<-f.published
	const why = "the panel did not show it: Selected sketch preview is unavailable."
	select {
	case got := <-f.stale:
		if got != why {
			t.Fatalf("panel marked %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("panel not marked out of date")
	}
	status, err := m.request(context.Background(), "prepare", "press", target)
	if err != nil || status.State != "preparing" || status.Sketches != 1 {
		t.Fatalf("out-of-date panel gated the press: %+v %v", status, err)
	}
}
