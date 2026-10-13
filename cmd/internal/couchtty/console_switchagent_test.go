package couchtty

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCopyOrientationUsesExactOSC52AndLeavesDraftUntouched(t *testing.T) {
	f := newFixture(t, 24, 80)
	request := orientation.Request{Tag: "work", Agent: "codex", Attempt: "one", Body: "Read the outgoing log.\nWait for instructions."}
	f.host.Reset()
	f.con.copyOrientation(request)
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(request.Body)) + "\x1b\\"
	if !strings.Contains(string(f.host.Written()), want) {
		t.Fatalf("missing exact copy sequence: %q", f.host.Written())
	}
}

func TestConsoleSwitchAgentFormAdmissionPreservesSelectionAndFocus(t *testing.T) {
	f := newFixture(t, 24, 100)
	address := menuAddress("couch-one")
	admitted := make(chan couchcore.SwitchAgentRequest, 1)
	f.con.SetAgentSwitchAdmission(func(ctx context.Context, request couchcore.SwitchAgentRequest) (string, error) {
		if ctx.Err() != nil {
			t.Error(ctx.Err())
		}
		admitted <- request
		return "form-request", nil
	})
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name == "prepare-switch-agent" {
			return couchcore.PreparedAgentSwitch{Address: address, SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: args["agent"], Argv: []string{"--model", "a b"}}, Fingerprint: "reviewed"}, nil
		}
		if name == "switch-agent" {
			t.Error("form bypassed admission")
		}
		return nil, fmt.Errorf("unexpected operation %s", name)
	})
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.focus = FocusPanel()
	f.con.panes["c1"].thread = address
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte("\t"))
	for range 3 {
		_, _ = f.stdin.Write([]byte("\x1b[B"))
	}
	_, _ = f.stdin.Write([]byte("\r\r"))
	waitFor(t, "switch prefill", func() bool {
		return f.con.menuSnapshot().CurrentFrame().SwitchStage == 1 && f.con.menuSnapshot().CurrentFrame().PreviewPending == 0
	})
	_, _ = f.stdin.Write([]byte("\t\r"))
	select {
	case request := <-admitted:
		if request.Address != address || request.Agent == "" || request.AcceptedFingerprint != "reviewed" || !reflect.DeepEqual(request.Argv, []string{"--model", "a b"}) {
			t.Fatalf("request %+v", request)
		}
	case <-time.After(time.Second):
		t.Fatal("form not admitted")
	}
	waitFor(t, "admission notice", func() bool {
		return f.con.menuSnapshot().InFlight.Operation == "" && strings.Contains(f.con.menuSnapshot().Notice.Text, "form-request")
	})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if !f.con.focus.IsPanel() {
		t.Fatal("admission stole focus")
	}
	if f.con.expectedExits.consume("c1") {
		t.Fatal("admission marked source exit expected")
	}
}

func TestConsoleSwitchAgentAdmissionDoesNotUseOperationQueue(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("couch-one")
	blocked := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	_, err := f.con.operationQueue.Enqueue(operationRequest{key: "blocked", run: func() (any, error) { close(blocked); <-release; return nil, nil }})
	if err != nil {
		t.Fatal(err)
	}
	<-blocked
	entered := make(chan struct{})
	allow := make(chan struct{})
	f.con.SetAgentSwitchAdmission(func(context.Context, couchcore.SwitchAgentRequest) (string, error) {
		close(entered)
		<-allow
		return "queued-request", nil
	})
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.menu.InFlight = MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address}
	f.con.mu.Unlock()
	f.con.runMenuOperation(MenuEffect{Operation: "switch-agent", Attempt: 1, Args: map[string]string{"agent": "codex", "argv": "[]", "fingerprint": "fp"}})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("admission waited on global queue")
	}
	close(allow)
	waitFor(t, "admission completed", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "queued-request") })
}

func TestConsoleSwitchAgentWithoutAdmissionRefuses(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("couch-one")
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.menu.InFlight = MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address}
	f.con.mu.Unlock()
	f.con.runMenuOperation(MenuEffect{Operation: "switch-agent", Attempt: 1, Args: map[string]string{"agent": "codex", "argv": "[]", "fingerprint": "fp"}})
	waitFor(t, "admission refused", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "admission unavailable") })
}

