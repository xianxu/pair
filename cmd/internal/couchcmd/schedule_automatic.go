package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

type scheduleJob struct {
	state  couchschedule.TaskState
	notice bool
}
type scheduleJobResult struct {
	job    scheduleJob
	notice string
	err    error
}

// automaticScheduler admits two execution and two notice jobs independently.
// Four joined workers prevent long scripts from occupying notice admission.
// Only its event loop mutates
// inFlight and the fairness cursor; workers communicate completions over results.
type automaticScheduler struct {
	coordinator *scheduleCoordinator
	runner      *manualTaskRunner
	notices     scheduleNoticePort
	jobs        chan scheduleJob
	results     chan scheduleJobResult
	inFlight    map[string]bool
	jobKinds    map[string]bool
	pending     map[string]scheduleJobResult
	retryAt     map[string]time.Time
	cursor      [2]int
}

func newAutomaticScheduler(c *scheduleCoordinator, r *manualTaskRunner, n scheduleNoticePort) *automaticScheduler {
	return &automaticScheduler{coordinator: c, runner: r, notices: n, jobs: make(chan scheduleJob, 4), results: make(chan scheduleJobResult, 4), inFlight: map[string]bool{}, jobKinds: map[string]bool{}, pending: map[string]scheduleJobResult{}, retryAt: map[string]time.Time{}}
}
func scheduledRunID(s couchschedule.TaskState) string {
	h := sha256.Sum256([]byte(s.Key.ID() + "\x00" + s.Revision + "\x00" + s.NextDue.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:16])
}
func (a *automaticScheduler) history() operationalrun.Store {
	return operationalrun.Store{Namespace: filepath.Dir(a.coordinator.store.Dir)}
}
func scheduleRunKey(s couchschedule.TaskState) operationalrun.RunKey {
	return operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: s.Last.ID}
}
func (a *automaticScheduler) publishSummary(ctx context.Context, s couchschedule.TaskState) error {
	if s.Last == nil || s.Last.Source != "scheduled" {
		return nil
	}
	record := runRecordForSummary(s)
	return a.history().Publish(ctx, record.Key, record, nil)
}

func runRecordForSummary(s couchschedule.TaskState) operationalrun.Record {
	last := s.Last
	revision := s.Revision
	if s.Active != nil && s.Active.ID == last.ID {
		revision = s.Active.Revision
	}
	return operationalrun.Record{Version: 1, Key: scheduleRunKey(s), DeclarationRevision: revision, Source: last.Source, ScheduledAt: last.DueAt, StartedAt: last.StartedAt, EndedAt: last.EndedAt, Exit: last.Exit, Outcome: last.Outcome, Notice: last.Notice}

}

// finishNotice publishes terminal history first. A failed ledger commit is
// recovered from that terminal record; it cannot cause another mailbox send.
func (a *automaticScheduler) finishNotice(ctx context.Context, s couchschedule.TaskState, next couchschedule.TaskState) error {
	if s.Last == nil || next.Last == nil || next.Last.Notice == "pending" {
		return errors.New("invalid notice completion")
	}
	revision := s.Revision
	if s.Active != nil {
		revision = s.Active.Revision
	}
	return a.history().FinalizeNotice(ctx, scheduleRunKey(s), revision, next.Last.Notice)
}
func (a *automaticScheduler) applyAdmission(ctx context.Context, result scheduleJobResult) error {
	c := a.coordinator
	return c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		s := l.Tasks[result.job.state.Key.ID()]
		if s.Active == nil || s.Active.ID != result.job.state.Active.ID || s.Active.Stage != "delivery" {
			return nil
		}
		notice := result.notice
		if result.err != nil && notice == "" {
			notice = "uncertain"
		}
		if notice == "" {
			return nil
		}
		next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: c.now(), Attempt: s.Active.ID, Revision: s.Active.Revision, Notice: notice, HubOnline: a.online(s.Key.Hub)})
		if next.Active == nil {
			if err := a.finishNotice(ctx, s, next); err != nil {
				return err
			}
		}
		l.Tasks[s.Key.ID()] = next
		return nil
	})
}
func (a *automaticScheduler) online(root string) bool { _, ok := a.notices.Binding(root); return ok }

