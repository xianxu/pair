package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, stderr *os.File) int {
	opts, ok := buildOptions(args, getenv)
	if !ok {
		return 0
	}
	logger := adapt.Open("session-watch", opts.Agent)
	defer logger.Close()
	if err := sessionwatch.Run(opts, sessionwatch.NewOSRuntime(logger)); err != nil {
		fmt.Fprintf(stderr, "pair-session-watch: %v\n", err)
		return 1
	}
	return 0
}

func buildOptions(args []string, getenv func(string) string) (sessionwatch.Options, bool) {
	if len(args) < 3 {
		return sessionwatch.Options{}, false
	}
	home := getenv("HOME")
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	return sessionwatch.Options{
		Agent:   args[0],
		Tag:     args[1],
		Cwd:     args[2],
		Args:    append([]string(nil), args[3:]...),
		Home:    home,
		DataDir: dataDir,
		PIDWait: sessionwatch.ParseDurationSeconds(getenv("PAIR_SESSION_WATCH_PID_WAIT_SECONDS"), 2*time.Second),
		Timeout: 60 * time.Second,
		Poll:    100 * time.Millisecond,
	}, true
}
