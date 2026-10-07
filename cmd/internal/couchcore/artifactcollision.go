package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// ErrPairSessionBindingAbsent means the exact scoped index was readable but
// contains no session binding for this address. It does not prove session absence.
var ErrPairSessionBindingAbsent = errors.New("exact Pair session binding is absent")

type PairSessionBinding struct {
	Name    string
	Present bool
	Owner   *launcher.SessionOwnerObservation
}

type SessionOwnerProber interface {
	Probe(context.Context, string, string, string, string) (launcher.SessionOwnerObservation, error)
	Revalidate(context.Context, launcher.SessionOwnerObservation) error
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
	OwnerProbe    SessionOwnerProber
	// Zellij is how the checker observes sessions. The zero value is the real
	// zellij on PATH; tests point Path at a stub and count the calls, because
	// "a reattach asks two sessions for their clients" is a count, not a timing
	// (pair#228).
	Zellij launcher.ZellijSource
	// Servers is the host's zellij server snapshot, the only place an orphaned
	// server is visible (#399). Nil is the real host.
	Servers launcher.ServerStates
}

func (c ScopedThreadArtifactCollisionChecker) serverStates() launcher.ServerStates {
	if c.Servers != nil {
		return c.Servers
	}
	return launcher.OSServerStates{}
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
	binding, err := c.PairSession(address)
	if errors.Is(err, ErrPairSessionBindingAbsent) {
		return nil
	}
	if err != nil {
		return err
	}
	if !binding.Present {
		return nil
	}
	if err := c.RevalidatePairSession(context.Background(), binding); err != nil {
		return err
	}
	if c.Sessions == nil {
		return errors.New("quiesce Pair session: nil session deleter")
	}
	return c.Sessions.DeleteSession(binding.Name)
}

func (c ScopedThreadArtifactCollisionChecker) QuiesceNamed(ctx context.Context, address ThreadAddress, name string) error {
	binding, err := c.NamedPairSessionContext(ctx, address, name)
	if err != nil {
		return err
	}
	if !binding.Present {
		return nil
	}
	if err := c.RevalidatePairSession(ctx, binding); err != nil {
		return err
	}
	if c.Sessions == nil {
		return errors.New("quiesce Pair session: nil session deleter")
	}
	return c.Sessions.DeleteSession(name)
}

// scopedIndexRead is one ReadSessionNameIndex result: the shared legacy file's
// rows, then one scope's own rows.
type scopedIndexRead struct {
	scope string
	index launcher.SessionNameIndex
}

// effectiveBindings is every thread's CURRENT session name over the union of the
// files the reads saw -- each thread once, at its newest binding. It is the ONLY
// derivation of that fact: PairSession and DetachedSessions both call it, so
// the name a thread is judged by cannot differ between them.
//
// The union is the whole difficulty. Every scoped read replays the legacy file
// before its own, so a thread bound only by a legacy row appears in EVERY read.
// Summing per-read counts therefore counted such a thread once per scope asked:
// with two or more scopes its session read as contested, it got no detached
// observation, it classified session-gone, and the pass would never seed it
// (pair#206 plan gate PQ-1, measured at 56 legacy-only bindings on the
// operator's host).
//
// So a thread's value comes from the read of its OWN scope when there is one --
// that read holds the legacy row and any newer scope row, in order -- and
// otherwise from any read at all, since a legacy-only row is identical in each.
// Read order does not matter.
func effectiveBindings(reads []scopedIndexRead) map[ThreadAddress]string {
	current := map[ThreadAddress]string{}
	authoritative := map[ThreadAddress]bool{}
	for _, read := range reads {
		// Newest row per thread within this read; entries are append-only.
		latest := make(map[ThreadAddress]string, len(read.index.Entries))
		for _, entry := range read.index.Entries {
			latest[ThreadAddress{RepoScope: entry.ScopeKey, Tag: ThreadTag(entry.Tag)}] = entry.SessionName
		}
		for address, name := range latest {
			switch {
			case address.RepoScope == read.scope:
				current[address], authoritative[address] = name, true
			case !authoritative[address]:
				current[address] = name
			}
		}
	}
	return current
}

