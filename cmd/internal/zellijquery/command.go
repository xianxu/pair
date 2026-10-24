package zellijquery

import (
	"context"
	"errors"
	"io"
	"os/exec"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// WithProbe owns bounded native admission through synchronous joined completion.
// It neither retries mutations nor changes the caller's process-cleanup owner.
func WithProbe(ctx context.Context, env []string, run func(context.Context) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	lease, err := AcquireProbe(ctx, env)
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

// RunNativeCommand is the shared shell/Lua/exec-helper argv seam. It accepts
// native argv, never a shell command, and joins the existing native group owner.
func RunNativeCommand(ctx context.Context, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if !IsBoundedNativeCommand(args) {
		return errors.New("internal native adapter refuses interactive or unsupported command")
	}
	snapshot := ChildEnvironment(env)
	return WithProbe(ctx, snapshot, func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "zellij", args...)
		cmd.Env = snapshot
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		return zellijexec.Run(cmd)
	})
}
