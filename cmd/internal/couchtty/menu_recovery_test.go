package couchtty

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"strings"
	"testing"
	"time"
)

func recoveryMenuRow() couchcore.ActionableThreadSummary {
	return couchcore.ActionableThreadSummary{Address: menuAddress("recovery"), State: couchcore.ThreadUnusable, Reason: couchcore.ReasonStaleIncarnation, Recovery: &couchcore.RecoveryDecision{Recover: true, FromCheckpoint: true, Archive: true, Diagnosis: "helper absent; exact session must be checked", CheckpointPath: "/saved/checkpoint.md", CheckpointDigest: "abcdef"}}
}
func TestRecoveryMenuEnterUsesOrdinaryRecoveryAndUnknownDoesNotLaunch(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	_, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "recover-thread" {
		t.Fatalf("stale Enter: %v", effects)
	}
	row.Recovery.Recover = false
	state = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	next, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || !strings.Contains(next.Notice.Text, row.Recovery.Diagnosis) {
		t.Fatalf("unknown Enter: %v, %s", effects, next.Notice.Text)
	}
}
func TestRecoveryCheckpointFormDispatchBoundsFailureAndRefresh(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "recover-checkpoint"
	state, _ = reduceActionKey(state, PanelKey{Kind: KeyEnter})
	if state.CurrentFrame().Kind != MenuFrameText {
		t.Fatal("no checkpoint path form")
	}
	lines, _ := renderMenuFrame(state, state.CurrentFrame(), 100, 20, time.Now(), false)
	if !strings.Contains(strings.Join(lines, " "), "new conversation") {
		t.Fatalf("unclear form: %v", lines)
	}
	state, effects := reduceTextKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("empty path dispatched")
	}
	frame := &state.Frames[len(state.Frames)-1]
	frame.Input = "/" + strings.Repeat("x", 4095)
	state, _ = reduceTextKey(state, PanelKey{Kind: KeyRune, Rune: 'y'})
	if len(state.CurrentFrame().Input) != 4096 {
		t.Fatal("path limit is not 4096 bytes")
	}
	state.Frames[len(state.Frames)-1].Input = "/other worktree/checkpoint.md"
	state, effects = reduceTextKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "recover-checkpoint" || effects[0].Args["path"] != "/other worktree/checkpoint.md" {
		t.Fatalf("checkpoint dispatch: %v", effects)
	}
	called := false
	_, err := couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: func(call couchcore.OperationCall) (any, error) { called = true; return nil, nil }}, couchcore.OperationCall{Name: effects[0].Operation, Args: effects[0].Args, Implicit: true})
	if err != nil || !called {
		t.Fatalf("generated payload failed schema: %v", err)
	}
	state = reduceOperationResult(state, MenuEvent{Operation: effects[0].Operation, Attempt: effects[0].Attempt, Address: row.Address, Error: "unreadable checkpoint"})
	if state.CurrentFrame().Input != "/other worktree/checkpoint.md" || !strings.Contains(state.Notice.Text, "unreadable") {
		t.Fatal("failed recovery lost path/diagnosis")
	}
	state = reconcileMenuFrames(state)
	if state.CurrentFrame().Kind != MenuFrameText {
		t.Fatal("refresh discarded valid recovery form")
	}
	row.Recovery = &couchcore.RecoveryDecision{Diagnosis: "ownership changed"}
	state.Inventory = []couchcore.ActionableThreadSummary{row}
	state = reconcileMenuFrames(state)
	if state.CurrentFrame().Kind == MenuFrameText {
		t.Fatal("invalidated recovery form remained actionable")
	}
}
func TestRecoveryCheckpointCancelAndIncompleteArchive(t *testing.T) {
	row := recoveryMenuRow()
	row.Continuation = &couchcore.ContinuationStatus{RequestID: "pending", Phase: checkpoint.Failed}
	if !slices.Contains(menuActionItems(row), "archive") {
		t.Fatal("empty failed request has no archive escape")
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "recover-checkpoint"
	state, _ = reduceActionKey(state, PanelKey{Kind: KeyEnter})
	state, effects := reduceTextKey(state, PanelKey{Kind: KeyEscape})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameActions {
		t.Fatal("cancel checkpoint form failed")
	}
}

func TestRecoveryActionsShowCheckpointIdentityAndDiagnosis(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	lines, _ := renderMenuFrame(state, state.CurrentFrame(), 140, 24, time.Now(), false)
	text := strings.Join(lines, "\n")
	for _, want := range []string{row.Recovery.Diagnosis, row.Recovery.CheckpointPath, row.Recovery.CheckpointDigest} {
		if !strings.Contains(text, want) {
			t.Errorf("recovery actions omit %q: %s", want, text)
		}
	}
}
func TestRecoveryArchiveConfirmationInvalidatesWhenEligibilityChanges(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "archive"
	state, _ = reduceActionKey(state, PanelKey{Kind: KeyEnter})
	row.Recovery = &couchcore.RecoveryDecision{Diagnosis: "session ownership changed"}
	state.Inventory = []couchcore.ActionableThreadSummary{row}
	next := reconcileMenuFrames(state)
	if next.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatal("archive confirmation survived loss of eligibility")
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "archive"
	_, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("archive dispatched after loss of eligibility")
	}
}

