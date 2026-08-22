package couchcore

import (
	"fmt"
	"sort"
)

// Couch is the composition root: every seam in one place, every operation a
// method on it. The terminal UI and (later) the advisor's tools are both
// clients of these methods -- never of two separate implementations.
type Couch struct {
	Runner Runner
	Path   PathOps
	Git    GitRunner
	Proc   ProcOps
	Store  Store
	Clock  Clock
	IDs    IDGen

	reg    Registry
	names  NamingTable
	policy PolicyTable
}

func New(r Runner, p PathOps, g GitRunner, proc ProcOps, s Store, c Clock, ids IDGen) (*Couch, error) {
	reg, names, policy, err := s.Load()
	if err != nil {
		return nil, err
	}
	return &Couch{
		Runner: r, Path: p, Git: g, Proc: proc, Store: s, Clock: c, IDs: ids,
		reg: reg, names: names, policy: policy,
	}, nil
}

// ResolveTree turns an operator-supplied path into a canonical worktree.
func (c *Couch) ResolveTree(path string) (Worktree, error) { return Resolve(path, c.Git, c.Path) }

// Spawn brings up an agent on a tree.
//
// Order matters: the snapshot is persisted BEFORE the caller waits on the
// child. `couch start` blocks for the child's lifetime, so if Save happened
// after Wait a second shell running `couch list` would see an empty registry
// for the entire session -- which is most of the time.
func (c *Couch) Spawn(args StartArgs) (ActorRecord, Handle, error) {
	tree, err := c.ResolveTree(args.WorkingDir())
	if err != nil {
		return ActorRecord{}, nil, err
	}
	args.Worktree = tree

	// Drop records whose process is gone BEFORE consulting the guard.
	//
	// Without this the guard refuses on registry membership alone, and since
	// `couch start` blocks until the child exits and nothing unregisters on
	// exit, the ordinary end of a session leaves a dead record that refuses
	// its tree forever. The registry is a cache of what is running, so a
	// record whose identity no longer matches is not evidence of anything.
	if err := c.PruneDead(); err != nil {
		return ActorRecord{}, nil, err
	}

	// The guard is evaluated before anything is forked: a refused spawn must
	// not leave a stray child behind.
	if err := c.reg.CheckAvailable(tree, args.SameTree, c.policy); err != nil {
		return ActorRecord{}, nil, err
	}

	argv := append([]string{"pair", "--layout2"}, args.ExtraArgs...)
	// The child is told which tree it is and where couch keeps state, so the
	// agent inside it can publish its own one-line description. Without this
	// the description cache has no source: an operator typing `couch describe`
	// is not "agent-supplied".
	env := []string{
		"COUCH_TREE=" + string(tree),
		"COUCH_STORE_DIR=" + c.Store.Dir(),
	}
	h, err := c.Runner.Start(args.WorkingDir(), argv, env)
	if err != nil {
		return ActorRecord{}, nil, fmt.Errorf("spawn %s: %w", tree, err)
	}

	record := ActorRecord{
		ID:        c.IDs.NewID(),
		Args:      args,
		StartedAt: c.Clock.Now(),
		PID:       h.PID(),
		Identity:  h.Identity(),
	}
	next, err := c.reg.RegisterWithPolicy(record, c.policy)
	if err != nil {
		return ActorRecord{}, h, err
	}
	c.reg = next

	if err := c.Store.Save(c.reg, c.names); err != nil {
		return record, h, fmt.Errorf("persist registry: %w", err)
	}
	return record, h, nil
}

// IsLive recomputes liveness for a persisted record. A PID that has been
// recycled by an unrelated process reports NOT live, because the kernel start
// token differs.
//
// Known narrow window: procutil.Alive is `kill -0`, which succeeds for a
// zombie, so a child that exited but has not yet been reaped by ITS OWN parent
// reads as live here. ExecRunner reaps in the background precisely so its own
// children are never zombies, and an orphan is reparented to init and reaped
// immediately -- so the window needs a couch that spawned a child, is not
// waiting on it, and is still running. `couch start` blocks on Wait, so that
// does not arise today; revisit if a non-blocking spawn ever lands.
func (c *Couch) IsLive(a ActorRecord) bool {
	if a.PID == 0 || !c.Proc.Alive(a.PID) {
		return false
	}
	if a.Identity == "" {
		return false
	}
	return c.Proc.Identity(a.PID) == a.Identity
}

func (c *Couch) List() []ActorRecord {
	out := c.reg.Records()
	sort.Slice(out, func(i, j int) bool { return out[i].Args.Worktree < out[j].Args.Worktree })
	return out
}

func (c *Couch) Get(w Worktree) []ActorRecord { return c.reg.Get(w) }
func (c *Couch) Entry(w Worktree) NameEntry   { return c.names.Entry(w) }
func (c *Couch) Policy() PolicyTable          { return c.policy }

func (c *Couch) SetName(w Worktree, name string) error {
	c.names = c.names.SetName(w, name)
	return c.Store.Save(c.reg, c.names)
}

func (c *Couch) SetDescription(w Worktree, desc string) error {
	c.names = c.names.SetDescription(w, desc)
	return c.Store.Save(c.reg, c.names)
}

// Forget drops an actor from the registry, freeing its tree.
func (c *Couch) Forget(w Worktree, id ActorID) error {
	c.reg = c.reg.RemoveActor(w, id)
	return c.Store.Save(c.reg, c.names)
}

