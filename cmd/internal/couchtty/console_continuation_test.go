package couchtty

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// failedContinuationCouch is a Couch on a real temp-dir store whose one live
// thread retains a FAILED continuation -- the shape the operator's pair thread
// was left in when typing interrupted automatic orientation (#280).
func failedContinuationCouch(t *testing.T) (*couchcore.Couch, couchcore.ThreadAddress, string) {
	t.Helper()
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewThreadStore(ns)
	address := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	cp, err := checkpoint.New("/repo/workshop/continuation/handoff.md", "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nContinue exact work.\n")
	if err != nil {
		t.Fatal(err)
	}
	request := checkpoint.Request{
		Version: checkpoint.Version, ID: checkpoint.RequestID(address.RepoScope, string(address.Tag), 2, cp.Digest), Checkpoint: cp,
		Source:    checkpoint.Source{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2, Helper: checkpoint.Process{PID: 41, Identity: "source-helper"}},
		CreatedAt: time.Unix(1, 0).UTC(), Phase: checkpoint.Failed, Attempt: "start-0102",
		Failure: "continuation delivery cancelled: operator input interrupted automatic orientation",
	}
	profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	if _, err := store.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion, Address: address,
		StartingPath: "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), LastActiveAt: time.Unix(1, 0).UTC(),
		Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "helper", State: couchcore.IncarnationLive, RepoIdentity: "/repo/.git", LaunchProfile: &profile}},
		LatestLaunchProfile: &profile, Revision: 1, Continuation: &request,
	}); err != nil {
		t.Fatal(err)
	}
	return &couchcore.Couch{Threads: store}, address, request.ID
}

func TestContinuationFailedRowKeepsAnExplicitRetry(t *testing.T) {
	address := menuAddress("failed")
	row := couchcore.ActionableThreadSummary{Address: address, State: couchcore.ThreadUnusable,
		Continuation: &couchcore.ContinuationStatus{Address: address, RequestID: "request", Phase: checkpoint.Failed}}
	if !slices.Contains(menuActionItems(row), "retry-continuation") || slices.Contains(menuActionItems(row), "archive") {
		t.Fatalf("failed continuation actions: %v", menuActionItems(row))
	}

	// The offered retry must reach RetryContinuation through the PRODUCTION
	// dispatcher and live-owner executor, not a fake: the switcher once sent
	// both `ref` and `tag`, which resolveOperationThread refuses outright, and a
	// fake executor could never see it (#280).
	c, live, requestID := failedContinuationCouch(t)
	liveRow := couchcore.ActionableThreadSummary{Address: live, State: couchcore.ThreadLive,
		Continuation: &couchcore.ContinuationStatus{Address: live, RequestID: requestID, Phase: checkpoint.Failed}}
	state := NewMenuState([]couchcore.ActionableThreadSummary{liveRow}, live)
	_, effects := dispatchThreadOperation(state, "retry-continuation", live)
	if len(effects) != 1 || effects[0].Args["request-id"] != requestID {
		t.Fatalf("retry lost selected request: %v", effects)
	}
	_, err := couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: couchcore.CouchLiveOwnerExecutor(c)},
		couchcore.OperationCall{Name: effects[0].Operation, Args: effects[0].Args, Implicit: true})
	// RetryContinuation's own first refusal on this Couch (no registration
	// observer is wired) proves the call got past thread resolution.
	if err == nil || err.Error() != "continuation target observer unavailable" {
		t.Fatalf("offered retry did not reach RetryContinuation: %v", err)
	}
}

func continuationConsole(t *testing.T) (*Console, couchcore.ContinuationStatus) {
	t.Helper()
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	address := menuAddress("continuation")
	c.attachThreadActor("source", "source", address, "/work", "source", ptychild.NewFakeChild(nil))
	c.menu = NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadLive}}, address)
	c.menuReady = true
	c.focus = FocusActor("source")
	c.active = "source"
	c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, errors.New("replacement failed") })
	t.Cleanup(c.Stop)
	return c, couchcore.ContinuationStatus{Address: address, RequestID: "request", Phase: checkpoint.Pending, Agent: "codex"}
}