// claimsFromBindings counts how many DISTINCT threads currently bind each
// session name. It is the counting half of ProjectDetachedSessions' duplicate
// rule, shared by the real checker and the fake so the rule cannot diverge
// between them (ARCH-MOCK).
func claimsFromBindings(bindings map[ThreadAddress]string) map[string]int {
	claims := make(map[string]int, len(bindings))
	for _, name := range bindings {
		if name != "" {
			claims[name]++
		}
	}
	return claims
}

func (c ScopedThreadArtifactCollisionChecker) PairSession(address ThreadAddress) (PairSessionBinding, error) {
	return c.PairSessionContext(context.Background(), address)
}

func (c ScopedThreadArtifactCollisionChecker) PairSessionContext(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	name, err := c.PairSessionName(ctx, address)
	if err != nil {
		return PairSessionBinding{}, err
	}
	return c.NamedPairSessionContext(ctx, address, name)
}

// PairSessionName reads the thread's recorded session name from the session
// index alone: no ownership probe, so no ps or zellij (#365).
func (c ScopedThreadArtifactCollisionChecker) PairSessionName(ctx context.Context, address ThreadAddress) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := validateThreadAddress(address); err != nil {
		return "", err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: c.GlobalDataDir, RepoScope: address.RepoScope, Tag: string(address.Tag),
	})
	if err != nil {
		return "", err
	}
	runtime := launcher.NewScopedOSRuntime(c.GlobalDataDir, paths.ScopeDir(), "")
	index, err := runtime.ReadSessionNameIndex()
	if err != nil {
		return "", fmt.Errorf("read exact Pair session index: %w", err)
	}
	name := effectiveBindings([]scopedIndexRead{{scope: address.RepoScope, index: index}})[address]
	if name == "" {
		return "", fmt.Errorf("%w for %+v", ErrPairSessionBindingAbsent, address)
	}
	return name, nil
}

func (c ScopedThreadArtifactCollisionChecker) ownerProbe() SessionOwnerProber {
	if c.OwnerProbe != nil {
		return c.OwnerProbe
	}
	return launcher.SessionOwnerProbe{}
}

// NamedPairSessionContext observes the binding supplied by a current or pending
// thread record. Name equality alone never confers ownership of a live server.
func (c ScopedThreadArtifactCollisionChecker) NamedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	if err := validateThreadAddress(address); err != nil {
		return PairSessionBinding{}, err
	}
	owner, err := c.ownerProbe().Probe(ctx, name, c.GlobalDataDir, address.RepoScope, string(address.Tag))
	binding := PairSessionBinding{Name: name, Owner: &owner}
	if err != nil {
		return binding, err
	}
	switch owner.State {
	case launcher.SessionOwnerOwned:
		binding.Present = true
	case launcher.SessionOwnerAbsent, launcher.SessionOwnerForeign:
	case launcher.SessionOwnerOrphaned:
		return binding, refuseResume(ResumeOrphanedServer, owner.Diagnostic)
	default:
		return binding, fmt.Errorf("session %q ownership unresolved: %s", name, owner.Diagnostic)
	}
	return binding, nil
}

func (c ScopedThreadArtifactCollisionChecker) RevalidatePairSession(ctx context.Context, binding PairSessionBinding) error {
	if !binding.Present || binding.Owner == nil || binding.Owner.Name != binding.Name {
		return errors.New("Pair session lacks live ownership proof")
	}
	return c.ownerProbe().Revalidate(ctx, *binding.Owner)
}

