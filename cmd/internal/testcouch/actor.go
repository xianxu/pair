package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

var nextCommand atomic.Uint64

func (f *Fleet) actorDirectory(n int) (string, error) {
	path := f.Workspaces.Primary
	if n != 0 {
		path = f.Workspaces.Workers[n]
	}
	if path == "" {
		return "", fmt.Errorf("unknown actor %d", n)
	}
	return agentfixture.ActorDirectory(f.Roots.Mutable["control"], path), nil
}

func (f *Fleet) Agent(n int) (agentfixture.Ready, error) {
	dir, err := f.actorDirectory(n)
	if err != nil {
		return agentfixture.Ready{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "ready.json"))
	if err != nil {
		return agentfixture.Ready{}, err
	}
	if len(raw) > agentfixture.MaxReadyBytes {
		return agentfixture.Ready{}, fmt.Errorf("oversized fixture readiness")
	}
	var ready agentfixture.Ready
	if err = json.Unmarshal(raw, &ready); err != nil {
		return ready, err
	}
	if ready.RunID != f.RunID || ready.PID <= 0 || ready.Generation == "" || procutil.Identity(fmt.Sprint(ready.PID)) != ready.Generation {
		return ready, fmt.Errorf("stale fixture readiness")
	}
	return ready, nil
}

func (f *Fleet) WaitAgent(ctx context.Context, n int) error {
	return waitUntil(ctx, func() (bool, error) { _, err := f.Agent(n); return err == nil, nil })
}

// ActorCLI never manufactures credentials: the already-running fixture spawns
// the real binary from its own inherited registered actor environment.
func (f *Fleet) ActorCLI(ctx context.Context, n int, args ...string) (result agentfixture.Result, outcome error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return agentfixture.Result{}, err
	}
	defer done()
	dir, err := f.actorDirectory(n)
	if err != nil {
		return agentfixture.Result{}, err
	}
	admitted := time.Now()
	timeout, err := actorCLITimeout(ctx, admitted, f.Options.RuntimeBudget)
	if err != nil {
		return agentfixture.Result{}, err
	}
	id := fmt.Sprintf("cli-%d", nextCommand.Add(1))
	if err = agentfixture.WriteCommand(dir, agentfixture.Command{ID: id, Kind: agentfixture.Exec, Timeout: timeout, DeadlineUnixNano: admitted.Add(timeout).UnixNano(), Args: append([]string(nil), args...)}); err != nil {
		return agentfixture.Result{}, err
	}
	// Publication transfers request lifetime to this invocation. Only proved
	// joined completion AND successful retirement release that ownership; every
	// other return hands the existing fleet owner an explicit uncertain outcome.
	retired := false
	defer func() {
		if !retired {
			outcome = errors.Join(agentfixture.ErrCleanupUncertain, outcome)
			f.record(RunEvent{Kind: "unknown", Error: outcome.Error()})
		}
	}()
	err = waitUntil(ctx, func() (bool, error) {
		var e error
		result, e = agentfixture.ReadResult(dir, id)
		if errors.Is(e, os.ErrNotExist) {
			return false, nil
		}
		return e == nil, e
	})
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		callerErr := err
		// Caller cancellation revokes the request, not just this observer.
		// Cleanup has the same explicit 15s bound as fleet teardown and remains
		// independent of the canceled context while the fixture joins its child.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if e := agentfixture.CancelCommand(dir, id); e != nil {
			return result, errors.Join(callerErr, fmt.Errorf("fixture CLI cancellation uncertain: %w", e))
		}
		e := waitUntil(cleanup, func() (bool, error) {
			var e error
			result, e = agentfixture.ReadResult(dir, id)
			if errors.Is(e, os.ErrNotExist) {
				return false, nil
			}
			return e == nil, e
		})
		if e != nil {
			return result, errors.Join(callerErr, fmt.Errorf("fixture CLI cancellation acknowledgment uncertain: %w", e))
		}
		if e := f.retainUncertainCLI(dir, result); e != nil {
			return result, errors.Join(callerErr, e)
		}
		if e := agentfixture.RetireResult(dir, id); e != nil {
			return result, errors.Join(callerErr, fmt.Errorf("fixture CLI terminal retirement: %w", e))
		}
		retired = true
		return result, callerErr
	}
	if err == nil {
		if e := f.retainUncertainCLI(dir, result); e != nil {
			return result, e
		}
		// This invocation is the last consumer of its unique correlated result.
		// Reclaim it before another observation needs the bounded mailbox.
		err = agentfixture.RetireResult(dir, id)
		retired = err == nil
		if result.Error != "" {
			err = errors.Join(err, fmt.Errorf("fixture CLI: %s", result.Error))
		}
	}
	return result, errors.Join(err, ctx.Err())
}

