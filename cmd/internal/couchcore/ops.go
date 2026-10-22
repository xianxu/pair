package couchcore

import (
	"sort"
)

const OpRecoverPark = "recover-park"

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
	ResultRecoveryPreview
	ResultStart
	ResultThreadInventory
	ResultStop
	ResultThread
	ResultConsole
	ResultWorkspace
	ResultRepositoryAlias
	ResultRecoverPlan
	ResultPeek
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
	// PresentationRecoverPlan is `couch --recover-plan-from-sdlc` (pair#367).
	PresentationRecoverPlan
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
	Configuration *AcceptedConfiguration `json:"-"`
	Record        ActorRecord
	Handle        Handle
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
func (s StartResult) Started() (StartResult, bool) {
	if s.Configuration == nil {
		s.Configuration = s.Record.configuration
	}
	return s, true
}

// Warning combines attachment notices and a failed accepted-configuration
// publication. These notices are not persisted.
func (s StartResult) Warning() string {
	s, _ = s.Started()
	warning := s.Configuration.publicationWarning()
	if warning == "" {
		return s.Record.AttachmentNotice
	}
	if s.Record.AttachmentNotice == "" {
		return warning
	}
	return s.Record.AttachmentNotice + "\n" + warning
}

// StopResult reports what stopping actually did: a record for an already-dead
// actor is forgotten without a signal, and saying so avoids implying a running
// agent was terminated.
type StopResult struct {
	Record    ActorRecord
	Signalled bool
}