// PaneSidecars observes the thread's agent pane sidecars in its own scope
// directory, the directory PairSessionContext reads the session index from.
// It uses a glob and a stat and asks zellij nothing, which is the point: it is
// what a cold resume watches before its first zellij call (#287).
// AgentFromPane keeps only this scope's pane files. It does NOT separate tags
// that share a prefix: with tags `work` and `work-2` in one scope,
// `pane-work-2-claude.json` reads as tag `work`, agent `2-claude`. Couch's
// fixed-shape `couch-<hex>` tags never collide that way. A sidecar that
// vanishes between the glob and the stat has been cleared, so it is left out,
// not reported as an error.
func (c ScopedThreadArtifactCollisionChecker) PaneSidecars(address ThreadAddress) (PaneMarks, error) {
	if err := validateThreadAddress(address); err != nil {
		return nil, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{
		DataDir: c.GlobalDataDir, RepoScope: address.RepoScope, Tag: string(address.Tag),
	})
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(paths.PaneGlob())
	if err != nil {
		return nil, err
	}
	marks := PaneMarks{}
	for _, path := range matches {
		if _, ok := paths.AgentFromPane(path); !ok {
			continue
		}
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("observe pane sidecar: %w", err)
		}
		marks[path] = info.ModTime()
	}
	return marks, nil
}

// DetachedSessionResolver observes which of the supplied threads currently have
// a live zellij session with no client attached.
//
// It takes addresses rather than returning the whole set because the
// session-name index is PER REPO SCOPE (`<dataDir>/repos/<scope>/session-names.jsonl`):
// a whole-set method would need a `repos/*` enumeration this checker does not
// have. Taking addresses mirrors PairSession(address) and ResolveEstablished.
type DetachedSessionResolver interface {
	DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error)
}

// DetachedCandidate names a thread and its saved agent profile. Its session
// ownership is resolved independently of native conversation evidence.
type DetachedCandidate struct {
	Address     ThreadAddress
	Agent       string
	SessionName string
}

func (c ScopedThreadArtifactCollisionChecker) resolveScopedBindings(ctx context.Context, addresses []ThreadAddress, agentOf func(ThreadAddress) string) ([]SessionNameBinding, map[ThreadAddress]string, map[string]bool, error) {
	byScope := make(map[string][]ThreadAddress, len(addresses))
	var scopes []string
	for _, address := range addresses {
		if err := validateThreadAddress(address); err != nil {
			return nil, nil, nil, err
		}
		if _, seen := byScope[address.RepoScope]; !seen {
			scopes = append(scopes, address.RepoScope)
		}
		byScope[address.RepoScope] = append(byScope[address.RepoScope], address)
	}

	var reads []scopedIndexRead
	readable := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		scoped := byScope[scope]
		paths, err := artifactpath.Resolve(artifactpath.Address{
			DataDir: c.GlobalDataDir, RepoScope: scope, Tag: string(scoped[0].Tag),
		})
		if err != nil {
			return nil, nil, nil, err
		}
		index, err := launcher.NewScopedOSRuntime(c.GlobalDataDir, paths.ScopeDir(), "").ReadSessionNameIndex()
		if err != nil {
			continue
		}
		reads = append(reads, scopedIndexRead{scope: scope, index: index})
		readable[scope] = true
	}
	// One derivation of "this thread's current session name", used for both the
	// binding and the claim count, so the name a thread is judged by and the
	// name its claim is counted under are the same value by construction rather
	// than by two lookups agreeing.
	current := effectiveBindings(reads)

	var bindings []SessionNameBinding
	for _, scope := range scopes {
		if !readable[scope] {
			continue // fail closed: see the reach rule above
		}
		for _, address := range byScope[scope] {
			name := current[address]
			if name == "" {
				continue
			}
			binding := SessionNameBinding{Address: address, SessionName: name}
			if agentOf != nil {
				binding.Agent = agentOf(address)
			}
			bindings = append(bindings, binding)
		}
	}
	return bindings, current, readable, nil
}

