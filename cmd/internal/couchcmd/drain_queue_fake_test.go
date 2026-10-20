package couchcmd

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// FakeDrainQueue has the same ownership and reconciliation seam as the CLI.
// Finishing an interruption is explicit; resume never restarts interrupted work.
type FakeDrainQueue struct {
	mu               sync.Mutex
	owners           map[string]QueuePauseOwner
	active           []QueueActivity
	waiters          []QueueActivity
	LoseNextReply    bool
	ObservationError error
}

func NewFakeDrainQueue() *FakeDrainQueue {
	return &FakeDrainQueue{owners: map[string]QueuePauseOwner{}}
}
func (f *FakeDrainQueue) reply() error {
	if f.LoseNextReply {
		f.LoseNextReply = false
		return errors.New("effect applied; response lost")
	}
	return nil
}
func (f *FakeDrainQueue) Pause(ctx context.Context, owner, reason string, fast bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := f.owners[owner]; !exists {
		f.owners[owner] = QueuePauseOwner{Owner: owner, Reason: reason, CancelRunning: fast}
	}
	if fast {
		// The real queue interrupts suite-owning operations, never a separate
		// merge/landing. Multiple kind rows of one operation share its phase.
		suites := map[string]bool{}
		for _, a := range f.active {
			if a.Kind == "suite" && (a.State == "running" || a.State == "interrupting") {
				suites[a.ID] = true
			}
		}
		for i := range f.active {
			if suites[f.active[i].ID] {
				f.active[i].State = "interrupting"
			}
		}
	}
	return f.reply()
}
func (f *FakeDrainQueue) Resume(ctx context.Context, owner string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(f.owners, owner)
	return f.reply()
}
func (f *FakeDrainQueue) Status(ctx context.Context) (QueueSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	err := ctx.Err()
	if err == nil {
		err = f.ObservationError
	}
	if err != nil {
		return QueueSnapshot{State: "unknown", Detail: err.Error()}, err
	}
	s := QueueSnapshot{SchemaVersion: 1, State: "running", Complete: true, PauseOwners: []QueuePauseOwner{}, Active: append([]QueueActivity{}, f.active...)}
	for _, o := range f.owners {
		s.PauseOwners = append(s.PauseOwners, o)
	}
	sort.Slice(s.PauseOwners, func(i, j int) bool { return s.PauseOwners[i].Owner < s.PauseOwners[j].Owner })
	if len(s.PauseOwners) > 0 {
		s.State = "paused"
	}
	return s, nil
}
func (f *FakeDrainQueue) Enqueue(a QueueActivity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waiters = append(f.waiters, a)
}
func (f *FakeDrainQueue) AdmitNext() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.owners) > 0 || len(f.waiters) == 0 {
		return false
	}
	f.active = append(f.active, f.waiters[0])
	f.waiters = f.waiters[1:]
	return true
}
func (f *FakeDrainQueue) Finish(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.active[:0]
	for _, a := range f.active {
		if a.ID != id {
			out = append(out, a)
		}
	}
	f.active = out
}