func (f *Fleet) retainUncertainCLI(dir string, result agentfixture.Result) error {
	err := result.CleanupFailure()
	if err == nil {
		return nil
	}
	// Keep all three correlated files until Close reconciles the private run.
	err = errors.Join(err, agentfixture.CancelCommand(dir, result.ID))
	return err
}

// The actor child shares the caller's remaining bound rather than imposing a
// shorter hidden timeout on real recovery operations.
func actorCLITimeout(ctx context.Context, now time.Time, fallback time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	budget := fallback
	if deadline, ok := ctx.Deadline(); ok {
		budget = deadline.Sub(now)
		if budget <= 0 {
			return 0, context.DeadlineExceeded
		}
	}
	if budget == 0 {
		budget = agentfixture.DefaultCLITimeout
	}
	if budget < 0 {
		return 0, errors.New("negative fixture CLI budget")
	}
	if budget > agentfixture.MaxCLITimeout {
		budget = agentfixture.MaxCLITimeout
	}
	return budget, nil
}

func (f *Fleet) Actors(ctx context.Context) (couchmessage.Response, error) {
	result, err := f.ActorCLI(ctx, 0, "--actors", "--json")
	if err != nil {
		return couchmessage.Response{}, err
	}
	if result.ExitCode != 0 {
		return couchmessage.Response{}, fmt.Errorf("actors exit%d: %s %s", result.ExitCode, result.Stdout, result.Stderr)
	}
	var response couchmessage.Response
	err = json.Unmarshal([]byte(result.Stdout), &response)
	if err == nil && response.Code != "ok" {
		err = fmt.Errorf("actors: %s %s", response.Code, response.Error)
	}
	return response, err
}

// ObserveDrive reads one current wrapper observation without sending input.
// The public actor registry supplies the complete binding; fixture readiness
// verifies the requested workspace and personality. Callers own any bounded
// polling and must still treat the later public command's admission as final.
// The test process must select this fleet's canonical COUCH_ISOLATED_ROOT so
// the existing endpoint client derives the owned socket directory.
func (f *Fleet) ObserveDrive(ctx context.Context, n int) (couchmessage.DriveObservation, error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return couchmessage.DriveObservation{}, err
	}
	defer done()
	root, err := filepath.EvalSymlinks(f.Roots.Root)
	if err != nil || !filepath.IsAbs(root) || root != f.Roots.Root || os.Getenv("COUCH_ISOLATED_ROOT") != root {
		return couchmessage.DriveObservation{}, fmt.Errorf("COUCH_ISOLATED_ROOT must select the canonical owned fleet root")
	}
	namespace := f.Roots.Mutable["store"]
	if !filepath.IsAbs(namespace) || !pathscope.Within(f.Roots.Root, namespace) {
		return couchmessage.DriveObservation{}, fmt.Errorf("owned store namespace required")
	}
	ready, err := f.Agent(n)
	if err != nil {
		return couchmessage.DriveObservation{}, err
	}
	actors, err := f.Actors(ctx)
	if err != nil {
		return couchmessage.DriveObservation{}, err
	}
	binding, err := f.driveBinding(n, ready, actors.Actors)
	if err != nil {
		return couchmessage.DriveObservation{}, err
	}
	endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}
	return endpoint.ObserveDrive(ctx)
}

