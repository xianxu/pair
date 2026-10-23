package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	bridge "github.com/xianxu/pair/cmd/internal/testcouchbridge"
)

// dependencies replaces effects only in tests; the executable has no runtime
// switches for selecting a fake producer or bypassing native ownership.
type dependencies struct {
	newPipes func(*os.File, *os.File, <-chan os.Signal) (bridge.BridgeIO, error)
	newOwner func(bridge.Options, io.Writer) bridge.Owner
	run      func(context.Context, bridge.Owner, bridge.BridgeIO) error
}

func run(args []string, input, output *os.File, diagnostics io.Writer, signals <-chan os.Signal, deps dependencies) (code int) {
	d := bridge.NewDiagnostics(diagnostics)
	root := ""
	defer func() {
		if err := d.Finish(root); err != nil {
			code = 1
		}
	}()
	flags := flag.NewFlagSet("couch-conformance-bridge", flag.ContinueOnError)
	flags.SetOutput(d)
	var options bridge.Options
	flags.StringVar(&options.Couch, "couch", "", "absolute candidate Couch binary (otherwise build this checkout)")
	flags.StringVar(&options.ScratchParent, "scratch-parent", "", "absolute existing scratch parent")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = io.WriteString(d, "bridge: unexpected positional arguments\n")
		return 2
	}
	if err := bridge.ValidateBridgeOptions(options); err != nil {
		_, _ = fmt.Fprintf(d, "bridge: %v\n", err)
		return 2
	}
	if deps.newPipes == nil {
		deps.newPipes = func(in, out *os.File, sig <-chan os.Signal) (bridge.BridgeIO, error) {
			return bridge.NewPipeIO(in, out, sig)
		}
	}
	if deps.newOwner == nil {
		deps.newOwner = func(o bridge.Options, d io.Writer) bridge.Owner { return bridge.NewNativeOwner(o, d) }
	}
	if deps.run == nil {
		deps.run = bridge.RunBridge
	}
	// Successful construction transfers both descriptors to the runner. Rejected
	// descriptors are closed by NewPipeIO, before any fixture resource exists.
	pipes, err := deps.newPipes(input, output, signals)
	if err != nil {
		_, _ = fmt.Fprintf(d, "bridge: %v\n", err)
		return 1
	}
	owner := deps.newOwner(options, d)
	if err = deps.run(context.Background(), owner, pipes); err != nil {
		root = owner.RetainedRoot()
		_, _ = fmt.Fprintf(d, "bridge: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	diagnostics, err := bridge.NewDiagnosticPipe(os.Stderr)
	if err != nil {
		// Non-pipe stderr cannot promise bounded best-effort writes. Refuse
		// before creating any fleet rather than attempting a blocking error.
		os.Exit(2)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	code := run(os.Args[1:], os.Stdin, os.Stdout, diagnostics, signals, dependencies{})
	signal.Stop(signals)
	os.Exit(code)
}
