package wrapcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type nativeAdmissionStore interface {
	InspectCodexScope(string) (string, error)
	Claim(context.Context, string, string, string) (launcher.NativeAdmissionClaim, error)
	PublishChild(context.Context, launcher.NativeAdmissionClaim, int) error
}

// startNativePTY is the single native start boundary for both wrapper command
// forms. A successful return permits readiness; failures never release durable
// ownership, even if the caller cleans up its newly spawned direct child.
func startNativePTY(ctx context.Context, store nativeAdmissionStore, grant, agent string, cmd *exec.Cmd, stderr io.Writer, start func(*exec.Cmd) (*os.File, error)) (*os.File, error) {
	return startNativePTYWithCodexConfigReader(ctx, store, grant, agent, cmd, stderr, start, func(ctx context.Context, q sessioninventory.CodexConfigInvocation, getenv func(string) string) (sessioninventory.CodexConfigProjection, error) {
		return sessioninventory.ReadCodexSelectionConfig(ctx, q, getenv)
	})
}

type nativeCodexConfigReader func(context.Context, sessioninventory.CodexConfigInvocation, func(string) string) (sessioninventory.CodexConfigProjection, error)

func startNativePTYWithCodexConfigReader(ctx context.Context, store nativeAdmissionStore, grant, agent string, cmd *exec.Cmd, stderr io.Writer, start func(*exec.Cmd) (*os.File, error), readConfig nativeCodexConfigReader) (*os.File, error) {
	if cmd == nil || len(cmd.Args) == 0 {
		return nil, errors.New("native invocation unavailable")
	}
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
		if launcher.CodexIndirectResumeCompatibility(selector) {
			fmt.Fprintln(stderr, launcher.CodexIndirectResumeCompatibilityNotice)
			return start(cmd)
		}
		return nil, errors.New("native resume requires admission; use ordinary Pair with an exact conversation ID, or start Fresh")
	}
	if selector.Kind != launcher.NativeResumeExplicit {
		return nil, errors.New("native resume selection is not pinned; use ordinary Pair to resolve one exact conversation ID, or start Fresh")
	}
	encodedScope, err := store.InspectCodexScope(grant)
	if err != nil {
		return nil, errors.New("native admission grant unavailable; retry launch")
	}
	if encodedScope != "" {
		if agent != "codex" {
			return nil, errors.New("native admission scope agent changed; retry launch")
		}
		cwd := cmd.Dir
		if cwd == "" || !filepath.IsAbs(cwd) {
			base, err := os.Getwd()
			if err != nil {
				return nil, errors.New("native admission cwd unavailable; retry launch")
			}
			cwd = filepath.Join(base, cwd)
		}
		startupCWD := cwd
		if selector.CWD != "" {
			if filepath.IsAbs(selector.CWD) {
				cwd = selector.CWD
			} else {
				cwd = filepath.Join(cwd, selector.CWD)
			}
		}
		invocation := selector.CodexConfig
		invocation.CWD = cwd
		invocation.StartupCWD = startupCWD
		env := cmd.Env
		if env == nil {
			env = os.Environ()
		}
		getenv := func(key string) string {
			if key != "HOME" && key != "CODEX_HOME" && key != "CODEX_SQLITE_HOME" && key != "CODEX_EXEC_SERVER_URL" {
				return ""
			}
			for i := len(env) - 1; i >= 0; i-- {
				k, v, ok := strings.Cut(env[i], "=")
				if ok && k == key {
					return v
				}
			}
			return ""
		}
		projection, err := readConfig(ctx, invocation, getenv)
		if err != nil || projection.Status != sessioninventory.CodexConfigComplete {
			return nil, errors.New("native admission Codex configuration unavailable; retry launch")
		}
		actual, err := launcher.EncodeNativeAdmissionCodexScope(projection.Scope)
		if err != nil || actual != encodedScope {
			return nil, errors.New("native admission Codex scope changed; retry launch")
		}
	}
	digest, err := launcher.NativeAdmissionScopedCommandDigest(agent, cmd.Args, encodedScope)
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