func TestRecoveryAndArchiveUseExistingOwnerQueue(t *testing.T) {
	for _, operation := range []string{"recover-thread", "recover-checkpoint", "archive"} {
		t.Run(operation, func(t *testing.T) {
			f := newFixture(t, 24, 100)
			entered, release := make(chan struct{}), make(chan struct{})
			defer close(release)
			_, err := f.con.operationQueue.Enqueue(operationRequest{key: "paused-lifecycle", name: "park", run: func() (any, error) { close(entered); <-release; return nil, nil }})
			if err != nil {
				t.Fatal(err)
			}
			<-entered
			calls := make(chan string, 1)
			f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				return couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: func(call couchcore.OperationCall) (any, error) { calls <- call.Name; return nil, nil }}, call)
			})
			row := recoveryMenuRow()
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			effect := threadEffect(operation, row.Address)
			if operation == "recover-checkpoint" {
				effect.Args["path"] = "/checkpoint.md"
			}
			state, effects := dispatchMenuOperation(state, effect, row.Address)
			f.con.mu.Lock()
			f.con.menu = state
			f.con.menuReady = true
			f.con.mu.Unlock()
			f.con.dispatchMenuEffects(effects)
			if len(f.con.operationQueue.requests) != 1 {
				t.Fatal("menu lifecycle was not queued behind paused operation")
			}
			select {
			case got := <-calls:
				t.Fatalf("%s bypassed paused owner queue", got)
			default:
			}
		})
	}
}

func TestWarmRecoveryAdoptsExactTerminalWithoutCreatingContinuation(t *testing.T) {
	f := newFixture(t, 24, 100)
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	started, _ := attachStartResult(t, "recovered-agent", row.Address)
	terminal := started.Handle.(couchcore.TerminalHandle).Terminal()
	terminal.Feed([]byte("SAME-RECOVERED-AGENT"))
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		if name != "recover-thread" {
			t.Fatalf("unexpected recovery operation %s", name)
		}
		return couchcore.ContinuationResult{Record: started.Record, Handle: started.Handle, SourceReattached: true}, nil
	})
	f.con.mu.Lock()
	f.con.menu = state
	f.con.menuReady = true
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	f.host.Reset()
	f.con.dispatchMenuEffects(effects)
	waitUpTo(t, time.Second, "warm recovered terminal adoption", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.active == started.Handle.ID() && f.con.focus == FocusActor(started.Handle.ID()) && f.con.menu.InFlight.Operation == "" && strings.Contains(f.host.Written(), "SAME-RECOVERED-AGENT")
	})
	if _, err := f.stdin.Write([]byte("recovered-input")); err != nil {
		t.Fatal(err)
	}
	waitUpTo(t, time.Second, "input to exact recovered agent", func() bool { return string(bytes.Join(terminal.Writes(), nil)) == "recovered-input" })
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if len(f.con.continuations) != 0 {
		t.Fatalf("warm attachment fabricated continuation watch: %+v", f.con.continuations)
	}
}

func TestRecoveryCheckpointFormSurvivesOwnInFlightProjection(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameActions, Thread: row.Address, SelectedItem: "recover-checkpoint"})
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameText, Thread: row.Address, Action: "recover-checkpoint", Input: "/checkpoint.md"})
	state, effects := reduceTextKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 {
		t.Fatal("checkpoint did not dispatch")
	}
	row.Recovery = nil
	row.State = couchcore.ThreadBusy
	state.Inventory = []couchcore.ActionableThreadSummary{row}
	state = reconcileMenuFrames(state)
	if state.CurrentFrame().Kind != MenuFrameText || state.CurrentFrame().Input != "/checkpoint.md" {
		t.Fatal("own recovery transition erased checkpoint input before completion")
	}
}
