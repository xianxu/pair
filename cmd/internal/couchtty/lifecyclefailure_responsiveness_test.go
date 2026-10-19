package couchtty

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type stalledLifecycleStore struct {
	entered      chan struct{}
	release      chan struct{}
	once         sync.Once
	finished     atomic.Bool
	contextError chan error
}

func (s *stalledLifecycleStore) Path() string                             { return "fixture/failures.json" }
func (s *stalledLifecycleStore) Read() ([]couchcore.LaunchFailure, error) { return nil, nil }
func (s *stalledLifecycleStore) Save(ctx context.Context, _ couchcore.LaunchFailure) error {
	s.once.Do(func() { close(s.entered) })
	defer s.finished.Store(true)
	defer func() { s.contextError <- ctx.Err() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.release:
		return nil
	}
}

func TestLifecycleFailureRunResponsiveWhileStoreStalled(t *testing.T) {
	for _, name := range []string{"menu", "reattach", "navigation"} {
		t.Run(name, func(t *testing.T) {
			store := &stalledLifecycleStore{entered: make(chan struct{}), release: make(chan struct{}), contextError: make(chan error, 1)}
			var release sync.Once
			defer release.Do(func() { close(store.release) })
			recorder := couchcore.NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "failure", nil })
			f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
				c.SetOperationAttemptFactory(func(call couchcore.OperationCall) (*couchcore.OperationAttempt, error) {
					return recorder.Begin(call.Name, couchcore.LifecycleFailureTarget{Address: menuAddress("target")})
				})
				c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, errors.New("failed lifecycle") })
			})
			effect := MenuEffect{Operation: "resume", Attempt: 77, Background: name == "reattach", Args: map[string]string{"repo-scope": "scope", "tag": "target"}}
			if name == "navigation" {
				attempt, err := recorder.Begin("switch", couchcore.LifecycleFailureTarget{Address: menuAddress("target")})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.con.operationQueue.Enqueue(operationRequest{key: "navigation", name: "switch", attempt: attempt, origin: MenuOperationOrigin{Operation: "switch"}, navigation: func() (any, error) { return nil, errors.New("navigation failed") }, run: func() (any, error) { t.Error("navigation failure replayed effects"); return nil, nil }}); err != nil {
					t.Fatal(err)
				}
			} else if name == "reattach" {
				f.con.runBackgroundOperation(effect)
			} else {
				f.con.mu.Lock()
				setTestMenuOperation(&f.con.menu, MenuOperationOrigin{Operation: "resume", Attempt: 77, Address: menuAddress("target")})
				f.con.mu.Unlock()
				f.con.runMenuOperation(effect)
			}
			select {
			case <-store.entered:
			case <-testwait.After(t):
				t.Fatal("store was not entered")
			}
			before := len(f.child.Writes())
			if _, err := f.stdin.Write([]byte("z")); err != nil {
				t.Fatal(err)
			}
			testwait.Until(t, "Run input while diagnostic store remains blocked", func() bool { return len(f.child.Writes()) > before || store.finished.Load() })
			if len(f.child.Writes()) == before || store.finished.Load() {
				release.Do(func() { close(store.release) })
				t.Fatal("Run input was not processed before diagnostic publication ended")
			}

			f.con.Stop()
			select {
			case <-f.done:
			case <-testwait.After(t):
				release.Do(func() { close(store.release) })
				t.Fatal("Stop did not cancel and join diagnostic publication")
			}
			if err := <-store.contextError; !errors.Is(err, context.Canceled) {
				t.Fatalf("store cancellation = %v, want console lifetime cancellation", err)
			}
		})
	}
}

func TestLifecycleFailureRemoteCallbackDoesNotBlockRun(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	f := newFixture(t, 24, 80)
	err := f.con.EnqueueRemoteOperation("remote-stalled", "park", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "park", Context: ctx}, nil
	}, func() {}, func(any, error) {
		close(entered)
		select {
		case <-release:
		case <-f.con.lifetime.Done():
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("callback not entered")
	}
	before := len(f.child.Writes())
	_, _ = f.stdin.Write([]byte("z"))
	testwait.Until(t, "Run input while remote completion callback remains blocked", func() bool { return len(f.child.Writes()) > before })

	f.con.Stop()
	select {
	case <-f.done:
	case <-testwait.After(t):
		once.Do(func() { close(release) })
		t.Fatal("Stop did not join remote callback")
	}
}

func TestLifecycleFailureBeforeEnqueueRefusalDoesNotPublish(t *testing.T) {
	r := newRemoteRig(t)
	r.c.operationQueue = newOperationQueue(1)
	if _, err := r.c.operationQueue.Enqueue(operationRequest{key: "full", run: func() (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewFakeLaunchFailureStore()
	recorder := couchcore.NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "failure", nil })
	r.c.SetOperationAttemptFactory(func(call couchcore.OperationCall) (*couchcore.OperationAttempt, error) {
		return recorder.Begin(call.Name, couchcore.LifecycleFailureTarget{Address: menuAddress("target")})
	})
	setTestMenuOperation(&r.c.menu, MenuOperationOrigin{Operation: "resume", Attempt: 77, Address: menuAddress("target")})
	r.c.runMenuOperation(MenuEffect{Operation: "resume", Attempt: 77})
	if len(store.Writes) != 0 {
		t.Fatal("pre-enqueue UI refusal performed diagnostic IO")
	}
}
