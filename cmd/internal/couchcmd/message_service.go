package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

// messageAuthority separates live evidence from registry mutation. All probes
// are read-only; none call mutable Couch domain operations from request workers.
type messageAuthority struct {
	thread     func(context.Context, couchmessage.Binding) (string, error)
	workspace  func(context.Context, string) (couchcore.WorkspaceIdentity, error)
	process    func(couchmessage.Binding) error
	wrapperPID func(couchmessage.Binding) (int, error)
	launch     func(context.Context, couchmessage.Binding) error
	endpoint   func(couchmessage.Binding) couchmessage.DeliveryEndpoint
	branch     func(context.Context, string) (couchcore.SlotGitStatus, error)
}

func (a messageAuthority) live(ctx context.Context, b couchmessage.Binding) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := b.Validate(); err != nil {
		return "", err
	}
	root, err := a.thread(ctx, b)
	if err != nil {
		return "", err
	}
	if err = a.process(b); err != nil {
		return "", err
	}
	pid, err := a.wrapperPID(b)
	if err != nil {
		return "", err
	}
	if pid != b.PID {
		return "", errors.New("wrapper does not own the current binding")
	}
	if err = a.launch(ctx, b); err != nil {
		return "", err
	}
	// External probes may yield while a pane is replaced or detached.
	current, err := a.thread(ctx, b)
	if err != nil {
		return "", err
	}
	if current != root {
		return "", errors.New("message checkout changed during verification")
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return root, nil
}

type messageService struct {
	server     *couchmessage.Server
	broker     *couchmessage.Broker
	cancel     context.CancelFunc
	workers    sync.WaitGroup
	authority  messageAuthority
	mu         sync.Mutex
	workspaces map[couchmessage.Binding]couchcore.WorkspaceIdentity
}

func (s *messageService) Close() {
	s.cancel()
	_ = s.server.Close()
	_ = s.broker.Close()
	s.workers.Wait()
}

// startMessageService is called only while the existing supervisor lease is
// held. Socket cleanup and all child registration live inside that lifetime.
func startMessageService(console *couchtty.Console, c *couchcore.Couch) (*messageService, error) {
	resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
	if !ok {
		return nil, nil
	}
	artifacts, ok := c.Artifacts.(couchcore.PairLifecycleEnvironment)
	if !ok || c.Proc == nil || c.Git == nil {
		return nil, nil
	}
	namespace := c.Namespace.Dir()
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return nil, err
	}
	reader := couchcore.OSOrientationStatusReader{DataDir: artifacts.PairLifecycleDataDir(), Session: artifacts.PairSession, Proc: c.Proc}
	session := func(ctx context.Context, address couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
		if err := ctx.Err(); err != nil {
			return couchcore.PairSessionBinding{}, err
		}
		v, e := artifacts.PairSession(address)
		if e == nil {
			e = ctx.Err()
		}
		return v, e
	}
	if source, ok := c.Artifacts.(interface {
		PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	}); ok {
		reader.SessionContext = source.PairSessionContext
		session = source.PairSessionContext
	}
	authority := messageAuthority{
		thread: console.MessageBinding, workspace: resolver.ResolveWorkspace,
		process: func(b couchmessage.Binding) error {
			if c.Proc.Exists(b.PID) != couchcore.Live {
				return errors.New("wrapper is not live")
			}
			identity, e := c.Proc.Identity(b.PID)
			if e != nil {
				return e
			}
			if identity != b.Start {
				return errors.New("wrapper process identity changed")
			}
			return nil
		},
		wrapperPID: func(b couchmessage.Binding) (int, error) {
			paths, e := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: b.Scope, Tag: b.Tag})
			if e != nil {
				return 0, e
			}
			return readMessageWrapperPID(paths.PairWrapPID())
		},
		launch: func(ctx context.Context, b couchmessage.Binding) error {
			address := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
			registered, e := reader.Registered(ctx, address, b.Agent, b.Nonce)
			if e != nil {
				return fmt.Errorf("wrapper launch registration: %w", e)
			}
			if !registered {
				return errors.New("wrapper launch registration unavailable")
			}
			current, e := session(ctx, address)
			if e != nil {
				return e
			}
			if !current.Present || current.Name != b.Session {
				return errors.New("wrapper conversation changed")
			}
			return nil
		},
		endpoint: func(b couchmessage.Binding) couchmessage.DeliveryEndpoint {
			return couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}
		},
		branch: func(ctx context.Context, root string) (couchcore.SlotGitStatus, error) {
			return couchcore.ProbeSlotGit(ctx, c.Git, root)
		},
	}
	service, err := newMessageService(context.Background(), socket, authority)
	if err == nil {
		console.SetMessageBroker(service.broker)
	}
	return service, err
}
func newMessageService(parent context.Context, socket string, authority messageAuthority) (*messageService, error) {
	if authority.thread == nil || authority.workspace == nil || authority.process == nil || authority.wrapperPID == nil || authority.launch == nil || authority.endpoint == nil || authority.branch == nil {
		return nil, errors.New("message authority is incomplete")
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	if err := prepareMessageSocket(socket); err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithCancel(parent)
	s := &messageService{cancel: cancel, authority: authority, workspaces: map[couchmessage.Binding]couchcore.WorkspaceIdentity{}}
	s.broker = couchmessage.NewBroker(lifetime, time.Now, func(ctx context.Context, b couchmessage.Binding) (bool, error) {
		root, err := authority.thread(ctx, b)
		if err != nil {
			return false, err
		}
		s.mu.Lock()
		identity, ok := s.workspaces[b]
		s.mu.Unlock()
		if !ok || identity.WorktreeRoot != root || identity.RestingBranch == nil {
			return false, errors.New("unknown message workspace")
		}
		status, err := authority.branch(ctx, root)
		return err == nil && !status.Detached && status.Branch == *identity.RestingBranch, err
	})
	server, err := couchmessage.StartServer(lifetime, socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return json.Marshal(couchmessage.Response{Code: "invalid-request", Error: err.Error()})
		}
		return json.Marshal(s.handle(ctx, request))
	})
	if err != nil {
		cancel()
		_ = s.broker.Close()
		return nil, err
	}
	s.server = server
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-lifetime.Done():
				return
			case <-ticker.C:
				s.reconcile(lifetime)
			}
		}
	}()
	return s, nil
}
func (s *messageService) register(ctx context.Context, b couchmessage.Binding) error {
	root, err := s.authority.live(ctx, b)
	if err != nil {
		return err
	}
	s.mu.Lock()
	identity, known := s.workspaces[b]
	s.mu.Unlock()
	if !known {
		identity, err = s.authority.workspace(ctx, root)
		if err != nil {
			return err
		}
	}
	if identity.Address == nil || *identity.Address != b.Slot || identity.RepoIdentity != b.Repository || identity.WorktreeRoot != root || identity.RestingBranch == nil {
		return errors.New("message slot does not match verified workspace")
	}
	endpoint := s.authority.endpoint(b)
	if endpoint == nil {
		return couchmessage.ErrUnsupported
	}
	observation, err := endpoint.Observe(ctx)
	if err != nil {
		return err
	}
	if _, err = s.authority.live(ctx, b); err != nil {
		return err
	}
	// Serialize publication with registration; failed probes never consume cache
	// capacity, and concurrent registration of the same binding consumes one row.
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.workspaces[b]; !exists && len(s.workspaces) >= couchmessage.MaxActors {
		return errors.New("message workspace capacity reached")
	}
	guarded := messageEndpoint{authority: s.authority, binding: b, endpoint: endpoint}
	if err = s.broker.Register(b, guarded); err != nil {
		return err
	}
	s.workspaces[b] = identity
	return s.broker.ReconcileObservation(b, observation)
}
func (s *messageService) handle(ctx context.Context, request couchmessage.Request) couchmessage.Response {
	if request.Binding == nil && couchmessage.ValidateRequest(request) == nil {
		binding, err := s.broker.Caller(request.Scope, request.Tag, request.Session, request.Nonce)
		if err == nil {
			_, err = s.authority.live(ctx, binding)
		}
		if err != nil {
			return couchmessage.Response{Code: "unavailable", Error: err.Error()}
		}
	}
	return couchmessage.Handle(ctx, s.broker, request, s.register)
}
func (s *messageService) reconcile(ctx context.Context) {
	s.mu.Lock()
	bindings := make([]couchmessage.Binding, 0, len(s.workspaces))
	for binding := range s.workspaces {
		bindings = append(bindings, binding)
	}
	s.mu.Unlock()
	for _, binding := range bindings {
		if ctx.Err() != nil {
			return
		}
		probe, cancel := context.WithTimeout(ctx, couchmessage.AdmissionTimeout)
		_, err := s.authority.live(probe, binding)
		cancel()
		if err != nil {
			s.broker.Disconnect(binding)
		}
	}
}

