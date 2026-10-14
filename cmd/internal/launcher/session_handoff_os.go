package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/zellijsession"
)

func terminalLaunchArgs(session, config, layout string, create bool) []string {
	if create {
		return []string{"--config-dir", config, "--new-session-with-layout", layout, "--session", session, "options", "--session-serialization", "false"}
	}
	return []string{"--config-dir", config, "attach", session}
}

func canonicalSocket(path string) (string, error) {
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(path)), nil
}

type terminalOSAttempt struct {
	rt                                                                 OSRuntime
	name, config, layout, socket, socketRoot, scope, tag, agent, nonce string
	create                                                             bool
}

func newTerminalOSAttempt(rt OSRuntime, name, config, layout string, create bool) (*terminalOSAttempt, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return nil, errors.New("invalid terminal session name")
	}
	root := os.Getenv("ZELLIJ_SOCKET_DIR")
	if root == "" {
		root = filepath.Join(os.TempDir(), "zellij-"+strconv.Itoa(os.Getuid()))
	}
	if !filepath.IsAbs(root) {
		return nil, errors.New("Zellij socket directory must be absolute")
	}
	if err := os.MkdirAll(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	a := &terminalOSAttempt{rt: rt, name: name, config: config, layout: layout, create: create, socketRoot: root, socket: filepath.Join(root, "contract_version_1", name), scope: os.Getenv("PAIR_SCOPE_KEY"), tag: os.Getenv("PAIR_TAG"), agent: os.Getenv("PAIR_AGENT"), nonce: os.Getenv("PAIR_LAUNCH_NONCE")}
	if create && a.tag == "" {
		return nil, errors.New("terminal thread identity unavailable")
	}
	return a, nil
}
func (a *terminalOSAttempt) Acquire(ctx context.Context) (io.Closer, error) {
	p, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return nil, err
	}
	path, err := p.TerminalLease(a.socket)
	if err != nil {
		return nil, err
	}
	return AcquireTerminalLease(ctx, path)
}
func (a *terminalOSAttempt) Check(ctx context.Context) error {
	if !a.create {
		return requireListedLiveTerminal(ctx, a.name, listTerminalSessions)
	}
	servers, err := (osSessionOwnerIO{}).SessionServers(ctx, a.name)
	if err != nil {
		return err
	}
	if len(servers) != 0 {
		return errors.New("terminal name still has a live or contested server → relaunch")
	}
	if _, err = os.Lstat(a.socket); !errors.Is(err, os.ErrNotExist) {
		return errors.New("terminal socket is occupied or unreadable → relaunch")
	}
	// A still-live agent is authority against a second launch even if its
	// server is gone. PID reuse is conservatively refused, never killed here.
	owner, err := artifactpath.NewStorageOwner(a.rt.globalDataDir(), a.scope, a.tag)
	if err != nil {
		return err
	}
	for _, agent := range AgentInventory() {
		path := AgentReadyPath(owner.Directory(), a.tag, agent)
		raw, e := osfs.ReadRegularMetadata(ctx, path, maxAgentReadyBytes)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return fmt.Errorf("cannot inspect retained agent readiness → relaunch: %w", e)
		}
		ready, e := readiness.Decode(string(raw))
		if e != nil || ready.Tag != a.tag || ready.Agent != agent {
			return errors.New("retained agent readiness is invalid → relaunch")
		}
		e = syscall.Kill(ready.PID, 0)
		if e == nil || errors.Is(e, syscall.EPERM) {
			return errors.New("thread still has a live agent → relaunch")
		}
		if !errors.Is(e, syscall.ESRCH) {
			return fmt.Errorf("retained agent presence is unknown → relaunch: %w", e)
		}
	}

	return nil
}

type terminalOSClient struct{ cmd *exec.Cmd }

func (c terminalOSClient) Wait() HandoffExit {
	err := c.cmd.Wait()
	if c.cmd.ProcessState != nil {
		return HandoffExit{Code: c.cmd.ProcessState.ExitCode()}
	}
	return HandoffExit{Code: 1, Err: err}
}
func (a *terminalOSAttempt) Start(ctx context.Context) (HandoffClient, error) {
	cmd := killOnCancel(exec.CommandContext(ctx, "zellij", terminalLaunchArgs(a.name, a.config, a.layout, a.create)...))
	cmd.Env = append(os.Environ(), "ZELLIJ_SOCKET_DIR="+a.socketRoot)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return terminalOSClient{cmd}, nil
}
func (a *terminalOSAttempt) Admit(ctx context.Context) error {
	if !a.create {
		return ctx.Err()
	}
	ticker := time.NewTicker(birthPoll)
	defer ticker.Stop()
	var last error
	for {
		if err := a.observeAdmission(ctx); err == nil {
			return requireListedLiveTerminal(ctx, a.name, listTerminalSessions)
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("terminal admission unverified: %w", errors.Join(last, ctx.Err()))
		case <-ticker.C:
		}
	}
}

func (a *terminalOSAttempt) observeAdmission(ctx context.Context) error {
	if !a.create {
		// Check already required a positive listing under the lease. Do not
		// replay the interactive client or introduce another admission policy.
		return ctx.Err()
	}
	// Do not send Zellij a query before its first pane runs: an early CLI
	// connection can kill a server still initializing its first client (#288).
	storageOwner, err := artifactpath.NewStorageOwner(a.rt.globalDataDir(), a.scope, a.tag)
	if err != nil {
		return err
	}
	readyBytes, err := osfs.ReadRegularMetadata(ctx, AgentReadyPath(storageOwner.Directory(), a.tag, a.agent), maxAgentReadyBytes)
	if err != nil {
		return err
	}
	ready, err := readiness.Decode(string(readyBytes))
	if err != nil {
		return err
	}
	if ready.Tag != a.tag || ready.Agent != a.agent || ready.Session != a.name || ready.Nonce != a.nonce {
		return errors.New("waiting for this launch's agent readiness")
	}
	return nil
}

func (r OSRuntime) runTerminalHandoff(ctx context.Context, name, config, layout string, create bool) (int, error) {
	a, err := newTerminalOSAttempt(r, name, config, layout, create)
	if err != nil {
		return 1, err
	}
	return CoordinateTerminalHandoff(ctx, a, birthBound)
}

const maxAgentReadyBytes int64 = 8192

func listTerminalSessions(ctx context.Context) ([]byte, error) {
	return (ZellijSource{}).runContext(ctx, "list-sessions", "--no-formatting")
}

// Retry observation once, never the interactive attachment. A failed query is
// not evidence even if it produced partial output containing the desired name.
func requireListedLiveTerminal(ctx context.Context, name string, query func(context.Context) ([]byte, error)) error {
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, err := query(ctx)
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		if err == nil {
			present, exited := zellijsession.RowState(string(raw), name)
			if present && !exited {
				return nil
			}
			err = fmt.Errorf("terminal %q is not listed live", name)
		}
		last = err
	}
	return fmt.Errorf("cannot attach terminal: %w", last)
}
