package couchcmd

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

type scheduleOwnerCommandKind uint8

const (
	ownerSubmit scheduleOwnerCommandKind = iota
	ownerObserve
	ownerCancel
	ownerExecutionReady
	ownerGateReady
)

type scheduleOwnerReply struct {
	status couchschedule.ManualStatus
	notice string
	err    error
}
type scheduleOwnerCommand struct {
	kind          scheduleOwnerCommandKind
	ctx           context.Context
	intent        couchschedule.ManualIntent
	hub, task, id string
	offset        int
	state         couchschedule.TaskState
	identity      couchschedule.OwnerIdentity
	declaration   couchschedule.Declaration
	send          func() (string, error)
	reply         chan scheduleOwnerReply
}
type scheduleOwnerJob struct {
	state  couchschedule.TaskState
	args   []string
	ctx    context.Context
	cancel context.CancelFunc
	notice bool
}
type scheduleOwnerResult struct {
	job           scheduleOwnerJob
	facts         scheduleExecutionFacts
	notice        string
	err           error
	settlementErr error
}

// Runtime channels have four command handoffs and four result slots. The only
// goroutines executing effects are two script workers and two notice workers.
// Maps belong exclusively to the owner loop, never to callers or workers.
type scheduleOwnerLoop struct {
	commands           chan scheduleOwnerCommand
	execution, notices chan scheduleOwnerJob
	results            chan scheduleOwnerResult
	done               chan struct{}
	worker             scheduleExecutionWorker
	port               scheduleNoticePort
	active             map[string]bool
	pending            map[string]scheduleOwnerResult
	retryAt            map[string]time.Time
	receipts           map[string]string
	clock              scheduleClock
	cursor             int
}

func (o *scheduleOwner) configure(worker scheduleExecutionWorker, notices scheduleNoticePort) {
	if o.loop != nil {
		return
	}
	o.loop = &scheduleOwnerLoop{commands: make(chan scheduleOwnerCommand, 4), execution: make(chan scheduleOwnerJob, 2), notices: make(chan scheduleOwnerJob, 2), results: make(chan scheduleOwnerResult, 4), done: make(chan struct{}), worker: worker, port: notices, active: map[string]bool{}, pending: map[string]scheduleOwnerResult{}, retryAt: map[string]time.Time{}, receipts: map[string]string{}}
}
func (o *scheduleOwner) command(ctx context.Context, cmd scheduleOwnerCommand) (scheduleOwnerReply, error) {
	if ctx == nil {
		return scheduleOwnerReply{}, errors.New("scheduler command requires context")
	}
	if err := ctx.Err(); err != nil {
		return scheduleOwnerReply{}, err
	}
	if o.loop == nil {
		return scheduleOwnerReply{}, errors.New("scheduler owner unavailable")
	}
	cmd.ctx = ctx
	cmd.reply = make(chan scheduleOwnerReply, 1)
	select {
	case <-ctx.Done():
		return scheduleOwnerReply{}, ctx.Err()
	case <-o.loop.done:
		return scheduleOwnerReply{}, errors.New("scheduler owner stopped")
	case o.loop.commands <- cmd:
	default:
		return scheduleOwnerReply{}, errors.New("scheduler owner busy")
	}
	select {
	case <-ctx.Done():
		return scheduleOwnerReply{}, ctx.Err()
	case <-o.loop.done:
		return scheduleOwnerReply{}, errors.New("scheduler owner stopped")
	case reply := <-cmd.reply:
		return reply, reply.err
	}
}
func (o *scheduleOwner) Submit(ctx context.Context, m couchschedule.ManualIntent) (couchschedule.ManualStatus, error) {
	if err := m.Validate(); err != nil {
		return couchschedule.ManualStatus{}, err
	}
	m.Args = append([]string(nil), m.Args...)
	r, e := o.command(ctx, scheduleOwnerCommand{kind: ownerSubmit, intent: m})
	return r.status, e
}
func (o *scheduleOwner) Observe(ctx context.Context, hub, task, id string, offset int) (couchschedule.ManualStatus, error) {
	r, e := o.command(ctx, scheduleOwnerCommand{kind: ownerObserve, hub: hub, task: task, id: id, offset: offset})
	return r.status, e
}
func (o *scheduleOwner) Cancel(ctx context.Context, hub, task, id string) (couchschedule.ManualStatus, error) {
	r, e := o.command(ctx, scheduleOwnerCommand{kind: ownerCancel, hub: hub, task: task, id: id})
	return r.status, e
}

