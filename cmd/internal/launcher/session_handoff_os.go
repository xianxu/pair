package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

func terminalLaunchArgs(session, config, layout string, create bool) []string {
	if create {
		return []string{"--config-dir", config, "--new-session-with-layout", layout, "--session", session, "options", "--session-serialization", "false"}
	}
	return []string{"--config-dir", config, "attach", session}
}

// TerminalSafety is an observation, never durable liveness. Usable requires a
// coherent live agent under this exact server AND this server's launch policy.
type TerminalSafety struct {
	Owner  SessionOwnerObservation
	Policy artifactpath.SessionLaunchPolicy
	State  TerminalUsability
	Reason string
}

type TerminalUsability uint8

const (
	TerminalUnknown TerminalUsability = iota
	TerminalUsable
	TerminalUnsafe
)

func (t TerminalSafety) Usable() bool { return t.State == TerminalUsable }

type unsafeTerminalError string

func (e unsafeTerminalError) Error() string { return string(e) }
func terminalObservationError(err error) TerminalSafety {
	result := TerminalSafety{Reason: err.Error()}
	var unsafe unsafeTerminalError
	var invalid *osfs.InvalidMetadataError
	if errors.Is(err, os.ErrNotExist) || errors.As(err, &unsafe) || errors.As(err, &invalid) {
		result.State = TerminalUnsafe
	}
	return result
}

// ObserveTerminalSafety is shared by launch admission and Couch projections.
// It never attaches, starts a process, or removes a saved layout.
func ObserveTerminalSafety(ctx context.Context, name, root, scope, tag string) (TerminalSafety, error) {
	owner, err := (SessionOwnerProbe{}).Probe(ctx, name, root, scope, tag)
	result := TerminalSafety{Owner: owner}
	if err != nil {
		return result, err
	}
	if owner.State != SessionOwnerOwned {
		result.Reason = owner.Diagnostic
		if owner.State != SessionOwnerUnknown {
			result.State = TerminalUnsafe
		}
		return result, nil
	}
	table, err := procutil.Table()
	if err != nil {
		return result, err
	}
	result = ObserveTerminalUsability(owner.Server, root, scope, tag, table)
	result.Owner = owner
	if !result.Usable() {
		return result, nil
	}
	if err = (SessionOwnerProbe{}).Revalidate(ctx, owner); err != nil {
		result.State = TerminalUnknown
		return result, err
	}
	return result, nil
}

// ObserveTerminalUsability is the bounded metadata half of the same safety
// observation. Refresh callers supply one host process snapshot and server
// generation; action callers also probe panes and revalidate before effects.
func ObserveTerminalUsability(server SessionServerIdentity, root, scope, tag string, table []procutil.Process) TerminalSafety {
	result := TerminalSafety{Owner: SessionOwnerObservation{Server: server, Name: server.Session}}
	policy, err := readSessionPolicy(server.Socket)
	if err != nil {
		return terminalObservationError(err)
	}
	result.Policy = policy
	canonicalObserved, err := canonicalSocket(server.Socket)
	if err != nil {
		return terminalObservationError(err)
	}
	if policy.Socket != canonicalObserved {
		result.State = TerminalUnsafe
		result.Reason = "terminal launch policy belongs to another socket → relaunch"
		return result
	}
	if policy.Scope != scope || policy.Tag != tag || policy.Name != server.Session || policy.ServerPID != server.PID || policy.ServerIdentity != server.Identity {
		result.State = TerminalUnsafe
		result.Reason = "terminal launch policy belongs to another generation → relaunch"
		return result
	}
	if err = validatePolicySocketFile(policy); err != nil {
		return terminalObservationError(err)
	}
	if err = checkNoSavedLayout(server.Session); err != nil {
		return terminalObservationError(err)
	}
	if err = observePolicyAgentWithTable(policy, root, table); err != nil {
		return terminalObservationError(err)
	}
	result.State = TerminalUsable
	result.Owner.State = SessionOwnerOwned
	return result
}

