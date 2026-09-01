package couchcore

import "time"

// ActorRecord is one live actor. Liveness is not stored: it is recomputed out
// of process by comparing Identity, because Couch owns the console and so every
// read runs in a second process with no Handle.
type ActorRecord struct {
	ID        ActorID       `json:"id"`
	Thread    ThreadAddress `json:"thread,omitempty"`
	Args      StartArgs     `json:"args"`
	StartedAt time.Time     `json:"started_at"`
	PID       int           `json:"pid"`
	Identity  string        `json:"identity"`
}

// Registry maps a worktree to the actors on it.
//
// This is a transitional display/handle cache, not an admission authority.
// ThreadStore plus normalized provider evidence owns admission.
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

// Insert adds a record without consulting the guard. It exists for replay:
// Store.Load must reproduce what was persisted, including legacy co-tenants.
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

// RemoveActor drops one incarnation, leaving any co-tenant.
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
