package couchtty

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestOperationQueueCoalescesAndRefusesOverloadWithoutEffects(t *testing.T) {
	queue := newOperationQueue(1)
	effects := 0
	want := couchcore.ThreadAddress{RepoScope: "scope", Tag: "tag"}
	request := operationRequest{key: "park/scope/tag", name: "park", run: func() (any, error) {
		effects++
		return want, nil
	}}
	accepted, err := queue.Enqueue(request)
	if !accepted || err != nil {
		t.Fatalf("first enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(request)
	if accepted || err != nil {
		t.Fatalf("duplicate enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(operationRequest{key: "park/scope/other", name: "park", run: func() (any, error) {
		effects++
		return nil, nil
	}})
	if accepted || !errors.Is(err, ErrOperationQueueOverloaded) || effects != 0 {
		t.Fatalf("overload = accepted %v err %v effects %d", accepted, err, effects)
	}

	stop := make(chan struct{})
	defer close(stop)
	go queue.Run(stop)
	select {
	case result := <-queue.results:
		if result.err != nil || result.name != "park" || result.key != request.key || !reflect.DeepEqual(result.value, want) || effects != 1 {
			t.Fatalf("result = %+v effects %d", result, effects)
		}
	case <-testwait.After(t):
		t.Fatal("timed out waiting for queued operation")
	}

	retry := operationRequest{key: "park/scope/other", name: "park", run: func() (any, error) {
		effects++
		return "retried", nil
	}}
	accepted, err = queue.Enqueue(retry)
	if !accepted || err != nil {
		t.Fatalf("overload did not restore rejected request = %v, %v", accepted, err)
	}
	result := <-queue.results
	if result.key != retry.key || result.value != "retried" || effects != 2 {
		t.Fatalf("retried result = %+v effects %d", result, effects)
	}
	select {
	case extra := <-queue.results:
		t.Fatalf("request emitted duplicate completion %+v", extra)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestOperationQueueCarriesTypedResultForEveryMenuOperation(t *testing.T) {
	for _, operation := range []string{"switch", "resume", "park", "reboot", "start", "leave"} {
		t.Run(operation, func(t *testing.T) {
			queue := newOperationQueue(1)
			stop := make(chan struct{})
			defer close(stop)
			go queue.Run(stop)
			want := struct {
				Operation string
				Sequence  int
			}{Operation: operation, Sequence: 7}
			accepted, err := queue.Enqueue(operationRequest{
				key: operation + "/scope/tag", name: operation,
				run: func() (any, error) { return want, nil },
			})
			if !accepted || err != nil {
				t.Fatalf("enqueue = %v, %v", accepted, err)
			}
			select {
			case result := <-queue.results:
				if result.name != operation || result.key != operation+"/scope/tag" || !reflect.DeepEqual(result.value, want) || result.err != nil {
					t.Fatalf("completion = %+v", result)
				}
			case <-testwait.After(t):
				t.Fatal("timed out waiting for typed result")
			}
		})
	}
}

func TestChildExitNeverProvesPark(t *testing.T) {
	for _, phase := range []string{"requested", "awaiting", "completion-retained", "finalized"} {
		t.Run(phase, func(t *testing.T) {
			ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/repo")
			if err != nil {
				t.Fatal(err)
			}
			store := couchcore.NewThreadStore(ns)
			profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
			record, err := store.CreateThread(couchcore.ThreadRecord{
				SchemaVersion: couchcore.ThreadSchemaVersion,
				Address:       couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"},
				StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), LastActiveAt: time.Unix(1, 0).UTC(),
				Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "helper", State: couchcore.IncarnationLive, RepoIdentity: "/repo/.git", LaunchProfile: &profile}},
				LatestLaunchProfile: &profile, Revision: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			identity := couchcore.ParkIdentity{Nonce: "park-exit-proof", Address: record.Address, PID: 42, ProcessIdentity: "helper"}
			record, err = store.BeginPark(record.Address, record.Revision, identity)
			if err != nil {
				t.Fatal(err)
			}
			if phase != "requested" {
				record, err = store.AdvancePark(context.Background(), record.Address, record.Revision, couchcore.ParkEvent{Kind: couchcore.ParkRequestCommitted, Identity: identity, Attempt: 1})
				if err != nil {
					t.Fatal(err)
				}
			}
			if phase == "finalized" {
				record, err = store.FinalizePark(context.Background(), record.Address, record.Revision, identity, 1, time.Unix(2, 0).UTC())
				if err != nil {
					t.Fatal(err)
				}
			}

			console := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			child := ptychild.NewFakeChild(nil)
			console.attachThreadActor("handle", "actor", record.Address, "/repo", "repo", child)
			console.onExit(childExit{id: "handle", code: 0})
			child.Exit(0)
			console.Stop()
			after, err := store.GetThread(record.Address)
			if err != nil || !reflect.DeepEqual(after, record) {
				t.Fatalf("child exit changed durable phase %s: %+v, %v", phase, after, err)
			}
		})
	}
}

// Several workers drain one queue (pair#205): up to N requests run at once,
// and every result still arrives on q.results for the console goroutine.
func TestOperationQueueRunsRequestsConcurrentlyAcrossWorkers(t *testing.T) {
	const workers = 3
	q := newOperationQueue(16)
	stop := make(chan struct{})
	defer close(stop)
	for i := 0; i < workers; i++ {
		go q.Run(stop)
	}
	started, release := make(chan struct{}, workers), make(chan struct{})
	for i := 0; i < workers; i++ {
		key := fmt.Sprintf("job-%d", i)
		if accepted, err := q.Enqueue(operationRequest{key: key, name: key, run: func() (any, error) {
			started <- struct{}{}
			<-release
			return key, nil
		}}); !accepted || err != nil {
			t.Fatalf("enqueue %s: %v %v", key, accepted, err)
		}
	}
	for i := 0; i < workers; i++ {
		select {
		case <-started:
		case <-testwait.After(t):
			t.Fatalf("only %d of %d requests ran at once", i, workers)
		}
	}
	close(release)
	got := map[any]bool{}
	for i := 0; i < workers; i++ {
		select {
		case completion := <-q.results:
			got[completion.value] = true
		case <-testwait.After(t):
			t.Fatal("a result never reached q.results")
		}
	}
	if len(got) != workers {
		t.Fatalf("results = %v", got)
	}
}

// These workers are joined at cleanup; held requests must register their release
// after this helper so cleanup releases them before joining the workers.
func startOperationQueueWorkers(t *testing.T, q *operationQueue, lifecycle, navigation int) {
	t.Helper()
	stop := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < lifecycle+navigation; i++ {
		workers.Add(1)
		go func(navigation bool) {
			defer workers.Done()
			if navigation {
				q.RunNavigation(stop)
			} else {
				q.Run(stop)
			}
		}(i >= lifecycle)
	}
	t.Cleanup(func() { close(stop); workers.Wait() })
}

func awaitOperationQueueSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-testwait.After(t):
		t.Fatal("queue worker did not reach handshake")
	}
}