func TestContinuationFailureKeepsLastActorPanelInEitherOrder(t *testing.T) {
	for _, completionFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "exit-first", true: "failure-first"}[completionFirst], func(t *testing.T) {
			c, status := continuationConsole(t)
			c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
			request := <-c.operationQueue.requests
			completion := operationCompletion{key: request.key, name: request.name, origin: request.origin, err: errors.New("replacement failed")}
			if completionFirst {
				c.finishOperation(completion)
			}
			if c.onExit(childExit{id: "source", code: 1}) {
				t.Fatal("continuation failure closed the last-actor recovery panel")
			}
			if !completionFirst {
				c.finishOperation(completion)
			}
			if !c.focus.IsPanel() {
				t.Fatal("failed replacement must leave the panel available")
			}
			addresses := c.continuationAddresses()
			if len(addresses) != 1 || addresses[0] != status.Address {
				t.Fatalf("lost request after source removal: %v", addresses)
			}
		})
	}
}

func TestContinuationPollingDeduplicatesUntilCompletionIsConsumed(t *testing.T) {
	c, status := continuationConsole(t)
	result := continuationScanResult{statuses: []couchcore.ContinuationStatus{status}}
	c.acceptContinuationRequests(result)
	request := <-c.operationQueue.requests
	// The queue releases its own key before Run consumes its completion. The
	// continuation controller must still treat the operation as outstanding.
	c.operationQueue.mu.Lock()
	delete(c.operationQueue.pending, request.key)
	c.operationQueue.mu.Unlock()
	c.acceptContinuationRequests(result)
	if len(c.operationQueue.requests) != 0 {
		t.Fatal("repeated scan queued a second replacement")
	}
}

func TestContinuationPartialScanKeepsHealthyAndUnseenRequests(t *testing.T) {
	c, status := continuationConsole(t)
	unreadable := menuAddress("unreadable")
	c.continuations[unreadable] = continuationWatch{status: couchcore.ContinuationStatus{Address: unreadable, RequestID: "retained", Phase: checkpoint.Failed}}
	c.acceptContinuationRequests(continuationScanResult{
		addresses: []couchcore.ThreadAddress{status.Address, unreadable},
		statuses:  []couchcore.ContinuationStatus{status}, err: errors.New("one slot unreadable"),
	})
	if len(c.operationQueue.requests) != 1 {
		t.Fatal("one unreadable slot blocked a healthy continuation")
	}
	if _, exists := c.continuations[unreadable]; !exists {
		t.Fatal("failed scan forgot the request whose source pane is gone")
	}
}

func TestContinuationSourceReattachmentResumesExecution(t *testing.T) {
	c, status := continuationConsole(t)
	c.continuations[status.Address] = continuationWatch{status: status, queued: true, handled: true}
	status.Phase = checkpoint.Running
	c.finishContinuationOperation(operationCompletion{
		origin: MenuOperationOrigin{Address: status.Address, ContinuationID: status.RequestID},
		value:  couchcore.ContinuationResult{Status: status, SourceReattached: true},
	}, nil)
	c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
	request := <-c.operationQueue.requests
	if request.name != "continue-thread" {
		t.Fatalf("reattached source only observes a nonexistent target: %s", request.name)
	}
}

func TestContinuationErrorResultPreservesRequestIdentity(t *testing.T) {
	c, status := continuationConsole(t)
	c.continuations[status.Address] = continuationWatch{status: status, queued: true, handled: true}
	c.finishContinuationOperation(operationCompletion{
		origin: MenuOperationOrigin{Address: status.Address, ContinuationID: status.RequestID},
		value:  couchcore.ContinuationResult{},
	}, errors.New("read failed"))
	if c.continuations[status.Address].status.RequestID != status.RequestID {
		t.Fatal("empty error result discarded the durable request identity")
	}
}

