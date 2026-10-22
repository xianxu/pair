package couchtty

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type submissionCast interface {
	RecordingPath() string
	SubmissionSelection() (board.Selection, error)
	PrepareSubmission(context.Context, string, core.Target, board.Selection, string, func(core.SubmissionProgress)) (core.Submission, map[string][]byte, error)
	ResolveSubmission(context.Context, core.Submission) error
}

// The manager owns one bounded background operation. Files are the authority;
// the one volatile receipt only describes work not yet published to the store.
type submissionManager struct {
	mu                  sync.Mutex
	store               *core.Store
	ctx                 context.Context
	spawn               func(func())
	guard               func(core.Target) error
	current             func() (submissionCast, error)
	replay              func(context.Context, string) (string, error)
	job                 *submissionJob
	previewState        core.PreviewState
	previewRun          *submissionPreviewRun
	deliveredGeneration uint64
	previewStart        sync.Mutex // serializes host-worker replacement and joined retirement
}
type submissionJob struct {
	status couchmessage.SubmissionStatus
	busy   bool
}

func (c *Console) SetSubmissionStore(store *core.Store) {
	c.submissions = &submissionManager{store: store, ctx: c.lifetime, spawn: c.GoTracked, guard: c.submissionTarget, current: c.submissionCast, replay: c.submissionReplay}
}
func (c *Console) Submission(ctx context.Context, op, id string, target core.Target) (couchmessage.SubmissionStatus, error) {
	if c.submissions == nil {
		return couchmessage.SubmissionStatus{}, errors.New("this Couch has no submission store")
	}
	return c.submissions.request(ctx, op, id, target)
}
func (c *Console) submissionTarget(target core.Target) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.lifetime.Err(); err != nil {
		return err
	}
	p := c.panes[c.active]
	if !messagePaneLive(p) || c.focus != FocusActor(c.active) || p.thread.RepoScope != target.Scope || string(p.thread.Tag) != target.Tag {
		return errors.New("submission requires the calling draft's agent in view")
	}
	return nil
}
func (c *Console) submissionCast() (submissionCast, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.wcast.phase != webcastRunning || c.wcast.cast == nil {
		return nil, nil
	}
	return c.wcast.cast, nil
}
func (c *Console) submissionReplay(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	st := c.ReplayShare("replay-share", id)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		switch st.State {
		case "ready":
			return st.Base, nil
		case "opening":
		default:
			return "", fmt.Errorf("replay unavailable: %s", st.Error)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-tick.C:
			st = c.ReplayShare("replay-status", id)
		}
	}
}
func (m *submissionManager) status(s core.Submission) couchmessage.SubmissionStatus {
	out := couchmessage.SubmissionStatus{HowToRead: s.HowToRead, ID: s.ID, State: string(s.Delivery), Path: m.store.ManifestPath(s.ID), Target: s.Target, Sketches: len(s.Sketches)}
	if m.job != nil && m.job.status.ID == s.ID {
		out.Remaining = m.job.status.Remaining
	}
	return out
}
func (m *submissionManager) request(ctx context.Context, op, id string, target core.Target) (couchmessage.SubmissionStatus, error) {
	if err := ctx.Err(); err != nil {
		return couchmessage.SubmissionStatus{}, err
	}
	if !core.ValidID(id) {
		return couchmessage.SubmissionStatus{}, errors.New("invalid submission ID")
	}
	switch op {
	case "prepare", "status", "begin", "ack":
	default:
		return couchmessage.SubmissionStatus{}, errors.New("unknown submission operation")
	}
	var work func()
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		// GoTracked may run inline before startup or after shutdown. Publish
		// the job under the lock, but never invoke its worker while holding it.
		if work != nil {
			m.spawn(work)
		}
	}()
	if err := m.ctx.Err(); err != nil {
		return couchmessage.SubmissionStatus{}, err
	}
	if m.job != nil && m.job.status.ID == id {
		if m.job.status.Target != target {
			return couchmessage.SubmissionStatus{}, errors.New("submission belongs to another target")
		}
		if m.job.busy || m.job.status.State == "failed" || m.job.status.State == "empty" {
			return m.job.status, nil
		}
	}
	s, err := m.store.ReadMetadata(id)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return couchmessage.SubmissionStatus{}, err
	}
	if exists && s.Target != target {
		return couchmessage.SubmissionStatus{}, errors.New("submission belongs to another target")
	}
	switch op {
	case "prepare":
		if err := m.guard(target); err != nil {
			return couchmessage.SubmissionStatus{}, err
		}
		if exists {
			return m.status(s), nil
		}
		if m.job != nil && m.job.busy {
			return couchmessage.SubmissionStatus{}, errors.New("a submission is already being prepared or resolved")
		}
		cast, e := m.current()
		if e != nil {
			return couchmessage.SubmissionStatus{}, e
		}
		job := &submissionJob{status: couchmessage.SubmissionStatus{ID: id, Target: target, State: "empty"}}
		if cast == nil {
			m.job = job
			return job.status, nil
		}
		selected, e := cast.SubmissionSelection()
		if e != nil {
			return couchmessage.SubmissionStatus{}, e
		}
		selected, e = m.freezeSubmissionPreview(cast, selected)
		if e != nil {
			return couchmessage.SubmissionStatus{}, e
		}
		if len(selected.Notes) == 0 && len(selected.Sketches) == 0 {
			m.job = job
			return job.status, nil
		}
		job.status.State = "preparing"
		job.status.Sketches = len(selected.Sketches)
		job.status.Remaining = selected.SketchRemaining
		job.status.Progress = &core.SubmissionProgress{Phase: "replay"}
		job.busy = true
		m.job = job
		work = func() { m.prepare(job, cast, selected) }
		return job.status, nil
	case "status":
		if !exists {
			return couchmessage.SubmissionStatus{}, os.ErrNotExist
		}
		out := m.status(s)
		if m.job != nil && m.job.status.ID == id {
			out.Error = m.job.status.Error
		}
		work = m.resolutionWork(s)
		return out, nil
	case "begin":
		if !exists {
			return couchmessage.SubmissionStatus{}, os.ErrNotExist
		}
		if err := m.guard(target); err != nil {
			return couchmessage.SubmissionStatus{}, err
		}
		s, err = m.store.Transition(id, target, core.Begin)
		if err != nil {
			return couchmessage.SubmissionStatus{}, err
		}
		return m.status(s), nil
	case "ack":
		if !exists {
			return couchmessage.SubmissionStatus{}, os.ErrNotExist
		}
		previousDelivery := s.Delivery
		s, err = m.store.Transition(id, target, core.Ack)
		if err != nil {
			return couchmessage.SubmissionStatus{}, err
		}
		if previousDelivery == core.Delivering {
			m.submissionDeliveredLocked()
		}
		work = m.resolutionWork(s)
		return m.status(s), nil
	default:
		return couchmessage.SubmissionStatus{}, errors.New("unknown submission operation")
	}
}
func (m *submissionManager) prepare(job *submissionJob, cast submissionCast, selected board.Selection) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()
	err := ctx.Err()
	if err == nil {
		if owner, ok := cast.(interface {
			ReconcilePendingSubmissions(context.Context, func(context.Context, string) (core.Submission, core.PublicationObservation, error)) error
		}); ok {
			err = owner.ReconcilePendingSubmissions(ctx, m.observePublication)
		}
	}
	var base string
	if err == nil {
		base, err = m.replay(ctx, filepath.Base(cast.RecordingPath()))
	}
	var s core.Submission
	if err == nil {
		var crops map[string][]byte
		published := false
		s, crops, err = cast.PrepareSubmission(ctx, job.status.ID, job.status.Target, selected, base, func(p core.SubmissionProgress) { m.progress(job, p) })
		if err == nil {
			err = m.guard(s.Target)
		}
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			m.progress(job, core.SubmissionProgress{Phase: "publishing"})
			published = true
			_, err = m.store.Prepare(s, crops)
		}
		if owner, ok := cast.(interface {
			ReconcilePreparedSubmission(context.Context, core.Submission, core.PublicationObservation) error
		}); ok && s.ID != "" {
			// Only reaching Prepare can make publication uncertain. A visible
			// manifest after an error still needs durability confirmation.
			obs := core.PublicationObservation{State: core.PublicationAbsent}
			if published {
				var confirmationErr error
				obs, confirmationErr = m.store.ConfirmPrepared(s, crops)
				err = errors.Join(err, confirmationErr)
			}
			// Release proven-absent claims even when the operation expired.
			// This bounded cleanup never dispatches or reports delivery.
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
			err = errors.Join(err, owner.ReconcilePreparedSubmission(cleanup, s, obs))
			stop()
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	job.busy = false
	if err != nil {
		job.status.Progress = nil
		job.status.State = "failed"
		job.status.Error = err.Error()
		return
	}
	job.status = m.status(s)
}

