// Package notifycmd implements the hook-facing Pair notification command.
package notifycmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
)

type Runtime interface {
	Getenv(string) string
	SendNotification(binding, message string) error
}

type OSRuntime struct{}

func (OSRuntime) Getenv(key string) string { return os.Getenv(key) }
func (OSRuntime) SendNotification(binding, message string) error {
	return notifytransport.Send(binding, message)
}

func Run(args []string, rt Runtime, stderr io.Writer) int {
	message, ok := parseArgs(args, stderr)
	if !ok {
		return 2
	}
	if rt.Getenv("PAIR_TAG") == "" {
		warn(stderr, "PAIR_TAG not set — not running inside a pair session")
		return 0
	}
	binding := rt.Getenv("PAIR_PAIR_WRAP_PID_PATH")
	if binding == "" {
		warn(stderr, "PAIR_PAIR_WRAP_PID_PATH not set; restart the pair session")
		return 0
	}
	if err := rt.SendNotification(binding, message); err != nil {
		warn(stderr, fmt.Sprintf("notification broker unavailable: %v", err))
	}
	return 0
}

func parseArgs(args []string, stderr io.Writer) (string, bool) {
	if len(args) > 0 && args[0] == "--osc" {
		if len(args) < 2 {
			usage(stderr, "missing --osc value")
			return "", false
		}
		if args[1] != "9" && args[1] != "777" {
			usage(stderr, "unsupported --osc value "+args[1])
			return "", false
		}
		args = args[2:]
	} else if len(args) > 0 && strings.HasPrefix(args[0], "--osc=") {
		value := strings.TrimPrefix(args[0], "--osc=")
		if value != "9" && value != "777" {
			usage(stderr, "unsupported --osc value "+value)
			return "", false
		}
		args = args[1:]
	}
	if len(args) != 1 || args[0] == "" {
		usage(stderr, "missing message argument")
		return "", false
	}
	return args[0], true
}

func usage(stderr io.Writer, message string) {
	fmt.Fprintf(stderr, "pair-notify: %s\nusage: pair-notify [--osc 9|777] \"message\"\n", message)
}

func warn(stderr io.Writer, message string) {
	fmt.Fprintf(stderr, "pair-notify: %s\n", message)
}