func awaitOperationQueueCompletion(t *testing.T, q *operationQueue) operationCompletion {
	t.Helper()
	select {
	case result := <-q.results:
		return result
	case <-testwait.After(t):
		t.Fatal("queue completion missing")
		return operationCompletion{}
	}
}

func requireOperationQueueAdmission(t *testing.T, q *operationQueue, request operationRequest) {
	t.Helper()
	if accepted, err := q.Enqueue(request); !accepted || err != nil {
		t.Fatalf("enqueue %q = %v, %v", request.key, accepted, err)
	}
}

func TestOperationQueueNavigationHasReservedWorker(t *testing.T) {
	q := newOperationQueue(16)
	startOperationQueueWorkers(t, q, 2, 1)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	started := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		requireOperationQueueAdmission(t, q, operationRequest{key: fmt.Sprint("held-", i), run: func() (any, error) { started <- struct{}{}; <-release; return nil, nil }})
	}
	awaitOperationQueueSignal(t, started)
	awaitOperationQueueSignal(t, started)
	var resumes atomic.Int32
	requireOperationQueueAdmission(t, q, operationRequest{key: "hosted", navigation: func() (any, error) { return "hosted", nil }, run: func() (any, error) { resumes.Add(1); return nil, nil }})
	result := awaitOperationQueueCompletion(t, q)
	if result.key != "hosted" || result.value != "hosted" || result.err != nil || resumes.Load() != 0 {
		t.Fatalf("navigation = %+v, resumes=%d", result, resumes.Load())
	}
}

func TestOperationQueueLanesShareQueuedCapacity(t *testing.T) {
	q := newOperationQueue(2)
	var effects atomic.Int32
	request := func(key string, navigation bool) operationRequest {
		r := operationRequest{key: key, run: func() (any, error) { effects.Add(1); return key, nil }}
		if navigation {
			r.navigation = r.run
		}
		return r
	}
	requireOperationQueueAdmission(t, q, request("navigation", true))
	requireOperationQueueAdmission(t, q, request("lifecycle", false))
	for _, navigation := range []bool{false, true} {
		if accepted, err := q.Enqueue(request(fmt.Sprint("overflow-", navigation), navigation)); accepted || !errors.Is(err, ErrOperationQueueOverloaded) {
			t.Fatalf("aggregate overload: %v, %v", accepted, err)
		}
	}
	if effects.Load() != 0 {
		t.Fatal("enqueue executed rejected work")
	}
	// A lifecycle worker must leave the queued navigation request untouched.
	startOperationQueueWorkers(t, q, 1, 0)
	result := awaitOperationQueueCompletion(t, q)
	if result.key != "lifecycle" {
		t.Fatalf("lifecycle lane consumed %+v", result)
	}
	startOperationQueueWorkers(t, q, 0, 1)
	result = awaitOperationQueueCompletion(t, q)
	if result.key != "navigation" || effects.Load() != 2 {
		t.Fatalf("navigation lane result=%+v effects=%d", result, effects.Load())
	}
}

