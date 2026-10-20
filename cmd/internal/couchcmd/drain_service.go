package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type drainLiveMember struct {
	Binding     couchmessage.Binding
	Session     string
	Observation couchcore.DrainObservation
}

func (m drainLiveMember) Identity() string {
	b, _ := json.Marshal(struct {
		Binding couchmessage.Binding
		Session string
	}{m.Binding, m.Session})
	return string(b)
}

type drainCoordinatorOptions struct {
	Census   func() []couchcore.DrainRow
	Broker   *couchmessage.Broker
	Queue    drainQueue
	Store    *drainStore
	Now      func() time.Time
	Snapshot func() map[string]drainLiveMember
	Endpoint func(couchmessage.Binding) couchmessage.DrainEndpoint
	Publish  func(couchcore.DrainBoard)
}
type drainCoordinator struct {
	opts        drainCoordinatorOptions
	namespace   string
	ctx         context.Context
	stop        context.CancelFunc
	done        chan struct{}
	wake        chan struct{}
	mu          sync.Mutex
	pass        sync.Mutex
	record      drainServiceRecord
	raw         json.RawMessage
	board       atomic.Value
	worker      atomic.Bool
	workers     sync.WaitGroup
	lastQueue   time.Time
	ackRevision map[string]uint64
	census      []couchcore.DrainRow
}

