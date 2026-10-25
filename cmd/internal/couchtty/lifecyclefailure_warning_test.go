package couchtty

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestLifecycleLocalQueuedCompletionWarningOwnsExactOrigin(t *testing.T) {
	for _, name := range []string{"cleanup", "diagnostic-persistence", "success"} {
		t.Run(name, func(t *testing.T) {
			r := newRemoteRig(t)
			target := menuAddress("local-original-target")
			store := couchcore.NewFakeLaunchFailureStore()
			if name == "diagnostic-persistence" {
				store.SaveError = errors.New("diagnostic store unavailable")
			}
			recorder := couchcore.NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "local-terminal-failure", nil })
			var attempt *couchcore.OperationAttempt
			r.c.SetOperationAttemptFactory(func(call couchcore.OperationCall) (*couchcore.OperationAttempt, error) {
				if call.FailureTarget == nil || call.FailureTarget.Address != target {
					t.Errorf("queue did not freeze selected origin: %+v", call.FailureTarget)
				}
				var err error
				attempt, err = recorder.Begin(call.Name, couchcore.LifecycleFailureTarget{Address: target})
				return attempt, err
			})
			r.resume = func(call couchcore.OperationCall) (any, error) {
				if couchcore.OperationAttemptFromContext(call.Context) != attempt {
					t.Error("local dispatch lost its queue-owned terminal attempt")
				}
				switch name {
				case "cleanup":
					return couchcore.ParkResult{CleanupError: errors.New("sidecar cleanup unavailable")}, nil
				case "diagnostic-persistence":
					return nil, errors.New("park failed")
				default:
					return couchcore.ParkResult{}, nil
				}
			}
			r.c.menu = NewMenuState([]couchcore.ActionableThreadSummary{{Address: target, State: couchcore.ThreadLive}}, target)
			origin := MenuOperationOrigin{Operation: "park", Attempt: 77, Address: target}
			setTestMenuOperation(&r.c.menu, origin)
			stop, joined := make(chan struct{}), make(chan struct{})
			go func() { defer close(joined); r.c.operationQueue.Run(stop) }()
			defer func() {
				close(stop)
				select {
				case <-joined:
				case <-testwait.After(t):
					t.Error("local queue worker did not join")
				}
			}()
			r.c.runMenuOperation(MenuEffect{Operation: "park", Attempt: 77})
			var completed operationCompletion
			select {
			case completed = <-r.c.operationQueue.results:
			case <-testwait.After(t):
				t.Fatal("local queued operation never published completion")
			}
			// Re-observing an already-published completion cannot create another
			// obligation, even though the recorder retains its terminal warning.
			r.c.publishOperationCompletion(completed)
			r.c.finishOperation(completed)
			items := warningSourceItems(r.c)
			if name == "success" {
				if len(items) != 0 || len(store.Writes) != 0 {
					t.Fatalf("ordinary success raised attention or diagnostics: %+v writes=%d", items, len(store.Writes))
				}
				return
			}
			if len(items) != 1 || items[0].Thread != target || len(store.Writes) != 1 {
				t.Fatalf("local warning lost its exact origin or terminal ownership: %+v writes=%d", items, len(store.Writes))
			}
			want := "unproven cleanup"
			if name == "diagnostic-persistence" {
				want = "diagnostics could not be saved"
			}
			if !strings.Contains(items[0].Body, want) {
				t.Fatalf("local warning lost diagnostic cause %q: %+v", want, items)
			}
		})
	}
}
