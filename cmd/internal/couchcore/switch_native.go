package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
)

// ExactNativeBindingResolver validates a retained native identity independently
// of the address's latest ledger entry, which may already describe the target.
type ExactNativeBindingResolver interface {
	ResolveExact(context.Context, string, string, string, string) (NativeBindingResolution, error)
}

func (r SessionInventoryNativeBindingResolver) ResolveExact(ctx context.Context, scope, tag, agent, nativeID string) (NativeBindingResolution, error) {
	if err := ctx.Err(); err != nil {
		return NativeBindingResolution{}, err
	}
	if r.Runtime == nil {
		return NativeBindingResolution{}, errors.New("exact native resolver unavailable")
	}
	if nativeID == "" || !sessioninventory.NativeSessionCandidateExists(r.Runtime, sessioninventory.Agent(agent), nativeID) {
		return NativeBindingResolution{}, refuseBinding(ResumeBindingRootMissing)
	}
	if err := ctx.Err(); err != nil {
		return NativeBindingResolution{}, err
	}
	return NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: nativeID}, nil
}

func (c ScopedThreadArtifactCollisionChecker) ResolveExact(ctx context.Context, scope, tag, agent, nativeID string) (NativeBindingResolution, error) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: c.GlobalDataDir, RepoScope: scope, Tag: tag})
	if err != nil {
		return NativeBindingResolution{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return NativeBindingResolution{}, err
	}
	return (SessionInventoryNativeBindingResolver{Runtime: sessioninventory.NewOSRuntime(home, paths.ScopeDir())}).ResolveExact(ctx, scope, tag, agent, nativeID)
}
