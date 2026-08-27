package couchcore

import (
	"sort"
)

// ArgSpec describes one argument of an operation, so a caller that is not a
// human -- the advisor's tool layer in #148 -- can construct a call without
// hardcoding couch's CLI.
type ArgSpec struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Required bool   `json:"required"`
	// FlagOnly arguments never bind positionally; they must be named with
	// --name. Use it for switches whose positional interpretation would be
	// surprising or unsafe.
	FlagOnly bool `json:"flag_only,omitempty"`
	// Implicit arguments are supplied by a trusted caller context rather than
	// accepted from CLI argv. The advisor/console dispatch schema can still name
	// them without exposing a user bypass flag.
	Implicit bool `json:"implicit,omitempty"`
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

// OperationEffect classifies what observable state an operation may change.
// Zero is invalid so adding a declaration cannot silently inherit authority.
type OperationEffect uint8

const (
	EffectUnknown OperationEffect = iota
	EffectRead
	EffectMetadata
	EffectProcess
	EffectConsole
)

// OperationConfirmation tells presentation layers whether an explicit human
// confirmation belongs before dispatch. Dispatch assumes that contract was
// satisfied; #151 owns the menu presentation.
type OperationConfirmation uint8

const (
	ConfirmUnknown OperationConfirmation = iota
	ConfirmNone
	ConfirmRequired
)

// OperationResult describes the stable result family without embedding Go
// execution in the declaration.
type OperationResult uint8

const (
	ResultUnknown OperationResult = iota
	ResultStart
	ResultThreadInventory
	ResultStop
	ResultThread
	ResultDescription
	ResultConsole
)

// Operation is one thing couch can do. The terminal UI and the advisor are
// both clients of this set; there is deliberately no second dispatch path, so
// the operator's surface and the advisor's cannot drift apart.
type Operation struct {
	Name         string
	Summary      string
	Args         []ArgSpec
	Execution    OperationExecution
	Effect       OperationEffect
	Confirmation OperationConfirmation
	Result       OperationResult
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
}

func Operations() []Operation {
	return []Operation{
		{
			Name: "start", Summary: "Start an agent on a peer repo (or a subdirectory of one)",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Args: []ArgSpec{
				// Optional, defaulting to "." in the start operation: `cd brain && couch
				// start` is what makes brain home, which is the Spec's
				// "whatever session couch launched in" delivered by convention
				// rather than by couch knowing about brain (Decision 1).
				{Name: "path", Summary: "repo or subdirectory to start in (default: .)", Required: false},
				// A stray positional word must not be able to turn off a whole
				// layer of terminal ownership.
				{Name: "no-console", Summary: "inherit couch's stdio instead of allocating a pty (--no-console)", Required: false, FlagOnly: true},
				{Name: "agent", Summary: "Pair agent to use instead of path/root history (--agent=<name>)", Required: false, FlagOnly: true},
			},
		},
		{
			Name: "list", Summary: "List every durable work thread",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultThreadInventory,
		},
		{
			Name: "show", Summary: "Show one work thread by tag, path, or name",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultThreadInventory,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or operator-assigned name", Required: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "stop", Summary: "Signal an actor's child and forget it",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStop,
			Args: []ArgSpec{{Name: "ref", Summary: "path or operator-assigned name", Required: true}},
		},
		{
			Name: "name", Summary: "Give a work thread a short human name",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or existing name", Required: true},
				{Name: "name", Summary: "the new short name", Required: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "describe", Summary: "Read or set a work thread's operator description",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultDescription,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: true},
				{Name: "description", Summary: "omit to read the cached value", Required: false},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "publish-description", Summary: "Publish this session's own one-line summary (run by the agent inside its thread)",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread,
			Args: []ArgSpec{
				{Name: "description", Summary: "what this session is working on", Required: true},
				{Name: "repo-scope", Summary: "exact thread scope from $COUCH_THREAD_SCOPE", Implicit: true},
				{Name: "tag", Summary: "exact thread tag from $COUCH_THREAD_TAG", Implicit: true},
			},
		},
		{
			Name: "switch", Summary: "Switch the operator terminal to a hosted work thread",
			Execution: ExecuteLiveOwner, Effect: EffectConsole, Confirmation: ConfirmNone, Result: ResultConsole,
			Args: []ArgSpec{
				{Name: "repo-scope", Summary: "exact hosted thread scope", Required: true, Implicit: true},
				{Name: "tag", Summary: "exact hosted thread tag", Required: true, Implicit: true},
			},
		},
		{
			Name: "attach", Summary: "Attach a newly started terminal to its durable work thread",
			Execution: ExecuteLiveOwner, Effect: EffectConsole, Confirmation: ConfirmNone, Result: ResultConsole,
			Args: []ArgSpec{
				{Name: "repo-scope", Summary: "exact started thread scope", Required: true, Implicit: true},
				{Name: "tag", Summary: "exact started thread tag", Required: true, Implicit: true},
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
