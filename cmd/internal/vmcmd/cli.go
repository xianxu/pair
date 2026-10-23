package vmcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/rowtext"
)

const runUsage = `Usage: pair vm run --checkout PATH [--lane adhoc|train] [--timeout 20m] [--artifacts DIR] -- COMMAND ARGS...

Run a literal command in a disposable VM using the qualified machine image.
The timeout includes waiting, snapshot preparation, boot and command execution:
timeout N + cleanup up to 10 s. Unconfirmed cleanup retains the occupied lane
claim and inputs; inspect the reported managed VM before attempting reuse.
`

func RunCLI(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runCLIWithRuntime(ctx, args, stdout, stderr, func() (Runtime, error) {
		driver, err := NewNativeDriver()
		if err != nil {
			return nil, err
		}
		account, err := user.Current()
		if err != nil {
			return nil, err
		}
		if err := validateRunPath(account.HomeDir); err != nil || account.HomeDir == "/" {
			return nil, fmt.Errorf("native account home unavailable")
		}
		return newHostRuntime(filepath.Join(account.HomeDir, ".tart", "pair-runner"), driver, stdout, stderr), nil
	})
}
func runCLIWithRuntime(ctx context.Context, args []string, stdout, stderr io.Writer, construct func() (Runtime, error)) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, runUsage)
		return 0
	}
	r, err := ParseRun(args)
	if err != nil {
		fmt.Fprintln(stderr, "pair vm run:", rowtext.Sanitize(err.Error()))
		return 2
	}
	rt, err := construct()
	if err != nil {
		fmt.Fprintln(stderr, "pair vm run:", rowtext.Sanitize(err.Error()))
		return 125
	}
	fmt.Fprintf(stdout, "timeout %s + cleanup up to 10 s\n", r.Timeout)
	state := Run(ctx, r, rt)
	code := state.ExitCode()
	if publisher, ok := rt.(interface{ WriteResult(RunState) error }); ok {
		if err := publisher.WriteResult(state); err != nil {
			fmt.Fprintln(stderr, "pair vm run result:", rowtext.Sanitize(err.Error()))
			if code == 0 {
				code = 125
			}
		}
	}
	if state.Detail() != "" {
		fmt.Fprintln(stderr, "pair vm run:", rowtext.Sanitize(state.Detail()))
	}
	return code
}
