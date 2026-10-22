package couchtty

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type submissionPreviewFake struct {
	*submissionCastFake
	muPreview  sync.Mutex
	selection  board.Selection
	host       core.PreviewHost
	life       chan struct{}
	updates    chan struct{}
	candidates chan struct{}
	published  chan core.PreviewPublication
	gate       chan struct{}
	freeze     func()
}

func newSubmissionPreviewFake(f *submissionCastFake) *submissionPreviewFake {
	return &submissionPreviewFake{submissionCastFake: f, selection: board.Selection{Document: 1, TabID: "tab", SketchGeneration: 1, Sketches: []board.SelectedSketch{{ID: "shape", NextSequence: 5, Digest: "a"}}}, host: core.PreviewHost{ID: "host", Generation: 1}, life: make(chan struct{}), updates: make(chan struct{}, 1), candidates: make(chan struct{}, 1), published: make(chan core.PreviewPublication, 8)}
}
func (f *submissionPreviewFake) SubmissionSelection() (board.Selection, error) {
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	s := f.selection
	s.Sketches = append([]board.SelectedSketch(nil), s.Sketches...)
	return s, nil
}
func (f *submissionPreviewFake) FreezeSubmissionPreview(host core.PreviewHost, g uint64, chosen []board.SelectedSketch) (board.Selection, error) {
	f.muPreview.Lock()
	defer f.muPreview.Unlock()
	if f.freeze != nil {
		f.freeze()
	}
	if host != f.host || g != f.selection.SketchGeneration {
		return board.Selection{}, errors.New("changed")
	}
	s := f.selection
	s.Sketches = append([]board.SelectedSketch(nil), chosen...)
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
func (f *submissionPreviewFake) PublishSubmissionPreview(ctx context.Context, p core.PreviewPublication) error {
	f.published <- p
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestSubmissionRequiresAcknowledgedPreviewAndAtomicFreeze(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	m.current = func() (submissionCast, error) { return f, nil }
	if _, err := m.request(context.Background(), "prepare", "before", target); err == nil {
		t.Fatal("unseen shapes admitted")
	}
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	p := <-f.published
	m.mu.Lock() // Worker must acquire this lock after its publication result.
	m.mu.Unlock()
	// Synchronize with explicit worker barrier rather than sleeps.
	_ = p
	deadline := time.After(2 * time.Second)
	for {
		m.mu.Lock()
		shown := m.previewState.Phase() == core.PreviewShown
		m.mu.Unlock()
		if shown {
			break
		}
		select {
		case <-deadline:
			t.Fatal("preview not acknowledged")
		case <-time.After(time.Millisecond):
		}
	}
	f.muPreview.Lock()
	f.freeze = func() { f.selection.SketchGeneration++ }
	f.muPreview.Unlock()
	if _, err := m.request(context.Background(), "prepare", "stale", target); err == nil {
		t.Fatal("changed version admitted after shown ACK")
	}
	base.mu.Lock()
	prepared := base.prepared
	base.mu.Unlock()
	if prepared != 0 {
		t.Fatal("stale preview started effects")
	}
	cancel()
	<-done
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
	m, base, target := newSubmissionManagerTest(t)
	f := newSubmissionPreviewFake(base)
	m.current = func() (submissionCast, error) { return f, nil }
	ctx, cancel := context.WithCancel(m.ctx)
	done := make(chan struct{})
	go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
	<-f.published
	waitSubmissionPreviewShown(t, m)
	f.muPreview.Lock()
	f.host.Generation++
	f.muPreview.Unlock()
	// The host change has not yet reached the manager. Atomic host+board freeze
	// must still reject it; a separate pre-check would leave this gap open.
	if _, err := m.request(context.Background(), "prepare", "replaced", target); err == nil {
		t.Fatal("retired host enabled preparation")
	}
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
		f.selection.Sketches = append(f.selection.Sketches, board.SelectedSketch{ID: string(rune('a' + i)), NextSequence: 5, Digest: "a"})
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
	ctx, cancel := context.WithCancel(ctx)
	m.runSubmissionPreview(ctx, m.newSubmissionPreviewRun(cast, cancel))
}