func (m *submissionManager) observePublication(ctx context.Context, id string) (core.Submission, core.PublicationObservation, error) {
	if err := ctx.Err(); err != nil {
		return core.Submission{}, core.PublicationObservation{State: core.PublicationUnknown}, err
	}
	return m.store.ObservePublication(id)
}
func (m *submissionManager) resolve(job *submissionJob, s core.Submission) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()
	err := ctx.Err()
	var cast submissionCast
	if err == nil {
		cast, err = m.current()
	}
	if err == nil && cast != nil && cast.RecordingPath() == s.Source.RecordingPath {
		err = cast.ResolveSubmission(ctx, s)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		s, err = m.store.Transition(s.ID, s.Target, core.Complete)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	job.busy = false
	if err != nil {
		job.status.Error = err.Error()
		return
	}
	job.status = m.status(s)
}

// resolutionWork admits confirmed delivery into the existing single worker.
// Both ack and status can resume after a busy worker, failure or restart.
func (m *submissionManager) resolutionWork(s core.Submission) func() {
	if s.Delivery != core.Dispatched || (m.job != nil && m.job.busy) {
		return nil
	}
	job := &submissionJob{status: m.status(s), busy: true}
	m.job = job
	return func() { m.resolve(job, s) }
}

// progress publishes an immutable snapshot; callbacks from retired jobs cannot
// overwrite a successor or resurrect progress after preparation finishes.
func (m *submissionManager) progress(job *submissionJob, p core.SubmissionProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.job == job && job.busy && job.status.State == "preparing" {
		job.status.Progress = &p
	}
}
