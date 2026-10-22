package agentfixture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

const retainedParentMode = "--testcouch-retained-parent"
const retainedChildMode = "--testcouch-retained-child"
const RetainedCount = 6

type RetainedProcess struct {
	PID        int    `json:"pid"`
	Generation string `json:"generation"`
}

// RetainedDirectory names a single immutable request's private fixture area.
func RetainedDirectory(control, id string) string { return filepath.Join(control, "retained-"+id) }

func retainedEnvironment(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "COUCH_SESSION_ID", "COUCH_CONNECTION_NONCE":
			continue
		}
		out = append(out, entry)
	}
	return out
}

func runRetained(ctx context.Context, c Config, request Command) Result {
	result := Result{ID: request.ID}
	dir := RetainedDirectory(c.ControlDir, request.ID)
	if err := os.Mkdir(dir, 0700); err != nil {
		result.Error = err.Error()
		result.ExitCode = -1
		return result
	}
	executable, err := os.Executable()
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = -1
		return result
	}
	// The helper's stdio is detached from the actor terminal. It inherits no
	// arbitrary program/argv authority: this executable implements both modes.
	cmd := exec.Command(executable, retainedParentMode, dir)
	cmd.Env = retainedEnvironment(os.Environ())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err = cmd.Start(); err != nil {
		result.Error = err.Error()
		result.ExitCode = -1
		return result
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case err = <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			err = <-done
		}
	}
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = -1
	}
	return result
}

func retainedRoot(dir string) error {
	root := os.Getenv("PAIR_TESTCOUCH_ROOT")
	run := os.Getenv("PAIR_TESTCOUCH_RUN_ID")
	if root == "" || run == "" || root != os.Getenv("COUCH_ISOLATED_ROOT") {
		return errors.New("retained helper requires private fleet ownership")
	}
	physical, err := pathscope.CanonicalFuture(dir)
	if err != nil || physical != dir || !pathscope.Within(root, dir) || dir == root {
		return errors.New("retained helper escapes private root")
	}
	raw, err := readBounded(filepath.Join(root, OwnershipFile), 256)
	if err != nil || strings.TrimSpace(string(raw)) != run {
		return errors.New("retained helper owner mismatch")
	}
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("retained helper directory unavailable")
	}
	return nil
}

// RetainedHelper handles only the fixed private descendant fixture modes. It
// never launches a caller-supplied executable and validates disposable ownership
// before creating processes or sidecars.
func RetainedHelper(args []string) (bool, int) {
	if len(args) == 0 || (args[0] != retainedParentMode && args[0] != retainedChildMode) {
		return false, 0
	}
	if len(args) != 2 && len(args) != 3 {
		return true, 2
	}
	dir := args[1]
	if err := retainedRoot(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 2
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()
	var err error
	if args[0] == retainedChildMode {
		if len(args) != 3 {
			return true, 2
		}
		index, e := strconv.Atoi(args[2])
		if e != nil || index < 0 || index >= RetainedCount {
			return true, 2
		}
		err = retainedChild(ctx, dir, index)
	} else {
		if len(args) != 2 {
			return true, 2
		}
		err = retainedParent(ctx, dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	return true, 0
}

func retainedChild(ctx context.Context, dir string, index int) error {
	pid := os.Getpid()
	generation := procutil.Identity(strconv.Itoa(pid))
	if generation == "" {
		return errors.New("retained child identity unavailable")
	}
	if err := atomicJSON(filepath.Join(dir, fmt.Sprintf("child-%d.json", index)), RetainedProcess{pid, generation}, true); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func retainedParent(ctx context.Context, dir string) (outcome error) {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	children := make([]*exec.Cmd, 0, RetainedCount)
	released := false
	defer func() {
		if !released {
			for _, cmd := range children {
				_ = cmd.Process.Kill()
			}
			for _, cmd := range children {
				_ = cmd.Wait()
			}
		}
	}()
	for i := 0; i < RetainedCount; i++ {
		cmd := exec.Command(executable, retainedChildMode, dir, strconv.Itoa(i))
		cmd.Env = retainedEnvironment(os.Environ())
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err = cmd.Start(); err != nil {
			return err
		}
		children = append(children, cmd)
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	ready := false
	for {
		if !ready {
			all := true
			for i, cmd := range children {
				raw, e := readBounded(filepath.Join(dir, fmt.Sprintf("child-%d.json", i)), 4096)
				var child RetainedProcess
				if e != nil || json.Unmarshal(raw, &child) != nil || child.PID != cmd.Process.Pid || child.Generation == "" {
					all = false
					break
				}
			}
			if all {
				if err = atomicJSON(filepath.Join(dir, "parent-ready.json"), RetainedProcess{os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid()))}, true); err != nil {
					return err
				}
				ready = true
			}
		}
		if ready {
			if _, e := os.Stat(filepath.Join(dir, "release-parent")); e == nil {
				released = true
				return nil
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
