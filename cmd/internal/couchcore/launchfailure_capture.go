package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// captureLaunchFailure records before cleanup can erase the failed attempt,
// then records what was observed after it. Persistence failure never skips cleanup.
func (c *Couch) captureLaunchFailure(in trackedThreadLaunch, stage string, h Handle, cause error, cleanup func() error) error {
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
	saved := c.saveLaunchFailure(f)
	var cleanupErr error
	if cleanup != nil {
		cleanupErr = cleanup()
	}
	if h != nil {
		f.AfterExit = observedLaunchExit(h)
	}
	if output, ok := h.(LaunchOutputHandle); ok {
		f.Stream, f.Output = output.LaunchOutput()
	}
	if cleanupErr != nil {
		if f.CleanupError != "" {
			f.CleanupError += "\n"
		}
		f.CleanupError += cleanupErr.Error()
	}
	saved = errors.Join(saved, c.saveLaunchFailure(f))
	return &launchFailureError{failure: f, cause: errors.Join(cause, cleanupErr), saveError: saved, recordPath: c.launchFailurePath()}
}
func (c *Couch) saveLaunchFailure(f LaunchFailure) error {
	if c.LaunchFailures == nil {
		return errors.New("launch diagnostic store unavailable")
	}
	if err := c.LaunchFailures.Save(f); err != nil {
		return fmt.Errorf("write launch diagnostics %s: %w", c.LaunchFailures.Path(), err)
	}
	return nil
}

func (c *Couch) finishSwitchFailure(in trackedThreadLaunch, err error) error {
	var failed *launchFailureError
	if !errors.As(err, &failed) {
		wrapped := c.captureLaunchFailure(in, "switch preparation", nil, err, nil)
		errors.As(wrapped, &failed)
	}
	f := failed.failure
	f.Operation = "switch-agent"
	f.FinalError = err.Error()
	return &launchFailureError{failure: f, cause: err, saveError: errors.Join(failed.saveError, c.saveLaunchFailure(f)), recordPath: c.launchFailurePath()}
}

func (c *Couch) launchFailurePath() string {
	if c.LaunchFailures == nil {
		return "unavailable"
	}
	return c.LaunchFailures.Path()
}