func (f *Fleet) driveBinding(n int, ready agentfixture.Ready, actors []couchmessage.Candidate) (couchmessage.Binding, error) {
	workspace := f.Workspaces.Primary
	if n != 0 {
		workspace = f.Workspaces.Workers[n]
	}
	if n < 0 || workspace == "" || ready.CWD != workspace || ready.Agent != f.Options.AgentKind {
		return couchmessage.Binding{}, fmt.Errorf("actor %d does not match owned workspace and personality", n)
	}
	slot := fmt.Sprintf("fixture:%d", n)
	var binding couchmessage.Binding
	found := false
	for _, actor := range actors {
		if actor.Binding.Slot != slot {
			continue
		}
		if found {
			return couchmessage.Binding{}, fmt.Errorf("duplicate actor %s", slot)
		}
		binding, found = actor.Binding, true
	}
	if !found {
		return couchmessage.Binding{}, fmt.Errorf("registered actor %s unavailable", slot)
	}
	if err := binding.Validate(); err != nil {
		return couchmessage.Binding{}, err
	}
	if binding.Repository != filepath.Join(f.Workspaces.Primary, ".git") || binding.Agent != ready.Agent {
		return couchmessage.Binding{}, fmt.Errorf("registered actor %s does not match owned repository and personality", slot)
	}
	return binding, nil
}

// WaitActors waits for at least count registrations. Restarting a supervisor
// may reconnect workers before its first steward-readiness observation.
func (f *Fleet) WaitActors(ctx context.Context, count int) error {
	if err := waitActorCount(ctx, count, f.Actors); err != nil {
		return err
	}
	return f.captureOwned(ctx)
}

func waitActorCount(ctx context.Context, count int, observe func(context.Context) (couchmessage.Response, error)) error {
	var last error
	err := waitUntil(ctx, func() (bool, error) {
		response, err := observe(ctx)
		last = err
		if err != nil {
			if errors.Is(err, agentfixture.ErrMailboxFull) || errors.Is(err, agentfixture.ErrCleanupUncertain) {
				return false, err
			}
			return false, nil
		}
		return len(response.Actors) >= count, nil
	})
	if err != nil {
		return fmt.Errorf("expected at least %d registered actors: %w; last=%v", count, err, last)
	}
	return nil
}

func (f *Fleet) Accepted(n int) ([]agentfixture.AcceptedInput, error) {
	dir, err := f.actorDirectory(n)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(dir, "accepted.jsonl"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var result []agentfixture.AcceptedInput
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var value agentfixture.AcceptedInput
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
		if len(result) > agentfixture.MaxRequests {
			return nil, fmt.Errorf("fixture input log limit")
		}
	}
	return result, nil
}

// Resources observes the private supervisor through the fixture's inherited
// namespace using its fixed readonly command. It does not manufacture caller
// credentials or alter generic CLI/frame/output limits.
func (f *Fleet) Resources(ctx context.Context) (agentfixture.ResourceObservation, error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return agentfixture.ResourceObservation{}, err
	}
	defer done()
	control, err := f.actorDirectory(0)
	if err != nil {
		return agentfixture.ResourceObservation{}, err
	}
	id := fmt.Sprintf("resources-%d", nextCommand.Add(1))
	if err = agentfixture.WriteCommand(control, agentfixture.Command{ID: id, Kind: agentfixture.Resources}); err != nil {
		return agentfixture.ResourceObservation{}, err
	}
	var result agentfixture.Result
	err = waitUntil(ctx, func() (bool, error) {
		var e error
		result, e = agentfixture.ReadResult(control, id)
		if errors.Is(e, os.ErrNotExist) {
			return false, nil
		}
		return e == nil, e
	})
	if err != nil {
		f.record(RunEvent{Kind: "unknown", Error: "readonly fixture observation not joined: " + err.Error()})
		return agentfixture.ResourceObservation{}, err
	}
	if err = agentfixture.RetireResult(control, id); err != nil {
		return agentfixture.ResourceObservation{}, err
	}
	if result.Error != "" || result.ExitCode != 0 || result.CleanupUncertain || result.Truncated {
		return agentfixture.ResourceObservation{}, fmt.Errorf("readonly fixture observation: %+v", result)
	}
	var observation agentfixture.ResourceObservation
	err = json.Unmarshal([]byte(result.Stdout), &observation)
	return observation, err
}
