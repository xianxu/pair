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
	// ValueRequired distinguishes named value flags from boolean switches.
	// Callers must provide --name=<non-empty-value>; absence of '=' or an empty
	// value is invalid rather than a synthetic boolean or fallback selection.
	ValueRequired bool `json:"value_required,omitempty"`
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
	EffectAuthority
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
	ResultStartResolution
	ResultStart
	ResultThreadInventory
	ResultStop
	ResultThread
	ResultDescription
	ResultConsole
	ResultOrientationStatus
	ResultWorkspace
)

// OperationPresentation assigns every typed operation exactly one UI/process
// home. Zero is deliberately invalid so a new operation cannot become argv
// reachable merely by being added to the registry.
type OperationPresentation uint8

const (
	PresentationUnknown OperationPresentation = iota
	PresentationTUI
	PresentationList
	PresentationShow
	PresentationInternal
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
	Presentation OperationPresentation
	// RowAction declares that the switcher offers this operation on a thread
	// row. It is membership only -- WHICH row states offer it stays in
	// menuActionItems, because that is real UI policy a bool cannot carry --
	// but membership now has one source instead of two.
	//
	// PresentationTUI is not the same question: switch, attach, leave and start
	// are all TUI operations that no row offers.
	RowAction bool
}

// StartResult is what `start` returns before the caller waits on the child.
type StartResult struct {
	Record ActorRecord
	Handle Handle
}

// StartedChild is implemented by any operation result that hands the console a
// newly started child to adopt.
//
// Adoption is a PROPERTY of the result, not a closed list of concrete types.
// finishOperation used to assert on StartResult alone, so relaunch -- which
// returns its own struct around the same Record and Handle -- spawned a child
// that was never adopted. It ran as couch's orphan: the record said live, the
// switcher rendered "live", the status bar had no pane, and Return could not
// reach it. A second operation that starts a child is exactly the case a type
// switch cannot be trusted to remember.
type StartedChild interface {
	Started() (StartResult, bool)
}

// Started makes a StartResult its own adoption payload.
func (s StartResult) Started() (StartResult, bool) { return s, true }

// StopResult reports what stopping actually did: a record for an already-dead
// actor is forgotten without a signal, and saying so avoids implying a running
// agent was terminated.
type StopResult struct {
	Record    ActorRecord
	Signalled bool
}