func TestOrientationFailureRetainsExactManualRecoveryAndIgnoresStaleStatus(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("couch-one")
	request := orientation.Request{Tag: string(address.Tag), Agent: "codex", Attempt: "new", Body: "Read /exact/outgoing.log and wait."}
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name != "orientation-status" || call.Args["attempt"] != "new" || call.Args["agent"] != "codex" {
			return nil, fmt.Errorf("wrong orientation identity")
		}
		return orientation.DeliveryState{Phase: orientation.DeliveryCancelled, BodyWritten: true, Reason: "operator input"}, nil
	})
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.mu.Unlock()
	f.con.watchOrientation(address, "c1", request)
	waitFor(t, "orientation failure", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "may already be") })
	state := f.con.menuSnapshot()
	if state.Orientation[address].Body != request.Body || !containsMenuItem(menuActionsFor(state, state.Inventory[0]), "copy-orientation") {
		t.Fatalf("lost recovery: %+v", state)
	}
	f.con.finishOrientation(orientationWatchResult{address: address, request: orientation.Request{Attempt: "old"}, state: orientation.DeliveryState{Phase: orientation.DeliverySubmitted}})
	if f.con.menuSnapshot().Orientation[address].Body != request.Body {
		t.Fatal("stale delivery removed recovery")
	}
}

func TestOrientationStatusErrorDoesNotClaimPromptWasNotSubmitted(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("couch-one")
	request := orientation.Request{Tag: string(address.Tag), Agent: "codex", Attempt: "uncertain", Body: "Read source."}
	f.con.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, fmt.Errorf("status read failed") })
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.mu.Unlock()
	f.con.watchOrientation(address, "c1", request)
	waitFor(t, "status read error", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "status read failed") })
	notice := f.con.menuSnapshot().Notice.Text
	if strings.Contains(notice, "was not submitted") || !strings.Contains(notice, "not confirmed") || !strings.Contains(notice, "Inspect") {
		t.Fatalf("uncertainty encourages duplicate submission: %s", notice)
	}
}

func TestOrientationCompletionOnceOutsideLock(t *testing.T) {
	for _, phase := range []orientation.DeliveryPhase{orientation.DeliverySubmitted, orientation.DeliveryCancelled} {
		t.Run(string(phase), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			address := menuAddress("couch-one")
			request := orientation.Request{Tag: string(address.Tag), Agent: "codex", Attempt: "one", Body: "exact\n"}
			calls := 0
			f.con.SetOrientationCompletion(func(a couchcore.ThreadAddress, r orientation.Request, s orientation.DeliveryState, err error) {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				calls++
				if a != address || r != request || s.Phase != phase || err != nil {
					t.Errorf("wrong callback: %v %v %v %v", a, r, s, err)
				}
			})
			f.con.mu.Lock()
			f.con.orientationWatches = map[couchcore.ThreadAddress]orientationWatch{address: {request: request, cancel: func() {}}}
			f.con.mu.Unlock()
			result := orientationWatchResult{address: address, request: request, state: orientation.DeliveryState{Phase: phase}}
			f.con.finishOrientation(result)
			f.con.finishOrientation(result)
			if calls != 1 {
				t.Fatalf("completion count %d", calls)
			}
		})
	}
}

func TestOrientationCompletionSupersededOutsideLock(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("couch-one")
	old := orientation.Request{Tag: string(address.Tag), Agent: "codex", Attempt: "old", Body: "old"}
	next := old
	next.Attempt = "next"
	callbacks := make(chan orientation.Request, 3)
	f.con.SetOrientationCompletion(func(a couchcore.ThreadAddress, r orientation.Request, s orientation.DeliveryState, err error) {
		f.con.mu.Lock()
		f.con.mu.Unlock()
		if r.Attempt == "old" && err == nil {
			t.Error("superseded completion must be unconfirmed")
		}
		callbacks <- r
	})
	f.con.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) {
		return orientation.DeliveryState{Phase: orientation.DeliverySubmitted}, nil
	})
	f.con.mu.Lock()
	f.con.orientationWatches = map[couchcore.ThreadAddress]orientationWatch{address: {request: old, cancel: func() {}}}
	f.con.mu.Unlock()
	f.con.watchOrientation(address, "c1", next)
	for range 2 {
		select {
		case <-callbacks:
		case <-time.After(time.Second):
			t.Fatal("missing completion")
		}
	}
	f.con.finishOrientation(orientationWatchResult{address: address, request: old, state: orientation.DeliveryState{Phase: orientation.DeliverySubmitted}})
	select {
	case r := <-callbacks:
		t.Fatalf("duplicate callback %v", r)
	default:
	}
}