func TestContinuationAdoptsReplacementAndPreservesUnrelatedFocus(t *testing.T) {
	for _, background := range []bool{false, true} {
		t.Run(map[bool]string{false: "focused", true: "background"}[background], func(t *testing.T) {
			c, status := continuationConsole(t)
			if background {
				c.attachThreadActor("other", "other", menuAddress("other"), "/other", "other", ptychild.NewFakeChild(nil))
				c.focus, c.active = FocusActor("other"), "other"
			}
			started, _ := attachStartResult(t, "replacement", status.Address)
			c.SetOperationDispatcher(c.ExecuteConsoleOperation)
			c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
			request := <-c.operationQueue.requests
			if c.onExit(childExit{id: "source", code: 0}) {
				t.Fatal("source retirement closed Console before replacement")
			}
			status.Phase = checkpoint.Running
			c.finishOperation(operationCompletion{name: request.name, origin: request.origin,
				value: couchcore.ContinuationResult{Status: status, Record: started.Record, Handle: started.Handle},
			})
			if _, attached := c.panes[started.Handle.ID()]; !attached {
				t.Fatal("registered continuation helper was not adopted")
			}
			wantFocus := FocusActor(started.Handle.ID())
			if background {
				wantFocus = FocusActor("other")
			}
			if c.focus != wantFocus {
				t.Fatalf("focus=%v want=%v", c.focus, wantFocus)
			}
			if c.expectedExits[started.Handle.ID()] {
				t.Fatal("new target death would be mistaken for expected source retirement")
			}
		})
	}
}

func TestContinuationFailedRequestIsNotRetriedAutomatically(t *testing.T) {
	c, status := continuationConsole(t)
	status.Phase = checkpoint.Failed
	for range 3 {
		c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
	}
	if len(c.operationQueue.requests) != 0 {
		t.Fatal("failed request was retried without an operator action")
	}
}

func TestContinuationFullQueueRetainsRequestWithoutRetiringSource(t *testing.T) {
	c, status := continuationConsole(t)
	c.operationQueue = newOperationQueue(1)
	if accepted, err := c.operationQueue.Enqueue(operationRequest{key: "occupied"}); !accepted || err != nil {
		t.Fatal("cannot prepare full queue")
	}
	scan := continuationScanResult{statuses: []couchcore.ContinuationStatus{status}}
	c.acceptContinuationRequests(scan)
	watch, retained := c.continuations[status.Address]
	if !retained || watch.queued || watch.handled || c.expectedExits["source"] || c.focus != FocusActor("source") {
		t.Fatal("queue overload lost request or prematurely retired the source")
	}
	<-c.operationQueue.requests
	c.acceptContinuationRequests(scan)
	if len(c.operationQueue.requests) != 1 || !c.continuations[status.Address].queued {
		t.Fatal("request was not admitted after queue capacity became available")
	}
}

func TestContinuationPendingRequestRetriesAdmissionAfterConflict(t *testing.T) {
	c, status := continuationConsole(t)
	c.continuations[status.Address] = continuationWatch{status: status, handled: true}
	// A revision conflict before Begin leaves the durable request pending:
	// there is no target receipt to observe yet.
	c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
	request := <-c.operationQueue.requests
	if request.name != "continue-thread" {
		t.Fatalf("pending admission conflict stalled in %s", request.name)
	}
}

