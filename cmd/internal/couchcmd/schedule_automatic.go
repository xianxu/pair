package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	coordinator    *scheduleCoordinator
	runner         *manualTaskRunner
	notices        scheduleNoticePort
	jobs           chan scheduleJob
	results        chan scheduleJobResult
	inFlight       map[string]bool
	jobKinds       map[string]bool
	pending        map[string]scheduleJobResult
	retryAt        map[string]time.Time
	passiveDurable map[string]couchschedule.RunSummary
	cursor         [2]int
}

func newAutomaticScheduler(c *scheduleCoordinator, r *manualTaskRunner, n scheduleNoticePort) *automaticScheduler {
	return &automaticScheduler{coordinator: c, runner: r, notices: n, jobs: make(chan scheduleJob, 4), results: make(chan scheduleJobResult, 4), inFlight: map[string]bool{}, jobKinds: map[string]bool{}, pending: map[string]scheduleJobResult{}, retryAt: map[string]time.Time{}, passiveDurable: map[string]couchschedule.RunSummary{}}
}
func scheduledRunID(s couchschedule.TaskState) string {
	h := sha256.Sum256([]byte(s.Key.ID() + "\x00" + s.Revision + "\x00" + s.NextDue.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:16])
}
func (a *automaticScheduler) history() operationalrun.Store {
	return a.coordinator.runs
}
func scheduleRunKey(s couchschedule.TaskState) operationalrun.RunKey {
	return operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: s.Last.ID}
}
func (a *automaticScheduler) publishSummary(ctx context.Context, s couchschedule.TaskState) (couchschedule.TaskState, error) {
	if s.Last == nil || s.Last.Source != "scheduled" {
		return s, nil
	}
	proposed := runRecordForSummary(s)
	record, _, err := a.history().PublishOccurrence(ctx, proposed.Key, proposed, nil)
	if err != nil {
		return s, err
	}
	return adoptScheduledRecord(s, record)
}

func runRecordForSummary(s couchschedule.TaskState) operationalrun.Record {
	last := s.Last
	revision := last.Revision
	if revision == "" && s.Active != nil && s.Active.ID == last.ID {
		revision = s.Active.Revision
	}
	return operationalrun.Record{Version: 1, Key: scheduleRunKey(s), DeclarationRevision: revision, Source: last.Source, ScheduledAt: last.DueAt, StartedAt: last.StartedAt, EndedAt: last.EndedAt, Exit: last.Exit, Outcome: last.Outcome, Notice: last.Notice}
}

// Same-occurrence stored facts are canonical. Source, revision and envelope
// structure remain authority checks; regenerated timestamps are not identity.
func adoptScheduledRecord(s couchschedule.TaskState, record operationalrun.Record) (couchschedule.TaskState, error) {
	if record.Key.Hub != s.Key.HubID() || record.Key.Task != s.Key.Task || !record.Completed() && record.Outcome != "unknown" {
		return s, errors.New("scheduled completion envelope differs from occurrence")
	}
	return adoptScheduledSummary(s, runSummaryForRecord(record))
}

// Both freshly read records and retained terminal proofs use this transition.
func adoptScheduledSummary(s couchschedule.TaskState, canonical couchschedule.RunSummary) (couchschedule.TaskState, error) {
	id, revision := "", ""
	if s.Active != nil {
		id, revision = s.Active.ID, s.Active.Revision
	} else if s.Last != nil {
		id, revision = s.Last.ID, s.Last.Revision
	}
	if canonical.ID != id || canonical.Source != "scheduled" || canonical.Revision == "" || revision != "" && canonical.Revision != revision || canonical.StartedAt.IsZero() || canonical.EndedAt.IsZero() {
		return s, errors.New("scheduled completion identity or structure differs from occurrence")
	}
	next, _ := couchschedule.Advance(s, couchschedule.NewRecoveryEvent(canonical.EndedAt, id, revision, canonical))
	if next.Last == nil || next.Last.ID != id || next.Last.Revision != canonical.Revision || next.Last.Source != canonical.Source || next.Last.Outcome != canonical.Outcome || next.Last.Notice != canonical.Notice || !next.Last.DueAt.Equal(canonical.DueAt) || !next.Last.StartedAt.Equal(canonical.StartedAt) || !next.Last.EndedAt.Equal(canonical.EndedAt) || (next.Last.Exit == nil) != (canonical.Exit == nil) || next.Last.Exit != nil && *next.Last.Exit != *canonical.Exit {
		return s, errors.New("scheduled canonical completion refused by reducer")
	}
	return next, nil
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
	var historyErr error
	err := c.store.Update(ctx, func(l *couchschedule.Ledger) error {
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
				historyErr = err
				return quarantineTaskHistory(ctx, l, s, "admission finalization", err)
			}
		}
		l.Tasks[s.Key.ID()] = next
		return nil
	})
	if err != nil {
		return err
	}
	if historyErr != nil {
		return &taskHistoryError{historyErr}
	}
	return nil
}

