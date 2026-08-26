package couchcore

import (
	"fmt"
	"sort"
	"strings"
)

// ArgSpec describes one argument of an operation, so a caller that is not a
// human -- the advisor's tool layer in #148 -- can construct a call without
// hardcoding couch's CLI.
type ArgSpec struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Required bool   `json:"required"`
	// FlagOnly arguments never bind positionally; they must be named with
	// --name. Set it on anything that bypasses a guard, so a stray positional
	// word cannot disable a refusal -- `couch start /repo true` silently
	// turned off the one-agent-per-tree guard before this existed.
	FlagOnly bool `json:"flag_only,omitempty"`
}

// OperationExecution names the authority required to perform an operation.
// Zero is deliberately non-authorizing: newly added operations must choose an
// owner before any dispatcher can execute them.
type OperationExecution uint8

const (
	ExecuteUnknown OperationExecution = iota
	ExecuteDirectStore
	ExecuteLiveOwner
)

// Operation is one thing couch can do. The terminal UI and the advisor are
// both clients of this set; there is deliberately no second dispatch path, so
// the operator's surface and the advisor's cannot drift apart.
type Operation struct {
	Name      string
	Summary   string
	Args      []ArgSpec
	Execution OperationExecution
	Invoke    func(c *Couch, args map[string]string) (any, error)
}

// StartResult is what `start` returns before the caller waits on the child.
type StartResult struct {
	Record ActorRecord
	Handle Handle
}

// StopResult reports what stopping actually did: a record for an already-dead
// actor is forgotten without a signal, and saying so avoids implying a running
// agent was terminated.
type StopResult struct {
	Record    ActorRecord
	Signalled bool
}

// ActorView is a record plus the state that must be computed rather than
// stored -- liveness, and whatever the operator or the agent has called it.
type ActorView struct {
	Record ActorRecord `json:"record"`
	Live   bool        `json:"live"`
	State  Liveness    `json:"state"`
	Name   string      `json:"name,omitempty"`
	Desc   string      `json:"description,omitempty"`
	Mode   Mode        `json:"mode"`
}

func Operations() []Operation {
	return []Operation{
		{
			Name:      "start",
			Summary:   "Start an agent on a peer repo (or a subdirectory of one)",
			Execution: ExecuteLiveOwner,
			Args: []ArgSpec{
				// Optional, defaulting to "." in the start operation: `cd brain && couch
				// start` is what makes brain home, which is the Spec's
				// "whatever session couch launched in" delivered by convention
				// rather than by couch knowing about brain (Decision 1).
				{Name: "path", Summary: "repo or subdirectory to start in (default: .)", Required: false},
				{Name: "same-tree", Summary: "override the one-agent-per-tree guard (--same-tree)", Required: false, FlagOnly: true},
				// FlagOnly for the same reason same-tree is: it bypasses the
				// console, and a stray positional word must not be able to
				// turn off a whole layer.
				{Name: "no-console", Summary: "inherit couch's stdio instead of allocating a pty (--no-console)", Required: false, FlagOnly: true},
			},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				path := a["path"]
				if path == "" {
					path = "."
				}
				rec, h, err := c.Spawn(StartArgs{
					Cwd:      path,
					SameTree: a["same-tree"] == "true",
				})
				if err != nil {
					return nil, err
				}
				return StartResult{Record: rec, Handle: h}, nil
			},
		},
		{
			Name:      "list",
			Summary:   "List every registered actor across all worktrees",
			Execution: ExecuteDirectStore,
			Invoke: func(c *Couch, _ map[string]string) (any, error) {
				return c.Summarize(nil), nil
			},
		},
		{
			Name:      "show",
			Summary:   "Show the actors on one tree, by path or by name",
			Execution: ExecuteDirectStore,
			Args:      []ArgSpec{{Name: "ref", Summary: "path or operator-assigned name", Required: true}},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				_, trees, err := c.ResolveRef(a["ref"])
				if err != nil {
					return nil, err
				}
				return c.Summarize(trees), nil
			},
		},
		{
			Name:      "stop",
			Summary:   "Signal an actor's child and forget it",
			Execution: ExecuteLiveOwner,
			Args:      []ArgSpec{{Name: "ref", Summary: "path or operator-assigned name", Required: true}},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				recs, _, err := c.ResolveRef(a["ref"])
				if err != nil {
					return nil, err
				}
				switch {
				case len(recs) == 0:
					// Absence is not ambiguity. A parked tree used to produce
					// "matches 0 actors; be specific", which reads as a
					// disambiguation problem it is not.
					return nil, fmt.Errorf("%q has no running actor to stop", a["ref"])
				case len(recs) > 1:
					// --same-tree co-tenants share a path and a label, so the
					// ActorID is the only handle that separates them. Name it,
					// or the escape hatch creates a state couch cannot exit.
					ids := make([]string, 0, len(recs))
					for _, r := range recs {
						ids = append(ids, string(r.ID))
					}
					return nil, fmt.Errorf("%q matches %d actors; stop one by id: %s",
						a["ref"], len(recs), strings.Join(ids, " "))
				}
				signalled, err := c.Stop(recs[0])
				if err != nil {
					return nil, err
				}
				return StopResult{Record: recs[0], Signalled: signalled}, nil
			},
		},
		{
			Name:      "name",
			Summary:   "Give a tree a short human name",
			Execution: ExecuteDirectStore,
			Args: []ArgSpec{
				{Name: "ref", Summary: "path or existing name", Required: true},
				{Name: "name", Summary: "the new short name", Required: true},
			},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				w, err := c.treeFor(a["ref"])
				if err != nil {
					return nil, err
				}
				return w, c.SetName(w, a["name"])
			},
		},
		{
			Name:      "describe",
			Summary:   "Read or set a tree's one-line description",
			Execution: ExecuteDirectStore,
			Args: []ArgSpec{
				{Name: "ref", Summary: "path or name", Required: true},
				{Name: "description", Summary: "omit to read the cached value", Required: false},
			},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				w, err := c.treeFor(a["ref"])
				if err != nil {
					return nil, err
				}
				if d := a["description"]; d != "" {
					return w, c.SetDescription(w, d)
				}
				return c.Describe(w), nil
			},
		},
		{
			Name:      "publish-description",
			Summary:   "Publish this session's own one-line description (run by the agent, inside its tree)",
			Execution: ExecuteDirectStore,
			Args: []ArgSpec{
				{Name: "description", Summary: "what this session is working on", Required: true},
				{Name: "tree", Summary: "tree to publish for; defaults to $COUCH_TREE", Required: false},
			},
			Invoke: func(c *Couch, a map[string]string) (any, error) {
				ref := a["tree"]
				if ref == "" {
					return nil, fmt.Errorf("no tree given and $COUCH_TREE is unset -- run this inside a couch-spawned session")
				}
				w, err := c.treeFor(ref)
				if err != nil {
					return nil, err
				}
				return w, c.PublishDescription(w, a["description"])
			},
		},
	}
}

// OperationNames is the sorted set of declared operations. The CLI's dispatch
// table is built from Operations(), and its audit asserts identity with this
// set -- not overlap with a hand-written list, which would not catch an
// operation reachable from the CLI but never declared.
func OperationNames() []string {
	var out []string
	for _, op := range Operations() {
		out = append(out, op.Name)
	}
	sort.Strings(out)
	return out
}
