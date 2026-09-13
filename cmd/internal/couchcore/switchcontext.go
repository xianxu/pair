package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// SwitchContextResolver captures outgoing source evidence before park.
type SwitchContextResolver interface {
	Resolve(context.Context, ThreadRecord) (orientation.OrientationContext, error)
}

// SwitchArchiveResolver refreshes only the exact capture committed by park.
type SwitchArchiveResolver interface {
	ResolveArchive(ThreadRecord, *orientation.OrientationContext) error
}

// OSSwitchContextResolver authorizes native evidence through the inventory owner
// query. Query/NativePath allow deterministic tests without scanning real homes.
type OSSwitchContextResolver struct {
	DataDir, HomeDir, Renderer string
	Query                      func(context.Context, sessioninventory.Runtime, string, string, sessioninventory.Agent) (sessioninventory.SessionQuery, error)
	NativePath                 func(sessioninventory.Runtime, sessioninventory.Artifact) (string, error)
}

func (r OSSwitchContextResolver) Resolve(ctx context.Context, record ThreadRecord) (orientation.OrientationContext, error) {
	result := orientation.OrientationContext{Tag: string(record.Address.Tag), WorkingPath: record.WorkingPath, Renderer: r.Renderer}
	if record.LatestLaunchProfile != nil {
		result.SourceAgent = record.LatestLaunchProfile.Agent
	}
	if len(record.Incarnations) == 1 && record.Incarnations[0].LaunchProfile != nil {
		result.SourceAgent = record.Incarnations[0].LaunchProfile.Agent
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: record.Address.RepoScope, Tag: result.Tag})
	if err != nil {
		return result, err
	}
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

func (r OSSwitchContextResolver) ResolveArchive(record ThreadRecord, result *orientation.OrientationContext) error {
	if result == nil {
		return errors.New("switch context is nil")
	}
	result.ScrollbackRaw, result.ScrollbackEvents = "", ""
	result.Renderer = r.Renderer
	if record.VerifiedPark == nil || record.VerifiedPark.Scrollback == nil {
		result.Unavailable = append(result.Unavailable, "park has no exact preserved Pair TTY descriptor")
		return nil
	}
	park := record.VerifiedPark
	if park.Identity.Address != record.Address {
		return errors.New("preserved Pair TTY identity does not match thread")
	}
	descriptor := park.Scrollback
	if err := pairlifecycle.ValidatePreservedScrollback(descriptor); err != nil {
		return err
	}
	if result.SourceAgent != "" && descriptor.Agent != result.SourceAgent {
		return errors.New("preserved Pair TTY agent does not match outgoing agent")
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: park.Identity.Address.RepoScope, Tag: string(park.Identity.Address.Tag)})
	if err != nil {
		return err
	}
	archive, err := paths.ParkedScrollbackArtifacts(descriptor.Token)
	if err != nil {
		return err
	}
	if readableSwitchFile(archive.Raw) {
		result.ScrollbackRaw = archive.Raw
	} else {
		result.Unavailable = append(result.Unavailable, "exact preserved Pair TTY raw capture is unavailable")
	}
	if descriptor.Events && result.ScrollbackRaw != "" && readableSwitchFile(archive.Events) {
		result.ScrollbackEvents = archive.Events
	}
	if result.ScrollbackRaw != "" && result.Renderer == "" {
		result.Unavailable = append(result.Unavailable, "Pair scrollback renderer is unavailable")
	}
	return nil
}

func readableSwitchFile(path string) bool {
	if path == "" {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && info.Mode().IsRegular()
}

func (c *Couch) ReadOrientationStatus(ctx context.Context, address ThreadAddress, agent, attempt string) (orientation.DeliveryState, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orientation.DeliveryState{}, err
	}
	if c == nil || c.Threads == nil {
		return orientation.DeliveryState{}, errors.New("orientation status services are unavailable")
	}
	record, err := c.Threads.GetThread(address)
	if err != nil {
		return orientation.DeliveryState{}, err
	}
	if agent == "" || attempt == "" || len(record.Incarnations) != 1 {
		return orientation.DeliveryState{}, errors.New("orientation target incarnation is obsolete")
	}
	incarnation := record.Incarnations[0]
	if incarnation.LaunchProfile == nil || incarnation.LaunchProfile.Agent != agent {
		return orientation.DeliveryState{}, errors.New("orientation target agent is obsolete")
	}
	if incarnation.Start != nil && incarnation.Start.Nonce != attempt {
		return orientation.DeliveryState{}, errors.New("orientation target attempt is obsolete")
	}
	if c.Proc != nil && incarnation.PID > 0 {
		if c.Proc.Exists(incarnation.PID) == Dead {
			return orientation.DeliveryState{}, errors.New("orientation target exited")
		}
		identity, err := c.Proc.Identity(incarnation.PID)
		if err != nil {
			return orientation.DeliveryState{}, err
		}
		if identity != incarnation.Identity {
			return orientation.DeliveryState{}, fmt.Errorf("orientation target process identity changed")
		}
	}
	if c.OrientationStatus == nil {
		return orientation.DeliveryState{}, nil
	}
	return c.OrientationStatus(ctx, address, agent, attempt)
}

// OSOrientationStatusReader reads the wrapper's existing exact ready artifact.
// Session supplies the authoritative Pair session binding for the thread.
var errObsoleteOrientationReady = errors.New("orientation ready record belongs to an obsolete target")

type OSOrientationStatusReader struct {
	DataDir string
	Session func(ThreadAddress) (PairSessionBinding, error)
	Proc    ProcOps
}

func (r OSOrientationStatusReader) readReady(ctx context.Context, address ThreadAddress, agent, attempt string) (*readiness.ReadyRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if agent == "" || attempt == "" {
		return nil, errors.New("orientation ready identity is empty")
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return nil, err
	}
	path, err := paths.AgentReadyChecked(agent)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return nil, err
	}
	if ready.Tag != string(address.Tag) || ready.Agent != agent || ready.Nonce != attempt {
		return nil, errObsoleteOrientationReady
	}
	if r.Session == nil {
		return nil, errors.New("orientation session binding unavailable")
	}
	session, err := r.Session(address)
	if err != nil {
		return nil, err
	}
	if !session.Present || session.Name != ready.Session {
		return nil, fmt.Errorf("%w: Pair session does not match", errObsoleteOrientationReady)
	}
	if r.Proc == nil || r.Proc.Exists(ready.PID) != Live {
		return nil, errors.New("orientation wrapper is not live")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &ready, nil
}

func (r OSOrientationStatusReader) Read(ctx context.Context, address ThreadAddress, agent, attempt string) (orientation.DeliveryState, error) {
	ready, err := r.readReady(ctx, address, agent, attempt)
	if err != nil || ready == nil || ready.Orientation == nil {
		return orientation.DeliveryState{}, err
	}
	return *ready.Orientation, nil
}

func (r OSOrientationStatusReader) Registered(ctx context.Context, address ThreadAddress, agent, attempt string) (bool, error) {
	ready, err := r.readReady(ctx, address, agent, attempt)
	if errors.Is(err, errObsoleteOrientationReady) {
		return false, nil
	}
	return ready != nil, err
}