func (a *automaticScheduler) online(root string) bool { _, ok := a.notices.Binding(root); return ok }

// recover runs once before any effect worker starts. A retained scheduled
// attempt is never executed again, even if the previous process died before
// recording whether authorization crossed the native boundary.
func (a *automaticScheduler) recover(ctx context.Context) error {
	c := a.coordinator
	err := c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		for _, prior := range l.Tasks {
			s, err := a.ensurePassive(ctx, prior)
			if err == nil && s.Active != nil && s.Active.Source == "scheduled" {
				s, err = a.recoverExecutionHistory(ctx, s, true)
			}
			if err != nil {
				if err := quarantineTaskHistory(ctx, l, prior, "startup recovery", err); err != nil {
					return err
				}
				continue
			}
			l.Tasks[s.Key.ID()] = s
		}
		return nil
	})
	if err != nil {
		return err
	}
	return ensureUncertainRuns(ctx, c.store, c.runs)
}

// recoverExecutionHistory consumes saved evidence without repeating work.
func (a *automaticScheduler) recoverExecutionHistory(ctx context.Context, s couchschedule.TaskState, startup bool) (couchschedule.TaskState, error) {
	at := s.Active
	key := operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: at.ID}
	record, _, err := a.history().ReadDurable(ctx, key)
	if err != nil {
		if !missingRunBundle(filepath.Dir(a.coordinator.store.Dir), key, err) {
			return s, err
		}
		if !retainedUnknownSummary(s) {
			s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: a.coordinator.now(), Attempt: at.ID, Revision: at.Revision})
		}
		return s, nil
	}
	if _, err = adoptScheduledRecord(s, record); err != nil {
		return s, err
	}
	if record.Notice == "pending" && (startup || at.Stage == "stopping" || s.Revision != at.Revision || !s.Allowed) {
		notice := "uncertain"
		if at.Stage == "stopping" || s.Revision != at.Revision || !s.Allowed {
			notice = "dropped: task cancelled"
		}
		if err = a.history().FinalizeNotice(ctx, record.Key, at.Revision, notice); err != nil {
			return s, err
		}
		record.Notice = notice
	}
	return adoptScheduledRecord(s, record)
}

func (a *automaticScheduler) reconcileExecution(ctx context.Context, result scheduleJobResult) error {
	c := a.coordinator
	var historyErr error
	err := c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		s := l.Tasks[result.job.state.Key.ID()]
		at := s.Active
		if at == nil || at.ID != result.job.state.Active.ID || at.Stage == "notifying" || at.Stage == "delivery" {
			return nil
		}
		next, err := a.recoverExecutionHistory(ctx, s, false)
		if err != nil {
			historyErr = err
			return quarantineTaskHistory(ctx, l, s, "execution reconciliation", err)
		}
		l.Tasks[s.Key.ID()] = next
		return nil
	})
	if err != nil {
		return err
	}
	if historyErr != nil {
		return &taskHistoryError{historyErr}
	}
	return ensureUncertainRuns(ctx, c.store, c.runs)
}

func retainedUnknownSummary(s couchschedule.TaskState) bool {
	return s.Active != nil && s.Active.Stage == "uncertain" && s.Last != nil && s.Last.ID == s.Active.ID && s.Last.Outcome == "unknown"
}

