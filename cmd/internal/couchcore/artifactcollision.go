package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type PairSessionBinding struct {
	Name    string
	Present bool
}

type PairSessionIO interface {
	PairSession(ThreadAddress) (PairSessionBinding, error)
	TriggerQuit(string, launcher.QuitIntent) error
}

// PairLifecycleEnvironment lets the Couch composition root install lifecycle
// recovery without teaching generic artifact collision fakes about Pair's
// durable request protocol.
type PairLifecycleEnvironment interface {
	PairSessionIO
	PairLifecycleIO() LifecycleIO
	PairLifecycleDataDir() string
}

// ThreadArtifactClaim is retained when ThreadStore accepts the same address and
// released only when its subsequent no-replace record claim fails.
type ThreadArtifactClaim interface {
	Release() error
}

// ThreadArtifactClaimer atomically serializes a prospective composite address
// with every current Pair artifact/session producer.
type ThreadArtifactClaimer interface {
	Claim(ThreadAddress) (ThreadArtifactClaim, error)
}

// ThreadArtifactController owns the full durable Pair-address lifecycle used
// by Couch. Allocation depends only on the narrower claimer capability.
type ThreadArtifactController interface {
	ThreadArtifactClaimer
	Release(ThreadAddress) error
	Registration(ThreadAddress) (RegistrationEvidence, error)
	Quiesce(ThreadAddress) error
}

type noopThreadArtifactClaim struct{}

func (noopThreadArtifactClaim) Release() error { return nil }

type NoThreadArtifactCollisions struct{}

func (NoThreadArtifactCollisions) Claim(ThreadAddress) (ThreadArtifactClaim, error) {
	return noopThreadArtifactClaim{}, nil
}
func (NoThreadArtifactCollisions) Release(ThreadAddress) error { return nil }
func (NoThreadArtifactCollisions) Registration(ThreadAddress) (RegistrationEvidence, error) {
	return RegistrationEstablished, nil
}
func (NoThreadArtifactCollisions) Quiesce(ThreadAddress) error { return nil }

type ScopedThreadArtifactCollisionChecker struct {
	GlobalDataDir string
	Sessions      launcher.SessionDeleter
}

func NewScopedThreadArtifactCollisionChecker(globalDataDir string) ScopedThreadArtifactCollisionChecker {
	return ScopedThreadArtifactCollisionChecker{GlobalDataDir: globalDataDir, Sessions: launcher.OSRuntime{}}
}

func (c ScopedThreadArtifactCollisionChecker) PairLifecycleIO() LifecycleIO {
	return PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}
}

func (c ScopedThreadArtifactCollisionChecker) PairLifecycleDataDir() string { return c.GlobalDataDir }