// recover runs once before any effect worker starts. A retained scheduled
// attempt is never executed again, even if the previous process died before
// recording whether authorization crossed the native boundary.
func (a *automaticScheduler) recover(ctx context.Context) error {
	c := a.coordinator
	return c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if s.Active == nil || s.Active.Source != "scheduled" {
				continue
			}
			at := s.Active
			record, _, err := a.history().Read(ctx, operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: at.ID})
			if err == nil && record.Completed() && record.Source == "scheduled" && record.DeclarationRevision == at.Revision && record.StartedAt.Equal(at.StartedAt) && record.ScheduledAt.Equal(at.DueAt) {
				if record.Notice == "pending" {
					if err = a.history().FinalizeNotice(ctx, record.Key, at.Revision, "uncertain"); err != nil {
						return err
					}
					record.Notice = "uncertain"
				}
				exit := -1
				if record.Exit != nil {
					exit = *record.Exit
				}
				s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ResultRecovered, At: record.EndedAt, Attempt: at.ID, Revision: at.Revision, Exit: exit, Outcome: record.Outcome, Notice: record.Notice})
			} else {
				next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: c.now(), Attempt: at.ID, Revision: at.Revision})
				if s.Last != nil && s.Last.Notice == "pending" && next.Active == nil {
					if err := a.finishNotice(ctx, s, next); err != nil {
						return err
					}
				}
				s = next
				if errors.Is(err, os.ErrNotExist) && s.Last != nil && s.Last.Outcome == "unknown" {
					if err := a.publishSummary(ctx, s); err != nil {
						return err
					}
				}
			}
			l.Tasks[key] = s
		}
		return nil
	})
}

// reconcileExecution consumes saved completion evidence without repeating work.
func (a *automaticScheduler) reconcileExecution(ctx context.Context, result scheduleJobResult) error {
	c := a.coordinator
	return c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		key := result.job.state.Key.ID()
		s := l.Tasks[key]
		at := s.Active
		if at == nil || at.ID != result.job.state.Active.ID || at.Stage == "notifying" || at.Stage == "delivery" {
			return nil
		}
		record, _, err := a.history().Read(ctx, operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: at.ID})
		if err == nil && record.Completed() {
			if record.Source != "scheduled" || record.DeclarationRevision != at.Revision || !record.StartedAt.Equal(at.StartedAt) || !record.ScheduledAt.Equal(at.DueAt) {
				return errors.New("scheduled completion identity differs from reservation")
			}
			if record.Notice == "pending" && (at.Stage == "stopping" || s.Revision != at.Revision || !s.Allowed) {
				if err := a.history().FinalizeNotice(ctx, record.Key, at.Revision, "dropped: task cancelled"); err != nil {
					return err
				}
				record.Notice = "dropped: task cancelled"
			}
			exit := -1
			if record.Exit != nil {
				exit = *record.Exit
			}
			s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ResultRecovered, At: record.EndedAt, Attempt: at.ID, Revision: at.Revision, Exit: exit, Outcome: record.Outcome, Notice: record.Notice})
		} else {
			if errors.Is(err, operationalrun.ErrBusy) {
				return err
			}
			// A damaged or missing completion is not execution permission.
			s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: c.now(), Attempt: at.ID, Revision: at.Revision})
			if errors.Is(err, os.ErrNotExist) {
				if err := a.publishSummary(ctx, s); err != nil {
					return err
				}
			}
		}
		l.Tasks[key] = s
		return nil
	})
}

