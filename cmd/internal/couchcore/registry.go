package couchcore

import (
	"fmt"
	"time"
)

// ActorRecord is one live actor. Liveness is not stored: it is recomputed out
// of process by comparing Identity, because `couch start` blocks and so every
// read runs in a second process with no Handle.
type ActorRecord struct {
	ID        ActorID       `json:"id"`
	Thread    ThreadAddress `json:"thread,omitempty"`
	Args      StartArgs     `json:"args"`
	StartedAt time.Time     `json:"started_at"`
	PID       int           `json:"pid"`
	Identity  string        `json:"identity"`
}

// TreeOccupiedError reports a refused registration. It carries the incumbents
// so the caller can name them, and the policy Mode so the offer is
// policy-shaped -- suggesting a new worktree is bad advice for a repo whose
// worktrees are expensive.
type TreeOccupiedError struct {
	Tree       Worktree
	Incumbents []ActorRecord
	Mode       Mode
}

func (e *TreeOccupiedError) Error() string {
	return fmt.Sprintf("%s already has %d agent(s)", e.Tree, len(e.Incumbents))
}

// Registry maps a worktree to the actors on it.
//
// The one-agent-per-tree invariant IS the collision guard -- Register failing
// when the key is taken is Erlang's register/2, not a separate check. The
// value is a slice so the --same-tree escape hatch produces two enumerable
// actors rather than overwriting the incumbent and orphaning a live child
// whose handle would then be unreachable.
type Registry struct {
	byTree map[string][]ActorRecord
}

func NewRegistry() Registry { return Registry{byTree: map[string][]ActorRecord{}} }

func (r Registry) Get(w Worktree) []ActorRecord { return r.byTree[w.Key()] }

// Records returns every registered actor, flattened.
//
// It deliberately does not expose the folded map keys: a caller tempted to
// rebuild a Worktree from a key would get the lowercased path and lose the
// original case, which matters because that string is fed to
// launcher.ResolveRepoScope and is what gets displayed. Every record already
// carries its unfolded Args.Worktree.
func (r Registry) Records() []ActorRecord {
	var out []ActorRecord
	for _, recs := range r.byTree {
		out = append(out, recs...)
	}
	return out
}

func (r Registry) Register(a ActorRecord) (Registry, error) {
	return r.RegisterWithPolicy(a, PolicyTable{})
}

// CheckAvailable reports whether a tree will accept a registration, without
// performing one. Spawn needs this so the guard is evaluated BEFORE a child
// process is started -- otherwise a refused spawn has already forked.
func (r Registry) CheckAvailable(tree Worktree, sameTree bool, p PolicyTable) error {
	existing := r.byTree[tree.Key()]
	if len(existing) > 0 && !sameTree {
		return &TreeOccupiedError{
			Tree:       tree,
			Incumbents: append([]ActorRecord(nil), existing...),
			Mode:       p.Mode(tree.Repo()),
		}
	}
	return nil
}

// RegisterWithPolicy copies before mutating. Registry wraps a map, which is a
// reference type, so a value-semantics signature over a shared map would be a
// lie: a failed Register would mutate the caller's state anyway.
func (r Registry) RegisterWithPolicy(a ActorRecord, p PolicyTable) (Registry, error) {
	tree := a.Args.Worktree
	existing := r.byTree[tree.Key()]
	if len(existing) > 0 && !a.Args.SameTree {
		return r, &TreeOccupiedError{
			Tree:       tree,
			Incumbents: append([]ActorRecord(nil), existing...),
			Mode:       p.Mode(tree.Repo()),
		}
	}
	next := Registry{byTree: r.copyMap()}
	next.byTree[tree.Key()] = append(append([]ActorRecord(nil), existing...), a)
	return next, nil
}

// Insert adds a record without consulting the guard. It exists for replay:
// Store.Load must reproduce what was persisted, including a co-tenant pair,
// without either tripping the refusal or fabricating SameTree on records that
// never used the escape hatch.
func (r Registry) Insert(a ActorRecord) Registry {
	next := Registry{byTree: r.copyMap()}
	key := a.Args.Worktree.Key()
	next.byTree[key] = append(next.byTree[key], a)
	return next
}

func (r Registry) Unregister(w Worktree) Registry {
	next := Registry{byTree: r.copyMap()}
	delete(next.byTree, w.Key())
	return next
}

// RemoveActor drops one incarnation, leaving any co-tenant under --same-tree.
func (r Registry) RemoveActor(w Worktree, id ActorID) Registry {
	next := Registry{byTree: r.copyMap()}
	kept := next.byTree[w.Key()][:0:0]
	for _, a := range next.byTree[w.Key()] {
		if a.ID != id {
			kept = append(kept, a)
		}
	}
	if len(kept) == 0 {
		delete(next.byTree, w.Key())
	} else {
		next.byTree[w.Key()] = kept
	}
	return next
}

func (r Registry) copyMap() map[string][]ActorRecord {
	out := make(map[string][]ActorRecord, len(r.byTree))
	for k, v := range r.byTree {
		out[k] = append([]ActorRecord(nil), v...)
	}
	return out
}
