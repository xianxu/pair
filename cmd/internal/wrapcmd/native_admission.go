package wrapcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type nativeAdmissionStore interface {
	Claim(context.Context, string, string, string) (launcher.NativeAdmissionClaim, error)
	PublishChild(context.Context, launcher.NativeAdmissionClaim, int) error
}

// startNativePTY is the single native start boundary for both wrapper command
// forms. A successful return permits readiness; failures never release durable
// ownership, even if the caller cleans up its newly spawned direct child.
func startNativePTY(ctx context.Context, couchHosted bool, store nativeAdmissionStore, grant, agent string, cmd *exec.Cmd, stderr io.Writer, start func(*exec.Cmd) (*os.File, error)) (*os.File, error) {
	selector, err := launcher.ParseNativeResumeSelector(agent, cmd.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("native resume selection: %w", err)
	}
	resume := selector.Kind == launcher.NativeResumeExplicit || selector.Kind == launcher.NativeResumeLatest || selector.Kind == launcher.NativeResumePicker
	if !resume {
		if grant != "" {
			return nil, errors.New("native admission does not match this Fresh, fork or non-native command")
		}
		return start(cmd)
	}
	if grant == "" {
		// A Couch-hosted pinned resume carries no grant (pair#597): Couch's
		// resume and recovery paths check its registry for the same
		// conversation in another live slot instead. Any other Couch launch
		// still gets a grant from the launcher.
		if couchHosted && selector.Kind == launcher.NativeResumeExplicit {
			return start(cmd)
		}
		if launcher.CodexIndirectResumeCompatibility(selector) {
			fmt.Fprintln(stderr, launcher.CodexIndirectResumeCompatibilityNotice)
			return start(cmd)
		}
		return nil, errors.New("native resume requires admission; use ordinary Pair with an exact conversation ID, or start Fresh")
	}
	if selector.Kind != launcher.NativeResumeExplicit {
		return nil, errors.New("native resume selection is not pinned; use ordinary Pair to resolve one exact conversation ID, or start Fresh")
	}
	digest, err := launcher.NativeAdmissionCommandDigest(agent, cmd.Args)
	if err != nil {
		return nil, err
	}
	claim, err := store.Claim(ctx, grant, agent, digest)
	if err != nil {
		return nil, fmt.Errorf("native admission refused: %w", err)
	}
	ptmx, err := start(cmd)
	if err != nil {
		return ptmx, err
	}
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return ptmx, errors.New("native admission child identity unavailable; launch ownership remains unresolved")
	}
	if err := store.PublishChild(ctx, claim, cmd.Process.Pid); err != nil {
		return ptmx, fmt.Errorf("native admission child publication unconfirmed; launch ownership retained: %w", err)
	}
	return ptmx, nil
}

func consumeNativeAdmissionEnvironment(getenv func(string) string, unsetenv func(string) error) (string, error) {
	grant := getenv(launcher.NativeAdmissionEnv)
	if err := unsetenv(launcher.NativeAdmissionEnv); err != nil {
		return "", fmt.Errorf("clear native admission environment: %w", err)
	}
	return grant, nil
}

func stripNativeAdmissionEnvironment(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		if !strings.HasPrefix(item, launcher.NativeAdmissionEnv+"=") {
			out = append(out, item)
		}
	}
	return out
}
