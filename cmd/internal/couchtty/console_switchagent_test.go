package couchtty

import (
	"encoding/base64"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"strings"
	"testing"
	"time"
)

func TestCopyOrientationUsesExactOSC52AndLeavesDraftUntouched(t *testing.T) {
	f := newFixture(t, 24, 80)
	request := orientation.Request{Tag: "work", Agent: "codex", Attempt: "one", Body: "Read the outgoing log.\nWait for instructions."}
	f.host.Reset()
	f.con.copyOrientation(request)
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(request.Body)) + "\x07"
	if !strings.Contains(string(f.host.Written()), want) {
		t.Fatalf("missing exact copy sequence: %q", f.host.Written())
	}
}

func TestConsoleSwitchAgentInputPreviewAndWarningAdoptsOnPanel(t *testing.T) {
	f := newFixture(t, 24, 100)
	address := menuAddress("couch-one")
	started, _ := attachStartResult(t, "switched-agent", address)
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		switch name {
		case "prepare-switch-agent":
			return couchcore.PreparedAgentSwitch{Address: address, SourceAgent: "claude", Profile: couchcore.LaunchProfile{Agent: args["agent"], Argv: []string{}}, Fingerprint: "reviewed"}, nil
		case "switch-agent":
			return couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Record: started.Record, Handle: started.Handle, Warning: "native transcript unavailable"}, nil
		default:
			return nil, fmt.Errorf("unexpected operation %s", name)
		}
	})
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte("\t"))
	for range 3 {
		_, _ = f.stdin.Write([]byte("\x1b[B"))
	}
	_, _ = f.stdin.Write([]byte("\r\r"))
	waitUpTo(t, time.Second, "switch prefill", func() bool {
		return f.con.menuSnapshot().CurrentFrame().SwitchStage == 1 && f.con.menuSnapshot().CurrentFrame().PreviewPending == 0
	})
	_, _ = f.stdin.Write([]byte("\r"))
	waitUpTo(t, time.Second, "switch final review", func() bool { return f.con.menuSnapshot().CurrentFrame().SwitchStage == 2 })
	_, _ = f.stdin.Write([]byte("\t\r"))
	waitUpTo(t, time.Second, "switch adoption", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, ok := f.con.panes[started.Handle.ID()]
		return ok && f.con.menu.InFlight.Operation == ""
	})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if !f.con.focus.IsPanel() {
		t.Fatal("panel-origin switch stole focus")
	}
	if !strings.Contains(f.con.menu.Notice.Text, "native transcript unavailable") {
		t.Fatalf("missing warning: %+v", f.con.menu.Notice)
	}
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
	waitUpTo(t, time.Second, "orientation failure", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "may already be") })
	state := f.con.menuSnapshot()
	if state.Orientation[address].Body != request.Body || !containsMenuItem(menuActionsFor(state, state.Inventory[0]), "copy-orientation") {
		t.Fatalf("lost recovery: %+v", state)
	}
	f.con.finishOrientation(orientationWatchResult{address: address, request: orientation.Request{Attempt: "old"}, state: orientation.DeliveryState{Phase: orientation.DeliverySubmitted}})
	if f.con.menuSnapshot().Orientation[address].Body != request.Body {
		t.Fatal("stale delivery removed recovery")
	}
}