// A positive terminal proof remains valid after normal history retention. Cache
// only canonical observed facts, so a failed ledger commit can adopt them again
// without regenerating timestamps or repeatedly recreating pruned passive runs.
func (a *automaticScheduler) ensurePassive(ctx context.Context, s couchschedule.TaskState) (couchschedule.TaskState, error) {
	last := s.Last
	passive := last != nil && last.Source == "scheduled" && (last.Outcome == "skipped" || last.Outcome == "refused" || last.Outcome == "completed")
	if !passive {
		return s, ensurePassiveRun(ctx, a.history(), s)
	}
	if proof, ok := a.passiveDurable[s.Key.ID()]; ok && proof.ID == last.ID {
		if s.Active == nil {
			return adoptScheduledSummary(s, proof)
		}
		return s, nil
	}
	record, err := ensureRetainedRun(ctx, a.history(), s, false)
	if err != nil {
		return s, err
	}
	if record.Source != "scheduled" || record.DeclarationRevision == "" || last.Revision != "" && record.DeclarationRevision != last.Revision {
		return s, errors.New("passive completion identity differs from retained occurrence")
	}
	if s.Active == nil {
		s, err = adoptScheduledRecord(s, record)
		if err != nil {
			return s, err
		}
	}
	if record.Completed() && record.Notice != "pending" {
		if a.passiveDurable == nil {
			a.passiveDurable = map[string]couchschedule.RunSummary{}
		}
		a.passiveDurable[s.Key.ID()] = runSummaryForRecord(record)
	}
	return s, nil
}
func (a *automaticScheduler) pruneStateCaches(ledger couchschedule.Ledger) {
	for key := range a.passiveDurable {
		if _, ok := ledger.Tasks[key]; !ok {
			delete(a.passiveDurable, key)
		}
	}
	for key := range a.retryAt {
		if _, ok := ledger.Tasks[key]; !ok {
			delete(a.retryAt, key)
		}
	}
}

