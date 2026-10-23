package couchtty

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestMenuBackgroundOperationAllowsOtherSlot(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, first := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(first) != 1 {
		t.Fatal("first operation not admitted")
	}
	_, second := dispatchThreadOperation(state, "switch", menuAddress("couch-two"))
	if len(second) != 1 || second[0].Operation != "switch" {
		t.Fatalf("running relaunch blocked another slot: %+v", second)
	}
}

func TestMenuBusyOperationExplainsSameSlotRefusal(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	next, effects := dispatchThreadOperation(state, "detach", menuAddress("couch-one"))
	if len(effects) != 0 || !strings.Contains(next.Notice.Text, "busy: relaunch") {
		t.Fatalf("same-slot refusal must name the owner: effects=%+v notice=%+v", effects, next.Notice)
	}
}

func TestMenuBackgroundOperationReturnsToRootImmediately(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Thread: menuAddress("couch-one"), Action: "relaunch", SelectedItem: "confirm"})
	next, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(effects) != 1 || next.CurrentFrame().Kind != MenuFrameRoot {
		t.Fatalf("admitted lifecycle action must return to usable root: frame=%+v effects=%+v", next.CurrentFrame(), effects)
	}
}

func TestMenuBackgroundOperationCompletionKeepsNewerActionFrame(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	before := state.CurrentFrame()
	if before.Kind != MenuFrameActions || before.Thread != menuAddress("couch-two") {
		t.Fatalf("bad fixture: %+v", before)
	}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "relaunch", Attempt: effects[0].Attempt, Address: menuAddress("couch-one"), Success: true})
	if state.CurrentFrame().Instance != before.Instance {
		t.Fatalf("late completion stole a newer navigation frame: before=%+v after=%+v", before, state.CurrentFrame())
	}
}

func TestMenuConcurrentOperationsKeepImmutableProgressAndIndependentResults(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			at := time.Unix(100, 0)
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			state.ObservedAt = at
			state, a := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
			beforeB := cloneMenuState(state)
			state, b := dispatchThreadOperation(state, "resume", menuAddress("couch-two"))
			if len(a) != 1 || len(b) != 1 || len(beforeB.Operations) != 1 || len(state.Operations) != 2 {
				t.Fatal("independent jobs did not retain separate owners")
			}
			beforeProgress := cloneMenuState(state)
			state = updateOperationProgress(state, a[0].Attempt, "restarting child", state.ObservedAt)

			keyA, _, _ := menuOperationForAttempt(state, a[0].Attempt)
			if beforeProgress.Operations[keyA].Detail != "" {
				t.Fatal("progress mutated previous job")
			}
			events := []MenuEvent{{Operation: "relaunch", Attempt: a[0].Attempt, Address: menuAddress("couch-one"), Error: "A failed"}, {Operation: "resume", Attempt: b[0].Attempt, Address: menuAddress("couch-two"), Success: true}}
			if reverse {
				events[0], events[1] = events[1], events[0]
			}
			for _, event := range events {
				previous := state
				before := cloneMenuState(state)
				state = reduceOperationResult(state, event)
				if !reflect.DeepEqual(previous, before) {
					t.Fatal("completion mutated preceding snapshot")
				}
			}
			jobA, ok := menuOperationForRow(state, menuThreads()[0])
			if !ok || jobA.Phase != MenuOperationFailed || jobA.Detail != "A failed" {
				t.Fatalf("A outcome lost: %+v", jobA)
			}
			jobB, ok := menuOperationForRow(state, menuThreads()[1])
			if !ok || jobB.Phase != MenuOperationSucceeded {
				t.Fatalf("B outcome lost: %+v", jobB)
			}
		})
	}
}

