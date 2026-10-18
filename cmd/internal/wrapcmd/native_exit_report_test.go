package wrapcmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestClassifyChildExit(t *testing.T) {
	for _, c := range []struct {
		restart, stopped bool
		want             childExitCause
	}{{false, false, childExitSelf}, {true, false, childExitRestart}, {true, true, childExitRestart}, {false, true, childExitStopped}} {
		if got := classifyChildExit(c.restart, c.stopped); got != c.want {
			t.Errorf("classifyChildExit(%v,%v)=%v, want %v", c.restart, c.stopped, got, c.want)
		}
	}
}

// fakeBroker records native-exit requests and answers with code.
func fakeBroker(t *testing.T, code string, hold time.Duration) (string, chan couchmessage.Request) {
	t.Helper()
	dir := t.TempDir()
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	requests := make(chan couchmessage.Request, 4)
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var r couchmessage.Request
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		requests <- r
		if hold > 0 {
			select {
			case <-time.After(hold):
			case <-ctx.Done():
			}
		}
		return json.Marshal(couchmessage.Response{Code: code})
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	return dir, requests
}

func exitBinding() couchmessage.Binding {
	return couchmessage.Binding{Slot: "pair:2", Repository: "/repo/.git", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: os.Getpid(), Start: "fixture"}
}

// startOwnedChild runs a real native child in its own group, as run does.
func startOwnedChild(t *testing.T, script string) *ownedGroup {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return newOwnedGroup(cmd, time.Second)
}

// The owned-child conformance check (pair#657): a real child exits on its
// own, the owned group reaps it, and exactly one native-exit report reaches
// the broker with the child's exit code and recorded launch; a child the
// wrapper stops (a signal) sends none.
func TestNativeExitReportOwnedChildConformance(t *testing.T) {
	t.Setenv("PAIR_LAUNCH_ORDINAL", "4")
	namespace, requests := fakeBroker(t, "ok", 0)
	rt := &peerRuntime{binding: exitBinding(), namespace: namespace}
	file, _ := os.Create(filepath.Join(t.TempDir(), "wrap.jsonl"))
	defer file.Close()

	p := &proxy{agentBasename: "codex", wrapEventsFD: file, readyProcessIdentity: "123:child"}
	group := startOwnedChild(t, "exit 3")
	// run calls Wait only after the PTY master reaches EOF, that is, after the
	// child has exited; Wait's stop signal then cannot change its status.
	time.Sleep(300 * time.Millisecond)
	werr := group.Wait()
	if classifyChildExit(p.restartFresh.Load(), p.stoppedByWrapper.Load()) != childExitSelf {
		t.Fatal("a self exit classified otherwise")
	}
	code, ok := childExitCode(werr)
	if !ok || code != 3 {
		t.Fatalf("exit code %d ok=%v (%v)", code, ok, werr)
	}
	p.reportNativeExit(rt, code)
	select {
	case r := <-requests:
		report, err := couchmessage.ParseNativeExit(r.Body)
		if err != nil || couchmessage.ValidateRequest(r) != nil || report != (couchmessage.NativeExit{LaunchOrdinal: 4, AgentIdentity: "123:child", ExitCode: 3}) || r.ID != couchmessage.NativeExitID(rt.binding, 4) {
			t.Fatalf("report %+v (%v)", r, err)
		}
	default:
		t.Fatal("no native-exit report")
	}

	// A child the wrapper stops is not the agent leaving.
	stopped := &proxy{agentBasename: "codex", wrapEventsFD: file, readyProcessIdentity: "124:child"}
	group = startOwnedChild(t, "sleep 30")
	locks := newWrapperNativeLocks(t.TempDir(), "codex", "test")
	defer locks.Close()
	stopped.stopForSignal(syscall.SIGHUP, locks, group)
	werr = group.Wait()
	if classifyChildExit(stopped.restartFresh.Load(), stopped.stoppedByWrapper.Load()) != childExitStopped {
		t.Fatal("a wrapper-stopped child classified as a self exit")
	}
	if code, ok := childExitCode(werr); !ok || code < 128 {
		t.Fatalf("a signalled child reports code %d ok=%v", code, ok)
	}
	select {
	case r := <-requests:
		t.Fatalf("a stopped child reported: %+v", r)
	default:
	}
}

// An older Couch refuses the unknown operation, and a Couch that never
// answers costs at most the timeout: the wrapper's exit is never blocked.
func TestNativeExitReportCompatibilityAndDeadline(t *testing.T) {
	t.Setenv("PAIR_LAUNCH_ORDINAL", "1")
	file, _ := os.Create(filepath.Join(t.TempDir(), "wrap.jsonl"))
	defer file.Close()
	p := &proxy{agentBasename: "codex", wrapEventsFD: file, readyProcessIdentity: "1:child"}

	old, requests := fakeBroker(t, "invalid-request", 0)
	p.reportNativeExit(&peerRuntime{binding: exitBinding(), namespace: old}, 0)
	if len(requests) != 1 {
		t.Fatal("the report did not reach the old broker")
	}

	saved := nativeExitTimeout
	nativeExitTimeout = 200 * time.Millisecond
	defer func() { nativeExitTimeout = saved }()
	silent, _ := fakeBroker(t, "ok", 10*time.Second)
	start := time.Now()
	p.reportNativeExit(&peerRuntime{binding: exitBinding(), namespace: silent}, 0)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("a silent Couch held the wrapper %v", elapsed)
	}

	// No recorded launch: nothing is sent.
	t.Setenv("PAIR_LAUNCH_ORDINAL", "")
	none, requests := fakeBroker(t, "ok", 0)
	p.reportNativeExit(&peerRuntime{binding: exitBinding(), namespace: none}, 0)
	if len(requests) != 0 {
		t.Fatal("a launch without an ordinal was reported")
	}
}