func (a *automaticScheduler) step(ctx context.Context) error {
	c := a.coordinator
	type hubBinding struct {
		raw    json.RawMessage
		online bool
	}
	bindings := map[string]hubBinding{}
	bindingFor := func(root string) (json.RawMessage, bool) {
		if old, ok := bindings[root]; ok {
			return old.raw, old.online
		}
		raw, online := a.notices.Binding(root)
		bindings[root] = hubBinding{raw, online}
		return raw, online
	}
	hubOnline := func(root string) bool { _, online := bindingFor(root); return online }
	for {
		select {
		case result := <-a.results:
			a.pending[result.job.state.Key.ID()] = result
		default:
			goto drained
		}
	}
drained:
	for key, result := range a.pending {
		var err error
		if result.job.notice {
			err = a.applyAdmission(ctx, result)
		} else {
			err = a.reconcileExecution(ctx, result)
		}
		if err != nil {
			return c.blocked(err)
		}
		if result.job.notice && result.notice == "not-admitted" {
			a.retryAt[key] = c.now().Add(time.Second)
		} else {
			delete(a.retryAt, key)
		}
		delete(a.pending, key)
		delete(a.inFlight, key)
		delete(a.jobKinds, key)
	}

	var candidates []scheduleJob
	executionSlots, noticeSlots := 0, 0
	for key := range a.inFlight {
		if a.jobKinds[key] {
			noticeSlots++
		} else {
			executionSlots++
		}
	}
	err := c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		keys := make([]string, 0, len(l.Tasks))
		for key, s := range l.Tasks {
			if containsScheduleRoot(c.roots, s.Key.Hub) {
				keys = append(keys, key)
			}
		}
		sort.Slice(keys, func(i, j int) bool {
			x, y := l.Tasks[keys[i]], l.Tasks[keys[j]]
			if x.Key.Hub != y.Key.Hub {
				return x.Key.Hub < y.Key.Hub
			}
			return x.Key.Task < y.Key.Task
		})
		var ready [3][]string
		now := c.now()
		for _, key := range keys {
			s := l.Tasks[key]
			kind := 2
			if !a.inFlight[key] && s.Allowed {
				at := s.Active
				due := at == nil && !s.NextDue.IsZero() && !now.Before(s.NextDue) && (now.Sub(s.NextDue) < time.Minute || s.Spec.Missed == "once")
				reserved := at != nil && at.Source == "scheduled" && at.Stage == "reserved"
				notifying := at != nil && at.Source == "scheduled" && at.Stage == "notifying" && now.Before(at.NoticeDeadline) && !now.Before(a.retryAt[key])
				if (due || reserved) && s.Spec.ScriptRun {
					kind = 0
				} else if (due || reserved || notifying) && hubOnline(s.Key.Hub) {
					kind = 1
				}
			}
			ready[kind] = append(ready[kind], key)
		}
		keys = append(interleaveScheduleHubs(ready[0], l.Tasks, a.cursor[0]), interleaveScheduleHubs(ready[1], l.Tasks, a.cursor[1])...)
		keys = append(keys, ready[2]...)

		for _, key := range keys {
			s := l.Tasks[key]
			if !s.NextDue.IsZero() && !c.now().Before(s.NextDue) {
				next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ClockObserved, At: c.now(), RunID: scheduledRunID(s), HubOnline: hubOnline(s.Key.Hub), Location: c.location})
				if next.Active == nil && next.Last != nil && (s.Last == nil || next.Last.ID != s.Last.ID) {
					if err := a.publishSummary(ctx, next); err != nil {
						return err
					}
				}
				s = next
			}
			at := s.Active
			if at != nil && at.Source == "scheduled" && !a.inFlight[key] {
				switch at.Stage {
				case "reserved":
					if s.Spec.ScriptRun {
						if executionSlots < 2 {
							candidates = append(candidates, scheduleJob{state: s})
							executionSlots++
						}
					} else {
						next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: c.now(), Attempt: at.ID, Revision: at.Revision, HubOnline: hubOnline(s.Key.Hub)})
						if err := a.publishSummary(ctx, next); err != nil {
							return err
						}
						s = next
					}
				case "delivery":
					notice, known, active := a.notices.Observe(at.ReceiptID)
					if !active && (!known || notice != "") {
						if !known {
							notice = "uncertain"
						}
						next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: c.now(), Attempt: at.ID, Revision: at.Revision, Notice: notice})
						if next.Active == nil {
							if err := a.finishNotice(ctx, s, next); err != nil {
								return err
							}
						}
						s = next
					}
				}
			}
			at = s.Active
			if at != nil && at.Source == "scheduled" && at.Stage == "notifying" && !a.inFlight[key] {
				if c.now().Before(a.retryAt[key]) && c.now().Before(at.NoticeDeadline) {
					l.Tasks[key] = s
					continue
				}
				binding, online := bindingFor(s.Key.Hub)
				if noticeSlots >= 2 && online && c.now().Before(at.NoticeDeadline) {
					l.Tasks[key] = s
					continue
				}
				// The exact body and binding are frozen once, including known-no-admission retries.
				if online && at.ReceiptID == "" {
					_, output, err := a.history().Read(ctx, scheduleRunKey(s))
					if err != nil {
						return err
					}
					s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeFrozen, At: c.now(), Attempt: at.ID, Revision: at.Revision, ReceiptID: "schedule-" + at.ID, NoticeBody: scheduleNoticeBody(s, output), NoticeBinding: binding})
				}
				at = s.Active
				next, effects := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeIntentDurable, At: c.now(), Attempt: at.ID, Revision: at.Revision, HubOnline: online})
				if next.Active == nil {
					if err := a.finishNotice(ctx, s, next); err != nil {
						return err
					}
				} else if len(effects) == 1 && effects[0].Kind == couchschedule.SendNotice {
					candidates = append(candidates, scheduleJob{state: next, notice: true})
					noticeSlots++
				}
				s = next
			}
			l.Tasks[key] = s
		}
		return nil
	})
	if err != nil {
		return c.blocked(err)
	}
	for _, job := range candidates {
		select {
		case a.jobs <- job:
			a.inFlight[job.state.Key.ID()] = true
			a.jobKinds[job.state.Key.ID()] = job.notice
			kind := 0
			if job.notice {
				kind = 1
			}
			a.cursor[kind]++
		default:
		}
	}
	return c.publish(ctx)
}