func TestMenuOperationDetailStorageIsBoundedUTF8AndRowSafe(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	text := strings.Repeat("界", menuTextLimit) + "\xff\x1b[2J\nforged"
	state = updateOperationProgress(state, effects[0].Attempt, text, state.ObservedAt)
	_, job, _ := menuOperationForAttempt(state, effects[0].Attempt)
	if len(job.Detail) > menuTextLimit || !utf8.ValidString(job.Detail) {
		t.Fatalf("progress detail bytes=%d valid=%v", len(job.Detail), utf8.ValidString(job.Detail))
	}
	state = reduceOperationResult(state, MenuEvent{Operation: "relaunch", Attempt: effects[0].Attempt, Address: menuAddress("couch-one"), Error: text})
	job, _ = menuOperationForRow(state, menuThreads()[0])
	if len(job.Detail) > menuTextLimit || !utf8.ValidString(job.Detail) {
		t.Fatalf("result detail bytes=%d valid=%v", len(job.Detail), utf8.ValidString(job.Detail))
	}
	row := menuOperationRowText(job, time.Unix(100, 0))
	if strings.ContainsAny(row, "\n\x1b") {
		t.Fatalf("unsafe row text: %q", row)
	}
}

func TestMenuOperationPruneRetainsOneLatestOutcomePerListedRow(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Operations = make(map[menuOperationTarget]MenuOperation)
	for i := 1; i <= 20; i++ {
		state.Operations[menuOperationTarget{StartPath: fmt.Sprint("/created-", i)}] = MenuOperation{Origin: MenuOperationOrigin{Operation: "start", Attempt: uint64(i)}, Phase: MenuOperationSucceeded, ResultAddress: menuAddress("couch-one"), Detail: "created"}
	}
	before := cloneMenuState(state)
	pruned := pruneMenuOperations(state)
	if len(pruned.Operations) != 1 || len(before.Operations) != 20 {
		t.Fatalf("outcome retention: %d, previous %d", len(pruned.Operations), len(before.Operations))
	}
	job, ok := menuOperationForRow(pruned, menuThreads()[0])
	if !ok || job.Origin.Attempt != 20 {
		t.Fatalf("latest row outcome=%+v", job)
	}
}

func TestMenuOperationElapsedAndResultsRenderAtMinimumGeometry(t *testing.T) {
	at := time.Unix(100, 0)
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.ObservedAt = at
	state, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	state = updateOperationProgress(state, effects[0].Attempt, "step A\ncontrol\x1b[2J", state.ObservedAt)
	_, job, _ := menuOperationForAttempt(state, effects[0].Attempt)
	if text := menuOperationRowText(job, at.Add(7*time.Second)); !strings.Contains(text, "7s") || !strings.Contains(text, "step A") || strings.ContainsAny(text, "\n\x1b") {
		t.Fatalf("elapsed row=%q", text)
	}
	before := cloneMenuState(state)
	for _, geometry := range [][2]int{{menuMinWidth, menuMinHeight}, {80, 24}} {
		assertRenderedBounds(t, RenderMenu(state, geometry[0], geometry[1], at.Add(7*time.Second), false), geometry[0], geometry[1])
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("render mutated running state")
	}
	state = reduceOperationResult(state, MenuEvent{Operation: "relaunch", Attempt: effects[0].Attempt, Address: menuAddress("couch-one"), Error: "retained failure"})
	state = pruneMenuOperations(state)
	job, _ = menuOperationForRow(state, menuThreads()[0])
	if !strings.Contains(menuOperationRowText(job, at), "retained failure") {
		t.Fatal("failed row result vanished")
	}
}

func TestMenuOperationPruneKeepsRunningMissingTargetUntilFinalResult(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	state.Inventory = nil
	state = pruneMenuOperations(state)
	if _, _, running := menuOperationForAttempt(state, effects[0].Attempt); !running {
		t.Fatal("inventory removal released active owner")
	}
	state = reduceOperationResult(state, MenuEvent{Operation: "relaunch", Attempt: effects[0].Attempt, Address: menuAddress("couch-one"), Error: "could not attach"})
	state = pruneMenuOperations(state)
	if len(state.Operations) != 0 || !strings.Contains(state.Notice.Text, "compiler") || !strings.Contains(state.Notice.Text, "scope/couch-one") || !strings.Contains(state.Notice.Text, "could not attach") {
		t.Fatalf("missing row result=%+v", state)
	}
}

