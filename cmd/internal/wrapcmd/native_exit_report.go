package wrapcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// childExitCause is why the native child is gone once group.Wait reaped it.
// Only a self exit is the agent leaving; a restart or a stop is the wrapper's
// own decision, which Couch already reports (relaunched, parked).
type childExitCause int

const (
	childExitSelf childExitCause = iota
	childExitRestart
	childExitStopped
)

func classifyChildExit(restartFresh, stoppedByWrapper bool) childExitCause {
	switch {
	case restartFresh:
		return childExitRestart
	case stoppedByWrapper:
		return childExitStopped
	default:
		return childExitSelf
	}
}

// childExitCode is the reaped child's status as one byte: its exit code, or
// 128+signal for a child a signal ended. ok is false for a wait error.
func childExitCode(werr error) (int, bool) {
	if werr == nil {
		return 0, true
	}
	var exitErr *exec.ExitError
	if !errors.As(werr, &exitErr) {
		return 0, false
	}
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal()), true
	}
	if code := exitErr.ExitCode(); code >= 0 && code <= 255 {
		return code, true
	}
	return 0, false
}

// afterChildReaped is run's one decision once group.Wait has reaped the
// native child, with the registry session still open: report a self exit, and
// nothing for a restart, a wrapper stop, a wait error or a wrapper without a
// registry session. It reports whether a report was sent.
func (p *proxy) afterChildReaped(rt *peerRuntime, werr error) bool {
	if rt == nil || classifyChildExit(p.restartFresh.Load(), p.stoppedByWrapper.Load()) != childExitSelf {
		return false
	}
	code, ok := childExitCode(werr)
	if !ok {
		return false
	}
	p.reportNativeExit(rt, code)
	return true
}

// nativeExitTimeout bounds the one report a wrapper sends as it exits.
var nativeExitTimeout = 2 * time.Second

// reportNativeExit sends Couch the wrapper's evidence that its native child
// exited on its own (pair#657), once, while the registry session is still
// open. It never changes the wrapper's exit: an older Couch answers
// invalid-request (unknown operation) and is simply not told.
func (p *proxy) reportNativeExit(rt *peerRuntime, code int) {
	ordinal, err := strconv.ParseUint(os.Getenv("PAIR_LAUNCH_ORDINAL"), 10, 64)
	if err != nil || ordinal == 0 || p.readyProcessIdentity == "" {
		p.traceWrap("native-exit-report", map[string]any{"skipped": "no recorded launch"})
		return
	}
	report := couchmessage.NativeExit{LaunchOrdinal: ordinal, AgentIdentity: p.readyProcessIdentity, ExitCode: code}
	raw, _ := json.Marshal(report)
	b := rt.binding
	request := couchmessage.Request{Op: "native-exit", Scope: b.Scope, Tag: b.Tag, Session: b.Session, Nonce: b.Nonce, ID: couchmessage.NativeExitID(b, ordinal), Body: string(raw)}
	socket, err := couchmessage.SocketPath(rt.namespace, "broker")
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), nativeExitTimeout)
		var response couchmessage.Response
		err = couchmessage.Call(ctx, socket, request, &response)
		cancel()
		if err == nil {
			p.traceWrap("native-exit-report", map[string]any{"exit_code": code, "launch_ordinal": ordinal, "code": response.Code, "error": response.Error})
			return
		}
	}
	p.traceWrap("native-exit-report", map[string]any{"exit_code": code, "launch_ordinal": ordinal, "error": err.Error()})
}
