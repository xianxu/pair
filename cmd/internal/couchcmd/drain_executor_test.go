package couchcmd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func receiveDrainExecutor[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(10 * time.Second):
		t.Fatal("drain executor did not reach controlled barrier")
	}
	var zero T
	return zero
}

func TestDrainAdmissionExecutorFourWorkersAndTargetFIFO(t *testing.T) {
	executor := newDrainAdmissionExecutor(context.Background())
	defer executor.close()
	started := make(chan string, 8)
	release := make(chan struct{})
	var mu sync.Mutex
	var order []int
	tasks := []drainAdmissionTask{}
	for i := range 4 {
		tasks = append(tasks, drainAdmissionTask{target: "a", run: func(ctx context.Context) {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			started <- "a"
			select {
			case <-release:
			case <-ctx.Done():
			}
		}})
	}
	for _, target := range []string{"b", "c", "d", "e"} {
		tasks = append(tasks, drainAdmissionTask{target: target, run: func(ctx context.Context) {
			started <- target
			select {
			case <-release:
			case <-ctx.Done():
			}
		}})
	}
	if err := executor.enqueue(tasks); err != nil {
		t.Fatal(err)
	}
	first := map[string]bool{}
	for range 4 {
		first[receiveDrainExecutor(t, started)] = true
	}
	if len(first) != 4 || !first["a"] || !first["b"] || !first["c"] || !first["d"] {
		t.Fatalf("oldest eligible tasks = %v", first)
	}
	select {
	case extra := <-started:
		t.Fatalf("more than four workers, or occupied lane ran twice: %s", extra)
	default:
	}
	close(release)
	for range 4 {
		receiveDrainExecutor(t, started)
	}
	executor.close()
	mu.Lock()
	defer mu.Unlock()
	for i, got := range order {
		if got != i {
			t.Fatalf("target order=%v", order)
		}
	}
	if len(order) != 4 {
		t.Fatalf("target order=%v", order)
	}
}

func TestDrainAdmissionExecutorBoundsActiveAndPendingAtomically(t *testing.T) {
	executor := newDrainAdmissionExecutor(context.Background())
	defer executor.close()
	started := make(chan struct{}, 1)
	if err := executor.enqueue([]drainAdmissionTask{{target: "same", run: func(ctx context.Context) { started <- struct{}{}; <-ctx.Done() }}}); err != nil {
		t.Fatal(err)
	}
	receiveDrainExecutor(t, started)
	capacity := maxSlotOperationReceipts * couchmessage.MaxActors
	completed := make(chan error, capacity)
	queued := make([]drainAdmissionTask, capacity-2)
	for i := range queued {
		queued[i] = drainAdmissionTask{target: "same", run: func(ctx context.Context) { completed <- ctx.Err() }}
	}
	if err := executor.enqueue(queued); err != nil {
		t.Fatal(err)
	}
	rejected := []drainAdmissionTask{
		{target: "overflow-a", run: func(context.Context) { t.Error("rejected batch partly executed") }},
		{target: "overflow-b", run: func(context.Context) { t.Error("rejected batch partly executed") }},
	}
	if err := executor.enqueue(rejected); err == nil {
		t.Fatal("accepted batch beyond pending+active bound")
	}
	if err := executor.enqueue(queued[:1]); err != nil {
		t.Fatalf("rejected batch consumed the final capacity: %v", err)
	}
	if err := executor.enqueue(rejected[:1]); err == nil {
		t.Fatal("active task did not count against capacity")
	}
	executor.close()
	for range capacity - 1 {
		if err := receiveDrainExecutor(t, completed); err != context.Canceled {
			t.Fatalf("queued cancellation=%v", err)
		}
	}
	if err := executor.enqueue(nil); err == nil {
		t.Fatal("accepted enqueue after close")
	}
}

func TestDrainAdmissionExecutorCloseCancelsAndJoins(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	executor := newDrainAdmissionExecutor(parent)
	defer executor.close()
	started := make(chan struct{}, 4)
	finished := make(chan error, 8)
	tasks := make([]drainAdmissionTask, 8)
	for i := range tasks {
		tasks[i] = drainAdmissionTask{target: string(rune('a' + i%4)), run: func(ctx context.Context) {
			if ctx.Err() == nil {
				started <- struct{}{}
				<-ctx.Done()
			}
			finished <- ctx.Err()
		}}
	}
	if err := executor.enqueue(tasks); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		receiveDrainExecutor(t, started)
	}
	cancel()
	executor.close()
	for range 8 {
		if err := receiveDrainExecutor(t, finished); err != context.Canceled {
			t.Fatalf("completion=%v", err)
		}
	}
	if err := executor.enqueue(tasks[:1]); err == nil {
		t.Fatal("accepted after parent cancellation")
	}
}

func TestDrainAdmissionExecutorBudgetStartsAtDispatch(t *testing.T) {
	executor := newDrainAdmissionExecutor(context.Background())
	defer executor.close()
	deadlines := make(chan time.Time, 5)
	tasks := make([]drainAdmissionTask, 5)
	for i := range tasks {
		tasks[i] = drainAdmissionTask{target: string(rune('a' + i)), run: func(ctx context.Context) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("missing admission deadline")
			}
			deadlines <- deadline
			<-ctx.Done()
		}}
	}
	before := time.Now()
	if err := executor.enqueue(tasks); err != nil {
		t.Fatal(err)
	}
	first := receiveDrainExecutor(t, deadlines)
	for range 3 {
		receiveDrainExecutor(t, deadlines)
	}
	fifth := receiveDrainExecutor(t, deadlines)
	if first.Before(before) || first.After(time.Now().Add(couchmessage.AdmissionTimeout)) {
		t.Fatalf("invalid first deadline %v", first)
	}
	if fifth.Sub(first) < couchmessage.AdmissionTimeout/2 {
		t.Fatalf("queued task inherited first task's deadline: first=%v fifth=%v", first, fifth)
	}
}
