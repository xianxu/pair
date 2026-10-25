package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
)

// prepareBenchOperation builds bench-start's or bench-retire's call on the
// slot as it stands now: a vacant slot by its path, otherwise the one row that
// stands for it, as reboot is. bench-start carries its exact profile.
func prepareBenchOperation(ctx context.Context, op, target, path string, row ActionableThreadSummary, rowErr error, bench *BenchProfile) (OperationCall, string, error) {
	var args map[string]string
	switch refusal, ok := rowErr.(*SlotOperationError); {
	case rowErr == nil:
		args = ActorOperationArgs(row, OpReboot)
	case ok && refusal.Code == SlotOpNoThread:
		args = map[string]string{"path": path, "ref": target}
	default:
		return OperationCall{}, "", rowErr
	}
	if op == OpBenchStart {
		if bench == nil {
			return OperationCall{}, "", &SlotOperationError{Code: SlotOpNotOffered, Detail: "bench-start needs an agent and explicit argv"}
		}
		raw, err := json.Marshal(bench.Argv)
		if err != nil {
			return OperationCall{}, "", err
		}
		args["agent"], args["argv"] = bench.Agent, string(raw)
	}
	return OperationCall{Name: op, Args: args, Implicit: true, Context: ctx}, "", nil
}

// LaunchProvenance is what a bench-start receipt proves about the launch
// (pair#847): where it ran, which thread and Pair session, which launch
// generation (the session's start nonce), and the exact profile recorded.
type LaunchProvenance struct {
	Root       string   `json:"root"`
	Tag        string   `json:"tag"`
	Session    string   `json:"session,omitempty"`
	Generation string   `json:"generation,omitempty"`
	Agent      string   `json:"agent"`
	Argv       []string `json:"argv"`
	ArgvSource string   `json:"argv_source"`
}

// BenchStartResult is a bench-start's reboot plus its provenance; embedding
// keeps the reboot's started child and warning for adoption and receipts.
type BenchStartResult struct {
	RebootResult
	Provenance *LaunchProvenance
}

// BenchStart is Reboot with an exact profile, plus the provenance its receipt
// carries, read back from the thread record the launch committed.
func (c *Couch) BenchStart(ctx context.Context, t RebootTarget) (BenchStartResult, error) {
	if t.Argv == nil {
		return BenchStartResult{}, fmt.Errorf("bench-start needs explicit argv")
	}
	result, err := c.Reboot(ctx, t)
	out := BenchStartResult{RebootResult: result}
	started, ok := result.Started()
	if err != nil || !ok {
		return out, err
	}
	record, err := c.Threads.GetThread(started.Record.Thread)
	if err != nil {
		return out, fmt.Errorf("bench-start launched %s but its record could not be read for provenance: %w", started.Record.Thread.Tag, err)
	}
	p := &LaunchProvenance{Root: string(started.Record.Args.Worktree), Tag: string(record.Address.Tag), ArgvSource: string(ArgvSourceExplicit)}
	if profile := record.LatestLaunchProfile; profile != nil {
		p.Agent, p.Argv = profile.Agent, append([]string{}, profile.Argv...)
	}
	if binding := record.SessionBinding; binding != nil {
		p.Session, p.Generation = binding.Name, binding.StartNonce
	}
	out.Provenance = p
	return out, nil
}