func canonicalSocket(path string) (string, error) {
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(path)), nil
}
func sessionPolicyPath(socket string) (string, error) {
	socket, err := canonicalSocket(socket)
	if err != nil {
		return "", err
	}
	p, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return "", err
	}
	return p.SessionPolicy(socket)
}
func readSessionPolicy(socket string) (artifactpath.SessionLaunchPolicy, error) {
	path, err := sessionPolicyPath(socket)
	if err != nil {
		return artifactpath.SessionLaunchPolicy{}, err
	}
	raw, err := osfs.ReadRegularMetadata(context.Background(), path, artifactpath.MaxSessionLaunchPolicyBytes)
	if err != nil {
		return artifactpath.SessionLaunchPolicy{}, err
	}
	policy, err := artifactpath.DecodeSessionLaunchPolicy(raw)
	if err != nil {
		return policy, unsafeTerminalError(err.Error())
	}
	return policy, nil
}
func socketFileID(path string) (uint64, uint64, error) {
	s, err := os.Lstat(path)
	if err != nil {
		return 0, 0, err
	}
	if s.Mode()&os.ModeSocket == 0 {
		return 0, 0, unsafeTerminalError("terminal socket is not a socket")
	}
	st, ok := s.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("terminal socket identity unavailable")
	}
	return uint64(st.Dev), uint64(st.Ino), nil
}
func validatePolicySocketFile(p artifactpath.SessionLaunchPolicy) error {
	dev, ino, err := socketFileID(p.Socket)
	if err != nil {
		return err
	}
	if dev != p.SocketDevice || ino != p.SocketInode {
		return unsafeTerminalError("terminal socket was replaced → relaunch")
	}
	return nil
}
func checkNoSavedLayout(name string) error {
	base, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	app := "zellij"
	if runtime.GOOS == "darwin" {
		app = "org.Zellij-Contributors.Zellij"
	}
	path := filepath.Join(base, app, "contract_version_1", "session_info", name, "session-layout.kdl")
	_, err = os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot inspect legacy terminal layout: %w", err)
	}
	return unsafeTerminalError("legacy saved terminal layout cannot be attached safely → relaunch")
}
func observePolicyAgent(p artifactpath.SessionLaunchPolicy, root string) error {
	table, err := procutil.Table()
	if err != nil {
		return err
	}
	return observePolicyAgentWithTable(p, root, table)
}
func observePolicyAgentWithTable(p artifactpath.SessionLaunchPolicy, root string, table []procutil.Process) error {
	owner, err := artifactpath.NewStorageOwner(root, p.Scope, p.Tag)
	if err != nil {
		return err
	}
	raw, err := osfs.ReadRegularMetadata(context.Background(), AgentReadyPath(owner.Directory(), p.Tag, p.Agent), maxAgentReadyBytes)
	if err != nil {
		return err
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return unsafeTerminalError(err.Error())
	}
	if ready.Tag != p.Tag || ready.Agent != p.Agent || ready.Session != p.Name || ready.Nonce != p.Nonce {
		return unsafeTerminalError("agent readiness belongs to another terminal generation")
	}
	server := SessionServerIdentity{PID: p.ServerPID, Identity: p.ServerIdentity}
	if !AgentBelongsToServer(server, ready.PID, ready.AgentIdentity, table) {
		return unsafeTerminalError("terminal has no live agent from its recorded launch → relaunch")
	}
	return nil
}

