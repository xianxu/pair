package couchtty

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// remoteRig is a console with an operator on actor "source", a dispatcher
// that records every call, and the outcome hooks a socket handler passes.
type remoteRig struct {
	c        *Console
	mu       sync.Mutex
	calls    []couchcore.OperationCall
	started  int
	finished []remoteOutcome
	// resume and attach answer the dispatcher's calls.
	resume func(couchcore.OperationCall) (any, error)
	attach func(couchcore.OperationCall) (any, error)
}

type remoteOutcome struct {
	value any
	err   error
}

func newRemoteRig(t *testing.T) *remoteRig {
	t.Helper()
	r := &remoteRig{c: New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)}
	t.Cleanup(r.c.Stop)
	source := menuAddress("source")
	r.c.attachThreadActor("source", "source", source, "/work", "source", ptychild.NewFakeChild(nil))
	r.c.menu = NewMenuState([]couchcore.ActionableThreadSummary{{Address: source, State: couchcore.ThreadLive}}, source)
	r.c.menuReady = true
	r.c.focus, r.c.active = FocusActor("source"), "source"
	r.attach = r.c.ExecuteConsoleOperation
	r.c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		r.mu.Lock()
		r.calls = append(r.calls, call)
		r.mu.Unlock()
		switch call.Name {
		case "attach":
			return r.attach(call)
		default:
			if r.resume == nil {
				return nil, errors.New("no " + call.Name + " fake")
			}
			return r.resume(call)
		}
	})
	return r
}

func (r *remoteRig) enqueue(key, op string, prepare func(context.Context) (couchcore.OperationCall, error)) error {
	return r.c.EnqueueRemoteOperation(key, op, prepare,
		func() { r.mu.Lock(); r.started++; r.mu.Unlock() },
		func(value any, err error) {
			r.mu.Lock()
			r.finished = append(r.finished, remoteOutcome{value, err})
			r.mu.Unlock()
		})
}

// drain runs one queued job the way operationQueue.Run does and hands its
// completion to finishOperation, as the console loop does.
func (r *remoteRig) drain(t *testing.T) {
	t.Helper()
	request := <-r.c.operationQueue.requests
	value, err := request.run()
	r.c.finishOperation(request.complete(value, err))
}

func prepared(call couchcore.OperationCall) func(context.Context) (couchcore.OperationCall, error) {
	return func(context.Context) (couchcore.OperationCall, error) { return call, nil }
}

func slotResumeCall() couchcore.OperationCall {
	return couchcore.OperationCall{Name: "resume", Args: map[string]string{"path": "/src/worktree/pair-slot1/pair"}, Implicit: true}
}

func TestRemoteResumeIsAdoptedWithoutTakingFocus(t *testing.T) {
	r := newRemoteRig(t)
	target := menuAddress("slot-one")
	start, _ := attachStartResult(t, "slot-one-actor", target)
	r.resume = func(call couchcore.OperationCall) (any, error) {
		if r.started != 1 {
			t.Errorf("dispatch before started: %d", r.started)
		}
		return start, nil
	}
	if err := r.enqueue("remote\x00pair:1", "resume", prepared(slotResumeCall())); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if len(r.calls) != 2 || r.calls[0].Name != "resume" || r.calls[0].Args["path"] != "/src/worktree/pair-slot1/pair" || r.calls[0].Context == nil || !r.calls[0].Implicit {
		t.Fatalf("calls = %+v", r.calls)
	}
	if r.calls[1].Name != "attach" || r.calls[1].Args["background"] != "true" {
		t.Fatalf("attach = %+v, want background", r.calls[1])
	}
	if _, adopted := r.c.panes[start.Handle.ID()]; !adopted {
		t.Fatal("remote resume's child was not adopted")
	}
	if r.c.focus != FocusActor("source") {
		t.Fatalf("focus moved to %v", r.c.focus)
	}
	if len(r.finished) != 1 || r.finished[0].err != nil || r.finished[0].value == nil {
		t.Fatalf("finished = %+v", r.finished)
	}
}

func TestRemoteAttachFailureReachesFinished(t *testing.T) {
	r := newRemoteRig(t)
	start, _ := attachStartResult(t, "slot-one-actor", menuAddress("slot-one"))
	r.resume = func(couchcore.OperationCall) (any, error) { return start, nil }
	r.attach = func(couchcore.OperationCall) (any, error) { return nil, errors.New("attach refused") }
	if err := r.enqueue("k", "resume", prepared(slotResumeCall())); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if len(r.finished) != 1 || r.finished[0].err == nil || r.finished[0].err.Error() != "attach refused" {
		t.Fatalf("finished = %+v, want the attach failure", r.finished)
	}
}

