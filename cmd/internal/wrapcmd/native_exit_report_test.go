package wrapcmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// startOwnedChild runs a real native child in its own group, as run does. The
// returned reader reaches EOF when the child exits, as run's PTY master does.
func startOwnedChild(t *testing.T, script string) (*ownedGroup, *os.File) {
	t.Helper()
	out, in, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Stdout = in
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	in.Close()
	if err != nil {
		t.Fatal(err)
	}
	return newOwnedGroup(cmd, time.Second), out
}

// The owned-child conformance check (pair#657), through run's own decision
// (afterChildReaped): a real child that exits on its own, reaped by the owned
// group, sends exactly one native-exit report with its exit code and recorded
// launch; a child the wrapper stops (SIGHUP) or restarts (SIGUSR2) sends none,
// and so does a wait error or a wrapper without a registry session.
func TestNativeExitReportOwnedChildConformance(t *testing.T) {
	t.Setenv("PAIR_LAUNCH_ORDINAL", "4")
	namespace, requests := fakeBroker(t, "ok", 0)
	rt := &peerRuntime{binding: exitBinding(), namespace: namespace}
	file, _ := os.Create(filepath.Join(t.TempDir(), "wrap.jsonl"))
	defer file.Close()
	none := func(what string) {
		t.Helper()
		select {
		case r := <-requests:
			t.Fatalf("%s reported: %+v", what, r)
		default:
		}
	}

	p := &proxy{agentBasename: "codex", wrapEventsFD: file, readyProcessIdentity: "123:child"}
	group, out := startOwnedChild(t, "exit 3")
	// run calls Wait only after the PTY master reaches EOF, that is, after the
	// child has exited; Wait's stop signal then cannot change its status.
	if _, err := io.Copy(io.Discard, out); err != nil {
		t.Fatal(err)
	}
	if !p.afterChildReaped(rt, group.Wait()) {
		t.Fatal("a self exit was not reported")
	}
	select {
	case r := <-requests:
		report, err := couchmessage.ParseNativeExit(r.Body)
		if err != nil || couchmessage.ValidateRequest(r) != nil || report != (couchmessage.NativeExit{LaunchOrdinal: 4, AgentIdentity: "123:child", ExitCode: 3}) || r.ID != couchmessage.NativeExitID(rt.binding, 4) {
			t.Fatalf("report %+v (%v)", r, err)
		}
	default:
		t.Fatal("no native-exit report")
	}
	none("a single self exit twice")

	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGUSR2} {
		stopped := &proxy{agentBasename: "codex", wrapEventsFD: file, readyProcessIdentity: "124:child"}
		group, _ := startOwnedChild(t, "sleep 30")
		locks := newWrapperNativeLocks(t.TempDir(), "codex", "test")
		stopped.stopForSignal(sig, locks, group)
		if stopped.afterChildReaped(rt, group.Wait()) {
			t.Fatalf("a child stopped by %v was reported", sig)
		}
		locks.Close()
		none(sig.String())
	}
	if p.afterChildReaped(rt, errors.New("wait failed")) || p.afterChildReaped(nil, nil) {
		t.Fatal("a wait error or a missing registry session was reported")
	}
	none("a wait error")
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
	//lint:allow test-shrunk-deadline — the broker never answers, so the deadline always fires; the test asserts only its bound
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
