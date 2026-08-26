package couchcore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const defaultPolicyTimeout = 5 * time.Second

type PolicyCommandOutput struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type PolicyCommand interface {
	Run(context.Context, string, ...string) (PolicyCommandOutput, error)
}

type PolicyCommandFunc func(context.Context, string, ...string) (PolicyCommandOutput, error)

func (f PolicyCommandFunc) Run(ctx context.Context, name string, args ...string) (PolicyCommandOutput, error) {
	return f(ctx, name, args...)
}

type OSPolicyCommand struct{}

func (OSPolicyCommand) Run(ctx context.Context, name string, args ...string) (PolicyCommandOutput, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	output := PolicyCommandOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		return output, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return output, ctxErr
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		output.ExitCode = exitErr.ExitCode()
		return output, nil
	}
	return output, err
}

type ExecPolicyResolver struct {
	Binary  string
	Timeout time.Duration
	Command PolicyCommand
}

func NewExecPolicyResolver(binary string) ExecPolicyResolver {
	return ExecPolicyResolver{Binary: binary, Timeout: defaultPolicyTimeout, Command: OSPolicyCommand{}}
}

func (r ExecPolicyResolver) ResolvePolicy(parent context.Context, path string) (PolicyResult, error) {
	if r.Binary == "" {
		return PolicyResult{}, errors.New("resolve fleet policy: empty sdlc binary")
	}
	if r.Command == nil {
		return PolicyResult{}, errors.New("resolve fleet policy: nil command runner")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultPolicyTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	output, err := r.Command.Run(ctx, r.Binary, "fleet", "policy", "--path", path, "--json")
	if err != nil {
		return PolicyResult{}, fmt.Errorf("run fleet policy for %q: %w", path, err)
	}
	result, err := DecodePolicyResponse(output.Stdout, output.Stderr, output.ExitCode)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("resolve fleet policy for %q: %w", path, err)
	}
	return result, nil
}