func newDrainCoordinator(parent context.Context, opts drainCoordinatorOptions) (*drainCoordinator, error) {
	if opts.Broker == nil || opts.Queue == nil || opts.Store == nil || opts.Snapshot == nil || opts.Endpoint == nil {
		return nil, errors.New("incomplete drain coordinator dependencies")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	namespace, err := couchmessage.DrainNamespaceKey(opts.Store.namespace)
	if err != nil {
		return nil, err
	}
	ctx, stop := context.WithCancel(parent)
	c := &drainCoordinator{namespace: namespace, opts: opts, ctx: ctx, stop: stop, done: make(chan struct{}), wake: make(chan struct{}, 1), ackRevision: map[string]uint64{}}
	c.board.Store(couchcore.DrainBoard{})
	raw, err := opts.Store.Read(ctx)
	if err == nil && len(raw) > 0 {
		c.record, err = decodeDrainServiceRecord(raw)
		c.raw = raw
		if err == nil && c.record.Namespace != namespace {
			err = errors.New("persisted drain belongs to another namespace")
		}
	}
	if err != nil {
		// Unreadable intent cannot grant a new start or admission. Keep a known
		// synthetic recovery generation solely as a fail-closed gate diagnostic.
		id := "00000000-0000-4000-8000-000000000000"
		c.record = drainServiceRecord{State: couchcore.DrainState{ID: id, Owner: "couch-drain:" + id, Phase: couchcore.DrainRecovered, Error: "drain storage unknown: " + err.Error()}}
		_ = opts.Broker.HoldDrain(id)
	} else if c.record.State.ID != "" && (!c.record.Released || c.record.State.Phase != couchcore.DrainFinished) {
		_ = opts.Broker.HoldDrain(c.record.State.ID)
		c.record.State, _ = couchcore.ReduceDrain(c.record.State, couchcore.DrainEvent{ID: c.record.State.ID, Kind: couchcore.DrainRecover})
	}
	c.publishLocked()
	go c.run()
	return c, nil
}
func (c *drainCoordinator) run() {
	defer close(c.done)
	defer c.workers.Wait()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			c.publishLocked()
			c.mu.Unlock()
		case <-c.wake:
		}
		if c.worker.CompareAndSwap(false, true) {
			c.workers.Add(1)
			go func() { defer c.workers.Done(); defer c.worker.Store(false); c.reconcile(c.ctx) }()
		}
	}
}
func (c *drainCoordinator) close() { c.stop(); <-c.done }
func (c *drainCoordinator) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}
func (c *drainCoordinator) view() couchcore.DrainBoard {
	return copyDrainServiceBoard(c.board.Load().(couchcore.DrainBoard))
}
func copyDrainServiceBoard(b couchcore.DrainBoard) couchcore.DrainBoard {
	b.Rows = append([]couchcore.DrainRow(nil), b.Rows...)
	b.Queue.Owners = append([]string(nil), b.Queue.Owners...)
	b.Queue.Active = append([]couchcore.DrainActivity(nil), b.Queue.Active...)
	return b
}
func (c *drainCoordinator) publishLocked() {
	b := couchcore.ProjectDrainBoard(c.record.State, c.opts.Now())
	if b.ID == "" {
		b.State = couchcore.DrainCancelled
		b.Detail = "no active drain"
	}
	if c.record.State.Phase == couchcore.DrainFinished && !c.record.Released {
		b.State = couchcore.DrainDraining
		b.Detail = "cancelling (dispatch held)"
		b.Recovery = "retry cancel"
	}
	if b.ID != "" && b.State != couchcore.DrainCancelled && len(c.census) > 0 {
		for _, r := range c.census {
			if len(b.Rows) >= couchcore.MaxDrainMembers {
				break
			}
			r.State = couchcore.DrainStuck
			if r.Recovery == "" {
				r.Recovery = "inspect status"
			}
			b.Rows = append(b.Rows, r)
		}
		b.State = couchcore.DrainStuck
		b.Detail = "fleet observation incomplete"
		b.Recovery = "inspect status"
	}
	c.board.Store(b)
	if c.opts.Publish != nil {
		c.opts.Publish(copyDrainServiceBoard(b))
	}
}
func (c *drainCoordinator) saveLocked() error {
	raw, err := encodeDrainServiceRecord(c.record, c.raw)
	if err == nil {
		err = c.opts.Store.Write(c.ctx, raw)
	}
	if err != nil {
		c.record.State.Error = "drain storage unknown: " + err.Error()
	} else {
		c.raw = raw
	}
	c.publishLocked()
	return err
}
func (c *drainCoordinator) event(e couchcore.DrainEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e.ID == "" {
		e.ID = c.record.State.ID
	}
	before := c.record.State
	c.record.State, _ = couchcore.ReduceDrain(c.record.State, e)
	if e.Kind == couchcore.DrainQueueObserved || e.Kind == couchcore.DrainMemberObserved || reflect.DeepEqual(before, c.record.State) {
		c.publishLocked()
		return nil
	}
	return c.saveLocked()
}
func (c *drainCoordinator) start(ctx context.Context, id, reason string, fast bool, timeout time.Duration, from couchmessage.Binding) (couchcore.DrainBoard, error) {
	if err := ctx.Err(); err != nil {
		return c.view(), err
	}
	if timeout < time.Second || timeout > 24*time.Hour {
		return c.view(), errors.New("drain timeout must be 1s to 24h")
	}
	if err := from.Validate(); err != nil {
		return c.view(), err
	}
	live := c.opts.Snapshot()
	if current, ok := live[from.Slot]; !ok || current.Binding != from || current.Session == "" {
		return c.view(), errors.New("steward missing from current drain inventory")
	}
	var census []couchcore.DrainRow
	if c.opts.Census != nil {
		census = c.opts.Census()
	}
	members := make([]couchcore.DrainMember, 0, len(live))
	controls := map[string]drainControlRecord{}
	for workspace, m := range live {
		if m.Binding.Slot != workspace || m.Session == "" {
			return c.view(), errors.New("unverified drain member")
		}
		if err := m.Binding.Validate(); err != nil {
			return c.view(), err
		}
		identity := m.Identity()
		members = append(members, couchcore.DrainMember{Workspace: workspace, Identity: identity})
		controls[workspace] = drainControlRecord{Member: m, NoticeID: drainEffectID(id, identity, "notice"), InterruptID: drainEffectID(id, identity, "interrupt")}
	}
	s, _, err := couchcore.StartDrain(couchcore.DrainStart{ID: id, Reason: reason, Fast: fast, Deadline: c.opts.Now().Add(timeout), Members: members})
	if err != nil {
		return c.view(), err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if old := c.record.State; old.ID != "" && (old.Phase != couchcore.DrainFinished || !c.record.Released) {
		if old.Reason == reason && old.Fast == fast {
			return c.view(), nil
		}
		return c.view(), errors.New("drain already active; inspect status or cancel")
	}
	if id == c.record.State.ID {
		if reason != c.record.State.Reason || fast != c.record.State.Fast {
			return c.view(), errors.New("conflicting completed drain ID; inspect status")
		}
		return c.view(), nil
	}
	epoch, err := nextDrainEpoch(c.record.Epoch)
	if err != nil {
		return c.view(), err
	}
	c.record = drainServiceRecord{SchemaVersion: 2, Namespace: c.namespace, Epoch: epoch, State: s, From: from, Controls: controls}
	c.raw = nil
	c.census = append([]couchcore.DrainRow(nil), census...)
	c.ackRevision = map[string]uint64{}
	err = c.saveLocked()
	holdErr := c.opts.Broker.HoldDrain(id)
	if err != nil {
		return c.view(), err
	}
	if holdErr != nil {
		c.record.State.Error = holdErr.Error()
		_ = c.saveLocked()
		return c.view(), holdErr
	}
	c.signal()
	return c.view(), nil
}
func (c *drainCoordinator) ack(ctx context.Context, id, workspace, identity string) (couchcore.DrainBoard, error) {
	if err := ctx.Err(); err != nil {
		return c.view(), err
	}
	live := c.opts.Snapshot()[workspace]
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.record.State
	m, ok := s.Members[workspace]
	if id != s.ID || !ok || m.Identity != identity || live.Identity() != identity || s.Phase != couchcore.DrainActive {
		return c.view(), errors.New("ack does not name current drain/member")
	}
	next, _ := couchcore.ReduceDrain(s, couchcore.DrainEvent{ID: id, Kind: couchcore.DrainAcknowledged, Workspace: workspace, Identity: identity})
	if !next.Members[workspace].Ack {
		return c.view(), errors.New("checkpoint notice not yet submitted")
	}
	if !m.Ack {
		c.ackRevision[workspace]++
	}
	c.record.State = next
	err := c.saveLocked()
	c.signal()
	return c.view(), err
}
func (c *drainCoordinator) cancel(ctx context.Context) (couchcore.DrainBoard, error) {
	if err := ctx.Err(); err != nil {
		return c.view(), err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.record.State
	if s.ID == "" || s.Phase == couchcore.DrainFinished && c.record.Released {
		return c.view(), nil
	}
	if c.record.SchemaVersion != 2 {
		return c.view(), errors.New("drain storage unknown; inspect persisted record before cancellation")
	}
	if err := c.opts.Broker.BeginDrainCancel(s.ID); err != nil {
		return c.view(), err
	}
	c.record.State, _ = couchcore.ReduceDrain(s, couchcore.DrainEvent{ID: s.ID, Kind: couchcore.DrainCancel})
	err := c.saveLocked()
	c.signal()
	return c.view(), err
}
func drainEffectID(id, identity, kind string) string {
	sum := sha256.Sum256([]byte(id + "\x00" + identity + "\x00" + kind))
	sum[6] = (sum[6] & 15) | 64
	sum[8] = (sum[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}
func (c *drainCoordinator) stateSnapshot() (couchcore.DrainState, map[string]drainControlRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.record.State
	controls := make(map[string]drainControlRecord, len(c.record.Controls))
	for k, v := range c.record.Controls {
		controls[k] = v
	}
	return s, controls
}
func (c *drainCoordinator) reconcile(parent context.Context) {
	c.pass.Lock()
	defer c.pass.Unlock()
	if c.ctx.Err() != nil {
		return
	}
	s, controls := c.stateSnapshot()
	if s.ID == "" {
		return
	}
	if c.opts.Census != nil {
		rows := c.opts.Census()
		c.mu.Lock()
		c.census = append([]couchcore.DrainRow(nil), rows...)
		c.publishLocked()
		c.mu.Unlock()
	}
	if s.Phase == couchcore.DrainFinished {
		c.finishRelease(s.ID)
		return
	}
	if s.Phase == couchcore.DrainPreparing {
		c.mu.Lock()
		started := c.record.PauseStarted
		if !started {
			c.record.PauseStarted = true
			if c.saveLocked() != nil {
				c.mu.Unlock()
				return
			}
		}
		c.mu.Unlock()
		if !started {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			_ = c.opts.Queue.Pause(ctx, s.Owner, "Couch drain "+s.ID+": "+s.Reason, s.Fast)
			cancel()
		}
	}
	c.observeQueue()
	s, controls = c.stateSnapshot()
	if s.Phase == couchcore.DrainRecovered {
		return
	}
	if s.Phase == couchcore.DrainCancelling {
		c.fanout(controls, func(workspace string, control drainControlRecord) { c.revoke(workspace, control) })
		s, _ = c.stateSnapshot()
		if s.ResumeIssued {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			_ = c.opts.Queue.Resume(ctx, s.Owner)
			cancel()
			c.observeQueue()
		}
		s, _ = c.stateSnapshot()
		if s.Phase == couchcore.DrainFinished {
			c.finishRelease(s.ID)
		}
		return
	}
	if s.Phase != couchcore.DrainActive {
		return
	}
	live := c.opts.Snapshot()
	for workspace, member := range live {
		old, exists := controls[workspace]
		if exists && old.Member.Identity() != member.Identity() {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainFailed, Detail: "member identity replaced: " + workspace + "; cancel and inspect status"})
			continue
		}
		if !exists {
			c.addMember(workspace, member)
		}
	}
	s, controls = c.stateSnapshot()
	pending := c.opts.Broker.DrainPending()
	c.fanout(controls, func(workspace string, control drainControlRecord) {
		member, ok := live[workspace]
		if !ok || member.Identity() != control.Member.Identity() {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainMemberObserved, Workspace: workspace, Identity: control.Member.Identity(), Observation: couchcore.DrainObservation{}})
			return
		}
		c.control(workspace, control)
		c.mu.Lock()
		revision := c.ackRevision[workspace]
		c.mu.Unlock()
		endpoint := c.opts.Endpoint(member.Binding)
		obs := member.Observation
		if endpoint == nil {
			obs.Known = false
		} else {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			fresh, err := endpoint.ObserveDrain(ctx)
			cancel()
			c.mu.Lock()
			current, present := c.record.Controls[workspace]
			expectedInstance := current.WrapperInstance
			expectedEpoch, expectedGeneration := c.record.Epoch, c.record.State.ID
			sameMember := present && current.Member.Identity() == member.Identity()
			c.mu.Unlock()
			obs.Known = obs.Known && err == nil && c.validDrainObservation(fresh) && sameMember && expectedInstance != "" && fresh.Instance == expectedInstance && fresh.Epoch == expectedEpoch && fresh.Generation == expectedGeneration
			obs.Idle = obs.Idle && fresh.Idle
			obs.Picker = obs.Picker || fresh.Picker
			obs.Background = obs.Background || fresh.AutomaticInput
			obs.Submission += fresh.Input
		}
		if _, held := pending[member.Binding]; held {
			obs.Background = true
		}
		obs.Sequence = 0 // Fresh calls are ordered by this serialized reconciliation pass.
		c.observeMember(workspace, member.Identity(), obs, revision, s.ID)
	})
	// An already-handed effect whose binding is outside the known current fleet
	// must block global completion, even when the SDLC queue is empty.
	for binding := range pending {
		if member, ok := live[binding.Slot]; !ok || member.Binding != binding {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainFailed, Detail: "pre-existing delivery outside current fleet"})
		}
	}
	_ = parent
}
func (c *drainCoordinator) observeQueue() {
	c.mu.Lock()
	generation := c.record.State.ID
	c.mu.Unlock()
	now := c.opts.Now()
	if !c.lastQueue.IsZero() && now.Sub(c.lastQueue) < 2*time.Second {
		return
	}
	c.lastQueue = now
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	q, err := c.opts.Queue.Status(ctx)
	cancel()
	o := couchcore.DrainQueueObservation{Known: err == nil && q.SchemaVersion == 1 && q.State != "unknown", Complete: q.Complete, Paused: q.State == "paused", Detail: q.Detail}
	if err != nil {
		o.Detail = err.Error()
	}
	for _, owner := range q.PauseOwners {
		o.Owners = append(o.Owners, owner.Owner)
	}
	for _, a := range q.Active {
		o.Active = append(o.Active, couchcore.DrainActivity{ID: a.ID, Workspace: a.Workspace, Kind: a.Kind, State: a.State, Head: a.Head})
	}
	_ = c.event(couchcore.DrainEvent{ID: generation, Kind: couchcore.DrainQueueObserved, Queue: o})
}

