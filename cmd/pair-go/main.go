// pair-go is a development-only dispatcher skeleton for the future primary Go
// CLI. The public launcher remains bin/pair.
package main

import (
	"io"
	"os"

	"github.com/xianxu/pair/cmd/internal/dispatcher"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	res := dispatcher.Dispatch(args)
	return writeResult(res, stdout, stderr)
}

func runWithLauncherRuntime(args []string, stdout, stderr io.Writer, rt dispatcher.LauncherRuntime) int {
	res := dispatcher.DispatchWithLauncherRuntime(args, rt)
	return writeResult(res, stdout, stderr)
}

func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
	if res.Stdout != "" {
		_, _ = io.WriteString(stdout, res.Stdout)
	}
	if res.Stderr != "" {
		_, _ = io.WriteString(stderr, res.Stderr)
	}
	return res.ExitCode
}
