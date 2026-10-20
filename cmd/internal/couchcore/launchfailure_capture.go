package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// launchFailureError keeps the error chain and diagnostic identity independent
// of the concise operator message. It is never an authority for cleanup.
type launchFailureError struct {
	failure    LaunchFailure
	cause      error
	saveError  error
	recordPath string
}

func (e *launchFailureError) Unwrap() error { return errors.Join(e.cause, e.saveError) }
func (e *launchFailureError) Error() string {
	text := e.operatorLine()
	if cause := e.Unwrap(); cause != nil {
		text += "\n" + cause.Error()
	}
	return text
}

// OperatorError keeps launch diagnostics complete for programmatic callers while
// presenting one bounded explanation and action at UI and receipt boundaries.
func OperatorError(err error) string {
	if err == nil {
		return ""
	}
	var failed *launchFailureError
	if errors.As(err, &failed) {
		line := failed.operatorLine()
		var unconfirmed *RecoveryLaunchUnconfirmed
		if errors.As(err, &unconfirmed) {
			line = "Launch unconfirmed; " + line
		}
		var recorded *lifecycleFailureError
		if failed.saveError == nil && errors.As(err, &recorded) && recorded.saveError != nil {
			line += " (diagnostics could not be saved: " + failureSummary(recorded.saveError.Error()) + ")"
		}
		return line
	}
	return err.Error()
}

func (e *launchFailureError) operatorLine() string {
	f := e.failure
	happened := "launch failed during " + f.Stage + ": " + failureSummary(f.LaunchError)
	if f.BeforeExit != nil {
		happened = fmt.Sprintf("Pair exited with exit %d during %s", *f.BeforeExit, f.Stage)
		if f.Stage == "registration" {
			happened = fmt.Sprintf("Pair exited before registering (exit %d)", *f.BeforeExit)
		}
	} else if errors.Is(e.cause, context.Canceled) {
		happened = "launch wait canceled; helper exit not observed before cleanup"
	}
	var cleanup *SessionCleanupUnconfirmed
	if errors.As(e.cause, &cleanup) {
		happened = "session cleanup unconfirmed; ownership retained"
	}
	if errors.Is(e.cause, ErrCleanupPersistenceUnconfirmed) {
		happened = "session cleanup and unknown state persistence unconfirmed; original occupied claim retained"
	}
	if f.Operation == "switch-agent" {
		happened = "switch-agent: " + happened
	}
	if e.saveError != nil {
		happened += " (diagnostics could not be saved: " + failureSummary(e.saveError.Error()) + ")"
	}
	happened += " [attempt " + failureSummary(f.Nonce) + "; diagnostics " + rowtext.Sanitize(e.recordPath) + "]"
	ref := f.StartingPath
	if ref == "" {
		ref = string(f.Address.Tag)
	}
	action := "couch --show '" + strings.ReplaceAll(ref, "'", "'\"'\"'") + "'"
	return rowtext.Sanitize(happened) + "; inspect with " + rowtext.Sanitize(action)
}

// Keep the operator line useful and bounded; the store retains full detail.
func failureSummary(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	s, _, _ = strings.Cut(s, ";")
	s = strings.Join(strings.Fields(rowtext.Sanitize(s)), " ")
	if len(s) > 240 {
		s = s[:240] + "…"
	}
	return strings.ToValidUTF8(s, "�")
}

func observedLaunchExit(h Handle) *int {
	if h == nil {
		return nil
	}
	select {
	case <-h.Exited():
		code := h.Wait()
		return &code
	default:
		return nil
	}
}

// captureLaunchFailure snapshots before cleanup can erase the failed attempt,
// then publishes once afterward. Optional persistence never postpones cleanup.
func (c *Couch) captureLaunchFailure(in trackedThreadLaunch, stage string, h Handle, cause error, cleanup func() error) error {
	captured := c.collectLaunchFailure(in, stage, h, cause, cleanup)
	return c.finishLaunchFailureCapture(in, captured.failure, captured.cause)
}

// collectLaunchFailure keeps cleanup evidence unpublished until its owner finishes.
func (c *Couch) collectLaunchFailure(in trackedThreadLaunch, stage string, h Handle, cause error, cleanup func() error) *launchFailureError {
	f := LaunchFailure{Address: in.Thread.Address, Nonce: in.Nonce, StartingPath: in.Thread.StartingPath, Agent: in.Args.Stack, Operation: "launch", Stage: stage, FailedAt: c.Clock.Now(), HelperStarted: h != nil, LaunchError: cause.Error(), BeforeExit: observedLaunchExit(h)}

	f.Stream = "unavailable"
	if output, ok := h.(LaunchOutputHandle); ok {
		f.Stream, f.Output = output.LaunchOutput()
	}
	var setup *blockedStartError
	if errors.As(cause, &setup) {
		f.HelperStarted = setup.HelperStarted
		f.BeforeExit, f.AfterExit = setup.BeforeExit, setup.AfterExit
		f.Stream, f.Output = setup.Stream, append([]byte(nil), setup.Output...)
		f.CleanupError = setup.CleanupError
	}
	f = cloneLaunchFailure(f)
	var cleanupErr error
	if cleanup != nil {
		cleanupErr = cleanup()
	}
	if h != nil {
		f.AfterExit = observedLaunchExit(h)
	}
	if output, ok := h.(LaunchOutputHandle); ok {
		if stream, raw := output.LaunchOutput(); len(raw) > 0 {
			f.Stream, f.Output = stream, append([]byte(nil), raw...)
		}
	}
	if cleanupErr != nil {
		if f.CleanupError != "" {
			f.CleanupError += "\n"
		}
		f.CleanupError += cleanupErr.Error()
	}
	return &launchFailureError{failure: f, cause: errors.Join(cause, cleanupErr), recordPath: c.launchFailurePath()}
}

func (c *Couch) finishLaunchFailureCapture(in trackedThreadLaunch, f LaunchFailure, cause error) error {
	captured := &launchFailureError{failure: f, cause: cause, recordPath: c.launchFailurePath()}
	if OperationAttemptFromContext(in.Context) != nil {
		return captured // The admitted operation still owns adoption/readiness.
	}
	// Standalone launch APIs retain their exact existing launch identity, while
	// sharing the same terminal recorder rather than writing at capture sites.
	attempt, err := c.BeginOperationAttempt(f.Operation, failureTarget(f))
	if err != nil {
		captured.saveError = err
		return captured
	}
	attempt.nonce = f.Nonce
	_, result := attempt.Finish(nil, captured)
	var recorded *lifecycleFailureError
	if errors.As(result, &recorded) {
		captured.saveError = recorded.saveError
	}
	return result
}

const launchDiagnosticsAdmissionTimeout = 10 * time.Second

func (c *Couch) finishSwitchFailure(in trackedThreadLaunch, err error) error {
	var failed *launchFailureError
	if !errors.As(err, &failed) {
		failed = c.collectLaunchFailure(in, "switch preparation", nil, err, nil)
	}
	f := failed.failure
	f.Operation = "switch-agent"
	f.FinalError = err.Error()
	return c.finishLaunchFailureCapture(in, f, err)
}

func (c *Couch) launchFailurePath() string {
	if c.LaunchFailures == nil {
		return "unavailable"
	}
	return c.LaunchFailures.Path()
}
