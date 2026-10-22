package couchtty

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type submissionPreviewCast interface {
	submissionCast
	FreezeSubmissionPreview(core.PreviewHost, uint64, []board.SelectedSketch) (board.Selection, error)
	SubmissionCandidateUpdates() (<-chan struct{}, func())
	SubmissionPreviewUpdates() (<-chan struct{}, func())
	SubmissionPreviewHost(context.Context) (core.PreviewHost, <-chan struct{}, error)
	PublishSubmissionPreview(context.Context, core.PreviewPublication) error
}
type submissionPreviewRun struct {
	cast   submissionPreviewCast
	wake   chan struct{}
	cancel context.CancelFunc
	done   chan struct{}
	host   core.PreviewHost
}

var errSubmissionPreview = errors.New("Selection changed; review the updated sketch preview and submit again")

func (m *submissionManager) startSubmissionPreview(cast submissionCast) {
	m.previewStart.Lock()
	defer m.previewStart.Unlock()
	c, ok := cast.(submissionPreviewCast)
	if !ok {
		return
	}
	m.mu.Lock()
	old := m.previewRun
	m.mu.Unlock()
	if old != nil {
		old.cancel()
		<-old.done
	}
	ctx, cancel := context.WithCancel(m.ctx)
	run := m.newSubmissionPreviewRun(c, cancel)
	// GoTracked owns the refresh worker and all of its bounded publication IO.
	m.spawn(func() { m.runSubmissionPreview(ctx, run) })
}

func (m *submissionManager) newSubmissionPreviewRun(cast submissionPreviewCast, cancel context.CancelFunc) *submissionPreviewRun {
	run := &submissionPreviewRun{cast: cast, wake: make(chan struct{}, 1), cancel: cancel, done: make(chan struct{})}
	m.mu.Lock()
	m.previewRun = run
	m.previewState = core.PreviewState{}
	m.mu.Unlock()
	return run
}

func (m *submissionManager) runSubmissionPreview(ctx context.Context, run *submissionPreviewRun) {
	cast := run.cast
	candidates, stopCandidates := cast.SubmissionCandidateUpdates()
	defer stopCandidates()
	hosts, stopHosts := cast.SubmissionPreviewUpdates()
	defer stopHosts()
	defer func() {
		run.cancel()
		m.mu.Lock()
		if m.previewRun == run {
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewDispose{})
			m.previewRun = nil
		}
		m.mu.Unlock()
		close(run.done)
	}()
	var delivered map[core.PreviewItem]struct{}
	var cachedDeliveryGeneration uint64
	deliveredLoaded := false
	for {
		if ctx.Err() != nil {
			return
		}
		host, life, err := cast.SubmissionPreviewHost(ctx)
		selected, selectionErr := cast.SubmissionSelection()
		m.mu.Lock()
		deliveredGeneration := m.deliveredGeneration
		m.mu.Unlock()
		if err == nil && selectionErr == nil && (!deliveredLoaded || cachedDeliveryGeneration != deliveredGeneration) {
			delivered, err = m.store.DeliveredSketchVersions(filepath.Base(cast.RecordingPath()))
			if err == nil {
				deliveredLoaded = true
				cachedDeliveryGeneration = deliveredGeneration
			}
		}
		var publication *core.PreviewPublication
		m.mu.Lock()
		if m.previewRun != run {
			m.mu.Unlock()
			return
		}
		if err != nil || selectionErr != nil || deliveredGeneration != m.deliveredGeneration {
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewHostChanged{})
		} else {
			run.host = host
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewHostChanged{Host: host})
			items := make([]core.PreviewItem, 0, len(selected.Sketches))
			for _, v := range selected.Sketches {
				items = append(items, core.PreviewItem{RecordingID: filepath.Base(cast.RecordingPath()), SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest})
			}
			selection := core.SelectSketchPreview(items, delivered, core.PreviewStamp{Board: selected.SketchGeneration, Delivered: deliveredGeneration})
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewCandidatesChanged{Selection: selection})
			var effects []core.PreviewEffect
			m.previewState, effects = core.NextPreview(m.previewState, core.PreviewRefresh{})
			for _, effect := range effects {
				if p, ok := effect.(core.PreviewPublish); ok {
					v := p.Publication
					publication = &v
				}
			}
		}
		m.mu.Unlock()
		if publication != nil {
			ioctx, stop := context.WithTimeout(ctx, 5*time.Second)
			result := make(chan error, 1)
			m.spawn(func() { result <- cast.PublishSubmissionPreview(ioctx, *publication) })
			refresh := false
			var publishErr error
			select {
			case publishErr = <-result:
			case <-ctx.Done():
				stop()
				<-result
				return
			case <-life:
				refresh = true
			case _, open := <-candidates:
				if !open {
					stop()
					<-result
					return
				}
				refresh = true
			case _, open := <-hosts:
				if !open {
					stop()
					<-result
					return
				}
				refresh = true
			case <-run.wake:
				refresh = true
			}
			stop()
			if refresh {
				<-result
				continue
			}
			m.mu.Lock()
			if m.previewRun == run {
				if publishErr == nil {
					m.previewState, _ = core.NextPreview(m.previewState, core.PreviewPublishAccepted{Publication: *publication})
				} else {
					m.previewState, _ = core.NextPreview(m.previewState, core.PreviewPublishFailed{Publication: *publication})
				}
			}
			m.mu.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-life:
		case _, open := <-candidates:
			if !open {
				return
			}
		case _, open := <-hosts:
			if !open {
				return
			}
		case <-run.wake:
		}
	}
}

// Called with manager.mu held; Board's generation fence is the atomic boundary.
func (m *submissionManager) freezeSubmissionPreview(cast submissionCast, selected board.Selection) (board.Selection, error) {
	if len(selected.Sketches) == 0 {
		return selected, nil
	}
	run := m.previewRun
	if run == nil || run.cast != cast {
		return board.Selection{}, errSubmissionPreview
	}
	var effects []core.PreviewEffect
	m.previewState, effects = core.NextPreview(m.previewState, core.PreviewPrepare{Host: run.host, Stamp: core.PreviewStamp{Board: selected.SketchGeneration, Delivered: m.deliveredGeneration}})
	for _, effect := range effects {
		if p, ok := effect.(core.PreviewFreezeProposal); ok {
			var chosen []board.SelectedSketch
			for _, v := range p.Publication.Selection().Items {
				chosen = append(chosen, board.SelectedSketch{ID: v.SketchID, NextSequence: v.Sequence, Digest: v.Digest})
			}
			frozen, err := run.cast.FreezeSubmissionPreview(p.Publication.Host(), p.Publication.Stamp().Board, chosen)
			if err == nil {
				frozen.SketchRemaining = p.Publication.Selection().Remaining
				return frozen, nil
			}
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewFreezeRejected{Publication: p.Publication})
		}
	}
	select {
	case run.wake <- struct{}{}:
	default:
	}
	return board.Selection{}, errSubmissionPreview
}
func (m *submissionManager) submissionDeliveredLocked() {
	m.deliveredGeneration++
	if m.previewRun != nil {
		select {
		case m.previewRun.wake <- struct{}{}:
		default:
		}
	}
}