type terminalOSAttempt struct {
	rt                                                                 OSRuntime
	name, config, layout, socket, socketRoot, scope, tag, agent, nonce string
	create                                                             bool
	expected                                                           TerminalSafety
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
	if a.tag == "" {
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
		observed, err := ObserveTerminalSafety(ctx, a.name, a.rt.globalDataDir(), a.scope, a.tag)
		if err != nil {
			return err
		}
		if !observed.Usable() {
			return fmt.Errorf("terminal needs relaunch: %s", observed.Reason)
		}
		if observed.Policy.Socket != a.socket {
			return errors.New("terminal socket namespace changed → relaunch")
		}
		a.expected = observed
		return nil
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
	if err = checkNoSavedLayout(a.name); err != nil {
		return err
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
	ticker := time.NewTicker(birthPoll)
	defer ticker.Stop()
	var last error
	for {
		if err := a.observeAdmission(ctx); err == nil {
			return nil
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

// retireSessionPolicy runs only after lifecycle quiescence. A concurrent new
// launch wins the same lease and makes the second absence check refuse cleanup.
// Saved layouts and lock inodes are never removed here.
func (r OSRuntime) retireSessionPolicy(ctx context.Context, name string) error {
	root := os.Getenv("ZELLIJ_SOCKET_DIR")
	if root == "" {
		root = filepath.Join(os.TempDir(), "zellij-"+strconv.Itoa(os.Getuid()))
	}
	socket, err := canonicalSocket(filepath.Join(root, "contract_version_1", name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	cache, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return err
	}
	lockPath, err := cache.TerminalLease(socket)
	if err != nil {
		return err
	}
	lease, err := AcquireTerminalLease(ctx, lockPath)
	if err != nil {
		return err
	}
	defer lease.Close()
	servers, err := (osSessionOwnerIO{}).SessionServers(ctx, name)
	if err != nil {
		return err
	}
	if len(servers) != 0 {
		return errors.New("terminal appeared during policy cleanup")
	}
	p, err := readSessionPolicy(socket)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return nil
	} // retain unknown evidence
	if p.Name != name || p.Socket != socket {
		return nil
	}
	path, err := cache.SessionPolicy(socket)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
func (a *terminalOSAttempt) observeAdmission(ctx context.Context) error {
	if !a.create {
		now, err := ObserveTerminalSafety(ctx, a.name, a.rt.globalDataDir(), a.scope, a.tag)
		if err != nil {
			return err
		}
		if !now.Usable() || now.Owner.Server != a.expected.Owner.Server || now.Policy != a.expected.Policy {
			return errors.New("terminal changed during attachment")
		}
		return nil
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
	owner, err := (SessionOwnerProbe{}).Probe(ctx, a.name, a.rt.globalDataDir(), a.scope, a.tag)
	if err != nil {
		return err
	}
	if owner.State != SessionOwnerOwned {
		return errors.New("new terminal ownership is not established")
	}
	socket, err := canonicalSocket(owner.Server.Socket)
	if err != nil || socket != a.socket {
		return errors.New("new server has unexpected socket")
	}
	dev, ino, err := socketFileID(socket)
	if err != nil {
		return err
	}
	config, err := readTerminalConfig(ctx, a.config)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(config)
	p := artifactpath.SessionLaunchPolicy{Version: 1, PolicyVersion: 1, Socket: socket, SocketDevice: dev, SocketInode: ino, Name: a.name, ServerPID: owner.Server.PID, ServerIdentity: owner.Server.Identity, Scope: a.scope, Tag: a.tag, Agent: a.agent, Nonce: a.nonce, ConfigDigest: hex.EncodeToString(digest[:])}
	if err = artifactpath.ValidateSessionLaunchPolicy(p); err != nil {
		return err
	}
	if err = observePolicyAgent(p, a.rt.globalDataDir()); err != nil {
		return err
	}
	if err = checkNoSavedLayout(a.name); err != nil {
		return err
	}
	if err = (SessionOwnerProbe{}).Revalidate(ctx, owner); err != nil {
		return err
	}
	path, err := sessionPolicyPath(socket)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return a.rt.WriteAtomic(path, string(raw))
}

func (r OSRuntime) runTerminalHandoff(ctx context.Context, name, config, layout string, create bool) (int, error) {
	a, err := newTerminalOSAttempt(r, name, config, layout, create)
	if err != nil {
		return 1, err
	}
	return CoordinateTerminalHandoff(ctx, a, birthBound)
}

const maxAgentReadyBytes int64 = 8192
const maxTerminalConfigBytes int64 = 1 << 20

func readTerminalConfig(ctx context.Context, dir string) ([]byte, error) {
	return osfs.ReadRegularMetadata(ctx, filepath.Join(dir, "config.kdl"), maxTerminalConfigBytes)
}
