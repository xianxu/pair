//go:build unix

package couchschedule

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"golang.org/x/sys/unix"
)

type groupUncertain struct {
	pid int
	err error
}

func (e *groupUncertain) Error() string {
	return fmt.Sprintf("work group %d cleanup uncertain: %v", e.pid, e.err)
}
func (e *groupUncertain) Unwrap() error { return e.err }

func ownedGroupCommand(ctx context.Context, program, dir string, env, args []string, stdout, stderr io.Writer, preserveSuccess bool) (int, error) {
	if err := ctx.Err(); err != nil {
		return -1, err
	}
	cmd := exec.Command(program, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var processErr error
	joined := false
	cancelled := false
	select {
	case processErr = <-done:
		joined = true
	case <-ctx.Done():
		cancelled = true
	}
	if preserveSuccess && joined && processErr == nil {
		return 0, nil
	}
	gone, probeErr := ownedGroupGone(pid)
	if !gone {
		_ = unix.Kill(-pid, unix.SIGTERM)
		grace := time.NewTimer(time.Second)
		for !gone && probeErr == nil {
			select {
			case <-grace.C:
				goto kill
			case <-time.After(10 * time.Millisecond):
				gone, probeErr = ownedGroupGone(pid)
			}
		}
		if !grace.Stop() {
			select {
			case <-grace.C:
			default:
			}
		}
	}
kill:
	if !gone {
		_ = unix.Kill(-pid, unix.SIGKILL)
	}
	if !joined {
		select {
		case processErr = <-done:
			joined = true
		case <-time.After(time.Second):
			return -1, &groupUncertain{pid, errors.New("direct child reaping unconfirmed")}
		}
	}
	end := time.Now().Add(time.Second)
	for {
		gone, probeErr = ownedGroupGone(pid)
		if gone {
			break
		}
		if time.Now().After(end) {
			return -1, &groupUncertain{pid, probeErr}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if cancelled {
		return -1, ctx.Err()
	}
	exit := 0
	if cmd.ProcessState != nil {
		exit = cmd.ProcessState.ExitCode()
	}
	if exit < 0 {
		exit = 1
	}
	return exit, processErr
}

func ownedGroupGone(pid int) (bool, error) {
	err := unix.Kill(-pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return true, nil
	}
	if err == nil {
		return false, nil
	}
	if errors.Is(err, unix.EPERM) {
		members, e := procutil.ProcessGroupMembers(pid)
		return e == nil && !members, e
	}
	return false, err
}

func executeOwned(ctx context.Context, r ExecutionRequest, stream *os.File) (ExecutionResult, func()) {
	result := ExecutionResult{Exit: -1, Outcome: "failed"}
	release, err := acquireExecutionPermits(ctx, r)
	if err != nil {
		result.Error = err.Error()
		setCancellationOutcome(ctx, &result)
		return result, nil
	}
	var artifact Artifact
	closeResources := func() {
		if artifact.Path != "" {
			_ = artifact.Close()
		}
		release()
	}
	finish := func(err error) (ExecutionResult, func()) {
		if err != nil {
			result.Error = err.Error()
		}
		setCancellationOutcome(ctx, &result)
		var unknown *groupUncertain
		if errors.As(err, &unknown) {
			result.Outcome = "uncertain"
			return result, func() {
				for {
					gone, _ := ownedGroupGone(unknown.pid)
					if gone {
						closeResources()
						return
					}
					_ = unix.Kill(-unknown.pid, unix.SIGKILL)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
		closeResources()
		return result, nil
	}
	heavyAcquired := false
	if r.Spec.Heavy {
		python, err := exec.LookPath("python3")
		if err != nil {
			return finish(err)
		}
		var out, stderr buildCapture
		_, err = ownedGroupCommand(ctx, python, r.HubRoot, executionEnvironment(r, false), []string{r.TestLockHelper, "acquire", "--watch", strconv.Itoa(os.Getpid()), "--slots", "1"}, &out, &stderr, true)
		if err != nil {
			return finish(fmt.Errorf("heavy admission: %w: %s", err, stderr.String()))
		}
		if out.overflow || strings.TrimSpace(out.String()) != "acquired" {
			return finish(fmt.Errorf("heavy admission refused: %s", out.String()))
		}
		heavyAcquired = true
	}
	goBinary := r.GoBinary
	if goBinary == "" {
		goBinary = "go"
	}
	goBinary, err = exec.LookPath(goBinary)
	if err != nil {
		return finish(err)
	}
	builder := Builder{CacheDir: r.CacheDir, GoBinary: goBinary}
	builder.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		// Builder supplies fixed Go settings. Strip inherited queue bypasses
		// and append only the acquired controller's nested-suite capability.
		var clean []string
		for _, entry := range env {
			key, _, _ := strings.Cut(entry, "=")
			if strings.HasPrefix(key, "WF_TEST_") || strings.HasPrefix(key, "WF_SDLC_QUEUE_") {
				continue
			}
			clean = append(clean, entry)
		}
		for _, entry := range executionEnvironment(r, heavyAcquired) {
			key, _, _ := strings.Cut(entry, "=")
			if strings.HasPrefix(key, "WF_TEST_") || strings.HasPrefix(key, "WF_SDLC_QUEUE_") {
				clean = append(clean, entry)
			}
		}
		var out, stderr buildCapture
		_, err := ownedGroupCommand(ctx, goBinary, dir, clean, args, &out, &stderr, false)
		if out.overflow || stderr.overflow {
			return nil, errors.Join(err, errors.New("build command output exceeds bound"))
		}
		if err != nil {
			return nil, fmt.Errorf("build command: %w: %s", err, stderr.String())
		}
		return out.Bytes(), nil
	}
	artifact, err = builder.Build(ctx, r.HubRoot, r.Task)
	if err != nil {
		return finish(err)
	}
	result.BuildKey = artifact.Key
	var gitUnknown error
	err = validateExecutionAuthority(ctx, Catalog{Git: ownedExecutionGit{request: r, heavy: heavyAcquired, unknown: &gitUnknown}}, r)
	if gitUnknown != nil {
		return finish(gitUnknown)
	}
	if err != nil {
		return finish(err)
	}
	output := operationalrun.NewOutputBuffer()
	writer := &ownerStreamWriter{output: output, file: stream, fd: ownerOutputFD}
	defer writer.close()
	exit, err := ownedGroupCommand(ctx, artifact.Path, r.HubRoot, executionEnvironment(r, heavyAcquired), r.Args, writer, writer, false)
	result.Output = output.Bytes()
	result.OutputBytes = output.TotalBytes()
	result.Exit = exit
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return finish(err)
		}
	}
	switch exit {
	case 0:
		result.Outcome = "quiet"
	case 10:
		result.Outcome = "wake"
	default:
		result.Outcome = "failed"
	}
	return finish(nil)
}

const ownerOutputFD = 3

func inheritedOwnerOutput(enabled bool) (*os.File, error) {
	if !enabled {
		return nil, nil
	}
	f := os.NewFile(ownerOutputFD, "schedule-output")
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		f.Close()
		return nil, errors.New("execution output requires a pipe")
	}
	unix.CloseOnExec(ownerOutputFD)
	if err = unix.SetNonblock(ownerOutputFD, true); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// The nonblocking FD carries only a contiguous prefix. A full or slow pipe
// ends live streaming while retained collection keeps draining.
type ownerStreamWriter struct {
	mu      sync.Mutex
	output  *operationalrun.OutputBuffer
	file    *os.File
	fd      int
	prefix  int
	stopped bool
}

func (w *ownerStreamWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.output.Write(p)
	if !w.stopped && w.file != nil && w.prefix < operationalrun.OutputLimit/2 {
		piece := p[:min(len(p), operationalrun.OutputLimit/2-w.prefix)]
		// File.Fd may restore blocking mode; retain the already-admitted raw
		// descriptor instead of querying it in this nonblocking writer.
		n, err := unix.Write(w.fd, piece)
		if n > 0 {
			w.prefix += n
		}
		if err != nil || n != len(piece) || w.prefix == operationalrun.OutputLimit/2 {
			w.stopped = true
			_ = w.file.Close()
		}
	}
	return len(p), nil
}
func (w *ownerStreamWriter) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	if w.file != nil {
		_ = w.file.Close()
	}
}

func validateExecutionAuthority(ctx context.Context, catalog Catalog, r ExecutionRequest) error {
	rows, err := catalog.Discover(ctx, []string{r.HubRoot})
	if err != nil {
		return fmt.Errorf("task authority changed: %w", err)
	}
	for _, d := range rows {
		if d.Key == (TaskKey{Hub: r.HubRoot, Task: r.Task}) {
			if !d.Allowed || d.Revision != r.Revision || d.Spec != r.Spec {
				return fmt.Errorf("task authority changed: %s", d.Reason)
			}
			return ctx.Err()
		}
	}
	return errors.New("task authority changed: declaration removed")
}

type ownedExecutionGit struct {
	request ExecutionRequest
	heavy   bool
	unknown *error
}

func (g ownedExecutionGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	var out, stderr buildCapture
	_, err = ownedGroupCommand(ctx, git, dir, executionEnvironment(g.request, g.heavy), args, &out, &stderr, false)
	var uncertain *groupUncertain
	if errors.As(err, &uncertain) && g.unknown != nil {
		*g.unknown = err
	}
	if out.overflow || stderr.overflow {
		return "", errors.Join(err, errors.New("Git identity output exceeds bound"))
	}
	if err != nil {
		return "", fmt.Errorf("Git identity: %w: %s", err, stderr.String())
	}
	return out.String(), nil
}

func setCancellationOutcome(ctx context.Context, result *ExecutionResult) {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.Outcome = "timeout"
	} else if ctx.Err() != nil {
		result.Outcome = "cancelled"
	}
}
