package couchcore

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
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
	// An empty path is refused rather than quietly meaning "wherever this
	// process happens to be".
	//
	// `filepath.Abs("")` returns the cwd, so an unset path used to spawn
	// somewhere plausible by accident -- which made the CLI's explicit `.`
	// default dead weight that could be deleted with every test still green
	// (found while deletion-checking M2 BR-24). Two mechanisms producing one
	// result means neither is pinned; this leaves the explicit one.
	if args.WorkingDir() == "" {
		return ActorRecord{}, nil, fmt.Errorf("spawn: no path given")
	}
	tree, err := c.ResolveTree(args.WorkingDir())
	if err != nil {
		return ActorRecord{}, nil, err
	}
	args.Worktree = tree
	// Canonicalise the recorded cwd. StartArgs is persisted so a revival can
	// reproduce the launch; storing the operator's relative path ("../pair")
	// makes that record meaningless from any other directory.
	if physical, err := c.Path.Physical(NormalizePath(args.WorkingDir())); err == nil {
		args.Cwd = physical
	} else {
		args.Cwd = NormalizePath(args.WorkingDir())
	}

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

	// `pair resume <tag> --layout2` rather than a bare `pair`.
	//
	// The tag: with none, launcher.DecideLaunch returns ActionPick as soon as a
	// detached session exists (decision.go:47), which inside couch's own pty is
	// an fzf picker waiting on an operator who only asked to start. `resume`
	// takes the ForcedTag branch -- attach if live or detached, create
	// otherwise -- and skips the name prompt (help.go:15). It derives from the
	// TREE, so going back in is deterministic: the same tree always resumes the
	// same session. `launcher.DefaultTag` is pair's own create-flow derivation,
	// reused rather than re-implemented.
	//
	// The layout: pinned to layout2 by operator decision 2026-08-22. couch owns
	// terminal switching now, so layout3's third pane -- pair's own user
	// terminal -- is the layer couch replaces. Provisional ("for now"), which is
	// why it is a literal here rather than a knob nobody has asked for.
	//
	// A correction worth keeping: an earlier version of this comment claimed
	// `resume` REFUSES a third argv element and that --layout2 was therefore
	// impossible. Only POSITIONALS are refused -- `ParseArgs` runs
	// `extractLayoutRequest` first (args.go:51), which strips layout flags
	// before the guard ever sees them, and `launchArgsAcceptLayout` admits them
	// for resume because its Command is "". Measured, not reasoned:
	// `resume mytag --layout2` parses to {tag, layout2}; `resume mytag stray`
	// is the thing that errors.
	//
	// This is a deliberate slice of #149, which makes the tag the space's
	// durable identity; #146 needs only that re-entry is deterministic.
	argv := append([]string{"pair", "resume", launcher.DefaultTag(string(tree)), "--layout2"}, args.ExtraArgs...)
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

// Liveness recomputes an actor's state from the persisted {PID, Identity}.
//
// Three-valued on purpose. A PID recycled by an unrelated process reports Dead
// because the kernel start token differs; a probe that cannot answer reports
// Unknown, and callers that act destructively must treat Unknown as "leave it
// alone".
func (c *Couch) Liveness(a ActorRecord) Liveness {
	if a.PID == 0 || a.Identity == "" {
		return Dead // nothing was ever recorded to check against
	}
	switch c.Proc.Exists(a.PID) {
	case Dead:
		return Dead
	case Unknown:
		return Unknown
	}
	id, err := c.Proc.Identity(a.PID)
	if err != nil {
		// The process exists but we could not read its token. That is not
		// evidence of anything; refusing to guess is the safe answer.
		return Unknown
	}
	if id != a.Identity {
		return Dead // same PID, different process
	}
	return Live
}

// IsLive is the display-side convenience. Do not branch on it for anything
// destructive -- it folds Unknown into false.
func (c *Couch) IsLive(a ActorRecord) bool { return c.Liveness(a) == Live }

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

// knownTrees is every tree couch knows about: those with actors and those with
// only a label. Both are addressable -- a parked tree is exactly the thread an
// operator loses track of.
func (c *Couch) knownTrees() []Worktree {
	seen := map[string]bool{}
	var out []Worktree
	add := func(w Worktree) {
		if w != "" && !seen[w.Key()] {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	for _, r := range c.reg.Records() {
		add(r.Args.Worktree)
	}
	for _, e := range c.names.All() {
		add(e.Tree)
	}
	return out
}

// LookupTrees resolves a fuzzy human reference to every tree it could mean.
//
// It matches the operator's name, the operator's typed description, AND the
// agent's own published line. All three answer "what is this thread called",
// so all three derive from one lookup -- displaying the agent's description
// while resolving only the operator's delivers half the behaviour.
func (c *Couch) LookupTrees(ref string) []Worktree {
	needle := strings.ToLower(strings.TrimSpace(ref))
	if needle == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []Worktree
	for _, w := range c.names.Lookup(ref) {
		if !seen[w.Key()] {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	for _, w := range c.knownTrees() {
		if seen[w.Key()] {
			continue
		}
		if strings.Contains(strings.ToLower(c.Describe(w)), needle) {
			seen[w.Key()] = true
			out = append(out, w)
		}
	}
	return out
}

// ResolveRef turns a human reference into the actors it names.
//
// A reference may be an ActorID, a label (operator name, operator description,
// or the agent's published line), or a path. The ActorID branch exists because
// --same-tree puts two actors on one tree sharing a path AND a label; without
// it that state cannot be exited, since stop refuses on ambiguity and the
// remedy it suggested did not exist.
func (c *Couch) ResolveRef(ref string) ([]ActorRecord, []Worktree, error) {
	trimmed := strings.TrimSpace(ref)

	for _, r := range c.reg.Records() {
		if string(r.ID) == trimmed {
			return []ActorRecord{r}, []Worktree{r.Args.Worktree}, nil
		}
	}

	trees := c.LookupTrees(trimmed)
	if len(trees) == 0 {
		if w, err := c.ResolveTree(trimmed); err == nil {
			trees = []Worktree{w}
		}
	}
	if len(trees) == 0 {
		return nil, nil, fmt.Errorf("no actor or tree matches %q", ref)
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
			Live:   c.Liveness(r) == Live,
			State:  c.Liveness(r),
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
	if trees := c.LookupTrees(ref); len(trees) == 1 {
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
		// Only a KNOWN-dead record is pruned. Pruning on Unknown deletes a
		// live actor's registration whenever the probe fails, and then lets a
		// second agent onto its tree -- observed in smoke testing, where a
		// sandboxed probe destroyed a running session's record.
		if c.Liveness(r) == Dead {
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
	// Signal on Live or Unknown: refusing to signal because we could not
	// confirm liveness would leave a running agent behind while freeing its
	// tree, which is the hazard Stop exists to close.
	if c.Liveness(a) != Dead {
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