// messageEndpoint repeats authority checks at use, so a still-running obsolete
// wrapper cannot receive during the background reconciliation interval.
type messageEndpoint struct {
	authority messageAuthority
	binding   couchmessage.Binding
	endpoint  couchmessage.DeliveryEndpoint
}

func (e messageEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	if _, err := e.authority.live(ctx, e.binding); err != nil {
		return couchmessage.Observation{}, err
	}
	return e.endpoint.Observe(ctx)
}
func (e messageEndpoint) Reserve(ctx context.Context, id string, seq uint64) error {
	if _, err := e.authority.live(ctx, e.binding); err != nil {
		return err
	}
	return e.endpoint.Reserve(ctx, id, seq)
}
func (e messageEndpoint) Release(ctx context.Context, id string) error {
	return e.endpoint.Release(ctx, id)
}
func (e messageEndpoint) Deliver(ctx context.Context, m couchmessage.Message) (couchmessage.Receipt, error) {
	if _, err := e.authority.live(ctx, e.binding); err != nil {
		return couchmessage.Receipt{Status: couchmessage.Cancelled, Detail: err.Error()}, nil
	}
	return e.endpoint.Deliver(ctx, m)
}

func readMessageWrapperPID(path string) (int, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return 0, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > 32 {
		return 0, errors.New("invalid wrapper PID binding file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 0 || len(raw) > 32 || strconv.Itoa(pid) != string(raw) {
		return 0, errors.New("invalid wrapper PID binding")
	}
	return pid, nil
}

// The supervisor lease establishes stale-listener authority. Validate the
// private parent before removing anything beneath it, including on restart.
func prepareMessageSocket(socket string) error {
	if !filepath.IsAbs(socket) || socket != filepath.Clean(socket) {
		return errors.New("invalid message socket path")
	}
	parent := filepath.Dir(socket)
	if err := os.Mkdir(parent, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	dir, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	stat, ok := dir.Sys().(*syscall.Stat_t)
	if !dir.IsDir() || dir.Mode().Perm() != 0700 || !ok || int(stat.Uid) != os.Getuid() {
		return errors.New("message socket parent is not a private owned directory")
	}
	info, err := os.Lstat(socket)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if info.Mode()&os.ModeSocket == 0 || !ok || int(owner.Uid) != os.Getuid() {
		return errors.New("message socket path is not an owned socket")
	}
	return os.Remove(socket)
}
