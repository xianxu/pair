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
	FreezeSubmissionPreview(core.PreviewHost, core.PreviewSelection) (board.Selection, error)
	SubmissionCandidateUpdates() (<-chan struct{}, func())
	SubmissionPreviewUpdates() (<-chan struct{}, func())
	ReadSubmissionVisibility(context.Context, core.PreviewHost) (core.FeedbackVisibility, error)
	ResolveSubmissionPreview(context.Context, core.PreviewHost, core.PreviewSelection) (core.PreviewSelection, error)
	SubmissionPreviewHost(context.Context) (core.PreviewHost, <-chan struct{}, error)
	PublishSubmissionPreview(context.Context, core.PreviewPublication) error
}
type submissionFeedbackCast interface {
	// This only updates a derived live projection, under the manager→Board
	// lock order. It performs no Store or browser IO.
	SetDeliveredFeedback(core.DeliveredFeedback) error
}
type submissionPreviewRun struct {
	cast   submissionPreviewCast
	wake   chan struct{}
	cancel context.CancelFunc
	done   chan struct{}
	host   core.PreviewHost
}

// submissionStaleCast marks the private panel out of date. Display only.
type submissionStaleCast interface {
	MarkSubmissionPreviewStale(context.Context, core.PreviewHost, string) error
}

var errPreviewHostRetired = errors.New("the private console host was replaced")

// previewProblem is the worker's one-line account of a refresh it could not
// complete, shown on the panel.
func previewProblem(host, selection, latest, delivery error, delivered, board bool) string {
	switch {
	case host != nil:
		return "the feedback could not be read: " + host.Error()
	case selection != nil:
		return "the board selection is unavailable: " + selection.Error()
	case delivery != nil:
		return "the delivered feedback could not be read: " + delivery.Error()
	case latest != nil:
		return "the board selection is unavailable: " + latest.Error()
	case delivered:
		return "a delivery completed while the preview was read"
	case board:
		return "the feedback changed while the preview was read"
	}
	return ""
}

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
	var delivered core.DeliveredFeedback
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
		var deliveryErr error
		if selectionErr == nil && (!deliveredLoaded || cachedDeliveryGeneration != deliveredGeneration) {
			delivered, deliveryErr = m.store.DeliveredFeedbackVersions(filepath.Base(cast.RecordingPath()))
			if deliveryErr == nil {
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
		// Live display does not depend on a private console host. The same
		// worker/confirmation cache serves viewers while that host is absent.
		if selectionErr == nil && deliveryErr == nil && deliveredLoaded && cachedDeliveryGeneration == m.deliveredGeneration {
			if display, ok := cast.(submissionFeedbackCast); ok {
				selectionErr = display.SetDeliveredFeedback(delivered)
				if selectionErr == nil {
					// Installing a confirmed note mask can change selection and its
					// generation. Never publish the pre-install witness.
					selected, selectionErr = cast.SubmissionSelection()
				}
			}
		}
		// Browser paint and recorder locator resolution never run under manager.mu.
		m.mu.Unlock()
		var selection core.PreviewSelection
		if err == nil && selectionErr == nil && deliveryErr == nil {
			selection, err = readVisibleSubmissionPreview(ctx, cast, host, selected, delivered, deliveredGeneration)
		}
		m.mu.Lock()
		if m.previewRun != run {
			m.mu.Unlock()
			return
		}
		latest, latestErr := cast.SubmissionSelection()
		select {
		case <-life:
			err = errPreviewHostRetired
		default:
		}
		stale := ""
		if err != nil || selectionErr != nil || latestErr != nil || deliveryErr != nil || deliveredGeneration != m.deliveredGeneration || latest.SketchGeneration != selected.SketchGeneration {
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewHostChanged{})
			if problem := previewProblem(err, selectionErr, latestErr, deliveryErr, deliveredGeneration != m.deliveredGeneration, latest.SketchGeneration != selected.SketchGeneration); problem != m.previewProblem {
				m.previewProblem, stale = problem, problem
			}
		} else {
			run.host = host
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewHostChanged{Host: host})
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
		if stale != "" && host.ID != "" {
			m.markPreviewStale(ctx, cast, host, stale)
		}
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
			// Cancellation joins IO but also has to settle its exact model operation.
			// Unchanged source notifications intentionally leave Host/Candidates alone;
			// without Failed, Refresh cannot advance a retired Publishing operation.
			accepted := !refresh && publishErr == nil && ioctx.Err() == nil
			stop()
			if refresh {
				<-result
			}
			failed := ""
			m.mu.Lock()
			if m.previewRun == run {
				if accepted && ctx.Err() == nil {
					m.previewState, _ = core.NextPreview(m.previewState, core.PreviewPublishAccepted{Publication: *publication})
					m.previewProblem = ""
				} else {
					m.previewState, _ = core.NextPreview(m.previewState, core.PreviewPublishFailed{Publication: *publication})
					if !refresh && ctx.Err() == nil {
						if publishErr == nil {
							publishErr = ioctx.Err()
						}
						m.previewProblem = "the panel did not show it: " + publishErr.Error()
						failed = m.previewProblem
					}
				}
			}
			m.mu.Unlock()
			if failed != "" {
				m.markPreviewStale(ctx, cast, publication.Host(), failed)
			}
			if refresh {
				continue
			}
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

// markPreviewStale is bounded display IO outside manager.mu. A host that is
// gone has no panel to mark, so its failure changes nothing.
func (m *submissionManager) markPreviewStale(ctx context.Context, cast submissionPreviewCast, host core.PreviewHost, reason string) {
	c, ok := cast.(submissionStaleCast)
	if !ok {
		return
	}
	ioctx, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	_ = c.MarkSubmissionPreviewStale(ioctx, host, reason)
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

// The caller supplies one captured authoritative generation. All browser/recorder
// IO is bounded by this shared deadline and occurs without manager.mu.
func readVisibleSubmissionPreview(ctx context.Context, cast submissionPreviewCast, host core.PreviewHost, selected board.Selection, delivered core.DeliveredFeedback, generation uint64) (core.PreviewSelection, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	view, err := cast.ReadSubmissionVisibility(ctx, host)
	if err != nil {
		return core.PreviewSelection{}, err
	}
	candidates := make([]core.PreviewSketchCandidate, 0, len(selected.Sketches))
	for _, v := range selected.Sketches {
		candidates = append(candidates, core.PreviewSketchCandidate{Item: core.PreviewItem{RecordingID: filepath.Base(cast.RecordingPath()), SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}, Source: v.Source, Finish: v.Finish})
	}
	notes := make([]core.PreviewNote, 0, len(selected.Notes))
	for _, v := range selected.Notes {
		notes = append(notes, core.PreviewNote{ID: v.ID, Revision: v.Revision})
	}
	selection, err := core.SelectVisibleFeedback(candidates, notes, delivered, core.PreviewStamp{Board: selected.SketchGeneration, Delivered: generation}, view)
	if err != nil {
		return core.PreviewSelection{}, err
	}
	selection, err = cast.ResolveSubmissionPreview(ctx, host, selection)
	if err == nil {
		err = ctx.Err()
	}
	return selection, err
}