// validDrainObservation accepts capability evidence only from this namespace.
// A lower watermark may precede first admission; a newer/conflicting one cannot
// establish authority for this intent.
func (c *drainCoordinator) validDrainObservation(o couchmessage.DrainObservation) bool {
	c.mu.Lock()
	epoch, generation := c.record.Epoch, c.record.State.ID
	c.mu.Unlock()
	return o.Validate() == nil && o.Complete && o.Protocol == 2 && o.Namespace == c.namespace && o.Epoch <= epoch && (o.Epoch != epoch || o.Generation == generation)
}

func (c *drainCoordinator) controlFence(control drainControlRecord, kind couchmessage.DrainControlKind) (couchmessage.DrainFence, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.record.Controls[control.Member.Binding.Slot]
	if !ok || current.Member.Identity() != control.Member.Identity() || current.NoticeID != control.NoticeID || current.InterruptID != control.InterruptID || current.WrapperInstance == "" || current.WrapperInstance != control.WrapperInstance {
		return couchmessage.DrainFence{}, errors.New("drain control authority changed")
	}
	f := couchmessage.DrainFence{Namespace: c.record.Namespace, Generation: c.record.State.ID, Epoch: c.record.Epoch, Instance: current.WrapperInstance, Kind: kind}
	return f, f.Validate()
}
func (c *drainCoordinator) fanout(controls map[string]drainControlRecord, fn func(string, drainControlRecord)) {
	type job struct {
		workspace string
		control   drainControlRecord
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < min(4, len(controls)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if c.ctx.Err() == nil {
					fn(job.workspace, job.control)
				}
			}
		}()
	}
	for workspace, control := range controls {
		select {
		case jobs <- job{workspace, control}:
		case <-c.ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}
func (c *drainCoordinator) addMember(workspace string, member drainLiveMember) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.record.State.Phase != couchcore.DrainActive {
		return
	}
	if len(c.record.Controls) >= couchcore.MaxDrainMembers {
		c.record.State.Error = "membership exceeds limit"
		_ = c.saveLocked()
		return
	}
	identity := member.Identity()
	c.record.Controls[workspace] = drainControlRecord{Member: member, NoticeID: drainEffectID(c.record.State.ID, identity, "notice"), InterruptID: drainEffectID(c.record.State.ID, identity, "interrupt")}
	c.record.State, _ = couchcore.ReduceDrain(c.record.State, couchcore.DrainEvent{ID: c.record.State.ID, Kind: couchcore.DrainMemberJoined, Workspace: workspace, Identity: identity})
	_ = c.saveLocked()
}
func (c *drainCoordinator) control(workspace string, control drainControlRecord) {
	s, _ := c.stateSnapshot()
	if s.Phase != couchcore.DrainActive {
		return
	}
	member := s.Members[workspace]
	endpoint := c.opts.Endpoint(control.Member.Binding)
	if control.WrapperInstance != "" {
		if endpoint == nil {
			return
		}
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		observed, err := endpoint.ObserveDrain(ctx)
		cancel()
		if err != nil || !c.validDrainObservation(observed) || observed.Instance != control.WrapperInstance {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainFailed, Detail: "wrapper incarnation changed or unknown: " + workspace})
			return
		}
	}
	if s.Fast && member.Interrupt != couchcore.DrainDone {
		outcome := couchcore.DrainUnknown
		detail := "wrapper drain capability unavailable; relaunch needed"
		if endpoint != nil {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			var result couchmessage.DrainInterruptResult
			var err error
			if !control.InterruptStarted {
				observation, observeErr := endpoint.ObserveDrain(ctx)
				if observeErr != nil || !c.validDrainObservation(observation) {
					err = errors.New("interrupt observation unknown")
				} else if !c.beginInterrupt(workspace, observation) {
					cancel()
					return
				} else {
					control.WrapperInstance = observation.Instance
					control.InterruptObserved = &observation
					fence, fenceErr := c.controlFence(control, couchmessage.DrainInterruptControl)
					if fenceErr != nil {
						err = fenceErr
					} else {
						result, err = endpoint.InterruptDrain(ctx, control.InterruptID, fence, observation)
					}
				}
			} else {
				fence, fenceErr := c.controlFence(control, couchmessage.DrainInterruptControl)
				if fenceErr != nil {
					err = fenceErr
				} else {
					result, err = endpoint.DrainInterruptStatus(ctx, control.InterruptID, fence)
				}
			}
			cancel()
			expected, fenceErr := c.controlFence(control, couchmessage.DrainInterruptControl)
			if err == nil && (fenceErr != nil || result.ID != control.InterruptID || result.Fence != expected || control.InterruptObserved != nil && result.Turn != control.InterruptObserved.Turn) {
				err = errors.New("interrupt reply identity mismatch")
			}
			if err == nil {
				switch result.Outcome {
				case "interrupted", "skipped":
					outcome = couchcore.DrainDone
				case "refused":
					outcome = couchcore.DrainRefused
				}
				detail = result.Detail
			} else {
				detail = err.Error()
			}
		}
		_ = c.event(couchcore.DrainEvent{ID: s.ID, Kind: couchcore.DrainInterruptResult, Workspace: workspace, Identity: control.Member.Identity(), Outcome: outcome, Detail: detail})
		if outcome != couchcore.DrainDone {
			return
		}
		s, _ = c.stateSnapshot()
	}
	if s.Phase != couchcore.DrainActive {
		return
	}
	if !control.NoticeStarted {
		if endpoint == nil {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainNoticeResult, Workspace: workspace, Identity: control.Member.Identity(), Outcome: couchcore.DrainUnknown, Detail: "wrapper drain capability unavailable; relaunch needed"})
			return
		}
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		observed, err := endpoint.ObserveDrain(ctx)
		cancel()
		if err != nil || !c.validDrainObservation(observed) {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainNoticeResult, Workspace: workspace, Identity: control.Member.Identity(), Outcome: couchcore.DrainUnknown, Detail: "notice wrapper observation unknown"})
			return
		}
		if !c.beginNotice(workspace, observed.Instance) {
			return
		}
		control.WrapperInstance = observed.Instance
		fence, err := c.controlFence(control, couchmessage.DrainNoticeControl)
		if err != nil {
			return
		}
		text, _ := couchcore.DrainNotice(s.ID, s.Reason, s.Fast)
		ctx, cancel = context.WithTimeout(c.ctx, 5*time.Second)
		_, err = c.opts.Broker.SendDrain(ctx, s.ID, c.recordFrom(), control.NoticeID, control.Member.Binding, text, fence)
		cancel()
		if err != nil {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainNoticeResult, Workspace: workspace, Identity: control.Member.Identity(), Outcome: couchcore.DrainUnknown, Detail: err.Error()})
			return
		}
	}
	receipt, known, _ := c.opts.Broker.ReceiptSnapshot(control.NoticeID)
	outcome := couchcore.DrainPending
	detail := ""
	if !known {
		outcome = couchcore.DrainUnknown
		detail = "notice handoff unknown"
	} else if receipt.Status == couchmessage.Submitted {
		outcome = couchcore.DrainDone
	} else if receipt.Status.Terminal() {
		outcome = couchcore.DrainUnknown
		detail = "notice not submitted: " + string(receipt.Status)
	}
	_ = c.event(couchcore.DrainEvent{ID: s.ID, Kind: couchcore.DrainNoticeResult, Workspace: workspace, Identity: control.Member.Identity(), Outcome: outcome, Detail: detail})
}
func (c *drainCoordinator) recordFrom() couchmessage.Binding {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.record.From
}
func (c *drainCoordinator) beginInterrupt(workspace string, o couchmessage.DrainObservation) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.record.State.Phase != couchcore.DrainActive {
		return false
	}
	control := c.record.Controls[workspace]
	if control.InterruptStarted {
		return false
	}
	control.WrapperInstance = o.Instance
	control.InterruptStarted = true
	control.InterruptObserved = &o
	c.record.Controls[workspace] = control
	return c.saveLocked() == nil
}
func (c *drainCoordinator) beginNotice(workspace, instance string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.record.State.Phase != couchcore.DrainActive {
		return false
	}
	control := c.record.Controls[workspace]
	if control.NoticeStarted {
		return false
	}
	if control.WrapperInstance != "" && control.WrapperInstance != instance {
		return false
	}
	control.WrapperInstance = instance
	control.NoticeStarted = true
	c.record.Controls[workspace] = control
	return c.saveLocked() == nil
}
func (c *drainCoordinator) revoke(workspace string, control drainControlRecord) {
	outcome := couchcore.DrainDone
	detail := ""
	endpoint := c.opts.Endpoint(control.Member.Binding)
	if control.WrapperInstance != "" {
		same := false
		if endpoint != nil {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			observed, err := endpoint.ObserveDrain(ctx)
			cancel()
			same = err == nil && c.validDrainObservation(observed) && observed.Instance == control.WrapperInstance
		}
		if !same {
			_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainRevoked, Workspace: workspace, Identity: control.Member.Identity(), Outcome: couchcore.DrainUnknown, Detail: "wrapper incarnation changed; old control disposition unknown"})
			return
		}
	}
	if control.NoticeStarted {
		if _, known, _ := c.opts.Broker.ReceiptSnapshot(control.NoticeID); known {
			if err := c.opts.Broker.CancelDrainControl(c.ctx, c.recordID(), control.NoticeID); err != nil {
				outcome = couchcore.DrainUnknown
				detail = err.Error()
			}
		}
	}
	// Both IDs are revoked, including before attempted execution: this tombstones
	// any late endpoint call that passed the coordinator's earlier phase check.
	if endpoint == nil {
		outcome = couchcore.DrainUnknown
		detail = "wrapper unavailable for cancellation"
	} else {
		if control.WrapperInstance == "" {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			observed, err := endpoint.ObserveDrain(ctx)
			cancel()
			if err != nil || !c.validDrainObservation(observed) {
				_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainRevoked, Workspace: workspace, Identity: control.Member.Identity(), Outcome: couchcore.DrainUnknown, Detail: "cancel wrapper observation unknown"})
				return
			}
			control.WrapperInstance = observed.Instance
			c.mu.Lock()
			current := c.record.Controls[workspace]
			current.WrapperInstance = observed.Instance
			c.record.Controls[workspace] = current
			err = c.saveLocked()
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
		for _, item := range []struct {
			id   string
			kind couchmessage.DrainControlKind
		}{{control.NoticeID, couchmessage.DrainNoticeControl}, {control.InterruptID, couchmessage.DrainInterruptControl}} {
			id := item.id
			fence, fenceErr := c.controlFence(control, item.kind)
			if fenceErr != nil {
				outcome, detail = couchcore.DrainUnknown, fenceErr.Error()
				continue
			}
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			r, err := endpoint.RevokeDrain(ctx, id, fence)
			cancel()
			known := err == nil && r.ID == id && r.Fence == fence && r.Outcome == "not-committed"
			if err == nil && r.ID == id && r.Fence == fence && r.Outcome == "already-committed" {
				if id == control.InterruptID {
					known = r.Interrupt != nil && r.Interrupt.ID == id && r.Interrupt.Fence == fence && r.Interrupt.Outcome == "interrupted"
				} else {
					s, _ := c.stateSnapshot()
					body, _ := couchcore.DrainNotice(fence.Generation, s.Reason, s.Fast)
					known = r.Receipt != nil && r.Receipt.Message.ID == id && r.Receipt.Message.To == control.Member.Binding && r.Receipt.Message.From == c.recordFrom() && r.Receipt.Message.Origin == couchmessage.OriginSystem && r.Receipt.Message.DrainFence == fence && r.Receipt.Message.Body == body && r.Receipt.Status.Terminal() && r.Receipt.Status != couchmessage.Indeterminate
				}
			}
			if !known {
				outcome = couchcore.DrainUnknown
				detail = "cancellation unconfirmed"
				if err != nil {
					detail += ": " + err.Error()
				}
			}
		}
	}
	_ = c.event(couchcore.DrainEvent{Kind: couchcore.DrainRevoked, Workspace: workspace, Identity: control.Member.Identity(), Outcome: outcome, Detail: detail})
}
func (c *drainCoordinator) recordID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.record.State.ID
}

func (c *drainCoordinator) finishRelease(id string) {
	c.mu.Lock()
	released := c.record.Released
	c.mu.Unlock()
	if released {
		return
	}
	if err := c.opts.Broker.ReleaseDrain(id); err != nil {
		c.mu.Lock()
		c.record.State.Phase = couchcore.DrainCancelling
		c.record.State.Error = "cancellation unconfirmed: " + err.Error()
		_ = c.saveLocked()
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	c.record.Released = true
	_ = c.saveLocked()
	c.mu.Unlock()
}
func (c *drainCoordinator) observeMember(workspace, identity string, obs couchcore.DrainObservation, revision uint64, generation ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(generation) > 0 && generation[0] != c.record.State.ID {
		return
	}
	if revision != c.ackRevision[workspace] {
		obs.Idle = false
	}
	c.record.State, _ = couchcore.ReduceDrain(c.record.State, couchcore.DrainEvent{ID: c.record.State.ID, Kind: couchcore.DrainMemberObserved, Workspace: workspace, Identity: identity, Observation: obs})
	c.publishLocked()
}