// Interleave only ready work; a future or held task cannot consume a hub turn.
func interleaveScheduleHubs(keys []string, tasks map[string]couchschedule.TaskState, cursor int) []string {
	var queues [][]string
	for _, key := range keys {
		if len(queues) == 0 || tasks[queues[len(queues)-1][0]].Key.Hub != tasks[key].Key.Hub {
			queues = append(queues, []string{key})
		} else {
			queues[len(queues)-1] = append(queues[len(queues)-1], key)
		}
	}
	if len(queues) == 0 {
		return nil
	}
	n := cursor % len(queues)
	queues = append(queues[n:], queues[:n]...)
	out := make([]string, 0, len(keys))
	for len(queues) > 0 {
		var remaining [][]string
		for _, q := range queues {
			out = append(out, q[0])
			if len(q) > 1 {
				remaining = append(remaining, q[1:])
			}
		}
		queues = remaining
	}
	return out
}

func containsScheduleRoot(roots []string, root string) bool {
	i := sort.SearchStrings(roots, root)
	return i < len(roots) && roots[i] == root
}
func scheduleNoticeBody(s couchschedule.TaskState, output []byte) string {
	body := fmt.Sprintf("Scheduled task %s is due. Run couch --schedule run %s -- --help for its task interface.", s.Key.Task, s.Key.Task)
	if s.Spec.ScriptRun && s.Last != nil {
		body = fmt.Sprintf("Scheduled task %s finished: %s. Inspect couch --schedule history %s --run %s for output.", s.Key.Task, s.Last.Outcome, s.Key.Task, s.Last.ID)
	}
	if s.Last != nil {
		body += "\nDue: " + s.Last.DueAt.UTC().Format(time.RFC3339)
	}
	if s.Spec.ScriptRun && len(output) > 0 {
		text := rowtext.Sanitize(strings.ToValidUTF8(string(output), "�"))
		if len(text) > 6000 {
			text = text[len(text)-6000:]
			for !utf8.ValidString(text) {
				text = text[1:]
			}
		}
		body += "\nOutput:\n" + text
	}
	body = rowtext.Sanitize(body)
	if len(body) > 8192 {
		body = body[:8192]
		for !utf8.ValidString(body) {
			body = body[:len(body)-1]
		}
	}
	return body
}

