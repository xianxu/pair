// Package diagnosticcmd is the shared optional-log append door for Lua/shell.
package diagnosticcmd

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"io"
)

func Run(args []string, getenv func(string) string, stdin io.Reader, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "--path" || args[1] == "" {
		fmt.Fprintln(stderr, "usage: pair diagnostic append --path PATH")
		return 2
	}
	data, err := io.ReadAll(io.LimitReader(stdin, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		fmt.Fprintln(stderr, "diagnostic record unavailable or too large")
		return 1
	}
	// Optional logging is best effort; contention must not fail its caller.
	_ = diagnosticlog.Append(args[1], data, diagnosticlog.EnvironmentOptions(getenv))
	return 0
}
