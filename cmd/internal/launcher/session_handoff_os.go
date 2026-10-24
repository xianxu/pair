package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
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
	creator                                                            *zellijquery.CreatorGuard
	panePublished                                                      bool
}

func newTerminalOSAttempt(rt OSRuntime, name, config, layout string, create bool) (*terminalOSAttempt, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return nil, errors.New("invalid terminal session name")
	}
	root, err := ResolveZellijSocketRoot(os.Getenv("ZELLIJ_SOCKET_DIR"), os.TempDir(), os.Getenv("XDG_RUNTIME_DIR"), runtime.GOOS, os.Getuid())
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		return nil, err
	}
	// Zellij uses this spelling as a Unix socket pathname. Expanding symlinks
	// can exceed the platform's socket limit even when its existing server's
	// original path fits. Only the lease identity needs canonicalization.
	transportRoot := root
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	a := &terminalOSAttempt{rt: rt, name: name, config: config, layout: layout, create: create, socketRoot: transportRoot, socket: filepath.Join(root, "contract_version_1", name), scope: os.Getenv("PAIR_SCOPE_KEY"), tag: os.Getenv("PAIR_TAG"), agent: os.Getenv("PAIR_AGENT"), nonce: os.Getenv("PAIR_LAUNCH_NONCE")}
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
	session, err := AcquireTerminalLease(ctx, path)
	if err != nil || !a.create {
		return session, err
	}
	a.creator, err = zellijquery.AcquireCreator(ctx, a.terminalEnv(), a.creatorIdentity())
	if err != nil {
		return nil, errors.Join(err, session.Close())
	}
	return terminalCreatorLease{creator: a.creator, session: session}, nil
}

type terminalCreatorLease struct {
	creator io.Closer
	session io.Closer
}

func (l terminalCreatorLease) Close() error {
	creatorErr := l.creator.Close()
	return errors.Join(creatorErr, l.session.Close())
}

func (a *terminalOSAttempt) terminalEnv() []string {
	return append(os.Environ(), "ZELLIJ_SOCKET_DIR="+a.socketRoot)
}

func (a *terminalOSAttempt) creatorIdentity() zellijquery.ReadyIdentity {
	return zellijquery.ReadyIdentity{Namespace: filepath.Dir(filepath.Dir(a.socket)), Scope: a.scope, Tag: a.tag, Agent: a.agent, Session: a.name, Nonce: a.nonce}
}

func (a *terminalOSAttempt) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.create {
		return requireListedLiveTerminal(ctx, a.name, listTerminalSessions, a.rt.liveSessionServers)
	}
	if a.creator == nil {
		return errors.New("terminal creator exclusion is unavailable")
	}
	return a.creator.Preflight(ctx, a.checkCreate)
}

func (a *terminalOSAttempt) checkCreate(ctx context.Context) error {
	servers, err := a.rt.liveSessionServers(ctx, a.name)
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
	if a.create {
		if a.creator == nil {
			return nil, errors.New("terminal creator exclusion is unavailable")
		}
		if err := a.creator.Prepare(ctx); err != nil {
			return nil, err
		}
	}
	cmd := killOnCancel(exec.CommandContext(ctx, "zellij", terminalLaunchArgs(a.name, a.config, a.layout, a.create)...))
	cmd.Env = a.terminalEnv()
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
	return admitTerminalAfterReady(ctx, ctx, a.observeCreateAdmission, func(queryCtx context.Context) error {
		return requireListedLiveTerminal(queryCtx, a.name, listTerminalSessions, a.rt.liveSessionServers)
	})
}

// Exact first-pane evidence releases native namespace exclusion independently
// of the agent's later readiness. Native admission still waits for AgentReady.
func (a *terminalOSAttempt) observeCreateAdmission(ctx context.Context) error {
	if !a.panePublished {
		if a.creator == nil {
			return errors.New("terminal creator exclusion is unavailable")
		}
		owner, err := artifactpath.NewStorageOwner(a.rt.globalDataDir(), a.scope, a.tag)
		if err != nil {
			return err
		}
		path, err := panebirth.Evidence(owner.Directory(), a.tag, a.agent)
		if err != nil {
			return err
		}
		raw, err := osfs.ReadRegularMetadata(ctx, path, panebirth.MaxRecordBytes)
		if err != nil {
			return err
		}
		pane, err := panebirth.DecodeRecord(raw)
		if err != nil {
			return err
		}
		if !pane.Identity.Equal(a.creatorIdentity()) {
			return errors.New("waiting for this launch's first pane")
		}
		if err := a.creator.PublishReady(ctx, pane.Identity); err != nil {
			return err
		}
		a.panePublished = true
	}
	return a.observeAdmission(ctx)
}