// gate is called after the worker owns the hub-effect gate. Once queued it
// waits for definitive resolution, even after cancellation: otherwise the gate
// could be released while the owner still executes the bounded send closure.
func (l *scheduleOwnerLoop) gate(ctx context.Context, state couchschedule.TaskState, send func() (string, error)) (string, error) {
	if err := ctx.Err(); err != nil {
		return "not-admitted", err
	}
	cmd := scheduleOwnerCommand{kind: ownerGateReady, ctx: ctx, state: state, send: send, reply: make(chan scheduleOwnerReply, 1)}
	select {
	case <-ctx.Done():
		return "not-admitted", ctx.Err()
	case <-l.done:
		return "not-admitted", errors.New("scheduler owner stopped")
	case l.commands <- cmd:
	}
	select {
	case <-l.done:
		return "not-admitted", errors.New("scheduler owner stopped")
	case reply := <-cmd.reply:
		return reply.notice, reply.err
	}
}
func (o *scheduleOwner) handleCommand(cmd scheduleOwnerCommand, clock scheduleClock) {
	reply := scheduleOwnerReply{}
	if err := cmd.ctx.Err(); err != nil {
		reply.err = err
		reply.notice = "not-admitted"
	} else {
		switch cmd.kind {
		case ownerSubmit:
			reply.status, reply.err = o.submitManual(cmd.ctx, cmd.intent)
		case ownerObserve:
			reply.status, reply.err = o.observeManual(cmd.ctx, cmd.hub, cmd.task, cmd.id, cmd.offset)
		case ownerCancel:
			reply.status, reply.err = o.cancelManual(cmd.ctx, cmd.hub, cmd.task, cmd.id)
		case ownerExecutionReady:
			reply.err = o.authorize(cmd.ctx, cmd.state, cmd.identity, cmd.declaration)
		case ownerGateReady:
			at := cmd.state.Active
			if at == nil {
				reply.notice = "dropped: task authority changed"
				break
			}
			if !clock.Now().Before(at.NoticeDeadline) {
				reply.notice = "expired"
				break
			}
			// Canonical terminal history wins even if a previous authority commit
			// failed. Never admit from the merely readable stale ledger tuple.
			_, reply.err = o.apply(cmd.ctx, cmd.state.Key, couchschedule.Event{At: clock.Now()}, nil)
			if reply.err != nil {
				reply.notice = "not-admitted"
				break
			}
			var ledger couchschedule.Ledger
			ledger, reply.err = o.state.ViewDurable(cmd.ctx)
			if reply.err != nil {
				reply.notice = "not-admitted"
				break
			}
			if !sameNoticeAuthority(ledger.Tasks[cmd.state.Key.ID()], cmd.state) {
				reply.notice = "dropped: task authority changed"
				break
			}
			if err := cmd.ctx.Err(); err != nil {
				reply.notice = "not-admitted"
				reply.err = err
				break
			}
			if !clock.Now().Before(at.NoticeDeadline) {
				reply.notice = "expired"
				break
			}
			reply.notice, reply.err = cmd.send()
		default:
			reply.err = errors.New("unknown scheduler owner command")
		}
	}
	cmd.reply <- reply
}

// dispatchOrder gives each eligible hub a turn within each independent worker
// pool. Ineligible rows must not consume a hub turn ahead of its ready rows.
func (o *scheduleOwner) dispatchOrder(keys []string, tasks map[string]couchschedule.TaskState, clock scheduleClock, online map[string]bool) []string {
	if o.loop == nil {
		return keys
	}
	l := o.loop
	var ready [2][]string
	var rest []string
	for _, key := range keys {
		s := tasks[key]
		a := s.Active
		kind := -1
		_, active := l.active[key]
		_, pending := l.pending[key]
		if s.Allowed && a != nil && !active && !pending {
			switch a.Stage {
			case "reserved":
				if o.busy[key] == "" {
					if a.Source == "manual" {
						intent, ok := o.intents[key]
						if ok && intent.ID == a.ID && intent.Digest() == a.ManualIntentDigest {
							kind = 0
						}
					} else if s.Spec.ScriptRun {
						kind = 0
					} else if online[s.Key.Hub] && !clock.Now().Before(l.retryAt[key]) {
						kind = 1
					}
				}
			case "notifying":
				if online[s.Key.Hub] && clock.Now().Before(a.NoticeDeadline) && !clock.Now().Before(l.retryAt[key]) {
					kind = 1
				}
			}
		}
		if kind < 0 {
			rest = append(rest, key)
		} else {
			ready[kind] = append(ready[kind], key)
		}
	}
	ordered := append(interleaveScheduleHubs(ready[0], tasks, l.cursor), interleaveScheduleHubs(ready[1], tasks, l.cursor)...)
	return append(ordered, rest...)
}

