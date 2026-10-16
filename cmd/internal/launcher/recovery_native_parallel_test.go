package launcher

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func orderedProofWrappers(n int) []recoveryNativeWrapper {
	wrappers := make([]recoveryNativeWrapper, n)
	for i := range wrappers {
		wrappers[i].Process = procutil.Process{PID: i}
	}
	return wrappers
}

func awaitOrderedProof[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("proof runner did not reach expected event")
		var zero T
		return zero
	}
}

func TestOrderedNativeProofsBoundedAndOrdered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan int, 12)
	release := make([]chan struct{}, 12)
	for i := range release {
		release[i] = make(chan struct{})
	}
	var active, peak atomic.Int32
	done := make(chan error, 1)
	var consumed []string
	go func() {
		done <- runOrderedNativeProofs(ctx, orderedProofWrappers(12), func(ctx context.Context, w recoveryNativeWrapper) (recoveryNativeProof, error) {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			started <- w.Process.PID
			select {
			case <-release[w.Process.PID]:
				return recoveryNativeProof{NativeID: fmt.Sprint(w.Process.PID)}, nil
			case <-ctx.Done():
				return recoveryNativeProof{}, ctx.Err()
			}
		}, func(w recoveryNativeWrapper, p recoveryNativeProof) error {
			consumed = append(consumed, p.NativeID)
			return nil
		})
	}()
	initial := make(map[int]bool)
	for range 4 {
		initial[awaitOrderedProof(t, started)] = true
	}
	if !initial[0] {
		t.Fatal("first four jobs omitted first wrapper")
	}
	// Finishing later jobs while index zero is blocked must keep the pool moving.
	for i := range initial {
		if i != 0 {
			close(release[i])
		}
	}
	for range 8 {
		close(release[awaitOrderedProof(t, started)])
	}
	close(release[0])
	if err := awaitOrderedProof(t, done); err != nil {
		t.Fatal(err)
	}
	if peak.Load() != 4 {
		t.Fatalf("peak proofs=%d, want 4", peak.Load())
	}
	if active.Load() != 0 {
		t.Fatal("proofs survived runner")
	}
	want := make([]string, 12)
	for i := range want {
		want[i] = fmt.Sprint(i)
	}
	if !reflect.DeepEqual(consumed, want) {
		t.Fatalf("consume order=%v, want %v", consumed, want)
	}
}

func TestOrderedNativeProofsStableErrors(t *testing.T) {
	for _, consumeFailure := range []bool{false, true} {
		t.Run(fmt.Sprint("consume=", consumeFailure), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			laterDone := make(chan struct{})
			expected := errors.New("first wrapper error")
			later := errors.New("later wrapper error")
			var consumed []int
			err := runOrderedNativeProofs(ctx, orderedProofWrappers(3), func(ctx context.Context, w recoveryNativeWrapper) (recoveryNativeProof, error) {
				switch w.Process.PID {
				case 0:
					select {
					case <-laterDone:
					case <-ctx.Done():
						return recoveryNativeProof{}, ctx.Err()
					}
					return recoveryNativeProof{NativeID: "0"}, nil
				case 1:
					if !consumeFailure {
						return recoveryNativeProof{}, expected
					}
					return recoveryNativeProof{NativeID: "1"}, nil
				default:
					close(laterDone)
					return recoveryNativeProof{}, later
				}
			}, func(w recoveryNativeWrapper, p recoveryNativeProof) error {
				consumed = append(consumed, w.Process.PID)
				if consumeFailure && w.Process.PID == 1 {
					return expected
				}
				return nil
			})
			if err != expected {
				t.Fatalf("error=%v, want %v", err, expected)
			}
			want := []int{0}
			if consumeFailure {
				want = append(want, 1)
			}
			if !reflect.DeepEqual(consumed, want) {
				t.Fatalf("consumed=%v, want %v", consumed, want)
			}
		})
	}
}

func TestOrderedNativeProofsFailureCancelsAndJoins(t *testing.T) {
	for _, parentCancellation := range []bool{false, true} {
		t.Run(fmt.Sprint("parent=", parentCancellation), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan int, 4)
			exited := make(chan int, 4)
			allowExit := make(chan struct{})
			firstReady := make(chan struct{})
			var releaseOnce sync.Once
			releaseWorkers := func() { releaseOnce.Do(func() { close(allowExit) }) }
			defer releaseWorkers()
			done := make(chan error, 1)
			failure := errors.New("consume failed")
			go func() {
				done <- runOrderedNativeProofs(ctx, orderedProofWrappers(4), func(ctx context.Context, w recoveryNativeWrapper) (recoveryNativeProof, error) {
					started <- w.Process.PID
					if !parentCancellation && w.Process.PID == 0 {
						select {
						case <-firstReady:
						case <-ctx.Done():
							return recoveryNativeProof{}, ctx.Err()
						}
						return recoveryNativeProof{}, nil
					}
					<-ctx.Done()
					exited <- w.Process.PID
					<-allowExit
					return recoveryNativeProof{}, ctx.Err()
				}, func(recoveryNativeWrapper, recoveryNativeProof) error { return failure })
			}()
			count := 4
			for range 4 {
				awaitOrderedProof(t, started)
			}
			if parentCancellation {
				cancel()
			} else {
				count = 3
				close(firstReady)
			}
			for range count {
				awaitOrderedProof(t, exited)
			}
			select {
			case err := <-done:
				t.Fatalf("runner returned before workers joined: %v", err)
			default:
			}
			releaseWorkers()
			err := awaitOrderedProof(t, done)
			if parentCancellation {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
			} else if err != failure {
				t.Fatalf("error=%v, want consume failure", err)
			}
		})
	}
}

func TestOrderedNativeProofsEmptyAndCanceled(t *testing.T) {
	prove := func(context.Context, recoveryNativeWrapper) (recoveryNativeProof, error) {
		t.Error("unexpected proof")
		return recoveryNativeProof{}, nil
	}
	consume := func(recoveryNativeWrapper, recoveryNativeProof) error { t.Error("unexpected consume"); return nil }
	if err := runOrderedNativeProofs(context.Background(), nil, prove, consume); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runOrderedNativeProofs(ctx, orderedProofWrappers(3), prove, consume); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want cancellation", err)
	}
}