// The first transaction persists occurrence identity and passive facts. The
// following pass may publish history or authorize effects only after it commits.
func (a *automaticScheduler) reserveDue(ctx context.Context, online func(string) bool) error {
	c := a.coordinator
	view, err := c.store.View(ctx)
	if err != nil {
		return err
	}
	a.pruneStateCaches(view)
	now := c.now()
	due := false
	for _, s := range view.Tasks {
		if containsScheduleRoot(c.roots, s.Key.Hub) && !s.NextDue.IsZero() && !now.Before(s.NextDue) {
			due = true
			break
		}
	}
	if !due {
		return nil
	}
	return c.store.Update(ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if !containsScheduleRoot(c.roots, s.Key.Hub) || s.NextDue.IsZero() || now.Before(s.NextDue) {
				continue
			}
			var err error
			s, err = a.ensurePassive(ctx, s)
			if err != nil {
				if err := quarantineTaskHistory(ctx, l, s, "passive repair", err); err != nil {
					return err
				}
				continue
			}
			s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ClockObserved, At: now, RunID: scheduledRunID(s), HubOnline: online(s.Key.Hub), Location: c.location})
			l.Tasks[key] = s
		}
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
		if c.now().Before(a.retryAt[key]) {
			continue
		}
		var err error
		if result.job.notice {
			err = a.applyAdmission(ctx, result)
		} else {
			err = a.reconcileExecution(ctx, result)
		}
		if err != nil {
			var historyErr *taskHistoryError
			if errors.As(err, &historyErr) {
				a.retryAt[key] = c.now().Add(time.Second)
				continue
			}
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

	if err := a.reserveDue(ctx, hubOnline); err != nil {
		return c.blocked(err)
	}

	var candidates []scheduleJob
	executionSlots, noticeSlots := 0, 0
	for key := range a.inFlight {
		if _, completed := a.pending[key]; completed {
			continue
		}
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

	taskLoop:
		for _, key := range keys {
			s := l.Tasks[key]
			prior := s
			var err error
			s, err = a.ensurePassive(ctx, s)
			if err != nil {
				if err := quarantineTaskHistory(ctx, l, prior, "passive repair", err); err != nil {
					return err
				}
				continue
			}
			if !s.Allowed {
				l.Tasks[key] = s
				continue
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
						record, _, err := a.history().ReadDurable(ctx, operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: at.ID})
						if err == nil {
							s, err = adoptScheduledRecord(s, record)
						} else if missingRunBundle(filepath.Dir(c.store.Dir), operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: at.ID}, err) {
							next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: c.now(), Attempt: at.ID, Revision: at.Revision, HubOnline: hubOnline(s.Key.Hub)})
							s, err = a.publishSummary(ctx, next)
						}
						if err != nil {
							if err := quarantineTaskHistory(ctx, l, prior, "occurrence reconciliation", err); err != nil {
								return err
							}
							continue taskLoop
						}
					}
				case "running", "stopping", "uncertain":
					// No worker owns this retained attempt: consume repaired evidence only.
					// Missing history classifies uncertainty; it never restarts execution.
					next, err := a.recoverExecutionHistory(ctx, s, false)
					if err != nil {
						if err := quarantineTaskHistory(ctx, l, prior, "retained execution recovery", err); err != nil {
							return err
						}
						continue taskLoop
					}
					s = next
				case "delivery":
					notice, known, active := a.notices.Observe(at.ReceiptID)
					if !active && (!known || notice != "") {
						if !known {
							notice = "uncertain"
						}
						next, _ := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: c.now(), Attempt: at.ID, Revision: at.Revision, Notice: notice})
						if next.Active == nil {
							if err := a.finishNotice(ctx, s, next); err != nil {
								if err := quarantineTaskHistory(ctx, l, prior, "notice finalization", err); err != nil {
									return err
								}
								continue taskLoop
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
					_, output, err := a.history().ReadDurable(ctx, scheduleRunKey(s))
					if err != nil {
						if err := quarantineTaskHistory(ctx, l, prior, "occurrence reconciliation", err); err != nil {
							return err
						}
						continue taskLoop
					}
					s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeFrozen, At: c.now(), Attempt: at.ID, Revision: at.Revision, ReceiptID: "schedule-" + at.ID, NoticeBody: scheduleNoticeBody(s, output), NoticeBinding: binding})
				}
				at = s.Active
				next, effects := couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeIntentDurable, At: c.now(), Attempt: at.ID, Revision: at.Revision, HubOnline: online})
				if next.Active == nil {
					if err := a.finishNotice(ctx, s, next); err != nil {
						if err := quarantineTaskHistory(ctx, l, prior, "notice finalization", err); err != nil {
							return err
						}
						continue taskLoop
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
	if err := ensureUncertainRuns(ctx, c.store, c.runs); err != nil {
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

// admitNotice uses one persisted budget for queue/gate/authority/broker waits.
// The adapter takes the hub-effect gate first, then this callback holds the
// scheduler authority lock until SendSystem has returned. No path may take
// those locks in reverse. Receipt reconciliation happens after both release.
func (a *automaticScheduler) admitNotice(ctx context.Context, clock scheduleClock, want couchschedule.TaskState) (string, error) {
	at := want.Active
	if at == nil {
		return "dropped: task authority changed", nil
	}
	if !clock.Now().Before(at.NoticeDeadline) {
		return "expired", nil
	}
	bounded, cancel := context.WithCancel(ctx)
	timer := clock.NewTimer(at.NoticeDeadline.Sub(clock.Now()))
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		select {
		case <-timer.C():
			cancel()
		case <-bounded.Done():
		}
	}()
	defer func() { cancel(); timer.Stop(); <-joined }()
	authority := func(ctx context.Context, send func() (string, error)) (string, error) {
		notice := "not-admitted"
		var admissionErr error
		err := a.coordinator.store.WithAuthority(ctx, func(ledger couchschedule.Ledger) error {
			if !clock.Now().Before(at.NoticeDeadline) {
				notice = "expired"
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			state := ledger.Tasks[want.Key.ID()]
			if !sameNoticeAuthority(state, want) {
				notice = "dropped: task authority changed"
				return nil
			}
			notice, admissionErr = send()
			return nil
		})
		if err != nil {
			return notice, err
		}
		return notice, admissionErr
	}
	notice, err := a.notices.Admit(bounded, want.Key.Hub, at.NoticeBinding, at.ReceiptID, at.NoticeBody, authority)
	// Known non-admission at the elapsed boundary expires. An accepted or
	// uncertain broker result retains its receipt authority regardless of time.
	if notice == "not-admitted" && !clock.Now().Before(at.NoticeDeadline) {
		return "expired", nil
	}
	return notice, err
}

func sameNoticeAuthority(state, want couchschedule.TaskState) bool {
	at := want.Active
	return at != nil && state.Allowed && state.Revision == at.Revision && state.Active != nil && state.Active.ID == at.ID && state.Active.Stage == "delivery" && state.Active.ReceiptID == at.ReceiptID && state.Active.NoticeBody == at.NoticeBody && string(state.Active.NoticeBinding) == string(at.NoticeBinding) && state.Active.NoticeDeadline.Equal(at.NoticeDeadline)
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
						result.notice, result.err = a.admitNotice(ctx, clock, job.state)
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
