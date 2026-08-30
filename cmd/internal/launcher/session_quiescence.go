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

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type sessionServerIdentity struct {
	PID      int
	Identity string
	Session  string
}

// sessionQuiescenceOps is the stateful external-zellij boundary used when a
// deletion result is safety evidence rather than best-effort cleanup.
type sessionQuiescenceOps interface {
	SessionPresent(context.Context, string) (bool, error)
	SessionServers(context.Context, string) ([]sessionServerIdentity, error)
	DeleteSessionRecord(context.Context, string) error
	KillServer(sessionServerIdentity) error
}

func quiesceZellijSession(parent context.Context, session string, ops sessionQuiescenceOps, timeout, poll time.Duration) error {
	if parent == nil {
		return errors.New("quiesce zellij session: nil context")
	}
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
	if err := parent.Err(); err != nil {
		return fmt.Errorf("quiesce zellij session: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	absentObservations := 0
	var attemptErr error
	for {
		present, err := ops.SessionPresent(ctx, session)
		if err != nil {
			attemptErr = errors.Join(attemptErr, fmt.Errorf("observe zellij session %q: %w", session, err))
			present = true // absence is unproven; keep retrying under ownership
		}
		servers, err := ops.SessionServers(ctx, session)
		if err != nil {
			attemptErr = errors.Join(attemptErr, fmt.Errorf("observe zellij servers for %q: %w", session, err))
			servers = []sessionServerIdentity{{}} // non-empty means unproven
		}
		if err == nil && !present && len(servers) == 0 {
			absentObservations++
			if absentObservations >= 2 {
				return nil
			}
		} else {
			absentObservations = 0
			if present {
				if err := ops.DeleteSessionRecord(ctx, session); err != nil {
					// Zellij can report an error after actually deleting an active
					// session. The status is retained for a timeout diagnostic, but
					// only subsequent observed absence decides success.
					attemptErr = errors.Join(attemptErr, fmt.Errorf("delete zellij session %q: %w", session, err))
				}
			}
			for _, server := range servers {
				if server.PID <= 0 {
					continue
				}
				if err := ops.KillServer(server); err != nil {
					attemptErr = errors.Join(attemptErr, fmt.Errorf("kill zellij server %d/%q for %q: %w", server.PID, server.Identity, session, err))
				}
			}
		}

		select {
		case <-ctx.Done():
			return errors.Join(fmt.Errorf("prove zellij session %q absent: %w", session, ctx.Err()), attemptErr)
		case <-time.After(poll):
		}
	}
}

type osSessionQuiescenceOps struct {
	processIdentity func(string) string
	processCommand  func(string) string
	killProcess     func(int) error
}

func newOSSessionQuiescenceOps() osSessionQuiescenceOps {
	return osSessionQuiescenceOps{
		processIdentity: procutil.Identity,
		processCommand:  procutil.Command,
		killProcess: func(pid int) error {
			process, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			err = process.Kill()
			if errors.Is(err, os.ErrProcessDone) {
				return nil
			}
			return err
		},
	}
}

func (osSessionQuiescenceOps) SessionPresent(ctx context.Context, session string) (bool, error) {
	out, err := exec.CommandContext(ctx, "zellij", "list-sessions", "--no-formatting").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	present, _ := sessionRowState(string(out), session)
	return present, nil
}

func (o osSessionQuiescenceOps) SessionServers(ctx context.Context, session string) ([]sessionServerIdentity, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	var result []sessionServerIdentity
	for _, pid := range zellijServerPIDs(string(out), session) {
		identity := o.identity()(strconv.Itoa(pid))
		if identity == "" {
			// The process may have exited between snapshots, but treating an
			// observed exact server as absent would be a false proof. Retry the
			// complete observation instead.
			return nil, fmt.Errorf("process identity unavailable for zellij server %d", pid)
		}
		result = append(result, sessionServerIdentity{PID: pid, Identity: identity, Session: session})
	}
	return result, nil
}

func (osSessionQuiescenceOps) DeleteSessionRecord(ctx context.Context, session string) error {
	out, err := exec.CommandContext(ctx, "zellij", "delete-session", session, "--force").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (o osSessionQuiescenceOps) KillServer(server sessionServerIdentity) error {
	pid := strconv.Itoa(server.PID)
	identity := o.identity()
	command := o.command()
	// Reauthorize on start token, exact argv, then start token once more. The
	// second identity read closes the command-inspection interval; a recycled
	// PID or a process that execs away from the exact server is never signalled.
	if identity(pid) != server.Identity || !isExactZellijServerCommand(command(pid), server.Session) || identity(pid) != server.Identity {
		return nil
	}
	return o.killer()(server.PID)
}

func (o osSessionQuiescenceOps) identity() func(string) string {
	if o.processIdentity != nil {
		return o.processIdentity
	}
	return procutil.Identity
}

func (o osSessionQuiescenceOps) command() func(string) string {
	if o.processCommand != nil {
		return o.processCommand
	}
	return procutil.Command
}

func (o osSessionQuiescenceOps) killer() func(int) error {
	if o.killProcess != nil {
		return o.killProcess
	}
	return newOSSessionQuiescenceOps().killProcess
}

func zellijServerPIDs(raw, session string) []int {
	var result []int
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 4 || !isExactZellijServerCommand(strings.Join(fields[1:], " "), session) {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err == nil && pid > 0 {
			result = append(result, pid)
		}
	}
	return result
}

func isExactZellijServerCommand(command, session string) bool {
	fields := strings.Fields(command)
	return len(fields) == 3 && filepath.Base(fields[0]) == "zellij" && fields[1] == "--server" && filepath.Base(fields[2]) == session
}
