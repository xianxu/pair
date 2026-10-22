package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// SwitchContextResolver resolves the current agent's transcript evidence.
type SwitchContextResolver interface {
	Resolve(context.Context, ThreadRecord) (orientation.OrientationContext, error)
}

// OSSwitchContextResolver authorizes native evidence through the inventory owner
// query. Query/NativePath allow deterministic tests without scanning real homes.
type OSSwitchContextResolver struct {
	DataDir, HomeDir, Renderer string
	Query                      func(context.Context, sessioninventory.Runtime, string, string, sessioninventory.Agent) (sessioninventory.SessionQuery, error)
	NativePath                 func(sessioninventory.Runtime, sessioninventory.Artifact) (string, error)
}

// RecordAgent is the agent a thread record says it runs: its only
// incarnation's, else its latest launch's; "" when it records none.
func RecordAgent(record ThreadRecord) string {
	if len(record.Incarnations) == 1 && record.Incarnations[0].LaunchProfile != nil {
		return record.Incarnations[0].LaunchProfile.Agent
	}
	if record.LatestLaunchProfile != nil {
		return record.LatestLaunchProfile.Agent
	}
	return ""
}

func (r OSSwitchContextResolver) Resolve(ctx context.Context, record ThreadRecord) (orientation.OrientationContext, error) {
	result := orientation.OrientationContext{Tag: string(record.Address.Tag), WorkingPath: record.WorkingPath, Renderer: r.Renderer, SourceAgent: RecordAgent(record)}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: record.Address.RepoScope, Tag: result.Tag})
	if err != nil {
		return result, err
	}
	owner, err := artifactpath.NewStorageOwner(r.DataDir, record.Address.RepoScope, result.Tag)
	if err != nil {
		return result, err
	}
	result.Owner = &owner
	if readableSwitchFile(paths.Log()) {
		result.PairLog = paths.Log()
	} else {
		result.Unavailable = append(result.Unavailable, "Pair sent-prompt history is unavailable")
	}
	if result.SourceAgent == "" {
		result.Unavailable = append(result.Unavailable, "outgoing agent identity is unavailable")
		return result, nil
	}
	runtime := sessioninventory.NewOSRuntime(r.HomeDir, paths.ScopeDir())
	query := r.Query
	if query == nil {
		query = sessioninventory.QuerySessionContext
	}
	queryContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	source, err := query(queryContext, runtime, record.Address.RepoScope, result.Tag, sessioninventory.Agent(result.SourceAgent))
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil || queryContext.Err() != nil || source.Status != sessioninventory.BindingEstablished || source.Root == nil {
		result.Unavailable = append(result.Unavailable, "native session has no exact established outgoing binding")
		return result, nil
	}
	result.SourceSession = source.Root.NativeID
	artifact, err := sessioninventory.RootTranscript(*source.Root)
	if err != nil {
		result.Unavailable = append(result.Unavailable, "native root has no unique transcript")
		return result, nil
	}
	nativePath := r.NativePath
	if nativePath == nil {
		nativePath = func(runtime sessioninventory.Runtime, artifact sessioninventory.Artifact) (string, error) {
			resolver, ok := runtime.(interface {
				ResolveArtifactPath(sessioninventory.Artifact) (string, error)
			})
			if !ok {
				return "", errors.New("native runtime cannot resolve authorized artifact paths")
			}
			return resolver.ResolveArtifactPath(artifact)
		}
	}
	path, err := nativePath(runtime, artifact)
	if err != nil || !readableSwitchFile(path) {
		result.Unavailable = append(result.Unavailable, "native root transcript is unavailable")
		return result, nil
	}
	result.NativeTranscripts = []string{path}
	return result, nil
}

// Open nonblocking and without following the final symlink so a stale artifact
// cannot stall context/status lookup or redirect it to another file.
func openSwitchRegularFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("switch artifact is not a regular file")
	}
	return file, nil
}

