package couchcore

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type leaveOwnerPassKey struct{}
type leaveOwnerPassArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	mu              sync.Mutex
	captures, reads int
	captured        context.Context
	requirePass     bool
}

func (f *leaveOwnerPassArtifacts) WithSessionOwnerPass(ctx context.Context) (context.Context, context.CancelFunc) {
	f.mu.Lock()
	f.captures++
	f.mu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, leaveOwnerPassKey{}, f)
	f.captured = ctx
	return ctx, cancel
}
func (f *leaveOwnerPassArtifacts) check(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.requirePass && ctx.Value(leaveOwnerPassKey{}) != f {
		return errors.New("owner observation outside detach pass")
	}
	if !f.requirePass && ctx.Value(leaveOwnerPassKey{}) != nil {
		return errors.New("standalone detach reused pass")
	}
	return nil
}
func (f *leaveOwnerPassArtifacts) PairSessionContext(ctx context.Context, a ThreadAddress) (PairSessionBinding, error) {
	if err := f.check(ctx); err != nil {
		return PairSessionBinding{}, err
	}
	return f.FakeThreadArtifactCollisionChecker.PairSessionContext(ctx, a)
}
func (f *leaveOwnerPassArtifacts) NamedPairSessionContext(ctx context.Context, a ThreadAddress, name string) (PairSessionBinding, error) {
	if err := f.check(ctx); err != nil {
		return PairSessionBinding{}, err
	}
	return f.FakeThreadArtifactCollisionChecker.NamedPairSessionContext(ctx, a, name)
}
func TestLeaveDetachScopesEveryObservationToOneOwnerPass(t *testing.T) {
	withParallelism(t, 6)
	c, _, threads := leaveFleet(t, 21)
	f := &leaveOwnerPassArtifacts{FakeThreadArtifactCollisionChecker: c.Artifacts.(*FakeThreadArtifactCollisionChecker), requirePass: true}
	c.Artifacts = f
	result, err := c.Leave(t.Context(), LeaveDetach)
	if err != nil || len(result.Detached) != len(threads) {
		t.Fatalf("leave=%+v err=%v", result, err)
	}
	if f.captures != 1 || f.reads != 3*len(threads) {
		t.Fatalf("captures=%d reads=%d", f.captures, f.reads)
	}
	if !errors.Is(f.captured.Err(), context.Canceled) {
		t.Fatal("completed pass retained its lifetime")
	}
	standalone, _, records := leaveFleet(t, 1)
	single := &leaveOwnerPassArtifacts{FakeThreadArtifactCollisionChecker: standalone.Artifacts.(*FakeThreadArtifactCollisionChecker)}
	standalone.Artifacts = single
	if _, err := standalone.Detach(t.Context(), records[0].Address); err != nil {
		t.Fatal(err)
	}
	if single.captures != 0 || single.reads != 2 {
		t.Fatalf("standalone captures=%d reads=%d", single.captures, single.reads)
	}
}
