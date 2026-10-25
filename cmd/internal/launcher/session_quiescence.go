package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// sessionQuiescenceOps observes liveness and signals only exact authorized
// server generations. Saved layouts are deliberately outside this interface.
type sessionQuiescenceOps interface {
	SessionPresent(context.Context, string) (bool, error)
	SessionServers(context.Context, string) ([]SessionServerIdentity, error)
	KillServer(SessionServerIdentity) error
}

// Ordinary lifecycle cleanup may authorize at most one initial server. A
// contested name requires explicit attribution by a separate recovery caller.
func quiesceZellijSession(parent context.Context, session string, ops sessionQuiescenceOps, timeout, poll time.Duration) error {
	ctx, cancel, err := quiescenceContext(parent, session, ops, timeout)
	if err != nil {
		return err
	}
	defer cancel()
	authorized, err := ops.SessionServers(ctx, session)
	if err != nil {
		return fmt.Errorf("observe initial zellij servers for %q: %w", session, err)
	}
	if len(authorized) > 1 {
		return fmt.Errorf("zellij session %q has contested server ownership", session)
	}
	return quiesceAuthorizedSession(ctx, session, authorized, ops, timeout, poll)
}

// quiesceAuthorizedSession never expands the caller's immutable authorization.
// Recovery may supply several positively attributed generations; ordinary
// cleanup uses quiesceZellijSession's single-server admission above.
func quiesceAuthorizedSession(parent context.Context, session string, authorized []SessionServerIdentity, ops sessionQuiescenceOps, timeout, poll time.Duration) error {
	ctx, cancel, err := quiescenceContext(parent, session, ops, timeout)
	if err != nil {
		return err
	}
	defer cancel()
	allowed := make(map[SessionServerIdentity]bool, len(authorized))
	pids := make(map[int]bool, len(authorized))
	for _, server := range authorized {
		if !validQuiescenceServer(server, session) || pids[server.PID] {
			return fmt.Errorf("invalid authorized zellij server identity for %q", session)
		}
		allowed[server] = true
		pids[server.PID] = true
	}
	if poll <= 0 {
		poll = 25 * time.Millisecond
	}
	absentObservations := 0
	var attemptErr error
	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(fmt.Errorf("prove zellij session %q absent: %w", session, err), attemptErr)
		}
		present, err := ops.SessionPresent(ctx, session)
		if err != nil {
			return fmt.Errorf("observe zellij session %q: %w", session, err)
		}
		servers, err := ops.SessionServers(ctx, session)
		if err != nil {
			return fmt.Errorf("observe zellij servers for %q: %w", session, err)
		}
		// Validate the whole observation before signalling any member of it.
		for _, server := range servers {
			if !validQuiescenceServer(server, session) || !allowed[server] {
				return fmt.Errorf("zellij session %q server ownership changed; refusing quiescence", session)
			}
		}
		if !present && len(servers) == 0 {
			absentObservations++
			if absentObservations >= 2 {
				return nil
			}
		} else {
			absentObservations = 0
			for _, server := range servers {
				if err := ctx.Err(); err != nil {
					return err
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

func validQuiescenceServer(server SessionServerIdentity, session string) bool {
	return server.PID > 0 && server.Identity != "" && server.Session == session && filepath.IsAbs(server.Socket) && filepath.Clean(server.Socket) == server.Socket && filepath.Base(server.Socket) == session
}
func quiescenceContext(parent context.Context, session string, ops sessionQuiescenceOps, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if parent == nil {
		return nil, nil, errors.New("quiesce zellij session: nil context")
	}
	if session == "" {
		return nil, nil, errors.New("quiesce zellij session: empty session")
	}
	if ops == nil {
		return nil, nil, errors.New("quiesce zellij session: nil operations")
	}
	if timeout <= 0 {
		return nil, nil, errors.New("quiesce zellij session: positive timeout required")
	}
	if err := parent.Err(); err != nil {
		return nil, nil, fmt.Errorf("quiesce zellij session: %w", err)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, nil
}

type osSessionQuiescenceOps struct {
	processes       procutil.Source
	processIdentity func(string) string
	processCommand  func(string) string
	killProcess     func(int) error
}

func newOSSessionQuiescenceOps(sources ...procutil.Source) osSessionQuiescenceOps {
	var source procutil.Source
	if len(sources) > 0 {
		source = sources[0]
	}
	return osSessionQuiescenceOps{
		processes:       source,
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
	out, err := (ZellijSource{explicitRecovery: true}).runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return false, err
	}
	present, exited, err := sessionListingState(out, session)
	return present && !exited, err
}

func (o osSessionQuiescenceOps) SessionServers(ctx context.Context, session string) ([]SessionServerIdentity, error) {
	rows, err := o.processes.SnapshotNamed(ctx, "zellij")
	if err != nil {
		return nil, err
	}
	servers, err := serverProcesses(rows)
	if err != nil {
		return nil, err
	}
	var result []SessionServerIdentity
	for _, server := range servers {
		if server.Session == session {
			result = append(result, server)
		}
	}
	return result, nil
}

func (o osSessionQuiescenceOps) KillServer(server SessionServerIdentity) error {
	pid := strconv.Itoa(server.PID)
	identity := o.identity()
	command := o.command()
	// Reauthorize on start token, exact argv, then start token once more. The
	// second identity read closes the command-inspection interval; a recycled
	// PID or a process that execs away from the exact server is never signalled.
	if !validQuiescenceServer(server, server.Session) || identity(pid) != server.Identity {
		return nil
	}
	fields := strings.Fields(command(pid))
	if len(fields) != 3 || filepath.Base(fields[0]) != "zellij" || fields[1] != "--server" || fields[2] != server.Socket || identity(pid) != server.Identity {
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
	for _, server := range ParseServerProcesses(raw) {
		if server.Session == session {
			result = append(result, server.PID)
		}
	}
	return result
}

func isExactZellijServerCommand(command, session string) bool {
	fields := strings.Fields(command)
	return len(fields) == 3 && filepath.Base(fields[0]) == "zellij" && fields[1] == "--server" && filepath.Base(fields[2]) == session
}
