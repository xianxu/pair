package couchtty

import (
	"bytes"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"strings"
	"testing"
	"time"
)

func recoveryMenuRow() couchcore.ActionableThreadSummary {
	return couchcore.ActionableThreadSummary{Address: menuAddress("recovery"), State: couchcore.ThreadUnusable, Reason: couchcore.ReasonSessionGone, Recovery: &couchcore.RecoveryDecision{Recover: true, FromCheckpoint: true, Archive: true, Diagnosis: "helper absent; exact session must be checked", CheckpointPath: "/saved/checkpoint.md", CheckpointDigest: "abcdef"}}
}
func TestRecoveryMenuPrimaryUsesContinuityOutcome(t *testing.T) {
	row := recoveryMenuRow()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	_, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" {
		t.Fatalf("existing checkpoint recovery unavailable: %+v", effects)
	}
	row.State, row.Reason = couchcore.ThreadDetached, ""
	state = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	_, effects = reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["tag"] != string(row.Address.Tag) || effects[0].Args["repo-scope"] != row.Address.RepoScope {
		t.Fatalf("Continue: %+v", effects)
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
func TestParkConfirmationInvalidatesWhenRowDisappears(t *testing.T) {
	row := recoveryMenuRow()
	row.State, row.Reason = couchcore.ThreadLive, ""
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "park"
	state, _ = reduceActionKey(state, PanelKey{Kind: KeyEnter})
	if state.CurrentFrame().Kind != MenuFrameConfirmation {
		t.Fatal("park did not open confirmation")
	}
	// A removed row cannot authorize the pending destructive action.
	state.Inventory = nil
	next := reconcileMenuFrames(state)
	if next.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatal("park confirmation survived loss of eligibility")
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "park"
	_, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("park dispatched after loss of eligibility")
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
	row.State, row.Reason = couchcore.ThreadDetached, ""
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["tag"] != string(row.Address.Tag) || effects[0].Args["repo-scope"] != row.Address.RepoScope {
		t.Fatalf("Continue did not dispatch exact row: %+v", effects)
	}
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
	waitFor(t, "warm recovered terminal adoption", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.active == started.Handle.ID() && f.con.focus == FocusActor(started.Handle.ID()) && f.con.menu.InFlight.Operation == "" && strings.Contains(f.host.Written(), "SAME-RECOVERED-AGENT")
	})
	if _, err := f.stdin.Write([]byte("recovered-input")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "input to exact recovered agent", func() bool { return string(bytes.Join(terminal.Writes(), nil)) == "recovered-input" })
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if len(f.con.continuations) != 0 {
		t.Fatalf("warm attachment fabricated continuation watch: %+v", f.con.continuations)
	}
}

func TestAdvertisedFreshSelectionReachesTheQueuedRecoveryRoute(t *testing.T) {
	for _, tc := range []struct {
		name, key, operation string
		explicitReboot       bool
	}{
		{name: "keyed Fresh", key: "v1.fresh-opaque_123", operation: "resume"},
		{name: "keyless Fresh", operation: "reboot"},
		{name: "explicit reboot", operation: "reboot", explicitReboot: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, 24, 120)
			row := menuSlotRow(1, "fresh-source")
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			var effects []MenuEffect
			if tc.explicitReboot {
				state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyTab}})
				state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
			} else {
				state, _ = openRecoveryPreview(state, row)
				preview := &couchcore.RecoveryPreview{Address: row.Address,
					Candidates: []couchcore.RecoveryChoice{{Key: "different-native-key", NativeID: "different-native", Label: "Resume from disk"}},
					Fresh:      &couchcore.RecoveryChoice{Key: tc.key, Label: "Start fresh"}}
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: state.CurrentFrame().Generation, RecoveryPreview: preview})
				if len(effects) != 0 {
					t.Fatal("read-only preview executed recovery")
				}
				state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}})
				paint := strings.Join(renderRecoveryMenu(state.CurrentFrame(), 120, 30), "\n")
				if !strings.Contains(paint, "▸ Start fresh") {
					t.Fatalf("fixture did not select advertised Fresh: %s", paint)
				}
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
			}
			if len(effects) != 1 || state.CurrentFrame().Kind == MenuFrameConfirmation {
				t.Fatalf("Fresh did not request common preparation: %+v", effects)
			}
			calls := make(chan couchcore.OperationCall, 2)
			f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				return couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: func(call couchcore.OperationCall) (any, error) {
					calls <- call
					return nil, nil
				}}, call)
			})
			f.con.mu.Lock()
			f.con.menu, f.con.menuReady = state, true
			f.con.mu.Unlock()
			f.con.dispatchMenuEffects(effects)
			select {
			case call := <-calls:
				if call.Name != tc.operation || call.Args["candidate"] != tc.key || call.Args["path"] != row.Target.Slot.WorktreeRoot || call.Args["warm-only"] != "" {
					t.Fatalf("advertised Fresh changed at queued dispatch: %+v", call)
				}
			case <-testwait.After(t):
				t.Fatal("Fresh did not reach the owner queue")
			}
		})
	}
}