func (c ScopedThreadArtifactCollisionChecker) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	if err := validateThreadAddress(address); err != nil {
		return nil, err
	}
	if c.GlobalDataDir == "" {
		return nil, errors.New("artifact claimer has no Pair data directory")
	}
	claim, err := launcher.ClaimNewThreadAddress(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func (c ScopedThreadArtifactCollisionChecker) Release(address ThreadAddress) error {
	paths := launcher.NewScopedPaths(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	return launcher.ReleaseThreadAddressClaim(paths.ThreadClaim())
}

func (c ScopedThreadArtifactCollisionChecker) Registration(address ThreadAddress) (RegistrationEvidence, error) {
	if err := validateThreadAddress(address); err != nil {
		return RegistrationUnknown, err
	}
	established, err := launcher.ThreadAddressEstablished(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err != nil {
		return RegistrationUnknown, err
	}
	if established {
		return RegistrationEstablished, nil
	}
	return RegistrationAbsent, nil
}

func (c ScopedThreadArtifactCollisionChecker) Quiesce(address ThreadAddress) error {
	if err := validateThreadAddress(address); err != nil {
		return err
	}
	if c.GlobalDataDir == "" {
		return errors.New("artifact claimer has no Pair data directory")
	}
	return launcher.QuiesceThreadSession(c.GlobalDataDir, address.RepoScope, string(address.Tag), c.Sessions)
}

// lookupSessionName finds the zellij session name bound to one address,
// scanning the index backwards so the newest binding wins. One implementation
// for PairSession and DetachedSessions, which asked the same question twice.
func lookupSessionName(index launcher.SessionNameIndex, address ThreadAddress) string {
	for i := len(index.Entries) - 1; i >= 0; i-- {
		entry := index.Entries[i]
		if entry.ScopeKey == address.RepoScope && entry.Tag == string(address.Tag) {
			return entry.SessionName
		}
	}
	return ""
}

func (c ScopedThreadArtifactCollisionChecker) PairSession(address ThreadAddress) (PairSessionBinding, error) {
	if err := validateThreadAddress(address); err != nil {
		return PairSessionBinding{}, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: c.GlobalDataDir, RepoScope: address.RepoScope, Tag: string(address.Tag),
	})
	if err != nil {
		return PairSessionBinding{}, err
	}
	runtime := launcher.NewScopedOSRuntime(c.GlobalDataDir, paths.ScopeDir(), "")
	index, err := runtime.ReadSessionNameIndex()
	if err != nil {
		return PairSessionBinding{}, fmt.Errorf("read exact Pair session index: %w", err)
	}
	name := lookupSessionName(index, address)
	if name == "" {
		return PairSessionBinding{}, fmt.Errorf("exact Pair session binding is absent for %+v", address)
	}
	sessions, err := runtime.Sessions()
	if err != nil {
		return PairSessionBinding{}, fmt.Errorf("observe exact Pair session: %w", err)
	}
	present := false
	for _, session := range sessions {
		if session.Name == name && session.State != launcher.SessionExited {
			present = true
			break
		}
	}
	return PairSessionBinding{Name: name, Present: present}, nil
}

// DetachedSessionResolver observes which of the supplied threads currently have
// a live zellij session with no client attached.
//
// It takes addresses rather than returning the whole set because the
// session-name index is PER REPO SCOPE (`<dataDir>/repos/<scope>/session-names.jsonl`):
// a whole-set method would need a `repos/*` enumeration this checker does not
// have. Taking addresses mirrors PairSession(address) and ResolveEstablished.
type DetachedSessionResolver interface {
	DetachedSessions(ctx context.Context, addresses []ThreadAddress) ([]DetachedSessionObservation, error)
}

// DetachedSessions reads each requested scope's session-name index once and
// takes ONE zellij snapshot for all of them -- the snapshot ignores scope, so a
// snapshot per scope would be the same query repeated.
//
// Cost, stated honestly: the snapshot is two `list-sessions` runs plus one
// `action list-clients` per non-exited session ON THE HOST. Passing candidates
// bounds WHETHER the snapshot runs -- a couch with nothing detachable pays
// nothing -- but it does not bound N once it does. Each query carries
// SnapshotContext's timeout so a hung zellij cannot wedge the refresh worker.
//
// Index reads fail closed per scope: a scope whose index cannot be read
// contributes no bindings rather than an empty answer that would silently hide
// its detached threads. A snapshot failure is returned, because that one IS the
// whole answer.
func (c ScopedThreadArtifactCollisionChecker) DetachedSessions(ctx context.Context, addresses []ThreadAddress) ([]DetachedSessionObservation, error) {
	if len(addresses) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	byScope := make(map[string][]ThreadAddress, len(addresses))
	var scopes []string
	for _, address := range addresses {
		if err := validateThreadAddress(address); err != nil {
			return nil, err
		}
		if _, seen := byScope[address.RepoScope]; !seen {
			scopes = append(scopes, address.RepoScope)
		}
		byScope[address.RepoScope] = append(byScope[address.RepoScope], address)
	}

	var bindings []SessionNameBinding
	for _, scope := range scopes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scoped := byScope[scope]
		paths, err := artifactpath.Resolve(artifactpath.Address{
			DataDir: c.GlobalDataDir, RepoScope: scope, Tag: string(scoped[0].Tag),
		})
		if err != nil {
			return nil, err
		}
		index, err := launcher.NewScopedOSRuntime(c.GlobalDataDir, paths.ScopeDir(), "").ReadSessionNameIndex()
		if err != nil {
			continue
		}
		for _, address := range scoped {
			name := lookupSessionName(index, address)
			if name != "" {
				bindings = append(bindings, SessionNameBinding{Address: address, SessionName: name})
			}
		}
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	sessions, err := (launcher.ZellijSource{}).SnapshotContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("observe zellij sessions: %w", err)
	}
	return ProjectDetachedSessions(bindings, sessions), nil
}

func (c ScopedThreadArtifactCollisionChecker) TriggerQuit(session string, intent launcher.QuitIntent) error {
	runtime := launcher.NewScopedOSRuntime(c.GlobalDataDir, c.GlobalDataDir, "")
	if err := runtime.WriteQuitIntent(session, intent); err != nil {
		return err
	}
	if c.Sessions == nil {
		return errors.New("trigger Pair quit: nil session deleter")
	}
	// Pair consumes the typed intent only after its blocking Zellij handoff
	// returns. Couch intercepts Alt+x, so writing the intent alone cannot make
	// that happen; quiescing this exact indexed session is the trigger that
	// returns control to Pair's shared full-quit cleanup.
	if err := c.Sessions.DeleteSession(session); err != nil {
		return fmt.Errorf("trigger Pair quit for %q: %w", session, err)
	}
	return nil
}

func (c ScopedThreadArtifactCollisionChecker) ResolveEstablished(ctx context.Context, repoScope, tag, agent string) (NativeBindingResolution, error) {
	paths, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: c.GlobalDataDir, RepoScope: repoScope, Tag: tag,
	})
	if err != nil {
		return NativeBindingResolution{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return NativeBindingResolution{}, err
	}
	return (SessionInventoryNativeBindingResolver{
		Runtime: sessioninventory.NewOSRuntime(home, paths.ScopeDir()),
	}).ResolveEstablished(ctx, repoScope, tag, agent)
}

var (
	_ NativeBindingResolver   = ScopedThreadArtifactCollisionChecker{}
	_ DetachedSessionResolver = ScopedThreadArtifactCollisionChecker{}
)