// Readiness consumes the selected startup window. Only after exact local evidence
// does native admission get its short observation budget, capped by startup.
func admitTerminalAfterReady(readinessCtx, lifetimeCtx context.Context, ready func(context.Context) error, query func(context.Context) error) error {
	ticker := time.NewTicker(birthPoll)
	defer ticker.Stop()
	var last error
	for {
		if err := readinessCtx.Err(); err != nil {
			return fmt.Errorf("terminal admission unverified: %w", errors.Join(last, err))
		}
		if err := ready(readinessCtx); err == nil {
			queryCtx, cancel := zellijreply.AdmissionContext(lifetimeCtx)
			defer cancel()
			if err := queryCtx.Err(); err != nil {
				return err
			}
			err := query(queryCtx)
			return errors.Join(err, queryCtx.Err())
		} else {
			last = err
		}
		select {
		case <-readinessCtx.Done():
			return fmt.Errorf("terminal admission unverified: %w", errors.Join(last, readinessCtx.Err()))
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
	queryCtx, cancel := zellijreply.AdmissionContext(ctx)
	defer cancel()
	return r.runTerminalHandoffContext(ctx, nil, queryCtx, name, config, layout, create)
}

func (r OSRuntime) runTerminalHandoffContext(lifetimeCtx, startupCtx, queryCtx context.Context, name, config, layout string, create bool) (int, error) {
	if err := queryCtx.Err(); err != nil {
		return 1, err
	}
	if err := lifetimeCtx.Err(); err != nil {
		return 1, err
	}
	startupCtx, stopStartup, startupErr := terminalStartupContext(lifetimeCtx, startupCtx)
	if startupErr != nil {
		return 1, startupErr
	}
	defer stopStartup()
	a, err := newTerminalOSAttempt(r, name, config, layout, create)
	if err != nil {
		return 1, err
	}
	return runTerminalHandoffAttempt(lifetimeCtx, startupCtx, queryCtx, a, create)
}

const maxAgentReadyBytes int64 = 8192

func listTerminalSessions(ctx context.Context) ([]byte, error) {
	return (ZellijSource{}).runContext(ctx, "list-sessions", "--no-formatting")
}

// ErrListingOmittedLiveServer is a listing that leaves out a session whose
// exact server is independently live (pair#692). Zellij 0.45.1 drops a
// session from list-sessions when its ConnStatus probe loses the reply to a
// concurrent connection (zellij-org/zellij#5270), so the omission is unknown,
// not absence. Callers refuse on it as on any unknown; they never retry.
var ErrListingOmittedLiveServer = errors.New("terminal listing omitted a live server; attach unverified")

// Each existing handoff fence consumes one fresh positive observation. Unknown
// is final for that fence, never a reason to try again inside this request.
// Only a listing that names the session live admits. An omission is checked
// against servers, process evidence, to name which refusal it is.
func requireListedLiveTerminal(ctx context.Context, name string, query func(context.Context) ([]byte, error), servers func(context.Context, string) ([]SessionServerIdentity, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := query(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("cannot attach terminal: %w", err)
	}
	present, exited, err := sessionListingState(raw, name)
	if err != nil {
		return fmt.Errorf("cannot attach terminal: %w", err)
	}
	if exited {
		return fmt.Errorf("terminal %q is not listed live", name)
	}
	if !present {
		live, err := servers(ctx, name)
		if err != nil || ctx.Err() != nil {
			return fmt.Errorf("terminal %q is not listed, and whether its server lives is unknown: %w", name, errors.Join(err, ctx.Err()))
		}
		if len(live) != 0 {
			return fmt.Errorf("terminal %q: %w", name, ErrListingOmittedLiveServer)
		}
		return fmt.Errorf("terminal %q is not listed live", name)
	}
	return nil
}

// This is the production budget-selection boundary, shared by OS attempts and
// channel-controlled admission tests. Keep query and interactive lifetimes distinct.
func runTerminalHandoffAttempt(lifetime, selected, query context.Context, a TerminalHandoffRuntime, create bool) (int, error) {
	startup, cancel, err := terminalStartupContext(lifetime, selected)
	if err != nil {
		return 1, err
	}
	defer cancel()
	return coordinateTerminalHandoffContexts(lifetime, query, startup, a, create)
}