func (o *scheduleOwner) launchExecution(ctx context.Context, state couchschedule.TaskState, args []string) bool {
	return o.launch(ctx, state, args, false)
}
func (o *scheduleOwner) launchNotice(ctx context.Context, clock scheduleClock, state couchschedule.TaskState) bool {
	if o.loop != nil && clock.Now().Before(o.loop.retryAt[state.Key.ID()]) {
		return false
	}
	return o.launch(ctx, state, nil, true)
}
func (o *scheduleOwner) launch(ctx context.Context, state couchschedule.TaskState, args []string, notice bool) bool {
	l := o.loop
	if l == nil || state.Active == nil || ctx.Err() != nil {
		return false
	}
	key := state.Key.ID()
	if _, ok := l.active[key]; ok {
		return false
	}
	if _, ok := l.pending[key]; ok {
		return false
	}
	if notice && l.receipts[key] == state.Active.ID {
		return false
	}
	count := 0
	for _, kind := range l.active {
		if kind == notice {
			count++
		}
	}
	if count >= 2 {
		return false
	}
	jobCtx, cancel := context.WithCancel(ctx)
	job := scheduleOwnerJob{state: state, args: append([]string(nil), args...), ctx: jobCtx, cancel: cancel, notice: notice}
	jobs := l.execution
	if notice {
		jobs = l.notices
	}
	select {
	case jobs <- job:
		l.active[key] = notice
		l.cursor++
		if !notice {
			o.operations[key] = &scheduleOwnerOperation{id: state.Active.ID, cancel: cancel}
		}
		return true
	default:
		cancel()
		return false
	}
}
func (o *scheduleOwner) work(ctx context.Context, jobs <-chan scheduleOwnerJob, clock scheduleClock) {
	l := o.loop
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-jobs:
			result := scheduleOwnerResult{job: job}
			if job.notice {
				result.notice, result.err = l.admit(job.ctx, clock, job.state)
			} else {
				result.facts = l.worker.execute(job.ctx, job.state, job.args, func(readyCtx context.Context, identity couchschedule.OwnerIdentity, declaration couchschedule.Declaration) error {
					_, err := o.command(readyCtx, scheduleOwnerCommand{kind: ownerExecutionReady, state: job.state, identity: identity, declaration: declaration})
					return err
				})
			}
			job.cancel()
			// At most four admitted jobs exist, so this buffer also holds every result
			// during shutdown. Completed execution output is never discarded by cancel.
			l.results <- result
		}
	}
}
func (l *scheduleOwnerLoop) admit(ctx context.Context, clock scheduleClock, state couchschedule.TaskState) (string, error) {
	at := state.Active
	if at == nil {
		return "dropped: task authority changed", nil
	}
	if !clock.Now().Before(at.NoticeDeadline) {
		return "expired", nil
	}
	if l.port == nil {
		return "dropped: hub unavailable", nil
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
	notice, err := l.port.Admit(bounded, state.Key.Hub, at.NoticeBinding, at.ReceiptID, at.NoticeBody, func(gateCtx context.Context, send func() (string, error)) (string, error) {
		return l.gate(gateCtx, state, send)
	})
	if notice == "not-admitted" && !clock.Now().Before(at.NoticeDeadline) {
		return "expired", nil
	}
	return notice, err
}
func (o *scheduleOwner) collect(result scheduleOwnerResult) {
	l := o.loop
	key := result.job.state.Key.ID()
	delete(l.active, key)
	if op := o.operations[key]; op != nil && op.id == result.job.state.Active.ID {
		delete(o.operations, key)
	}
	l.pending[key] = result
}
func (o *scheduleOwner) retry(ctx context.Context, clock scheduleClock) error {
	var failures []error
	l := o.loop
	for key, result := range l.pending {
		if clock.Now().Before(l.retryAt[key]) {
			var local *taskHistoryError
			if result.settlementErr != nil && !errors.As(result.settlementErr, &local) {
				failures = append(failures, result.settlementErr)
			}
			continue
		}
		var err error
		if result.job.notice {
			notice := result.notice
			if notice == "" && result.err != nil {
				notice = "uncertain"
			}
			if notice != "" {
				online := false
				if l.port != nil {
					_, online = l.port.Binding(result.job.state.Key.Hub)
				}
				at := result.job.state.Active
				_, err = o.apply(ctx, result.job.state.Key, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: clock.Now(), Attempt: at.ID, Revision: at.Revision, Notice: notice, HubOnline: online}, nil)
			}
			if err == nil && notice != "not-admitted" {
				l.receipts[key] = result.job.state.Active.ID
			}
		} else {
			err = o.complete(ctx, result.job.state, result.facts)
		}
		if err != nil {
			result.settlementErr = err
			l.pending[key] = result
			l.retryAt[key] = clock.Now().Add(time.Second)
			var local *taskHistoryError
			if !errors.As(err, &local) {
				o.blocked(err)
				failures = append(failures, err)
			}
			continue
		}
		delete(l.pending, key)
		if result.job.notice && result.notice == "not-admitted" {
			l.retryAt[key] = clock.Now().Add(time.Second)
		} else {
			delete(l.retryAt, key)
		}
	}
	return errors.Join(failures...)
}
func (o *scheduleOwner) run(ctx context.Context, clock scheduleClock) {
	if o.loop == nil {
		o.configure(scheduleExecutionWorker{}, nil)
	}
	l := o.loop
	l.clock = clock
	o.coordinator.now = clock.Now
	for o.state == nil {
		acquire, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := o.open(acquire)
		cancel()
		if err == nil {
			break
		}
		o.blocked(err)
		timer := clock.NewTimer(30 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			close(l.done)
			return
		case <-timer.C():
			timer.Stop()
		}
	}
	workersCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		jobs := l.execution
		if i >= 2 {
			jobs = l.notices
		}
		workers.Add(1)
		go func(jobs <-chan scheduleOwnerJob) { defer workers.Done(); o.work(workersCtx, jobs, clock) }(jobs)
	}
	defer func() {
		// No queued gate closure may execute after done closes. Then release callback
		// waiters, stop scripts, join all workers, settle their returned facts, and
		// only then release the namespace fence.
		close(l.done)
		cancel()
		workers.Wait()
		for {
			select {
			case result := <-l.results:
				o.collect(result)
			default:
				goto drained
			}
		}
	drained:
		finalCtx, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		o.retry(finalCtx, clock)
		_ = o.close()
	}()
	_ = o.refresh(ctx)
	nextScan := clock.Now().Add(30 * time.Second)
	for ctx.Err() == nil {
		if !clock.Now().Before(nextScan) {
			_ = o.refresh(ctx)
			nextScan = clock.Now().Add(30 * time.Second)
		}
		if pendingErr := o.retry(ctx, clock); pendingErr == nil {
			if err := o.tick(ctx, clock); err != nil {
				o.blocked(err)
			}
		} else {
			o.blocked(pendingErr)
		}
		retained := map[string]string{}
		for _, row := range o.coordinator.view().Tasks {
			if row.Active != nil {
				retained[(couchschedule.TaskKey{Hub: row.Hub, Task: row.Task}).ID()] = row.Active.ID
			}
		}
		for key, id := range l.receipts {
			if retained[key] != id {
				delete(l.receipts, key)
			}
		}
		for key := range l.retryAt {
			if retained[key] == "" {
				if _, pending := l.pending[key]; !pending {
					delete(l.retryAt, key)
				}
			}
		}
		wake := clock.Now().Add(time.Second)
		for _, row := range o.coordinator.view().Tasks {
			if row.NextDue.After(clock.Now()) && row.NextDue.Before(wake) {
				wake = row.NextDue
			}
		}
		timer := clock.NewTimer(wake.Sub(clock.Now()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case cmd := <-l.commands:
			timer.Stop()
			o.handleCommand(cmd, clock)
		case result := <-l.results:
			timer.Stop()
			o.collect(result)
		case <-timer.C():
			timer.Stop()
		}
	}
}