func TestContinuationCompletionPreservesInterveningFocus(t *testing.T) {
	for _, scenario := range []string{"other-actor", "back-to-panel", "during-attach", "switch-before-source-exit"} {
		t.Run(scenario, func(t *testing.T) {
			c, status := continuationConsole(t)
			c.attachThreadActor("other", "other", menuAddress("other"), "/other", "other", ptychild.NewFakeChild(nil))
			c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
			request := <-c.operationQueue.requests
			if scenario != "switch-before-source-exit" {
				c.onExit(childExit{id: "source", code: 0})
			}
			if scenario != "during-attach" {
				c.switchTo("other", true, arrivalOrdinary)
			}
			if scenario == "switch-before-source-exit" {
				c.onExit(childExit{id: "source", code: 0})
			}
			want := FocusActor("other")
			if scenario == "back-to-panel" {
				c.onHotkey()
				want = FocusPanel()
			}
			c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				if scenario == "during-attach" && call.Name == "attach" {
					c.switchTo("other", true, arrivalOrdinary)
				}
				return couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: c.ExecuteConsoleOperation}, call)
			})
			started, _ := attachStartResult(t, "replacement", status.Address)
			status.Phase = checkpoint.Running
			c.finishOperation(operationCompletion{name: request.name, origin: request.origin,
				value: couchcore.ContinuationResult{Status: status, Record: started.Record, Handle: started.Handle},
			})
			if _, attached := c.panes[started.Handle.ID()]; !attached {
				t.Fatal("replacement was not adopted")
			}
			if c.focus != want {
				t.Fatalf("completion overrode newer choice: focus=%v want=%v", c.focus, want)
			}
		})
	}
}

func TestContinuationWorkerPicksUpQuietActorAndJoinsOnStop(t *testing.T) {
	c, status := continuationConsole(t)
	reader, writer := io.Pipe()
	c.stdin = reader
	t.Cleanup(func() { _ = writer.Close(); _ = reader.Close() })
	called := make(chan struct{}, 1)
	c.SetContinuationProvider(func(ctx context.Context, addresses []couchcore.ThreadAddress) ([]couchcore.ContinuationStatus, error) {
		select {
		case called <- struct{}{}:
		default:
		}
		return []couchcore.ContinuationStatus{status}, nil
	})
	dispatched := make(chan string, 1)
	c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		dispatched <- call.Name
		return nil, errors.New("replacement failed")
	})
	done := make(chan int, 1)
	go func() { done <- c.Run() }()
	waitFor(t, "quiet actor continuation pickup", func() bool { return len(dispatched) > 0 })
	if name := <-dispatched; name != "continue-thread" {
		t.Fatalf("operation = %q", name)
	}
	c.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Console did not join continuation worker")
	}
}

func TestRecoveryResultStartsWatchOnlyForPublishedRequest(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(map[bool]string{false: "warm", true: "checkpoint"}[published], func(t *testing.T) {
			c, status := continuationConsole(t)
			result := couchcore.ContinuationResult{}
			if published {
				status.Phase = checkpoint.Running
				result.Status = status
				result.SourceReattached = true
			}
			c.finishContinuationOperation(operationCompletion{name: "recover-thread", origin: MenuOperationOrigin{Address: status.Address}, value: result}, nil)
			watch, found := c.continuations[status.Address]
			if found != published {
				t.Fatalf("watch exists = %v, want %v", found, published)
			}
			if published && (watch.status.RequestID != status.RequestID || watch.handled) {
				t.Fatalf("lost request or followup execution: %+v", watch)
			}
			if _, bad := c.continuations[couchcore.ThreadAddress{}]; bad {
				t.Fatal("warm attachment created empty request watch")
			}
		})
	}
}

func TestRecoveryCompletionDoesNotReplaceNewerWatchedRequest(t *testing.T) {
	c, status := continuationConsole(t)
	newer := status
	newer.RequestID = "newer"
	c.continuations[status.Address] = continuationWatch{status: newer, queued: true}
	c.finishContinuationOperation(operationCompletion{name: "recover-checkpoint", origin: MenuOperationOrigin{Address: status.Address}, value: couchcore.ContinuationResult{Status: status}}, nil)
	if got := c.continuations[status.Address]; got.status.RequestID != newer.RequestID || !got.queued {
		t.Fatalf("obsolete recovery replaced accepted request: %+v", got)
	}
}
