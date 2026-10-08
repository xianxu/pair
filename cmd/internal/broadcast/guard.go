package broadcast

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// GuardSubcommand is the hidden couch subcommand that runs RunGuard.
const GuardSubcommand = "__broadcast-guard"

const (
	guardChildPrefix  = "broadcast-guard: child "
	defaultGuardGrace = 3 * time.Second
)

// RunGuard runs a tunnel process so that it can't outlive its owner. The
// owner (Couch) starts the guard holding a pipe to the guard's stdin; when
// that pipe closes, because the owner exited, crashed or was SIGKILLed, the
// guard terminates the child's whole process group (TERM, then KILL after the
// grace period) and removes the --remove directory. If the child exits on its
// own, the guard exits with its code and leaves cleanup to the living owner.
//
// args: [--remove DIR] [--grace DURATION] -- PROGRAM [ARGS...]
//
// The first stderr line is "broadcast-guard: child <pid>" (ParseGuardChild);
// the child's own stderr follows on the same stream. Exit code 2 is a usage
// error.
func RunGuard(args []string, stdin io.Reader, stderr io.Writer) int {
	remove, grace, argv, err := parseGuardArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "broadcast-guard: %v\n", err)
		return 2
	}
	child := exec.Command(argv[0], argv[1:]...)
	child.Stdout, child.Stderr = stderr, stderr
	// Its own group, so the stop reaches every process the tunnel starts.
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		fmt.Fprintf(stderr, "broadcast-guard: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "%s%d\n", guardChildPrefix, child.Process.Pid)

	exited := make(chan int, 1)
	go func() {
		err := child.Wait()
		var exit *exec.ExitError
		switch {
		case err == nil:
			exited <- 0
		case errors.As(err, &exit):
			exited <- exit.ExitCode()
		default:
			exited <- 1
		}
	}()
	ownerGone := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, stdin)
		close(ownerGone)
	}()

	select {
	case code := <-exited:
		return code
	case <-ownerGone:
	}
	group := -child.Process.Pid
	_ = syscall.Kill(group, syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(grace):
		_ = syscall.Kill(group, syscall.SIGKILL)
		<-exited
	}
	// Anything the child left in its group goes too.
	_ = syscall.Kill(group, syscall.SIGKILL)
	if remove != "" {
		_ = os.RemoveAll(remove)
	}
	return 0
}

func parseGuardArgs(args []string) (remove string, grace time.Duration, argv []string, err error) {
	grace = defaultGuardGrace
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--":
			if i+1 >= len(args) {
				return "", 0, nil, errors.New("no program after --")
			}
			return remove, grace, args[i+1:], nil
		case "--remove":
			if i+1 >= len(args) || args[i+1] == "" {
				return "", 0, nil, errors.New("--remove needs a directory")
			}
			i++
			remove = args[i]
		case "--grace":
			if i+1 >= len(args) {
				return "", 0, nil, errors.New("--grace needs a duration")
			}
			i++
			if grace, err = time.ParseDuration(args[i]); err != nil || grace <= 0 {
				return "", 0, nil, fmt.Errorf("--grace %q: not a positive duration", args[i])
			}
		default:
			return "", 0, nil, fmt.Errorf("unexpected argument %q (the program goes after --)", args[i])
		}
	}
	return "", 0, nil, errors.New("missing -- PROGRAM")
}

// ParseGuardChild reads the guard's first stderr line.
func ParseGuardChild(line string) (int, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), guardChildPrefix)
	if !ok {
		return 0, false
	}
	pid, err := strconv.Atoi(rest)
	return pid, err == nil && pid > 0
}
