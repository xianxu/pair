// pair-go is the development dispatcher for the future primary Go CLI. Its
// launch route is a compatibility handoff to the current shell launcher.
package main

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/dispatcher"
	"github.com/xianxu/pair/cmd/internal/entrypoint"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
}

type legacyRuntime interface {
	Executable() (string, error)
	Stat(path string) error
	Environ() []string
	Exec(path string, argv []string, env []string) int
}

func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
	if len(args) > 0 && args[0] == "launch" {
		return runLegacyLaunch(args[1:], stderr, rt)
	}
	res := dispatcher.Dispatch(args)
	return writeResult(res, stdout, stderr)
}

func runLegacyLaunch(args []string, stderr io.Writer, rt legacyRuntime) int {
	exe, err := rt.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
		return 1
	}
	req := entrypoint.ResolveLegacyLaunch(exe, args)
	if err := rt.Stat(req.Path); err != nil {
		_, _ = fmt.Fprintf(stderr, "pair-go launch: pair launcher not found at %s (%v); run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", req.Path, err)
		return 1
	}
	return rt.Exec(req.Path, req.Argv, rt.Environ())
}

type osLegacyRuntime struct{}

func (osLegacyRuntime) Executable() (string, error) {
	return os.Executable()
}

func (osLegacyRuntime) Stat(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	return nil
}

func (osLegacyRuntime) Environ() []string {
	return os.Environ()
}

func (osLegacyRuntime) Exec(path string, argv []string, env []string) int {
	if err := syscall.Exec(path, argv, env); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "pair-go launch: exec %s failed: %v\n", path, err)
		return 1
	}
	return 0
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