func Operations() []Operation {
	return []Operation{
		{Name: "keys", Summary: "Drive bounded named keys in an idle owned slot", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultConsole, Presentation: PresentationTUI},
		{Name: "set-model", Summary: "Select and verify a model in an idle owned slot", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultConsole, Presentation: PresentationTUI},
		{Name: "interrupt", Summary: "Send one Ctrl+C into a working turn in an owned slot", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultConsole, Presentation: PresentationTUI},
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
			// Reconcile converges one :1+ slot to a working slot (pair#387):
			// the explicit form of what open, resume, reboot and add slot do
			// first. It ignores a remembered setup failure, so it is how an
			// operator retries after fixing a cause outside the slot. It runs
			// in the calling process; the host creation lease serializes it
			// with a running Couch.
			Name: "reconcile", Summary: "Converge a slot's workspace and show what it did",
			Execution: ExecuteDirectStore, Effect: EffectProcess, Confirmation: ConfirmNone,
			Result: ResultWorkspace, Presentation: PresentationShow,
			Args: []ArgSpec{
				{Name: "ref", Summary: "slot reference (repo:N) or a path in the slot", Required: true},
			},
		},
		{
			// Read-only look at another slot (pair#362): the coordinator's
			// evidence that a peer message reached the composer. It replays a
			// recording, and each replay parks one emulator goroutine for the
			// life of the process (scrollbackcmd.RenderLines), so it is a CLI
			// operation: the long-running console must not dispatch it.
			Name: "peek", Summary: "Show a slot's recent terminal, and with --transcripts where its logs live (read-only)",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone,
			Result: ResultPeek, Presentation: PresentationShow,
			Args: []ArgSpec{
				{Name: "ref", Summary: "slot reference (repo:N; several as repo:1:2,other:0), thread tag or path", Required: true},
				{Name: "lines", Summary: "how many recent lines to show per slot (default 40)", FlagOnly: true, ValueRequired: true},
				{Name: "json", Summary: "print the result as JSON", FlagOnly: true},
				{Name: "previous", Summary: "select the exact previous incarnation", FlagOnly: true},
				{Name: "all", Summary: "replay a complete recording within safety limits (heavy)", FlagOnly: true},
				{Name: "transcripts", Summary: "also name the sent-prompt log and native transcripts (slower)", FlagOnly: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
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
			Name: "switch-agent", Summary: "Switch to another coding agent using the saved startup configuration",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: switchAgentArguments(true),
		},
		{
			Name: "move-slot", Summary: "Move exact slots to another active hub",
			Execution: ExecuteLiveOwner, Effect: EffectConsole, Confirmation: ConfirmNone, Result: ResultConsole,
			Presentation: PresentationTUI,
		},
		{
			Name: "start-hub", Summary: "Open an explicitly selected hub using normal startup continuity",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI,
			Args:         []ArgSpec{{Name: "path", Summary: "canonical hub directory", Required: true, Implicit: true}},
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
			Name: "recover-plan", Summary: "Report per slot what to do after a restart, from sdlc's claims and Couch's threads",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultRecoverPlan,
			Presentation: PresentationRecoverPlan,
		},
		{
			Name: "show", Summary: "Show one work thread by tag or path",
			Execution: ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultThreadInventory,
			Presentation: PresentationShow,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag or path", Required: true},
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
			Name: "alias", Summary: "Read or set a repository's short name for slot references",
			Execution: ExecuteDirectStore, Effect: EffectMetadata, Confirmation: ConfirmNone, Result: ResultRepositoryAlias,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "repository name, alias, repo:N, or path", Required: true},
				{Name: "alias", Summary: "the new short name; omit to read it", Required: false},
				{Name: "clear", Summary: "remove the alias", FlagOnly: true},
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
			Name: "continue-thread", Summary: "Execute an accepted continuation, or resume one Couch left mid-way",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart, Presentation: PresentationInternal,
			Args: continuationArguments(),
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
				{Name: "ref", Summary: "thread tag or path", Required: false},
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
				{Name: "ref", Summary: "thread tag or path", Required: false},
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
			Name: OpRecoverPark, Summary: "Resolve an interrupted park, preserving its conversation without starting an agent",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultThread,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "tag", Summary: "exact retained thread tag", Required: true, Implicit: true},
				{Name: "repo-scope", Summary: "exact retained repository scope", Required: true, Implicit: true},
				{Name: "park-nonce", Summary: "original park transaction, rechecked before effects", Implicit: true},
			},
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
				{Name: "ref", Summary: "thread tag or path", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				{Name: "require-settled", Summary: "admission's idle evidence, known or forced; re-checked before parking (pair#421)", Implicit: true},
			},
		},
		{
			// Reload-context (pair#421) is Shift+Alt+N from outside the slot:
			// a fresh agent conversation in the same Pair process. Confirmed
			// because it ends a conversation; remote callers only (no row key).
			Name: OpReloadContext, Summary: "Start a fresh agent conversation in a live slot's Pair",
			// TUI presentation like the other socket slot verbs (its home is
			// atlas/couch.md); not a row action, so the switcher never offers it.
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultThread,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag or path", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				{Name: "require-settled", Summary: "admission's idle evidence, known or forced (pair#421)", Implicit: true},
			},
		},
		{
			// Prepare (pair#434) is `couch --prepare`'s admission and nothing
			// else: the CLI then runs weave refresh itself, off the console's
			// serial queue, and reload-context. Confirmed because the verb it
			// admits ends a conversation; a read, since this half changes nothing.
			Name: OpPrepare, Summary: "Admit an idle, clean slot on its resting branch for couch --prepare",
			Execution: ExecuteLiveOwner, Effect: EffectRead, Confirmation: ConfirmRequired, Result: ResultThread,
			Presentation: PresentationTUI,
			Args: []ArgSpec{
				{Name: "ref", Summary: "thread tag or path", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				{Name: "require-settled", Summary: "admission's idle evidence, known or forced (pair#421)", Implicit: true},
				{Name: "path", Summary: "the slot's checkout, as admission resolved it", Implicit: true},
			},
		},
		{
			// Reboot is the operator's way out of a conversation that cannot
			// come back (pair#363): it archives the record with its evidence,
			// so it is reversible in the way archive was, and confirmed because
			// it stops a detached row's running agent.
			Name: "reboot", Summary: "Archive this conversation and start a fresh agent in the same slot or path",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "ref", Summary: "exact slot reference"},
				{Name: "path", Summary: "slot host checkout", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Implicit: true},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "agent", Summary: "agent for the fresh conversation", FlagOnly: true, ValueRequired: true},
			},
		},
		{
			// Reap ends an orphaned server's whole tree (#399): its agent may
			// still be writing, so it is confirmed, and it admits only a row
			// whose orphan verdict holds on a second observation.
			Name: "reap", Summary: "End an orphaned zellij server and everything under it so its thread can be resumed",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmRequired, Result: ResultStop,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "path", Summary: "slot host checkout", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Implicit: true},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
			},
		},
		{
			// Recover runs the report's steps for one row: resume, reap then
			// resume, or reboot (#399). Choosing it is the consent (operator,
			// 2026-10-07): within its envelope it only stops a server nothing
			// can reach or reboots a conversation that cannot be resolved, and
			// it never changes the slot's files. A held row refuses.
			Name: "recover", Summary: "Run the recovery report's steps for one thread: resume, reap then resume, or reboot",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "path", Summary: "slot host checkout", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Implicit: true},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
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
		{Name: "recovery-confirmation", Summary: "Confirm the exact recovery stop set", Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart, Presentation: PresentationInternal, Args: []ArgSpec{{Name: "token", Summary: "short-lived capability for the exact previously offered stop set", Required: true, FlagOnly: true, ValueRequired: true}}},
		{Name: "resume-candidates", Summary: "List explicit conversation recovery choices", Execution: ExecuteLiveOwner, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultRecoveryPreview, Presentation: PresentationTUI,
			Args: []ArgSpec{{Name: "ref", Summary: "thread or exact slot reference"}, {Name: "path", Summary: "slot host checkout", Implicit: true}, {Name: "repo-scope", Summary: "repository scope", Implicit: true}, {Name: "tag", Summary: "exact thread tag", Implicit: true}},
		},
		{
			Name: "resume", Summary: "Reattach a detached work thread, or resume one whose conversation still resolves",
			Execution: ExecuteLiveOwner, Effect: EffectProcess, Confirmation: ConfirmNone, Result: ResultStart,
			Presentation: PresentationTUI, RowAction: true,
			Args: []ArgSpec{
				{Name: "candidate", Summary: "opaque recovery choice key", FlagOnly: true, ValueRequired: true},
				{Name: "ref", Summary: "thread tag or path", Required: false},
				{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
				{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
				// Implicit, so the CLI can never send it: only couch's own background
				// reattach pass may ask for a resume that refuses to start an agent
				// (pair#206). Its absence is today's behaviour.
				{Name: "warm-only", Summary: "refuse unless the thread is detached; never start an agent", Implicit: true},
				{Name: "path", Summary: "slot host checkout", Implicit: true},
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
		{Name: "argv", Summary: "JSON array of startup parameters; empty array selects no parameters", Implicit: true},
	}
	if accepted {
		args = append(args,
			ArgSpec{Name: "fingerprint", Summary: "accepted preview fingerprint", Required: true, Implicit: true},
			ArgSpec{Name: "interrupt-first", Summary: "explicit cooperative interruption consent; fresh idle still required", Implicit: true},
			ArgSpec{Name: "known-config", Summary: "use existing saved target configuration", Implicit: true},
			ArgSpec{Name: "source", Summary: "authenticated source incarnation token", Implicit: true},
		)
	}
	return args
}

func continuationArguments() []ArgSpec {
	return []ArgSpec{{Name: "repo-scope", Summary: "exact repository scope for the continuation", Required: true, Implicit: true}, {Name: "tag", Summary: "exact thread tag supplied by the owner", Implicit: true}, {Name: "request-id", Summary: "stable retained continuation request ID", Required: true, Implicit: true}}
}
