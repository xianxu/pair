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

	// The guard is evaluated before anything is forked: a refused spawn must
	// not leave a stray child behind.
	if err := c.reg.CheckAvailable(tree, args.SameTree, c.policy); err != nil {
		return ActorRecord{}, nil, err
	}

	argv := append([]string{"pair", "--layout2"}, args.ExtraArgs...)
	h, err := c.Runner.Start(args.WorkingDir(), argv, nil)
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