func Operations() []Operation {
	return []Operation{
		{Name: "open-slot", Summary: "Open or recover this durable slot", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart, Presentation: PresentationTUI, RowAction: true, Args: []ArgSpec{{Name: "path", Summary: "slot host checkout", Required: true}, {Name: "agent", Summary: "agent when initialization needs a profile", FlagOnly: true, ValueRequired: true}}},
		{Name: "fresh-slot", Summary: "Start a fresh conversation in this slot", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart, Presentation: PresentationTUI, RowAction: true, Args: []ArgSpec{{Name: "path", Summary: "slot host checkout", Required: true}, {Name: "agent", Summary: "agent for the fresh conversation", FlagOnly: true, ValueRequired: true}}},
		{
			Name: "provision-workspace", Summary: "Prepare a durable numbered workspace",
			Execution: ExecuteDirectStore, Effect: EffectProcess, Confirmation: ConfirmNone,
			Result: ResultWorkspace, Presentation: PresentationInternal,
			Args: []ArgSpec{
				{Name: "path", Summary: "primary repository path", Required: true},
				{Name: "slot", Summary: "positive workspace number", Required: true, FlagOnly: true, ValueRequired: true},
				{Name: "remote", Summary: "configured Git remote", FlagOnly: true, ValueRequired: true},
			},
		},
		{
			Name: "prepare-start", Summary: "Resolve a start request without starting anything",
			Execution: ExecuteLiveOwner, Effect: EffectAuthority, Confirmation: ConfirmNone, Result: ResultStartResolution,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "path", Summary: "repo or subdirectory to start in (default: .)", Required: false},
				{Name: "action", Summary: "open or create", FlagOnly: true, ValueRequired: true},
				{Name: "agent", Summary: "Pair agent to use instead of path/root history (--agent=<name>)", Required: false, FlagOnly: true, ValueRequired: true},
			},
		},
		{
			Name: "prepare-switch-agent", Summary: "Preview fresh agent startup parameters for a thread",
			Execution: ExecuteLiveOwner, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultStartResolution,
			Presentation: PresentationTUI,
			Args:         switchAgentArguments(false),
		},
		{
			Name: "switch-agent", Summary: "Start a fresh agent context using the outgoing thread's evidence",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: switchAgentArguments(true),
		},
		{
			Name: "orientation-status", Summary: "Read delivery status for one fresh agent launch",
			Execution: ExecuteLiveOwner, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultOrientationStatus,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "repo-scope", Summary: "exact thread repository scope", Required: true, Implicit: true},
				{Name: "tag", Summary: "exact thread tag", Required: true, Implicit: true},
				{Name: "agent", Summary: "target coding agent", Required: true, Implicit: true},
				{Name: "attempt", Summary: "unique fresh launch attempt", Required: true, Implicit: true},
			},
		},
		{
			Name: "start", Summary: "Start an agent on a peer repo (or a subdirectory of one)",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "path", Summary: "canonical path the preview resolved", Required: true, Implicit: true},
				{Name: "action", Summary: "accepted open or create intent", Implicit: true},
				{Name: "agent", Summary: "agent the operator explicitly requested, if any", Required: false, Implicit: true},
				{Name: "issue", Summary: "issue the preview resolved, if any", Required: false, Implicit: true},
				{Name: "fingerprint", Summary: "fingerprint of the resolution the preview accepted", Required: true, Implicit: true},
			},
		},
		{
			Name: "list", Summary: "List every durable work thread",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultThreadInventory,
			Presentation: PresentationList,
		},
		{
			Name: "show", Summary: "Show one work thread by tag, path, or name",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultThreadInventory,
			Presentation: PresentationShow,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or operator-assigned name", Required: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "stop", Summary: "Signal an actor's child and forget it",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStop,
			Presentation: PresentationTUI,
			Args:         []ArgSpec{{Name: "ref", Summary: "path or operator-assigned name", Required: true}},
		},
		{
			Name: "name", Summary: "Give a work thread a short human name",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or existing name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "name", Summary: "the new short name", Required: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "describe", Summary: "Read or set a work thread's operator description",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultDescription,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "description", Summary: "omit to read the cached value", Required: false},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},

		{
			Name: "request-continuation", Summary: "Accept an exact checkpoint for this hosted thread",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread, Presentation: PresentationInternal,
			Args: []ArgSpec{
				{Name: "path", Summary: "absolute path to the saved checkpoint", Required: true},
				{Name: "repo-scope", Summary: "exact repository scope from the hosted thread", Required: true, Implicit: true}, {Name: "tag", Summary: "exact hosted thread tag", Required: true, Implicit: true},
				{Name: "agent", Summary: "source agent matching the checkpoint", Required: true, Implicit: true}, {Name: "session", Summary: "exact source Pair session", Required: true, Implicit: true}, {Name: "launch-ordinal", Summary: "current source launch generation", Required: true, Implicit: true}, {Name: "expected-digest", Summary: "SHA-256 of the checkpoint accepted by the writer", Required: true, Implicit: true},
			},
		},
		{
			Name: "continue-thread", Summary: "Execute or reconcile an accepted continuation",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart, Presentation: PresentationInternal,
			Args: continuationArguments(false),
		},
		{
			Name: "retry-continuation", Summary: "Retry the retained continuation without duplicating its target",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart, Presentation: PresentationInternal, RowAction: true,
			Args: continuationArguments(true),
		},
		{
			// A record write that stops nothing: direct-store, and no confirmation,
			// like retry. Arguments take name's shape -- an optional ref for the CLI
			// and the exact implicit tag from the switcher, never both (#280).
			Name: "dismiss-continuation", Summary: "Drop a failed continuation the thread has moved on from",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread, Presentation: PresentationInternal, RowAction: true,
			Args: continuationArguments(true),
		},
		{
			Name: "continuation-status", Summary: "Reconcile the exact continuation delivery receipt",
			Execution: ExecuteLiveOwner, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread, Presentation: PresentationInternal,
			Args: append(continuationArguments(false), ArgSpec{Name: "attempt", Summary: "exact continuation launch attempt", Implicit: true}),
		},
		{
			Name: "publish-description", Summary: "Publish this session's own one-line summary (run by the agent inside its thread)",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultThread,
			Presentation: PresentationInternal,
			Args: []ArgSpec{
				{Name: "description", Summary: "what this session is working on", Required: true},
				{Name: "repo-scope", Summary: "exact thread scope from $COUCH_THREAD_SCOPE", Implicit: true},
				{Name: "tag", Summary: "exact thread tag from $COUCH_THREAD_TAG", Implicit: true},
			},
		},
		{
			Name: "switch", Summary: "Switch the operator terminal to a hosted work thread",
			Execution: ExecuteLiveOwner, Effect: EffectConsole, Confirmation: ConfirmNone, Result: ResultConsole,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "repo-scope", Summary: "exact hosted thread scope", Required: true, Implicit: true},
				{Name: "tag", Summary: "exact hosted thread tag", Required: true, Implicit: true},
			},
		},
		{
			Name: "attach", Summary: "Attach a newly started terminal to its durable work thread",
			Execution: ExecuteLiveOwner, Effect: EffectConsole, Confirmation: ConfirmNone, Result: ResultConsole,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "repo-scope", Summary: "exact started thread scope", Required: true, Implicit: true},
				{Name: "tag", Summary: "exact started thread tag", Required: true, Implicit: true},
				// Implicit: only couch's own reattach pass sets it (pair#206). A
				// background attach adds its pane without taking focus.
				{Name: "background", Summary: "attach without taking focus (the reattach pass)", Implicit: true},
			},
		},
		{
			Name: "park", Summary: "Fully quit a work thread after verified Pair cleanup",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultThread,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "mode", Summary: "normal, retry, recover, or abandon (--mode=<mode>)", FlagOnly: true, ValueRequired: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			// Detach is the warm counterpart to park: the agent keeps running
			// behind its zellij session and only the client goes. Nothing is
			// destroyed, so unlike park it needs no confirmation.
			Name: "detach", Summary: "Stop a work thread's Pair client and leave its session running",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultThread,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			// The archive is inspectable on purpose: retiring a thread is a
			// decision the operator can undo, and an undo they cannot see is
			// not one they will trust.
			Name: "archived", Summary: "List work threads removed from Couch",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone,
			Result: ResultThreadInventory, Presentation: PresentationList,
		},
		{
			// Relaunch replaces a thread's Pair process with the current binary
			// and keeps the agent conversation. Confirmed because it stops a
			// running agent; live-owner because both halves it composes are.
			Name: "relaunch", Summary: "Restart a work thread's Pair, keeping its agent conversation",
			// ResultStart, like resume: a completed relaunch hands back a child
			// the console must adopt. Declaring ResultThread said otherwise and
			// the declaration was simply false.
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			// Archiving is the operator's "delete": remove a thread from the
			// switcher so they can start anew. It is reversible by design --
			// the record moves rather than being destroyed -- but it is still
			// confirmed, because a row leaving the working set is exactly the
			// kind of change that should not happen by a mistyped keystroke.
			Name: "archive", Summary: "Remove a work thread from Couch, keeping its record in the archive",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultThread,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "leave", Summary: "Apply one disposition to every live work thread and leave Couch",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultConsole,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "mode", Summary: "detach (default) or park every live thread (--mode=<mode>)", FlagOnly: true, ValueRequired: true},
			},
		},
		{
			Name: "recover-thread", Summary: "Recover a surviving session or the retained checkpoint",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name"},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
			},
		},
		{
			Name: "recover-checkpoint", Summary: "Start a new conversation from a selected checkpoint",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name"},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				{Name: "path", Summary: "absolute checkpoint path", Required: true, FlagOnly: true, ValueRequired: true},
			},
		},
		{
			Name: "resume", Summary: "Reattach a detached work thread, or resume one whose conversation still resolves",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag, path, or name", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				// Implicit, so the CLI can never send it: only couch's own background
				// reattach pass may ask for a resume that refuses to start an agent
				// (pair#206). Its absence is today's behaviour.
				{Name: "warm-only", Summary: "refuse unless the thread is detached; never start an agent", Implicit: true},
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

func switchAgentArguments(accepted bool) []ArgSpec {
	args := []ArgSpec{
		{Name: "repo-scope", Summary: "exact thread repository scope", Required: true, Implicit: true},
		{Name: "tag", Summary: "exact thread tag", Required: true, Implicit: true},
		{Name: "agent", Summary: "selected coding agent", Required: true, Implicit: true},
		{Name: "argv", Summary: "JSON array of startup parameters; empty array selects no parameters", Required: accepted, Implicit: true},
	}
	if accepted {
		args = append(args, ArgSpec{Name: "fingerprint", Summary: "accepted preview fingerprint", Required: true, Implicit: true})
	}
	return args
}

func continuationArguments(operatorFacing bool) []ArgSpec {
	args := []ArgSpec{{Name: "repo-scope", Summary: "exact repository scope for the continuation", Required: true, Implicit: true}, {Name: "tag", Summary: "exact thread tag supplied by the owner", Implicit: true}, {Name: "request-id", Summary: "stable retained continuation request ID", Required: !operatorFacing, Implicit: true}}
	// Operator-facing entries (retry, dismiss) are addressed by the switcher's
	// exact implicit tag or a CLI ref, and default to the retained request.
	if operatorFacing {
		// Optional, like name's and describe's: the switcher addresses the row by
		// its exact implicit tag, and resolveOperationThread refuses a call that
		// carries both. A required ref forced the switcher to send both, so its
		// Retry continuation never reached the thread (#280).
		args = append([]ArgSpec{{Name: "ref", Summary: "thread tag, path, or operator-assigned name"}}, args...)
	}
	return args
}