func TestMenuOperationCompletionFencesReplacedRootAndCanRearmBusyRow(t *testing.T) {
	for _, operation := range []string{"switch", "start", "resume", "relaunch"} {
		t.Run(operation, func(t *testing.T) {
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			address := menuAddress("couch-one")
			if operation == "start" {
				address = couchcore.ThreadAddress{}
				state.Frames[0].PreviewResolution.Worktree = "/repo/new"
			}
			state, effects := dispatchMenuOperation(state, MenuEffect{Operation: operation}, address)
			if len(effects) != 1 {
				t.Fatal("operation not admitted")
			}
			state.FrameSequence++
			state.Frames[0].Instance = state.FrameSequence
			before := cloneMenuState(state)
			resultAddress := menuAddress("couch-one")
			if operation == "start" {
				resultAddress = menuAddress("new")
			}
			after := reduceOperationResult(state, MenuEvent{Operation: operation, Attempt: effects[0].Attempt, Address: resultAddress, Success: true})
			if after.LandingAttempt != 0 || !reflect.DeepEqual(after.Frames, before.Frames) {
				t.Fatalf("replaced root was authorized: %+v", after)
			}
			if operation == "start" {
				return
			}
			state, selected := selectBusyMenuRow(state, menuThreads()[0])
			if !selected {
				t.Fatal("busy row could not rearm landing")
			}
			after = reduceOperationResult(state, MenuEvent{Operation: operation, Attempt: effects[0].Attempt, Address: resultAddress, Success: true})
			if after.LandingAttempt != effects[0].Attempt {
				t.Fatal("explicit busy-row selection did not restore current landing")
			}
		})
	}
}

func TestMenuOperationSelectedFailureCopiesItsOwnDetail(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, a := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	state, b := dispatchThreadOperation(state, "resume", menuAddress("couch-two"))
	for i, effect := range []MenuEffect{a[0], b[0]} {
		state = reduceOperationResult(state, MenuEvent{Operation: effect.Operation, Attempt: effect.Attempt, Address: menuThreads()[i].Address, Error: fmt.Sprintf("failure %d\nfull reason", i)})
	}
	state = pruneMenuOperations(state)
	state.Notice = infoMenuNotice("unrelated latest notice")
	before := cloneMenuState(state)
	text, ok := selectedMenuOperationDetail(state)
	if !ok || text != "failure 0\nfull reason" {
		t.Fatalf("selected failure=%q,%v", text, ok)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("copy projection changed state")
	}
	state.Frames[0].SelectedAddress = menuAddress("couch-two")
	state.Frames[0].SelectedKey = menuRowKey(menuThreads()[1])
	text, ok = selectedMenuOperationDetail(state)
	if !ok || text != "failure 1\nfull reason" {
		t.Fatalf("other row failure=%q,%v", text, ok)
	}
}

func TestMenuNewSameActionFrameCannotBorrowInFlightApplicability(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Action: "relaunch", Thread: menuAddress("couch-one")})
	state, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(effects) != 1 {
		t.Fatal("operation not admitted")
	}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Action: "relaunch", Thread: menuAddress("couch-one")})
	if menuFrameOperationInFlight(state, state.CurrentFrame()) {
		t.Fatal("new confirmation borrowed old action authority")
	}
	parked := menuThreads()
	parked[0].State = couchcore.ThreadParked
	state.Inventory = parked
	state = reconcileMenuFrames(state, menuThreads())
	if state.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatal("new stale confirmation remained applicable")
	}
}

func TestMenuStartupAdoptionOwnsItsTarget(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Reattach.InFlight = map[uint64]couchcore.ThreadAddress{1: menuAddress("couch-one")}
	state.OperationSequence = 1
	next, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(effects) != 0 || !strings.Contains(next.Notice.Text, "busy: attaching") {
		t.Fatalf("UI action overlapped startup adoption: effects=%+v notice=%+v", effects, next.Notice)
	}
	_, effects = dispatchThreadOperation(state, "relaunch", menuAddress("couch-two"))
	if len(effects) != 1 {
		t.Fatal("startup adoption blocked independent row")
	}
}

func TestMenuUnconfirmedOperationDoesNotClaimFailure(t *testing.T) {
	text := menuOperationRowText(MenuOperation{Phase: MenuOperationUnconfirmed, Detail: "inspect current slot before retrying"}, time.Unix(0, 0))
	if !strings.HasPrefix(text, "unconfirmed:") {
		t.Fatalf("uncertain outcome presented as confirmed failure: %q", text)
	}
}
