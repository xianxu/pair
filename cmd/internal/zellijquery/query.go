// Package zellijquery owns bounded, synchronous Zellij reads and their evidence.
package zellijquery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

const Timeout = 5 * time.Second

var ErrNoReply = errors.New("zellij query returned no reply")

// Source configures one read boundary. Nil Env inherits the caller environment;
// disposable fixtures supply their complete environment explicitly.
type Source struct {
	// ExplicitRecovery identifies a lifecycle observation at the existing caller
	// seam. Both modes preserve legacy endpoints and exclude live creators.
	ExplicitRecovery bool
	Path             string
	Env              []string
	// Run defaults to zellijexec.Run, which owns and joins a private query
	// group. A CLI already owning its whole group must select InheritedGroup
	// so queries remain inside that owner's EOF/shutdown cleanup boundary.
	Run func(*exec.Cmd) error
}

// InheritedGroup collects a query within an existing CLI-owned process group.
// The caller must own that complete group for its whole lifetime and terminate
// it on shutdown (as the title CLI does). It is not a standalone cleanup owner.
func InheritedGroup(cmd *exec.Cmd) error {
	cmd.WaitDelay = time.Second
	return cmd.Run()
}

// Result keeps original collection evidence apart from read classification.
// Only a direct ProcessErr *exec.ExitError can authorize native diagnostic
// interpretation; a joined collection error is never a native exit observation.
type Result struct {
	Stdout, Stderr []byte
	ProcessErr     error
}

// Observation retains bounded copies per attempt; counts describe full streams.
type Observation struct {
	Executable                        string
	Args                              []string
	Started, Finished                 time.Time
	CallerDeadline, EffectiveDeadline time.Time
	Duration                          time.Duration
	Stdout, Stderr                    []byte
	StdoutBytes, StderrBytes          int
	StdoutTruncated, StderrTruncated  bool
	ExitCode                          int
	Err                               error
}

type observerKey struct{}

// WithObserver installs a synchronous, nonblocking measurement callback.
func WithObserver(ctx context.Context, observe func(Observation)) context.Context {
	return context.WithValue(ctx, observerKey{}, observe)
}

// Observer lets a scoped diagnostic collector chain the previously installed observer.
func Observer(ctx context.Context) func(Observation) {
	observe, _ := ctx.Value(observerKey{}).(func(Observation))
	return observe
}

func (s Source) Output(ctx context.Context, args ...string) ([]byte, error) {
	r, err := s.ReadContext(ctx, args...)
	return r.Stdout, err
}

// ReadContext owns one original deadline and at most three joined attempts for
// exactly allowlisted reads. Unknown argv and mutations are never replayed.
// Caller-side strict validation and independent agreement remain mandatory.
func (s Source) ReadContext(ctx context.Context, args ...string) (Result, error) {
	callerDeadline, _ := ctx.Deadline()
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	var result Result
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			result.ProcessErr = errors.Join(result.ProcessErr, err)
			return result, result.ProcessErr
		}
		result, err = s.readAttempt(ctx, callerDeadline, args...)
		if canceled := ctx.Err(); canceled != nil {
			if !errors.Is(result.ProcessErr, canceled) {
				result.ProcessErr = errors.Join(result.ProcessErr, canceled)
			}
			return result, result.ProcessErr
		}
		retry := errors.Is(err, ErrSessionLookupMiss) || len(result.Stderr) == 0 &&
			(errors.Is(err, ErrNoReply) || errors.Is(err, ErrCrossedReply))
		if ReadKind(args) == UnknownReply || !retry {
			return result, err
		}
	}
	return result, err
}

func (s Source) readAttempt(ctx context.Context, callerDeadline time.Time, args ...string) (result Result, readErr error) {
	path := s.Path
	if path == "" {
		path = "zellij"
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = ChildEnvironment(s.Env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if observe, _ := ctx.Value(observerKey{}).(func(Observation)); observe != nil {
		effectiveDeadline, _ := ctx.Deadline()
		started := time.Now()
		defer func() {
			finished := time.Now()
			exitCode := -1
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}
			const bound = 64 * 1024
			observe(Observation{
				CallerDeadline: callerDeadline, EffectiveDeadline: effectiveDeadline,
				Executable: cmd.Path, Args: append([]string(nil), args...), Started: started, Finished: finished,
				Duration: finished.Sub(started), ExitCode: exitCode, Err: readErr,
				Stdout:      append([]byte(nil), stdout.Bytes()[:min(stdout.Len(), bound)]...),
				Stderr:      append([]byte(nil), stderr.Bytes()[:min(stderr.Len(), bound)]...),
				StdoutBytes: stdout.Len(), StderrBytes: stderr.Len(),
				StdoutTruncated: stdout.Len() > bound, StderrTruncated: stderr.Len() > bound,
			})
		}()
	}
	acquire := AcquireProbe
	if s.ExplicitRecovery {
		acquire = AcquireExplicitCommand
	}
	lease, err := acquire(ctx, cmd.Env)
	if err != nil {
		result.ProcessErr = err
		return result, err
	}
	defer func() {
		if err := lease.Close(); err != nil {
			result.ProcessErr = errors.Join(result.ProcessErr, err)
			readErr = errors.Join(readErr, err)
		}
	}()
	run := s.Run
	if run == nil {
		run = zellijexec.Run
	}
	result.ProcessErr = run(cmd)
	result.Stdout, result.Stderr = stdout.Bytes(), stderr.Bytes()
	if ctx.Err() != nil {
		result.ProcessErr = errors.Join(result.ProcessErr, ctx.Err())
		return result, result.ProcessErr
	}
	if result.ProcessErr != nil {
		exit, direct := result.ProcessErr.(*exec.ExitError)
		if direct {
			exit.Stderr = append([]byte(nil), result.Stderr...)
			if exit.ExitCode() == 1 && len(args) > 1 &&
				RetryableSessionLookupMiss(args, result.Stdout, result.Stderr, 1, true) && s.sessionSocketPresent(args[1]) {
				return result, ErrSessionLookupMiss
			}
			if len(args) > 0 && args[0] == "list-sessions" && exit.ExitCode() == 1 &&
				len(result.Stdout) == 0 && strings.TrimSpace(string(result.Stderr)) == "No active zellij sessions found." {
				result.Stdout = nil
				return result, nil
			}
		}
		return result, result.ProcessErr
	}
	if RetryableSessionLookupMiss(args, result.Stdout, result.Stderr, 0, false) {
		return result, ErrSessionLookupMiss
	}
	if ReadKind(args) != UnknownReply && len(bytes.TrimSpace(result.Stdout)) == 0 {
		return result, ErrNoReply
	}
	if len(result.Stderr) == 0 && CrossedReply(ReadKind(args), result.Stdout) {
		return result, ErrCrossedReply
	}
	return result, nil
}

// Error preserves the traditional launcher diagnostic prefix without changing
// the retained raw ProcessErr's direct-exit provenance.
func Error(args []string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("zellij %s: %w", strings.Join(args, " "), err)
}