// The production checker must satisfy every seam the evidence pass reaches it
// through. Those are TYPE ASSERTIONS on c.Artifacts, which fail SILENTLY: drop a
// method and the assertion simply stops matching, the evidence is never
// gathered, and every thread reads `unknown` -- fail-closed, but indistinguishable
// from a host that could not be asked. A compile-time binding turns that into a
// build error.
var (
	_ SessionPresenceResolver = ScopedThreadArtifactCollisionChecker{}
	_ DetachedSessionResolver = ScopedThreadArtifactCollisionChecker{}
	_ NativeBindingResolver   = ScopedThreadArtifactCollisionChecker{}
	_ PairSessionIO           = ScopedThreadArtifactCollisionChecker{}
	_ SessionPresenceResolver = (*FakeThreadArtifactCollisionChecker)(nil)
	_ DetachedSessionResolver = (*FakeThreadArtifactCollisionChecker)(nil)
)

// SessionPresence answers EXISTENCE for every supplied address from one
// host-wide liveness snapshot.
//
// Liveness, not a full snapshot: present is "listed and not exited", so no
// session is asked for its clients. That is what makes this affordable for every
// record rather than only the resume-shaped ones -- a `list-clients` costs about
// 250 ms against a real detached session (#228), and #256 needs a session answer
// for records carrying an incarnation, which is exactly the set the detached
// question was never asked about.
//
// The attached-versus-detached distinction stays with DetachedSessions, which the
// ACTION path uses and which `RequireAttachState` guards. A thread whose session
// someone else attached to reads present here and is refused at the action, which
// is the deliberate optimistic-inventory trade.
//
// A scope whose index could not be read contributes no binding, so its threads
// are absent from the result and read UNRESOLVED -- never "no session".
func (c ScopedThreadArtifactCollisionChecker) SessionPresence(ctx context.Context, addresses []ThreadAddress) (map[ThreadAddress]SessionObservation, error) {
	if len(addresses) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bindings, current, readable, err := c.resolveScopedBindings(ctx, addresses, nil)
	if err != nil {
		return nil, err
	}
	out := map[ThreadAddress]SessionObservation{}
	if len(bindings) > 0 {
		sessions, sessionErr := c.Zellij.LivenessContext(ctx)
		if sessionErr != nil {
			return nil, fmt.Errorf("observe zellij sessions: %w", sessionErr)
		}
		servers, serverErr := c.serverStates().ServerStates(ctx)
		if serverErr != nil {
			// Without the server snapshot an orphan would read absent -- the
			// "no session" that offers a resume onto a running agent. Fail closed.
			return nil, fmt.Errorf("observe zellij servers: %w", serverErr)
		}
		out = ProjectSessionPresence(bindings, sessions, servers, claimsFromBindings(current))
	}
	// An address in a READABLE scope with no index row was asked about, and
	// there is no session: absent, not unresolved. Only an address whose scope
	// could not be read stays out of the map, where it reads the zero value.
	//
	// The distinction is the whole point. A thread that was spawned and never
	// bound -- #273's shape, `last_active_at` at the zero time -- would
	// otherwise read `checking…` forever instead of being recognised as debris
	// the operator can clear.
	for _, address := range addresses {
		if _, answered := out[address]; answered {
			continue
		}
		if readable[address.RepoScope] {
			out[address] = SessionObservation{State: SessionAbsent}
		}
	}
	return out, nil
}

