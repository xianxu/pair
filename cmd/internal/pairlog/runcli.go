package pairlog

import (
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type persistFunc func(string, []byte, time.Time, string) error

// RunCLI implements the streaming `pair session-log append` route.
func RunCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer) int {
	return runCLI(args, stdin, getenv, now, stderr, PersistSessionLog)
}

func runCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer, persist persistFunc) int {
	if len(args) != 2 || args[0] != "--append-id" || !sessioninventory.ValidPairLogAppendID(args[1]) {
		_, _ = fmt.Fprintln(stderr, "usage: pair session-log append --append-id ID")
		return 2
	}
	path := getenv("PAIR_LOG_PATH")
	if path == "" {
		_, _ = fmt.Fprintln(stderr, "pair session-log append: PAIR_LOG_PATH is unset")
		return 2
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pair session-log append: read stdin: %v\n", err)
		return 1
	}
	if err := persist(path, body, now, args[1]); err != nil {
		if commitoutcome.Of(err) == commitoutcome.Committed {
			_, _ = fmt.Fprintf(stderr, "pair session-log append: committed with cleanup warning: %v\n", err)
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "pair session-log append: %v\n", err)
		return 1
	}
	return 0
}
