package couchtty

import (
	"bytes"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
	"time"
)

func recoveryMenuRow() couchcore.ActionableThreadSummary {
	return couchcore.ActionableThreadSummary{Address: menuAddress("recovery"), State: couchcore.ThreadUnusable, Reason: couchcore.ReasonSessionGone, Recovery: &couchcore.RecoveryDecision{Recover: true, FromCheckpoint: true, Archive: true, Diagnosis: "helper absent; exact session must be checked", CheckpointPath: "/saved/checkpoint.md", CheckpointDigest: "abcdef"}}
}
func TestRecoveryMenuEnterResumesAndAnUnrecoverableRowNamesReboot(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	_, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" {
		t.Fatalf("recoverable Enter: %v", effects)
	}
	row.Recovery.Recover = false
	state = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	next, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || !strings.Contains(next.Notice.Text, row.Recovery.Diagnosis) || !strings.Contains(next.Notice.Text, "Tab → reboot") {
		t.Fatalf("unrecoverable Enter: %v, %s", effects, next.Notice.Text)
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
func TestRecoveryRebootConfirmationInvalidatesWhenEligibilityChanges(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	state, _ = reduceActionKey(state, PanelKey{Kind: KeyEnter})
	// "checking…" is not a verdict, and reboot stops a session.
	row.Reason, row.Recovery = couchcore.ReasonUnknown, nil
	state.Inventory = []couchcore.ActionableThreadSummary{row}
	next := reconcileMenuFrames(state)
	if next.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatal("reboot confirmation survived loss of eligibility")
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	_, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("reboot dispatched after loss of eligibility")
	}
}

func TestRecoveryAndRebootUseExistingOwnerQueue(t *testing.T) {
	for _, operation := range []string{"resume", "reboot"} {
		t.Run(operation, func(t *testing.T) {
			f := newFixture(t, 24, 100)
			// Pause EVERY queue worker (pair#205: the queue has several), so an
			// operation that goes through the owner queue must wait behind them.
			release := make(chan struct{})
			defer close(release)
			workers := max(1, f.con.queueWorkers)
			entered := make(chan struct{}, workers)
			for i := 0; i < workers; i++ {
				_, err := f.con.operationQueue.Enqueue(operationRequest{key: fmt.Sprintf("paused-lifecycle-%d", i), name: "park", run: func() (any, error) { entered <- struct{}{}; <-release; return nil, nil }})
				if err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < workers; i++ {
				<-entered
			}
			calls := make(chan string, 1)
			f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				return couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: func(call couchcore.OperationCall) (any, error) { calls <- call.Name; return nil, nil }}, call)
			})
			row := recoveryMenuRow()
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			effect := threadEffect(operation, row.Address)
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
		if name != "resume" {
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