// ResolveRef composes the naming layer with the registry: a fuzzy human
// reference yields the actors it names. Fuzzy in, exact out -- an ambiguous
// reference returns every candidate rather than guessing.
func (c *Couch) ResolveRef(ref string) ([]ActorRecord, []Worktree, error) {
	trees := c.names.Lookup(ref)
	if len(trees) == 0 {
		if w, err := c.ResolveTree(ref); err == nil {
			trees = []Worktree{w}
		}
	}
	if len(trees) == 0 {
		return nil, nil, fmt.Errorf("no actor matches %q", ref)
	}
	var out []ActorRecord
	for _, t := range trees {
		out = append(out, c.reg.Get(t)...)
	}
	return out, trees, nil
}

// Views decorates records with the state that is computed rather than stored.
func (c *Couch) Views(recs []ActorRecord) []ActorView {
	out := make([]ActorView, 0, len(recs))
	for _, r := range recs {
		e := c.names.Entry(r.Args.Worktree)
		out = append(out, ActorView{
			Record: r,
			Live:   c.IsLive(r),
			Name:   e.Name,
			Desc:   c.Describe(r.Args.Worktree),
			Mode:   c.policy.Mode(r.Args.Worktree.Repo()),
		})
	}
	return out
}

// Describe returns the agent-supplied one-liner, preferring the sidecar the
// live agent writes and falling back to the last value couch stored.
//
// This is not the published-status artifact returning: it is a one-line LABEL,
// and a stale label still finds the right tree. Labels tolerate staleness;
// state does not.
func (c *Couch) Describe(w Worktree) string {
	if s, err := c.Store.ReadDescription(w); err == nil && s != "" {
		return s
	}
	return c.names.Entry(w).Description
}

// treeFor resolves a ref to exactly one tree, erroring on ambiguity rather
// than guessing -- fuzzy in, exact out.
func (c *Couch) treeFor(ref string) (Worktree, error) {
	if trees := c.names.Lookup(ref); len(trees) == 1 {
		return trees[0], nil
	} else if len(trees) > 1 {
		return "", fmt.Errorf("%q matches %d trees; be specific", ref, len(trees))
	}
	return c.ResolveTree(ref)
}

// TreeSummary is a worktree and whatever couch knows about it. A tree with no
// live actor is still a row: a named tree nobody is running is exactly the
// parked thread this project exists to stop losing.
type TreeSummary struct {
	Tree   Worktree    `json:"tree"`
	Name   string      `json:"name,omitempty"`
	Desc   string      `json:"description,omitempty"`
	Mode   Mode        `json:"mode"`
	Actors []ActorView `json:"actors,omitempty"`
}

// Live reports whether any actor on this tree is running.
func (s TreeSummary) Live() bool {
	for _, a := range s.Actors {
		if a.Live {
			return true
		}
	}
	return false
}

// Summarize groups actors by tree and folds in every tree that has a name or
// description but no actor, so parked threads stay visible.
func (c *Couch) Summarize(trees []Worktree) []TreeSummary {
	seen := map[string]*TreeSummary{}
	order := []string{}

	add := func(w Worktree) *TreeSummary {
		if s, ok := seen[w.Key()]; ok {
			return s
		}
		e := c.names.Entry(w)
		s := &TreeSummary{Tree: w, Name: e.Name, Desc: c.Describe(w), Mode: c.policy.Mode(w.Repo())}
		seen[w.Key()] = s
		order = append(order, w.Key())
		return s
	}

	// A non-empty filter RESTRICTS the result. Folding in every registry
	// record regardless would make the argument additive only, so `show <ref>`
	// would print exactly what `list` prints.
	want := map[string]bool{}
	for _, w := range trees {
		want[w.Key()] = true
		add(w)
	}
	for _, r := range c.reg.Records() {
		if len(want) > 0 && !want[r.Args.Worktree.Key()] {
			continue
		}
		sum := add(r.Args.Worktree)
		sum.Actors = append(sum.Actors, c.Views([]ActorRecord{r})...)
	}
	if len(trees) == 0 {
		for _, e := range c.names.All() {
			add(e.Tree)
		}
	}

	out := make([]TreeSummary, 0, len(order))
	for _, k := range order {
		out = append(out, *seen[k])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tree < out[j].Tree })
	return out
}

// PruneDead removes records whose process is gone and persists the result.
//
// Liveness is recomputed rather than stored, so the registry accumulates
// records for children that have exited. Pruning is what keeps the
// one-agent-per-tree guard meaningful: without it the guard protects a tree
// against a process that no longer exists.
func (c *Couch) PruneDead() error {
	var dead []ActorRecord
	for _, r := range c.reg.Records() {
		if !c.IsLive(r) {
			dead = append(dead, r)
		}
	}
	if len(dead) == 0 {
		return nil
	}
	for _, r := range dead {
		c.reg = c.reg.RemoveActor(r.Args.Worktree, r.ID)
	}
	return c.Store.Save(c.reg, c.names)
}

// Stop signals an actor's child and then forgets it.
//
// Order matters and is the opposite of what it was: forgetting first would
// free the tree while the agent kept running, so the next `couch start` would
// be allowed and two agents would share one index lock and one branch --
// opening the exact hazard the registry exists to close.
//
// The identity token is re-checked immediately before signalling, because a
// stale record's PID may have been recycled by an unrelated process and
// SIGTERM to the wrong pid is not recoverable.
func (c *Couch) Stop(a ActorRecord) (signalled bool, err error) {
	if c.IsLive(a) {
		if err := c.Proc.Signal(a.PID, TermSignal); err != nil {
			return false, fmt.Errorf("stop %s: %w", a.ID, err)
		}
		signalled = true
	}
	return signalled, c.Forget(a.Args.Worktree, a.ID)
}

// PublishDescription is the agent-facing half of the description: a session
// running inside a tree writes its own one-liner, which Describe then prefers
// over anything the operator typed.
func (c *Couch) PublishDescription(w Worktree, text string) error {
	return c.Store.WriteDescription(w, text)
}