// DetachedSessions answers which of the supplied threads have a live zellij
// session with NO CLIENT attached. It is the ACTION path's authority; the
// refresh asks SessionPresence instead.
//
// Cost: the shared index read (resolveScopedBindings) plus one
// `action list-clients` per candidate session that is live -- the candidates'
// OWN sessions, not every session on the host (pair#228). It used to ask every
// live pair session, about 250 ms each against a real detached one, so proving
// one thread detached scaled with the operator's whole session set.
//
// A snapshot failure is returned, because that one IS the whole answer. The
// per-scope fail-closed rule lives in resolveScopedBindings and is pinned by
// TestDetachedSessionsBindsNothingForAnUnreadableScope.
func (c ScopedThreadArtifactCollisionChecker) DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error) {
	addresses := make([]ThreadAddress, 0, len(candidates))
	var exact []SessionNameBinding
	proof := make(map[ThreadAddress]DetachedCandidate, len(candidates))
	for _, candidate := range candidates {
		if candidate.SessionName != "" {
			exact = append(exact, SessionNameBinding{Address: candidate.Address, Agent: candidate.Agent, SessionName: candidate.SessionName})
		} else {
			addresses = append(addresses, candidate.Address)
		}
		proof[candidate.Address] = candidate
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bindings, _, _, err := c.resolveScopedBindings(ctx, addresses, func(address ThreadAddress) string {
		return proof[address].Agent
	})
	if err != nil {
		return nil, err
	}
	bindings = append(bindings, exact...)
	if len(bindings) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		names = append(names, binding.SessionName)
	}
	// Only the bindings' names are asked for clients. ProjectDetachedSessions
	// reads state for exactly those names, so the answer is the one a full
	// snapshot gives -- including its duplicate-row check, since both rows of a
	// duplicated name pass the same filter. It refuses a snapshot that did not
	// ask, so a liveness form swapped in here fails loudly instead of proving
	// every thread "not detached".
	sessions, err := c.Zellij.SnapshotSessionsContext(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("observe zellij sessions: %w", err)
	}
	// Unlike passive inventory this is a selected action. Positive runtime
	// ownership supersedes stale index claim counts, while a foreign server
	// cannot select the warm path for this conversation.
	live := indexSessionsByName(sessions)
	var owned []SessionNameBinding
	claims := map[string]int{}
	for _, binding := range bindings {
		if !live.live[binding.SessionName] || live.ambiguous[binding.SessionName] {
			continue
		}
		observed, err := c.NamedPairSessionContext(ctx, binding.Address, binding.SessionName)
		if err != nil {
			return nil, err
		}
		if !observed.Present {
			continue
		}
		owned = append(owned, binding)
		claims[binding.SessionName]++
	}
	return ProjectDetachedSessions(owned, sessions, claims)
}

func (c ScopedThreadArtifactCollisionChecker) TriggerQuit(session string, intent launcher.QuitIntent) error {
	if intent.Request == nil || intent.Request.DataDir != c.GlobalDataDir {
		return errors.New("trigger Pair quit requires exact Couch address")
	}
	address := ThreadAddress{RepoScope: intent.Request.RepoScope, Tag: ThreadTag(intent.Request.Tag)}
	binding, err := c.NamedPairSessionContext(context.Background(), address, session)
	if err != nil {
		return err
	}
	return c.TriggerBoundQuit(context.Background(), binding, intent)
}

func (c ScopedThreadArtifactCollisionChecker) TriggerBoundQuit(ctx context.Context, binding PairSessionBinding, intent launcher.QuitIntent) error {
	if intent.Request == nil || binding.Owner == nil || intent.Request.DataDir != c.GlobalDataDir || binding.Owner.Owner.DataDir != intent.Request.DataDir || binding.Owner.Owner.RepoScope != intent.Request.RepoScope || binding.Owner.Owner.Tag != intent.Request.Tag {
		return errors.New("quit intent does not match observed session owner")
	}
	if err := c.RevalidatePairSession(ctx, binding); err != nil {
		return err
	}
	session := binding.Name
	runtime := launcher.NewScopedOSRuntime(c.GlobalDataDir, c.GlobalDataDir, "")
	if err := runtime.WriteQuitIntent(session, intent); err != nil {
		return err
	}
	if c.Sessions == nil {
		return errors.New("trigger Pair quit: nil session deleter")
	}
	if err := c.RevalidatePairSession(ctx, binding); err != nil {
		return err
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