func TestOperationQueueNavigationTransfersSamePendingRequest(t *testing.T) {
	q := newOperationQueue(2)
	startOperationQueueWorkers(t, q, 0, 1)
	var navigations, lifecycles atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	request := operationRequest{key: "lost-host", name: "switch", navigation: func() (any, error) {
		navigations.Add(1)
		close(entered)
		<-release
		return nil, fmt.Errorf("host replaced: %w", errNavigationNeedsLifecycle)
	}, run: func() (any, error) { lifecycles.Add(1); return "resumed", nil }}
	// A second navigation provides an explicit handshake after the first transfer.
	requireOperationQueueAdmission(t, q, request)
	awaitOperationQueueSignal(t, entered)
	requireOperationQueueAdmission(t, q, operationRequest{key: "nav-barrier", navigation: func() (any, error) { return "barrier", nil }})
	releaseOnce.Do(func() { close(release) })
	result := awaitOperationQueueCompletion(t, q)
	if result.key != "nav-barrier" {
		t.Fatalf("transfer completed on navigation worker: %+v", result)
	}
	if accepted, err := q.Enqueue(request); accepted || err != nil {
		t.Fatalf("transfer lost pending owner: %v,%v", accepted, err)
	}
	if lifecycles.Load() != 0 {
		t.Fatal("navigation worker executed lifecycle")
	}
	startOperationQueueWorkers(t, q, 1, 0)
	result = awaitOperationQueueCompletion(t, q)
	if result.key != request.key || result.value != "resumed" || result.err != nil || navigations.Load() != 1 || lifecycles.Load() != 1 {
		t.Fatalf("transfer result=%+v nav=%d life=%d", result, navigations.Load(), lifecycles.Load())
	}
}

func TestOperationQueueNavigationTransferRefusesFullQueueOnce(t *testing.T) {
	// The lifecycle quota is full although the navigation reservation is free.
	q := newOperationQueue(2)
	startOperationQueueWorkers(t, q, 0, 1)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var resumes atomic.Int32
	request := operationRequest{key: "lost-host", navigation: func() (any, error) { close(entered); <-release; return nil, errNavigationNeedsLifecycle }, run: func() (any, error) { resumes.Add(1); return nil, nil }}
	requireOperationQueueAdmission(t, q, request)
	awaitOperationQueueSignal(t, entered)
	if accepted, err := q.Enqueue(request); accepted || err != nil {
		t.Fatalf("navigation execution lost pending owner: %v, %v", accepted, err)
	}
	requireOperationQueueAdmission(t, q, operationRequest{key: "queued-lifecycle", run: func() (any, error) { return "queued", nil }})
	releaseOnce.Do(func() { close(release) })
	result := awaitOperationQueueCompletion(t, q)
	if result.key != request.key || !errors.Is(result.err, ErrOperationQueueOverloaded) || resumes.Load() != 0 {
		t.Fatalf("transfer overload=%+v resumes=%d", result, resumes.Load())
	}
	// Completion relinquishes queue ownership; the same key can be admitted anew.
	startOperationQueueWorkers(t, q, 1, 0)
	result = awaitOperationQueueCompletion(t, q)
	if result.key != "queued-lifecycle" {
		t.Fatalf("unexpected duplicate completion: %+v", result)
	}
	requireOperationQueueAdmission(t, q, operationRequest{key: request.key, navigation: func() (any, error) { return "retry", nil }})
	result = awaitOperationQueueCompletion(t, q)
	if result.value != "retry" || result.err != nil {
		t.Fatalf("retry=%+v", result)
	}
}

func TestOperationQueueLifecycleSaturationReservesNavigationWithinAggregateCap(t *testing.T) {
	q := newOperationQueue(16)
	var effects atomic.Int32
	for i := 0; i < 15; i++ {
		requireOperationQueueAdmission(t, q, operationRequest{key: fmt.Sprint("life-", i), run: func() (any, error) { effects.Add(1); return nil, nil }})
	}
	if accepted, err := q.Enqueue(operationRequest{key: "life-overflow"}); accepted || !errors.Is(err, ErrOperationQueueOverloaded) {
		t.Fatalf("lifecycle must preserve navigation reservation: %v,%v", accepted, err)
	}
	requireOperationQueueAdmission(t, q, operationRequest{key: "reserved-navigation", navigation: func() (any, error) { effects.Add(1); return "selected", nil }})
	for _, request := range []operationRequest{{key: "overflow-life"}, {key: "overflow-nav", navigation: func() (any, error) { effects.Add(1); return nil, nil }}} {
		if accepted, err := q.Enqueue(request); accepted || !errors.Is(err, ErrOperationQueueOverloaded) {
			t.Fatalf("aggregate cap: %v,%v", accepted, err)
		}
	}
	q.mu.Lock()
	pending := len(q.pending)
	q.mu.Unlock()
	if pending != 16 || effects.Load() != 0 {
		t.Fatalf("queued owners=%d effects=%d", pending, effects.Load())
	}
	startOperationQueueWorkers(t, q, 0, 1)
	result := awaitOperationQueueCompletion(t, q)
	if result.key != "reserved-navigation" || result.value != "selected" || effects.Load() != 1 {
		t.Fatalf("reserved navigation=%+v effects=%d", result, effects.Load())
	}
}
