package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
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
	if len(raw) > 16<<10 {
		return agentfixture.Ready{}, fmt.Errorf("oversized fixture readiness")
	}
	var ready agentfixture.Ready
	if err = json.Unmarshal(raw, &ready); err != nil {
		return ready, err
	}
	if ready.RunID != f.RunID || ready.PID <= 0 || !procutil.Alive(fmt.Sprint(ready.PID)) {
		return ready, fmt.Errorf("stale fixture readiness")
	}
	return ready, nil
}

func (f *Fleet) WaitAgent(ctx context.Context, n int) error {
	return waitUntil(ctx, func() (bool, error) { _, err := f.Agent(n); return err == nil, nil })
}

// ActorCLI never manufactures credentials: the already-running fixture spawns
// the real binary from its own inherited registered actor environment.
func (f *Fleet) ActorCLI(ctx context.Context, n int, args ...string) (agentfixture.Result, error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return agentfixture.Result{}, err
	}
	defer done()
	dir, err := f.actorDirectory(n)
	if err != nil {
		return agentfixture.Result{}, err
	}
	id := fmt.Sprintf("cli-%d", nextCommand.Add(1))
	if err = agentfixture.WriteCommand(dir, agentfixture.Command{ID: id, Kind: agentfixture.Exec, Args: append([]string(nil), args...)}); err != nil {
		return agentfixture.Result{}, err
	}
	var result agentfixture.Result
	err = waitUntil(ctx, func() (bool, error) {
		var e error
		result, e = agentfixture.ReadResult(dir, id)
		if errors.Is(e, os.ErrNotExist) {
			return false, nil
		}
		return e == nil, e
	})
	if err == nil && result.Error != "" {
		err = fmt.Errorf("fixture CLI: %s", result.Error)
	}
	return result, err
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

func (f *Fleet) WaitActors(ctx context.Context, count int) error {
	var last error
	err := waitUntil(ctx, func() (bool, error) {
		response, err := f.Actors(ctx)
		last = err
		if err != nil {
			return false, nil
		}
		return len(response.Actors) == count, nil
	})
	if err != nil {
		return fmt.Errorf("expected %d registered actors: %w; last=%v", count, err, last)
	}
	return f.captureOwned(ctx)
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
