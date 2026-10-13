package couchtty

import (
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

func TestConsoleSwitchAgentUsesNormalQueueAndPreservesEditedArgv(t *testing.T) {
	f := newFixture(t, 24, 100)
	address := menuAddress("couch-one")
	called := make(chan couchcore.OperationCall, 1)
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		called <- call
		return couchcore.SwitchAgentResult{}, nil
	})
	f.con.mu.Lock()
	f.con.menu = NewMenuState(menuThreads(), address)
	f.con.menuReady = true
	f.con.focus = FocusPanel()
	f.con.menu.InFlight = MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address}
	f.con.mu.Unlock()
	args := map[string]string{"agent": "codex", "argv": `["--model","a b"]`, "fingerprint": "reviewed"}
	f.con.runMenuOperation(MenuEffect{Operation: "switch-agent", Attempt: 1, Args: args})
	select {
	case call := <-called:
		if call.Name != "switch-agent" || !reflect.DeepEqual(call.Args, args) {
			t.Fatalf("wrong dispatch: %+v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("switch form did not use normal dispatcher")
	}
	waitFor(t, "switch completion", func() bool { return f.con.menuSnapshot().InFlight.Operation == "" })
	if len(f.con.menuSnapshot().Orientation) != 0 {
		t.Fatal("switch created an orientation prompt")
	}
	select {
	case call := <-called:
		t.Fatalf("unexpected follow-up operation: %s", call.Name)
	default:
	}
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if !f.con.focus.IsPanel() {
		t.Fatal("switch stole panel focus")
	}
}

func TestConsoleSwitchAgentRefusalUsesNormalNotice(t *testing.T) {
	for _, reason := range []string{"source is busy", "composer is dirty"} {
		t.Run(reason, func(t *testing.T) {
			f := newFixture(t, 24, 80)
			address := menuAddress("couch-one")
			f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) { return nil, fmt.Errorf("%s", reason) })
			f.con.mu.Lock()
			f.con.menu = NewMenuState(menuThreads(), address)
			f.con.menuReady = true
			f.con.menu.InFlight = MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address}
			f.con.mu.Unlock()
			f.con.runMenuOperation(MenuEffect{Operation: "switch-agent", Attempt: 1, Args: map[string]string{"agent": "codex", "argv": "[]", "fingerprint": "fp"}})
			waitFor(t, "switch refused", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, reason) })
		})
	}
}