func TestRemoteCompletionLeavesOperatorInFlight(t *testing.T) {
	r := newRemoteRig(t)
	start, _ := attachStartResult(t, "slot-one-actor", menuAddress("slot-one"))
	r.resume = func(couchcore.OperationCall) (any, error) { return start, nil }
	operator := MenuOperationOrigin{Operation: "park", Attempt: 3, Address: menuAddress("source")}
	r.c.menu.InFlight = operator
	if err := r.enqueue("k", "resume", prepared(slotResumeCall())); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if r.c.menu.InFlight != operator {
		t.Fatalf("in flight = %+v, want the operator's %+v", r.c.menu.InFlight, operator)
	}
}

func TestRemotePrepareErrorReachesFinishedWithoutDispatch(t *testing.T) {
	r := newRemoteRig(t)
	refusal := &couchcore.SlotOperationError{Code: couchcore.SlotOpNotOffered, Detail: "pair:1 is live"}
	if err := r.enqueue("k", "resume", func(context.Context) (couchcore.OperationCall, error) { return couchcore.OperationCall{}, refusal }); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if len(r.calls) != 0 {
		t.Fatalf("dispatched %+v after a refused prepare", r.calls)
	}
	if r.started != 1 || len(r.finished) != 1 || !errors.Is(r.finished[0].err, refusal) {
		t.Fatalf("started %d finished %+v", r.started, r.finished)
	}
}

// A prepared call for another operation than the one admitted is refused:
// the receipt names the operation the caller asked for.
func TestRemotePreparedCallMustMatchTheAdmittedOperation(t *testing.T) {
	r := newRemoteRig(t)
	reboot := slotResumeCall()
	reboot.Name = "reboot"
	if err := r.enqueue("k", "resume", prepared(reboot)); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if len(r.calls) != 0 || len(r.finished) != 1 || r.finished[0].err == nil {
		t.Fatalf("calls %+v finished %+v", r.calls, r.finished)
	}
}

// Single outcome owner: an enqueue that returns an error has not called, and
// never will call, started or finished.
func TestRemoteEnqueueRefusalsNeverReport(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		r := newRemoteRig(t)
		if err := r.enqueue("same", "resume", prepared(slotResumeCall())); err != nil {
			t.Fatal(err)
		}
		if err := r.enqueue("same", "resume", prepared(slotResumeCall())); !errors.Is(err, ErrRemotePending) {
			t.Fatalf("duplicate key: %v", err)
		}
		r.drain(t)
		if r.started != 1 || len(r.finished) != 1 {
			t.Fatalf("started %d finished %d, want only the first job reported", r.started, len(r.finished))
		}
	})
	t.Run("overloaded", func(t *testing.T) {
		r := newRemoteRig(t)
		r.c.operationQueue = newOperationQueue(1)
		if accepted, err := r.c.operationQueue.Enqueue(operationRequest{key: "occupied", run: func() (any, error) { return nil, nil }}); !accepted || err != nil {
			t.Fatal("cannot fill the queue")
		}
		if err := r.enqueue("k", "resume", prepared(slotResumeCall())); !errors.Is(err, ErrOperationQueueOverloaded) {
			t.Fatalf("full queue: %v", err)
		}
		r.drain(t) // the occupying job
		if r.started != 0 || len(r.finished) != 0 {
			t.Fatalf("started %d finished %+v after a refused enqueue", r.started, r.finished)
		}
	})
	t.Run("no dispatcher", func(t *testing.T) {
		r := newRemoteRig(t)
		r.c.SetOperationDispatcher(nil)
		if err := r.enqueue("k", "resume", prepared(slotResumeCall())); err == nil {
			t.Fatal("enqueued with no dispatcher")
		}
		if len(r.c.operationQueue.requests) != 0 || r.started != 0 || len(r.finished) != 0 {
			t.Fatal("a refused enqueue queued or reported")
		}
	})
}

