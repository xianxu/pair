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

// ErrPairSessionBindingAbsent means the exact scoped index was readable but
// contains no session binding for this address. It does not prove session absence.
var ErrPairSessionBindingAbsent = errors.New("exact Pair session binding is absent")

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
	// Zellij is how the checker observes sessions. The zero value is the real
	// zellij on PATH; tests point Path at a stub and count the calls, because
	// "a reattach asks two sessions for their clients" is a count, not a timing
	// (pair#228).
	Zellij launcher.ZellijSource
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
	if err := ctx.Err(); err != nil {
		return PairSessionBinding{}, err
	}
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
	name := effectiveBindings([]scopedIndexRead{{scope: address.RepoScope, index: index}})[address]
	if name == "" {
		return PairSessionBinding{}, fmt.Errorf("%w for %+v", ErrPairSessionBindingAbsent, address)
	}
	// Liveness, not a full snapshot: Present is "listed and not exited", so no
	// session needs asking for its clients. This is couch's registration poll on
	// every reattach, and detach and park call it too (pair#228).
	sessions, err := c.Zellij.LivenessContext(ctx)
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
	DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error)
}

// DetachedCandidate names a thread and its saved agent profile. Its session
// ownership is resolved independently of native conversation evidence.
type DetachedCandidate struct {
	Address ThreadAddress
	Agent   string
}

// DetachedSessions reads each requested scope's session-name index once and
// takes ONE zellij snapshot for all of them -- the snapshot ignores scope, so a
// snapshot per scope would be the same query repeated.
//
// Cost: two `list-sessions` runs plus one `action list-clients` per candidate
// session that is live -- the candidates' OWN sessions, not every session on the
// host (pair#228). It used to ask every live pair session, about 250 ms each
// against a real detached one, so proving one thread detached scaled with the
// operator's whole session set. Candidates also bound WHETHER the snapshot runs:
// a couch with nothing detachable pays nothing. Each query carries the zellij
// query timeout, so a hung zellij cannot wedge the refresh worker.
//
// Index reads fail closed per scope: a scope whose index cannot be read binds
// none of its threads -- not even from legacy rows that another scope's read
// replayed, because the unreadable file may hold a NEWER row that supersedes
// them, and judging a thread by a name it has left is the wrong-answer failure
// this rule exists to prevent. Its rows still count as claims where another read
// saw them. Pinned by TestDetachedSessionsBindsNothingForAnUnreadableScope. A
// snapshot failure is returned, because that one IS the whole answer.
// resolveScopedBindings reads each requested scope's session-name index ONCE and
// returns the {address -> session name} bindings it established, plus the
// effective binding map the claim count derives from.
//
// Extracted so the two session questions -- detached (needs clients) and
// presence (existence only) -- read the index the same way. Splitting the
// QUESTION without splitting the READ is what would let them disagree about
// which name a thread is bound to (ARCH-DRY).
//
// Index reads fail closed per scope: a scope whose index cannot be read binds
// none of its threads -- not even from legacy rows another scope's read
// replayed, because the unreadable file may hold a NEWER row that supersedes
// them, and judging a thread by a name it has left is the wrong-answer failure
// this rule exists to prevent. Its rows still count as claims where another read
// saw them.
func (c ScopedThreadArtifactCollisionChecker) resolveScopedBindings(ctx context.Context, addresses []ThreadAddress, agentOf func(ThreadAddress) string) ([]SessionNameBinding, map[ThreadAddress]string, error) {
	byScope := make(map[string][]ThreadAddress, len(addresses))
	var scopes []string
	for _, address := range addresses {
		if err := validateThreadAddress(address); err != nil {
			return nil, nil, err
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
			return nil, nil, err
		}
		scoped := byScope[scope]
		paths, err := artifactpath.Resolve(artifactpath.Address{
			DataDir: c.GlobalDataDir, RepoScope: scope, Tag: string(scoped[0].Tag),
		})
		if err != nil {
			return nil, nil, err
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
	return bindings, current, nil
}

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
	bindings, current, err := c.resolveScopedBindings(ctx, addresses, nil)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		// Asked, and nothing is bound. An empty map is a RESOLVED answer for
		// nobody: every address reads the zero value, which is unresolved.
		return map[ThreadAddress]SessionObservation{}, nil
	}
	sessions, err := c.Zellij.LivenessContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("observe zellij sessions: %w", err)
	}
	return ProjectSessionPresence(bindings, sessions, claimsFromBindings(current)), nil
}

func (c ScopedThreadArtifactCollisionChecker) DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error) {
	addresses := make([]ThreadAddress, 0, len(candidates))
	proof := make(map[ThreadAddress]DetachedCandidate, len(candidates))
	for _, candidate := range candidates {
		addresses = append(addresses, candidate.Address)
		proof[candidate.Address] = candidate
	}
	if len(addresses) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bindings, current, err := c.resolveScopedBindings(ctx, addresses, func(address ThreadAddress) string {
		return proof[address].Agent
	})
	if err != nil {
		return nil, err
	}
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
	return ProjectDetachedSessions(bindings, sessions, claimsFromBindings(current))
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