func (a *automaticScheduler) authorizeNotice(ctx context.Context, want couchschedule.TaskState) error {
	ledger, err := a.coordinator.store.View(ctx)
	if err != nil {
		return err
	}
	state := ledger.Tasks[want.Key.ID()]
	if !state.Allowed || state.Revision != want.Active.Revision || state.Active == nil || state.Active.ID != want.Active.ID || state.Active.Stage != "delivery" || state.Active.ReceiptID != want.Active.ReceiptID || state.Active.NoticeBody != want.Active.NoticeBody || string(state.Active.NoticeBinding) != string(want.Active.NoticeBinding) {
		return errors.New("notice authority changed before admission")
	}
	return nil
}

// scheduleTimer is a single cancelable clock seam; tests drive exact instants.
type scheduleTimer interface {
	C() <-chan time.Time
	Stop()
}
type scheduleClock interface {
	Now() time.Time
	NewTimer(time.Duration) scheduleTimer
}
type nativeScheduleClock struct{}

func (nativeScheduleClock) Now() time.Time { return time.Now() }
func (nativeScheduleClock) NewTimer(d time.Duration) scheduleTimer {
	return nativeScheduleTimer{time.NewTimer(d)}
}

type nativeScheduleTimer struct{ *time.Timer }

func (t nativeScheduleTimer) C() <-chan time.Time { return t.Timer.C }
func (t nativeScheduleTimer) Stop()               { t.Timer.Stop() }

func (a *automaticScheduler) run(ctx context.Context, clock scheduleClock) {
	c := a.coordinator
	c.now = clock.Now
	for {
		err := c.refresh(ctx)
		if err == nil {
			err = a.recover(ctx)
		}
		if err == nil {
			break
		}
		c.blocked(err)
		timer := clock.NewTimer(30 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
			timer.Stop()
		}
	}
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-a.jobs:
					result := scheduleJobResult{job: job}
					if job.notice {
						at := job.state.Active
						if err := a.authorizeNotice(ctx, job.state); err != nil {
							result.notice = "dropped: task authority changed"
							result.err = err
						} else {
							result.notice, result.err = a.notices.Admit(ctx, job.state.Key.Hub, at.NoticeBinding, at.ReceiptID, at.NoticeBody)
						}
					} else if a.runner != nil {
						_, result.err = a.runner.runReserved(ctx, job.state, []string{"script-run"})
					}
					select {
					case a.results <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	defer workers.Wait()
	nextScan := clock.Now().Add(30 * time.Second)
	catalogueReady := true
	for {
		if ctx.Err() != nil {
			return
		}
		if !clock.Now().Before(nextScan) {
			catalogueReady = c.refresh(ctx) == nil
			nextScan = clock.Now().Add(30 * time.Second)
		}
		if catalogueReady {
			_ = a.step(ctx)
		}
		wake := nextScan
		if len(a.pending) > 0 {
			wake = clock.Now().Add(time.Second)
		}
		for _, s := range c.view().Tasks {
			if !s.NextDue.IsZero() && s.NextDue.After(clock.Now()) && s.NextDue.Before(wake) {
				wake = s.NextDue
			}
		}
		// Canonical receipt observation is memory-only and bounded by current tasks.
		for _, s := range c.view().Tasks {
			if s.Active != nil && s.Active.Stage == "notifying" {
				deadline := s.Active.NoticeDeadline
				if deadline.After(clock.Now()) && deadline.Before(wake) {
					wake = deadline
				}
				if retry := a.retryAt[(couchschedule.TaskKey{Hub: s.Hub, Task: s.Task}).ID()]; retry.After(clock.Now()) && retry.Before(wake) {
					wake = retry
				}
			}
			if s.Active != nil && s.Active.Stage == "delivery" && clock.Now().Add(time.Second).Before(wake) {
				wake = clock.Now().Add(time.Second)
			}
		}
		timer := clock.NewTimer(wake.Sub(clock.Now()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case result := <-a.results:
			timer.Stop()
			a.pending[result.job.state.Key.ID()] = result
		case <-timer.C():
			timer.Stop()
		}
	}
}