// A remote resume clears the row's reattach-failure mark, as the switcher's
// resume does, whether the row is named by tag or by slot path, and whether
// or not the resume succeeds.
func TestRemoteResumeClearsTheReattachMark(t *testing.T) {
	slot := menuSlotRow(1, "couch-slot")
	slot.State, slot.Reason = couchcore.ThreadDetached, ""
	primary := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadDetached}
	for name, c := range map[string]struct {
		row  couchcore.ActionableThreadSummary
		args map[string]string
		fail bool
	}{
		"slot path":     {slot, map[string]string{"path": slot.Target.Slot.WorktreeRoot}, false},
		"primary tag":   {primary, map[string]string{"repo-scope": "scope", "tag": "couch-primary", "warm-only": "true"}, false},
		"failed resume": {primary, map[string]string{"repo-scope": "scope", "tag": "couch-primary", "warm-only": "true"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRemoteRig(t)
			r.c.menu = NewMenuState([]couchcore.ActionableThreadSummary{r.c.menu.Inventory[0], c.row}, r.c.menu.Inventory[0].Address)
			r.c.menu.Reattach.Failed = map[couchcore.ThreadAddress]string{c.row.Address: "failed"}
			r.resume = func(couchcore.OperationCall) (any, error) { return nil, errors.New("not detached") }
			if !c.fail {
				start, _ := attachStartResult(t, "resumed", c.row.Address)
				r.resume = func(couchcore.OperationCall) (any, error) { return start, nil }
			}
			if err := r.enqueue("k", "resume", prepared(couchcore.OperationCall{Name: "resume", Args: c.args, Implicit: true})); err != nil {
				t.Fatal(err)
			}
			r.drain(t)
			if _, marked := r.c.menu.Reattach.Failed[c.row.Address]; marked {
				t.Fatalf("reattach mark kept: %v", r.c.menu.Reattach.Failed)
			}
		})
	}
}

// remoteResumeCompletion recognizes a remote resume with no origin field of
// its own: an Attempt-0 resume with no continuation ID. A continuation
// replacement (ContinuationID set) and the operator's resume (Attempt > 0)
// never match; a reboot is not a resume.
func TestRemoteResumeRecognitionIsPinnedBothSides(t *testing.T) {
	for _, c := range []struct {
		origin MenuOperationOrigin
		want   bool
	}{
		{MenuOperationOrigin{Operation: "resume", PreserveFocus: true}, true},
		{MenuOperationOrigin{Operation: "resume", PreserveFocus: true, ContinuationID: "request"}, false},
		{MenuOperationOrigin{Operation: "resume", Attempt: 4}, false},
		{MenuOperationOrigin{Operation: "resume", Attempt: 4, Background: true}, false},
		{MenuOperationOrigin{Operation: "reboot", PreserveFocus: true}, false},
	} {
		if got := remoteResumeCompletion(c.origin); got != c.want {
			t.Errorf("%+v: %v, want %v", c.origin, got, c.want)
		}
	}
	// The producer side: the origin a remote enqueue builds is the one the
	// predicate recognizes, and the continuation controller's is not.
	r := newRemoteRig(t)
	if err := r.enqueue("k", "resume", prepared(slotResumeCall())); err != nil {
		t.Fatal(err)
	}
	request := <-r.c.operationQueue.requests
	if !remoteResumeCompletion(request.origin) || !request.origin.PreserveFocus || request.origin.Attempt != 0 {
		t.Fatalf("remote origin %+v not recognized", request.origin)
	}
	c, status := continuationConsole(t)
	c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
	replacement := <-c.operationQueue.requests
	if remoteResumeCompletion(replacement.origin) {
		t.Fatalf("continuation replacement %+v recognized as remote", replacement.origin)
	}
}

// A remote job's failure carries its own step timings (queue wait, prepare,
// operation), so a slow step before the launch cannot hide (pair#367 smoke
// test), and the typed error still unwraps.
func TestRemoteFailureCarriesItsStepTimings(t *testing.T) {
	r := newRemoteRig(t)
	refusal := &couchcore.ResumeRefusal{Code: couchcore.ResumeNotDetached, Diagnostic: "registration timed out"}
	r.resume = func(couchcore.OperationCall) (any, error) { return nil, refusal }
	if err := r.enqueue("k", "resume", prepared(slotResumeCall())); err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if len(r.finished) != 1 || r.finished[0].err == nil {
		t.Fatalf("finished = %+v", r.finished)
	}
	err := r.finished[0].err
	if !regexp.MustCompile(`\[remote job: queued [0-9.]+m?s, prepare [0-9.]+m?s, operation [0-9.]+m?s\]`).MatchString(err.Error()) {
		t.Fatalf("error %q lacks the remote job's step timings", err)
	}
	if couchcore.ResumeDiagnosticOf(err) != couchcore.ResumeNotDetached {
		t.Fatalf("the timings hid the typed refusal: %v", err)
	}
}
