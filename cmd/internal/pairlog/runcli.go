package pairlog

import (
	"fmt"
	"io"
	"time"
)

type persistFunc func(string, []byte, time.Time) error

// RunCLI implements the streaming `pair session-log append` route.
func RunCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer) int {
	return runCLI(args, stdin, getenv, now, stderr, PersistSessionLog)
}

func runCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer, persist persistFunc) int {
	if len(args) != 0 {
		_, _ = fmt.Fprintln(stderr, "usage: pair session-log append")
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
	if err := persist(path, body, now); err != nil {
		_, _ = fmt.Fprintf(stderr, "pair session-log append: %v\n", err)
		return 1
	}
	return 0
}
