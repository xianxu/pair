package couchtty

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// attachFailure freezes observations before presenter retirement or the
// caller's AbortStarted can dispose of the rejected terminal. It carries
// diagnostics only; the admitted operation retains identity and cleanup authority.
type attachFailure struct {
	cause    error
	evidence couchcore.LifecycleFailure
}

func (e *attachFailure) Error() string { return e.cause.Error() }
func (e *attachFailure) Unwrap() error { return e.cause }
func (e *attachFailure) TerminalFailureEvidence() couchcore.LifecycleFailure {
	f := e.evidence
	f.Output = append([]byte(nil), f.Output...)
	if f.BeforeExit != nil {
		code := *f.BeforeExit
		f.BeforeExit = &code
	}
	return f
}

func captureAttachFailure(stage string, child *ptychild.Child, cause error) error {
	raw := child.Snapshot()
	if len(raw) > couchcore.LaunchFailureOutputLimit {
		raw = raw[len(raw)-couchcore.LaunchFailureOutputLimit:]
	}
	f := couchcore.LifecycleFailure{
		Stage: stage, HelperStarted: true, Stream: "pty stdout/stderr",
		Output:      append([]byte(nil), raw...),
		LaunchError: fmt.Sprintf("%s; endpoint=%s input_ended=%t termination_cause=unavailable", cause, child.Endpoint().ID(), child.Endpoint().InputEnded()),
	}
	// InputEnded precedes reaping. Do not block for an exit or mistake a later
	// rollback-induced exit for the status at rejection.
	if child.Done() {
		code := child.Wait()
		f.BeforeExit = &code
	}
	return &attachFailure{cause: cause, evidence: f}
}
