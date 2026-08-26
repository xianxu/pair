package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// sessionQuiescenceOps is the stateful external-zellij boundary used when a
// deletion result is safety evidence rather than best-effort cleanup.
type sessionQuiescenceOps interface {
	SessionPresent(context.Context, string) (bool, error)
	SessionServerPIDs(context.Context, string) ([]int, error)
	DeleteSessionRecord(context.Context, string) error
	KillProcess(int) error
}

func quiesceZellijSession(session string, ops sessionQuiescenceOps, timeout, poll time.Duration) error {
	if session == "" {
		return errors.New("quiesce zellij session: empty session")
	}
	if ops == nil {
		return errors.New("quiesce zellij session: nil operations")
	}
	if timeout <= 0 {
		return errors.New("quiesce zellij session: positive timeout required")
	}
	if poll <= 0 {
		poll = 25 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	absentObservations := 0
	for {
		present, err := ops.SessionPresent(ctx, session)
		if err != nil {
			return fmt.Errorf("observe zellij session %q: %w", session, err)
		}
		servers, err := ops.SessionServerPIDs(ctx, session)
		if err != nil {
			return fmt.Errorf("observe zellij servers for %q: %w", session, err)
		}
		if !present && len(servers) == 0 {
			absentObservations++
			if absentObservations >= 2 {
				return nil
			}
		} else {
			absentObservations = 0
			if present {
				if err := ops.DeleteSessionRecord(ctx, session); err != nil {
					return fmt.Errorf("delete zellij session %q: %w", session, err)
				}
			}
			for _, pid := range servers {
				if err := ops.KillProcess(pid); err != nil {
					return fmt.Errorf("kill zellij server %d for %q: %w", pid, session, err)
				}
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("prove zellij session %q absent: %w", session, ctx.Err())
		case <-time.After(poll):
		}
	}
}

type osSessionQuiescenceOps struct{}

func (osSessionQuiescenceOps) SessionPresent(ctx context.Context, session string) (bool, error) {
	out, err := exec.CommandContext(ctx, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	present, _ := sessionRowState(string(out), session)
	return present, nil
}

func (osSessionQuiescenceOps) SessionServerPIDs(ctx context.Context, session string) ([]int, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	return zellijServerPIDs(string(out), session), nil
}

func (osSessionQuiescenceOps) DeleteSessionRecord(ctx context.Context, session string) error {
	out, err := exec.CommandContext(ctx, "zellij", "delete-session", session, "--force").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (osSessionQuiescenceOps) KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = process.Kill()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}

func zellijServerPIDs(raw, session string) []int {
	var result []int
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 4 || filepath.Base(fields[1]) != "zellij" || fields[2] != "--server" || filepath.Base(fields[3]) != session {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err == nil && pid > 0 {
			result = append(result, pid)
		}
	}
	return result
}
