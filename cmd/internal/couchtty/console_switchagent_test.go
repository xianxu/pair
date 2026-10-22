package couchtty

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"reflect"
	"strings"
	"testing"
	"time"
)

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
	setTestMenuOperation(&f.con.menu, MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address})
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
	waitFor(t, "switch completion", func() bool { return f.con.menuSnapshot().operationForTest().Operation == "" })
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
			setTestMenuOperation(&f.con.menu, MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address})
			f.con.mu.Unlock()
			f.con.runMenuOperation(MenuEffect{Operation: "switch-agent", Attempt: 1, Args: map[string]string{"agent": "codex", "argv": "[]", "fingerprint": "fp"}})
			waitFor(t, "switch refused", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, reason) })
		})
	}
}

func TestConsoleSwitchAgentEditsParametersWithoutSuccessfulPreview(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, mode := range []string{"held", "refused"} {
			t.Run(agent+"/"+mode, func(t *testing.T) {
				address := menuAddress("couch-one")
				release := make(chan struct{})
				var once sync.Once
				unblock := func() { once.Do(func() { close(release) }) }
				defer unblock()
				calls := make(chan string, 8)
				f := newFixtureBeforeRun(t, 24, 100, func(con *Console) {
					con.menu = NewMenuState(menuThreads(), address)
					appendMenuFrame(&con.menu, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: address, Agent: agent})
					con.menuReady = true
					con.focus = FocusPanel()
					con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
						calls <- call.Name
						if mode == "refused" {
							return nil, errors.New("source preview refused")
						}
						select {
						case <-release:
						case <-call.Context.Done():
							return nil, call.Context.Err()
						}
						return couchcore.PreparedAgentSwitch{Address: address, SourceAgent: "agy", Profile: couchcore.LaunchProfile{Agent: agent, Argv: []string{"stale-default"}}, Fingerprint: "late"}, nil
					})
				})
				if _, err := f.stdin.Write([]byte("\r")); err != nil {
					t.Fatal(err)
				}
				select {
				case name := <-calls:
					if name != "prepare-switch-agent" {
						t.Fatalf("unexpected operation %s", name)
					}
				case <-testwait.After(t):
					t.Fatal("preview did not start")
				}
				if mode == "refused" {
					waitFor(t, "preview refusal", func() bool { return strings.Contains(f.con.menuSnapshot().Notice.Text, "source preview refused") })
				}
				// Real terminal bytes exercise Unicode, quoting, cursor movement and backspace.
				if _, err := f.stdin.Write([]byte("'a b'\x1b[D\x7f界\t")); err != nil {
					t.Fatal(err)
				}
				waitFor(t, "input focus sentinel", func() bool { return f.con.menuSnapshot().CurrentFrame().SelectedItem == "switch" })
				frame := f.con.menuSnapshot().CurrentFrame()
				if frame.Input != "'a 界'" || !frame.SwitchEdited || frame.PreviewPending != 0 || frame.SwitchPrepared != nil {
					t.Fatalf("preview blocked or corrupted draft: %+v", frame)
				}
				unblock()
				waitFor(t, "late preview completion", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.previewRunning == 0 })
				if got := f.con.menuSnapshot().CurrentFrame().Input; got != "'a 界'" {
					t.Fatalf("late preview replaced edited draft: %q", got)
				}
				if _, err := f.stdin.Write([]byte("\x1b")); err != nil {
					t.Fatal(err)
				}
				waitFor(t, "cancelled parameter form", func() bool { return f.con.menuSnapshot().CurrentFrame().Kind == MenuFrameRoot })
				select {
				case name := <-calls:
					t.Fatalf("editing or cancel dispatched %s", name)
				default:
				}
			})
		}
	}
}

func TestConsoleForceSwitchPreservesLocalPayload(t *testing.T) {
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
	setTestMenuOperation(&f.con.menu, MenuOperationOrigin{Operation: "switch-agent", Attempt: 1, Address: address})
	f.con.mu.Unlock()
	consent := (&couchcore.AgentSwitchForceOffer{}).Confirm()
	effect := MenuEffect{Operation: "switch-agent", Attempt: 1, Args: map[string]string{"agent": "codex", "argv": "[]", "fingerprint": "accepted"}, TypedPayload: consent}
	raw, err := json.Marshal(effect)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "TypedPayload") || strings.Contains(string(raw), "offer") {
		t.Fatal("local consent escaped serialization")
	}
	f.con.runMenuOperation(effect)
	select {
	case call := <-called:
		if call.TypedPayload != consent {
			t.Fatal("normal lifecycle queue lost local consent")
		}
	case <-testwait.After(t):
		t.Fatal("no force dispatch")
	}
	waitFor(t, "force operation completion", func() bool { return f.con.menuSnapshot().operationForTest().Operation == "" })
}
