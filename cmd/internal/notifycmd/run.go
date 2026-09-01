// Package notifycmd implements the hook-facing Pair notification command.
package notifycmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

type Runtime interface {
	Getenv(string) string
	ReadFile(string) ([]byte, error)
	WriteNonblocking(string, []byte) error
}

type OSRuntime struct {
	write func(fd int, p []byte) (int, error)
}

func (OSRuntime) Getenv(key string) string             { return os.Getenv(key) }
func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (rt OSRuntime) WriteNonblocking(path string, p []byte) error {
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	write := rt.write
	if write == nil {
		write = unix.Write
	}
	n, err := write(fd, p)
	if err != nil {
		return err
	}
	if n != len(p) {
		return io.ErrShortWrite
	}
	return nil
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
	sidecar := rt.Getenv("PAIR_OUTER_TTY_PATH")
	if sidecar == "" {
		warn(stderr, "PAIR_OUTER_TTY_PATH not set; restart the pair session")
		return 0
	}
	b, err := rt.ReadFile(sidecar)
	if err != nil {
		warn(stderr, fmt.Sprintf("%s missing; outer TTY not recorded", sidecar))
		return 0
	}
	tty := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	if tty == "" {
		warn(stderr, "recorded outer TTY is empty or stale")
		return 0
	}
	if err := rt.WriteNonblocking(tty, notifyosc.Encode(message)); err != nil {
		warn(stderr, fmt.Sprintf("outer TTY %q not writable (likely stale): %v", tty, err))
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
