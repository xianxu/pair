package zellijquery

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// WithProbe owns bounded native admission through synchronous joined completion.
// It neither retries mutations nor changes the caller's process-cleanup owner.
func WithProbe(ctx context.Context, env []string, run func(context.Context) error) (err error) {
	return withNativeAdmission(ctx, env, false, run)
}

// WithReadyProbe adds positive readiness qualification for unattended probes.
func WithReadyProbe(ctx context.Context, env []string, run func(context.Context) error) error {
	return withNativeAdmission(ctx, env, true, run)
}

func withNativeAdmission(ctx context.Context, env []string, ready bool, run func(context.Context) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	acquire := AcquireExplicitCommand
	if ready {
		acquire = AcquireProbe
	}
	lease, err := acquire(ctx, env)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := lease.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	err = run(ctx)
	if cause := ctx.Err(); cause != nil {
		return errors.Join(err, cause)
	}
	return err
}

// IsBoundedNativeCommand limits the internal argv adapter to existing short
// commands. Creation and interactive clients keep their separate owners.
func IsBoundedNativeCommand(args []string) bool {
	_, args = nativeCommandArgv(args)
	if len(args) > 1 && (args[0] == "--session" || args[0] == "-s") {
		if !validSessionLookupName(args[1]) {
			return false
		}
		args = args[2:]
	}
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "action":
		return len(args) > 1
	case "list-sessions":
		return true
	case "kill-session", "delete-session":
		return len(args) == 2 && validSessionLookupName(args[1])
	case "run", "edit":
		return len(args) > 1
	default:
		return false
	}
}

// An explicit native override may select an absolute zellij executable. It is
// still an argv adapter for Zellij, never an arbitrary program runner.
func nativeCommandArgv(args []string) (string, []string) {
	if len(args) > 0 && filepath.IsAbs(args[0]) && filepath.Base(args[0]) == "zellij" {
		return args[0], args[1:]
	}
	return "zellij", args
}

func nativeCommandReads(args []string) bool {
	if len(args) > 1 && (args[0] == "--session" || args[0] == "-s") {
		args = args[2:]
	}
	return len(args) > 0 && (args[0] == "list-sessions" || len(args) > 1 && args[0] == "action" && len(args[1]) >= 5 && args[1][:5] == "list-")
}

// RunNativeCommand is the shared shell/Lua/exec-helper argv seam. It accepts
// native argv, never a shell command, and joins the existing native group owner.
func RunNativeCommand(ctx context.Context, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if !IsBoundedNativeCommand(args) {
		return errors.New("internal native adapter refuses interactive or unsupported command")
	}
	snapshot := ChildEnvironment(env)
	executable, args := nativeCommandArgv(args)
	return withNativeAdmission(ctx, snapshot, nativeCommandReads(args), func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Env = snapshot
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		return zellijexec.Run(cmd)
	})
}
