package agentcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

type Runtime interface {
	ReadFile(string) ([]byte, error)
	Signal(int, syscall.Signal) error
}

func RunRestart(args []string, getenv func(string) string, rt Runtime, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: pair agent restart")
		return 2
	}
	tag := getenv("PAIR_TAG")
	if tag == "" {
		fmt.Fprintln(stderr, "pair agent restart: PAIR_TAG is not set")
		return 1
	}
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		dataDir = adapt.DataDir()
	}
	raw, err := rt.ReadFile(filepath.Join(dataDir, "pair-wrap-pid-"+tag))
	if err != nil {
		fmt.Fprintf(stderr, "pair agent restart: read wrapper pid: %v\n", err)
		return 1
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		fmt.Fprintln(stderr, "pair agent restart: invalid wrapper pid")
		return 1
	}
	if err := rt.Signal(pid, syscall.SIGUSR2); err != nil {
		fmt.Fprintf(stderr, "pair agent restart: signal wrapper: %v\n", err)
		return 1
	}
	return 0
}

type OSRuntime struct{}

func (OSRuntime) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSRuntime) Signal(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}