func readableSwitchFile(path string) bool {
	if path == "" {
		return false
	}
	file, err := openSwitchRegularFile(path)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

var errObsoleteReady = errors.New("ready record belongs to an obsolete target")

// OSReadyReader reads the wrapper's exact ready artifact: a fresh launch's
// registration, its recorded session and its generation.
// Session supplies the authoritative Pair session binding for the thread.
type OSReadyReader struct {
	DataDir        string
	Session        func(ThreadAddress) (PairSessionBinding, error)
	SessionContext func(context.Context, ThreadAddress) (PairSessionBinding, error)
	Proc           ProcOps
}

// sessionBinding prefers the cancellable production observer. Session remains
// compatible with in-memory observers and older embedders.
func (r OSReadyReader) sessionBinding(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	if err := ctx.Err(); err != nil {
		return PairSessionBinding{}, err
	}
	if r.SessionContext != nil {
		return r.SessionContext(ctx, address)
	}
	if r.Session == nil {
		return PairSessionBinding{}, errors.New("ready session binding unavailable")
	}
	binding, err := r.Session(address)
	if err == nil {
		err = ctx.Err()
	}
	return binding, err
}

func (r OSReadyReader) readReadyFile(ctx context.Context, address ThreadAddress, agent, attempt string) (*readiness.ReadyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if agent == "" || attempt == "" {
		return nil, errors.New("ready identity is empty")
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return nil, err
	}
	path, err := paths.AgentReadyChecked(agent)
	if err != nil {
		return nil, err
	}
	file, err := openSwitchRegularFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 16*1024+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 16*1024 {
		return nil, errors.New("ready record exceeds size limit")
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return nil, err
	}
	if ready.Tag != string(address.Tag) || ready.Agent != agent || ready.Nonce != attempt {
		return nil, errObsoleteReady
	}
	return &ready, nil
}

func (r OSReadyReader) readReady(ctx context.Context, address ThreadAddress, agent, attempt string) (*readiness.ReadyRecord, error) {
	ready, err := r.readReadyFile(ctx, address, agent, attempt)
	if err != nil || ready == nil {
		reportProgress(ctx, "waiting for agent readiness")
		return ready, err
	}
	reportProgress(ctx, "verifying terminal registration")
	session, err := r.sessionBinding(ctx, address)
	if err != nil {
		return nil, err
	}
	if !session.Present || session.Name != ready.Session {
		return nil, fmt.Errorf("%w: Pair session does not match", errObsoleteReady)
	}
	if r.Proc == nil || r.Proc.Exists(ready.PID) != Live {
		return nil, errors.New("ready agent process is not live")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return ready, nil
}

// RecordedSession is the session named by the launch's ready file, read from
// the file alone: it proves the launch nonce is the recorded one without an
// ownership probe (#365). ok is false when no current ready file exists.
func (r OSReadyReader) RecordedSession(ctx context.Context, address ThreadAddress, agent, attempt string) (session string, ok bool, err error) {
	ready, err := r.readReadyFile(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteReady) || (err == nil && ready == nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return ready.Session, true, nil
}

// RecordedChild names the native child the launch's ready file records:
// its launch ordinal and process identity, from the file alone (pair#657).
// ok is false when no current ready file exists.
func (r OSReadyReader) RecordedChild(ctx context.Context, address ThreadAddress, agent, attempt string) (ordinal uint64, identity string, ok bool, err error) {
	ready, err := r.readReadyFile(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteReady) || (err == nil && ready == nil) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	return ready.LaunchOrdinal, ready.AgentIdentity, true, nil
}

func (r OSReadyReader) Registered(ctx context.Context, address ThreadAddress, agent, attempt string) (bool, error) {
	ready, err := r.readReady(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteReady) {
		return false, nil
	}
	return ready != nil, err
}

// Generation reads durable attempt correlation even after the target dies.
// Unlike Registered, it proves provenance, never current process liveness.
func (r OSReadyReader) Generation(ctx context.Context, address ThreadAddress, agent, attempt string) (*checkpoint.TargetGeneration, error) {
	ready, err := r.readReadyFile(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteReady) {
		return nil, nil
	}
	if err != nil || ready == nil {
		return nil, err
	}
	if ready.LaunchOrdinal == 0 {
		return nil, nil
	} // legacy receipts prove no generation
	session, err := r.sessionBinding(ctx, address)
	if err != nil {
		return nil, err
	}
	if session.Name != ready.Session {
		return nil, errors.New("continuation target generation has a foreign session")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &checkpoint.TargetGeneration{Agent: agent, Session: ready.Session, Attempt: attempt, LaunchOrdinal: ready.LaunchOrdinal}, nil
}
