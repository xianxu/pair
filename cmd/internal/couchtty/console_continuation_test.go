package couchtty

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

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
				c.finishPreparedOperationForTest(completion)
			}
			if c.onExit(childExit{id: "source", code: 1}) {
				t.Fatal("continuation failure closed the last-actor recovery panel")
			}
			if !completionFirst {
				c.finishPreparedOperationForTest(completion)
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
	c.continuations[unreadable] = continuationWatch{status: couchcore.ContinuationStatus{Address: unreadable, RequestID: "retained", Phase: checkpoint.Running}}
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
	c.continuations[status.Address] = continuationWatch{status: status, queued: true}
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
	c.continuations[status.Address] = continuationWatch{status: status, queued: true}
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
			c.finishPreparedOperationForTest(operationCompletion{name: request.name, origin: request.origin,
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
			if c.expectedExits.has(started.Handle.ID()) {
				t.Fatal("new target death would be mistaken for expected source retirement")
			}
		})
	}
}

// A request that didn't finish ends its watch and says its one line, once
// (#522): there is nothing left to drive or retry.
func TestAMissedContinuationEndsItsWatchWithOneLine(t *testing.T) {
	for _, routed := range []bool{false, true} {
		c, status := continuationConsole(t)
		origin := MenuOperationOrigin{Address: status.Address, ContinuationID: status.RequestID}
		name := "continue-thread"
		if routed {
			origin.ContinuationID, name = "", "resume"
		}
		c.continuations[status.Address] = continuationWatch{status: status, queued: !routed}
		missed := &checkpoint.Miss{RequestID: status.RequestID, Path: "/repo/c.md", Reason: "fork unavailable", At: time.Unix(1, 0)}
		c.finishContinuationOperation(operationCompletion{name: name, origin: origin, value: couchcore.ContinuationResult{Missed: missed}}, errors.New("fork unavailable"))
		if _, watched := c.continuations[status.Address]; watched {
			t.Fatalf("routed=%v: the watch outlived its request", routed)
		}
		if got := c.feed.Row().Body; !strings.Contains(got, "compaction didn't finish: fork unavailable; continuation at /repo/c.md") {
			t.Fatalf("routed=%v: notice %q", routed, got)
		}
		c.acceptContinuationRequests(continuationScanResult{addresses: []couchcore.ThreadAddress{status.Address}})
		if len(c.operationQueue.requests) != 0 {
			t.Fatalf("routed=%v: a missed request was driven again", routed)
		}
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
	if !retained || watch.queued || c.expectedExits.has("source") || c.focus != FocusActor("source") {
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
	c.continuations[status.Address] = continuationWatch{status: status}
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
			c.finishPreparedOperationForTest(operationCompletion{name: request.name, origin: request.origin,
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
	case <-testwait.After(t):
		t.Fatal("Console did not join continuation worker")
	}
}

// Resume routes a retained request to RecoverThread or Continue
// (pair#363), and their ContinuationResult must start the console's watch the
// way the old recover-thread completion did.
func TestRoutedResumeResultStartsWatchOnlyForPublishedRequest(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(map[bool]string{false: "warm", true: "checkpoint"}[published], func(t *testing.T) {
			c, status := continuationConsole(t)
			result := couchcore.ContinuationResult{}
			if published {
				status.Phase = checkpoint.Running
				result.Status = status
				result.SourceReattached = true
			}
			c.finishContinuationOperation(operationCompletion{name: "resume", origin: MenuOperationOrigin{Address: status.Address}, value: result}, nil)
			watch, found := c.continuations[status.Address]
			if found != published {
				t.Fatalf("watch exists = %v, want %v", found, published)
			}
			if published && watch.status.RequestID != status.RequestID {
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
	c.finishContinuationOperation(operationCompletion{name: "resume", origin: MenuOperationOrigin{Address: status.Address}, value: couchcore.ContinuationResult{Status: status}}, nil)
	if got := c.continuations[status.Address]; got.status.RequestID != newer.RequestID || !got.queued {
		t.Fatalf("obsolete recovery replaced accepted request: %+v", got)
	}
}

// A continue-thread refused because another operation held the thread
// (pair#205) never ran: it is re-armed for the next scan, its expected-exit
// marks are dropped, and nothing is announced -- the retry is silent.
func TestABusyContinuationIsRetriedSilently(t *testing.T) {
	c, status := continuationConsole(t)
	c.attachThreadActor("source", "source", status.Address, "/repo", "source", ptychild.NewFakeChild(nil))
	status.Phase = checkpoint.Running
	// An unrelated handle's mark (set by some other operation) must survive.
	c.expectedExits.mark("parked")
	c.mu.Lock()
	marked := c.markThreadExitsLocked(status.Address)
	c.mu.Unlock()
	if len(marked) != 1 || marked[0] != "source" {
		t.Fatalf("enqueue recorded marks %v, want its own mark on the thread's pane", marked)
	}
	c.continuations[status.Address] = continuationWatch{status: status, queued: true, marked: marked}
	before := c.feed.Row().Body

	c.finishContinuationOperation(operationCompletion{
		name:   "continue-thread",
		origin: MenuOperationOrigin{Address: status.Address, ContinuationID: status.RequestID},
	}, &couchcore.ThreadBusyError{Address: status.Address, Running: "relaunch"})

	if c.expectedExits.has("source") {
		t.Fatal("a refused continuation left its pane's exit marked as expected")
	}
	if !c.expectedExits.has("parked") {
		t.Fatal("a refused continuation cleared the mark a park set on the same thread")
	}
	if got := c.feed.Row().Body; got != before {
		t.Fatalf("a busy continuation announced itself: %q", got)
	}
	c.acceptContinuationRequests(continuationScanResult{statuses: []couchcore.ContinuationStatus{status}})
	request := <-c.operationQueue.requests
	if request.name != "continue-thread" {
		t.Fatalf("the next scan sent %q, want the refused continue-thread retried", request.name)
	}
}

// GoTracked runs work off the caller's goroutine while the console is live,
// and the console's shutdown joins it (pair#205: an abort waiting on a
// thread's gate must not block the console).
func TestGoTrackedDoesNotBlockTheCallerAndIsJoined(t *testing.T) {
	// A bare console: the join below waits on every tracked worker, so the
	// fixture must not carry pane watchers of its own.
	c := &Console{started: true}
	release, finished := make(chan struct{}), make(chan struct{})
	returned := make(chan struct{})
	go func() {
		c.GoTracked(func() { <-release; close(finished) })
		close(returned)
	}()
	select {
	case <-returned:
	case <-testwait.After(t):
		t.Fatal("GoTracked blocked its caller")
	}
	joined := make(chan struct{})
	go func() { c.workers.Wait(); close(joined) }()
	select {
	case <-joined:
		t.Fatal("the console's join did not wait for tracked work")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	<-finished
	<-joined
}

// Marks are owned (pair#205 M1 review): whichever order a park's mark and a
// continuation's mark land in, the continuation's refusal removes only its own.
func TestARefusalNeverRemovesAnotherOperationsExitMark(t *testing.T) {
	for _, parkFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "park marks first", false: "park re-marks after"}[parkFirst], func(t *testing.T) {
			c, status := continuationConsole(t)
			c.attachThreadActor("source", "source", status.Address, "/repo", "source", ptychild.NewFakeChild(nil))
			status.Phase = checkpoint.Running
			if parkFirst {
				c.expectedExits.mark("source")
			}
			c.mu.Lock()
			marked := c.markThreadExitsLocked(status.Address)
			c.mu.Unlock()
			if !parkFirst {
				c.expectedExits.mark("source")
			}
			c.continuations[status.Address] = continuationWatch{status: status, queued: true, marked: marked}
			c.finishContinuationOperation(operationCompletion{
				name:   "continue-thread",
				origin: MenuOperationOrigin{Address: status.Address, ContinuationID: status.RequestID},
			}, &couchcore.ThreadBusyError{Address: status.Address, Running: "park"})
			if !c.expectedExits.has("source") {
				t.Fatal("a refused continuation removed the park's expected-exit mark")
			}
			if !c.expectedExits.consume("source") || c.expectedExits.has("source") {
				t.Fatal("the pane's real exit must consume every remaining mark")
			}
		})
	}
}
