package couchtty

import (
	"errors"
	"sync"
)

var errOperationQueueOverloaded = errors.New("operation queue is full")

type operationRequest struct {
	key  string
	name string
	run  func() (any, error)
}

type operationCompletion struct {
	key   string
	name  string
	value any
	err   error
}

type operationQueue struct {
	requests chan operationRequest
	results  chan operationCompletion
	mu       sync.Mutex
	pending  map[string]bool
}

func newOperationQueue(capacity int) *operationQueue {
	if capacity < 1 {
		capacity = 1
	}
	return &operationQueue{
		requests: make(chan operationRequest, capacity),
		results:  make(chan operationCompletion, capacity),
		pending:  map[string]bool{},
	}
}

// Enqueue returns accepted=false,nil for an already pending exact request.
func (q *operationQueue) Enqueue(request operationRequest) (accepted bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.pending[request.key] {
		return false, nil
	}
	q.pending[request.key] = true
	select {
	case q.requests <- request:
		return true, nil
	default:
		delete(q.pending, request.key)
		return false, errOperationQueueOverloaded
	}
}

func (q *operationQueue) Run(stop <-chan struct{}) {
	for {
		select {
		case request := <-q.requests:
			value, err := request.run()
			q.mu.Lock()
			delete(q.pending, request.key)
			q.mu.Unlock()
			select {
			case q.results <- operationCompletion{key: request.key, name: request.name, value: value, err: err}:
			case <-stop:
				return
			}
		case <-stop:
			return
		}
	}
}
