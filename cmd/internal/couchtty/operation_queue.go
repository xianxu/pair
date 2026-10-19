package couchtty

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

var errNavigationNeedsLifecycle = errors.New("hosted navigation needs lifecycle")

var ErrOperationQueueOverloaded = errors.New("operation queue is full")

type operationRequest struct {
	attempt *couchcore.OperationAttempt
	key     string
	name    string
	origin  MenuOperationOrigin
	run     func() (any, error)
	// navigation is present only for a currently hosted selection; run remains
	// its lifecycle fallback when the hosted endpoint proof disappears.
	navigation func() (any, error)
	// remote is a socket-originated job's completion state (pair#367 M2);
	// nil for every console-originated request.
	remote *remoteOperation
}

type operationCompletion struct {
	attempt  *couchcore.OperationAttempt
	key      string
	name     string
	origin   MenuOperationOrigin
	value    any
	err      error
	remote   *remoteOperation
	owner    *atomic.Bool
	prepared bool
}

// complete is the request's completion carrying its outcome.
func (r operationRequest) complete(value any, err error) operationCompletion {
	return operationCompletion{attempt: r.attempt, key: r.key, name: r.name, origin: r.origin, value: value, err: err, remote: r.remote, owner: new(atomic.Bool)}
}

type operationQueue struct {
	requests           chan operationRequest
	navigationRequests chan operationRequest
	capacity           int
	results            chan operationCompletion
	mu                 sync.Mutex
	pending            map[string]bool
	// prepare remains on lifecycle workers; navigation never runs adoption.
	prepare func(operationCompletion) operationCompletion
	abandon func(operationCompletion)
}

func newOperationQueue(capacity int) *operationQueue {
	if capacity < 1 {
		capacity = 1
	}
	lifecycleCapacity := capacity - 1
	if lifecycleCapacity < 1 {
		lifecycleCapacity = 1
	}
	return &operationQueue{
		requests:           make(chan operationRequest, lifecycleCapacity),
		navigationRequests: make(chan operationRequest, 1),
		capacity:           capacity,
		results:            make(chan operationCompletion, capacity),
		pending:            map[string]bool{},
	}
}

// Enqueue returns accepted=false,nil for an already pending exact request.
func (q *operationQueue) Enqueue(request operationRequest) (accepted bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pending[request.key] {
		return false, nil
	}
	if err := q.enqueueLocked(request, request.navigation != nil); err != nil {
		return false, err
	}
	q.pending[request.key] = true
	return true, nil
}

// enqueueLocked preserves one queued navigation position within the shared cap.
// Receivers can only decrease occupancy while the mutex serializes all sends.
func (q *operationQueue) enqueueLocked(request operationRequest, navigation bool) error {
	if len(q.requests)+len(q.navigationRequests) >= q.capacity {
		return ErrOperationQueueOverloaded
	}
	lane := q.requests
	if navigation {
		lane = q.navigationRequests
	}
	select {
	case lane <- request:
		return nil
	default:
		return ErrOperationQueueOverloaded
	}
}

// Run owns lifecycle execution, including requests transferred by navigation.
func (q *operationQueue) Run(stop <-chan struct{}) {
	for {
		select {
		case request := <-q.requests:
			value, err := request.run()
			completed := request.complete(value, err)
			if q.prepare != nil {
				completed = q.prepare(completed)
			}
			if !q.finish(stop, completed) {
				return
			}
		case <-stop:
			return
		}
	}
}

// RunNavigation never invokes lifecycle work. A vanished hosted endpoint hands
// the same pending owner to the lifecycle lane once, or completes as overloaded.
func (q *operationQueue) RunNavigation(stop <-chan struct{}) {
	for {
		select {
		case request := <-q.navigationRequests:
			value, err := request.navigation()
			if errors.Is(err, errNavigationNeedsLifecycle) {
				q.mu.Lock()
				err = q.enqueueLocked(request, false)
				q.mu.Unlock()
				if err == nil {
					continue
				}
				value = nil
			}
			if !q.finish(stop, request.complete(value, err)) {
				return
			}
		case <-stop:
			return
		}
	}
}

func (q *operationQueue) finish(stop <-chan struct{}, completed operationCompletion) bool {
	q.mu.Lock()
	delete(q.pending, completed.key)
	q.mu.Unlock()
	select {
	case q.results <- completed:
		return true
	case <-stop:
		if q.abandon != nil {
			q.abandon(completed)
		}
		return false
	}
}

// abandonPending is called only after all queue workers have joined. Every
// accepted caller then gets its final cancellation even when Run has stopped.
func (q *operationQueue) abandonPending() {
	for {
		var completed operationCompletion
		select {
		case completed = <-q.results:
		case request := <-q.requests:
			completed = request.complete(nil, context.Canceled)
		case request := <-q.navigationRequests:
			completed = request.complete(nil, context.Canceled)
		default:
			q.mu.Lock()
			clear(q.pending)
			q.mu.Unlock()
			return
		}
		if q.abandon != nil {
			q.abandon(completed)
		}
	}
}
