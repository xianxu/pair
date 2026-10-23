package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OperationCall is the transport-neutral request consumed by every operation
// client. TypedPayload is reserved for owner-local capabilities (currently a
// terminal-bearing StartResult); it is never inferred from argv.
type OperationCall struct {
	Name          string
	Args          map[string]string
	Implicit      bool
	TypedPayload  any
	Context       context.Context
	Operation     Operation
	FailureTarget *LifecycleFailureTarget // owner-local frozen facts, never argv
}

type OperationExecutor func(OperationCall) (any, error)

// OperationExecutors supplies capability rather than hiding it in Operation.
// A store-only client intentionally leaves LiveOwner nil until #147 can route
// to the singleton process that owns the actors and terminal.
type OperationExecutors struct {
	DirectStore       OperationExecutor
	LiveOwner         OperationExecutor
	Attempt           func(OperationCall) (*OperationAttempt, error)
	DiagnosticWarning func(string)
}

// OwnerRoutingRequiredError is the typed refusal for an owner-required call
// made outside the singleton Couch process.
type OwnerRoutingRequiredError struct{ Operation string }

func (e *OwnerRoutingRequiredError) Error() string {
	return fmt.Sprintf("%s requires the live couch owner; routing requires #147", e.Operation)
}

// DispatchOperation resolves one declaration, validates its schema, and calls
// exactly the executor named by that declaration.
func DispatchOperation(executors OperationExecutors, call OperationCall) (value any, retErr error) {
	op, ok := operationByName(call.Name)
	if !ok {
		return nil, fmt.Errorf("unknown operation %q", call.Name)
	}
	switch op.Execution {
	case ExecuteDirectStore:
		if executors.DirectStore == nil {
			return nil, fmt.Errorf("%s has no direct-store executor", op.Name)
		}
	case ExecuteLiveOwner:
		if executors.LiveOwner == nil {
			return nil, &OwnerRoutingRequiredError{Operation: op.Name}
		}
	default:
		return nil, fmt.Errorf("%s has no execution owner", op.Name)
	}
	if err := validateOperationCall(op, call); err != nil {
		return nil, err
	}
	call.Operation = op
	call.Args = cloneStringMap(call.Args)
	if executors.Attempt != nil && OperationRecordsLifecycleFailure(op) && OperationAttemptFromContext(call.Context) == nil {
		attempt, err := executors.Attempt(call)
		if err != nil {
			return nil, err
		}
		if attempt != nil {
			call.Context = WithOperationAttempt(call.Context, attempt)
			defer func() {
				value, retErr = attempt.Finish(value, retErr)
				if warning := attempt.DiagnosticWarning(); warning != "" && executors.DiagnosticWarning != nil {
					executors.DiagnosticWarning(warning)
				}
			}()
		}
	}
	if op.Execution == ExecuteDirectStore {
		return executors.DirectStore(call)
	}
	return executors.LiveOwner(call)
}

// OperationRecordsLifecycleFailure derives capture membership from the operation
// declaration. Reads cannot clear an earlier lifecycle failure.
func OperationRecordsLifecycleFailure(op Operation) bool {
	return op.Name == "prepare-start" || op.Execution == ExecuteLiveOwner && (op.Effect == EffectProcess || op.Effect == EffectAuthority || op.Effect == EffectConsole)
}

// OperationConfirms reports whether a named operation DECLARES that it needs an
// operator confirmation, and whether it is declared at all.
//
// The switcher carried three separate hand-written lists of "which actions
// confirm" -- the action list's Enter, the confirmation's own dispatch guard,
// and the refresh's frame-validity check -- none of which read the declaration
// they were transcribing. relaunch joined the action list and none of the three,
// so Enter on it did nothing whatsoever, silently. Confirmation is declared
// exactly once, in Operations(); this is how the terminal asks.
func OperationConfirms(name string) (confirms bool, declared bool) {
	op, ok := operationByName(name)
	if !ok {
		return false, false
	}
	return op.Confirmation == ConfirmRequired, true
}

func operationByName(name string) (Operation, bool) {
	for _, op := range Operations() {
		if op.Name == name {
			return op, true
		}
	}
	return Operation{}, false
}

func validateOperationCall(op Operation, call OperationCall) error {
	known := make(map[string]ArgSpec, len(op.Args))
	for _, arg := range op.Args {
		known[arg.Name] = arg
	}
	for name, value := range call.Args {
		arg, ok := known[name]
		if !ok {
			return fmt.Errorf("%s: unknown argument %q", op.Name, name)
		}
		if arg.Implicit && !call.Implicit {
			return fmt.Errorf("%s: argument %q requires trusted caller context", op.Name, name)
		}
		if arg.ValueRequired && value == "" {
			return fmt.Errorf("%s: argument %q requires a non-empty value", op.Name, name)
		}
	}
	// Presence, not non-emptiness: an empty value is MEANINGFUL for some
	// required arguments -- `set-name` clears a name with one and
	// `publish-description` clears a description. Arguments whose emptiness is
	// nonsense guard themselves at the layer that knows (see SpawnPrepared's
	// empty-fingerprint refusal).
	for _, arg := range op.Args {
		if _, supplied := call.Args[arg.Name]; arg.Required && !supplied {
			return fmt.Errorf("%s: missing required argument %q", op.Name, arg.Name)
		}
	}
	return nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// DirectStoreExecutor performs the direct-store operation families. The
// declaration chooses this executor; this switch only implements that closed
// capability and refuses any owner operation passed around DispatchOperation.
func DirectStoreExecutor(c *Couch) OperationExecutor {
	return func(call OperationCall) (any, error) {
		a := call.Args
		switch call.Operation.Name {
		case "provision-workspace":
			request, err := ParseProvisionRequest(a["path"], a["slot"], a["remote"])
			if err != nil {
				return nil, err
			}
			if c.Workspaces == nil {
				return nil, fmt.Errorf("workspace provisioner is unavailable")
			}
			request.Progress = c.WorkspaceProgress
			return c.Workspaces.Ensure(call.Context, request)
		case "reconcile":
			return c.ReconcileSlot(call.Context, a["ref"])
		case "peek":
			lines := 0
			if raw := a["lines"]; raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n <= 0 {
					return nil, fmt.Errorf("--lines takes a positive number, not %q", raw)
				}
				lines = n
			}
			refs, err := ExpandPeekReferences(a["ref"])
			if err != nil {
				return nil, err
			}
			selection := PeekSelection{Previous: a["previous"] == "true", All: a["all"] == "true", Transcripts: a["transcripts"] == "true", Lines: lines}
			if len(refs) > 1 && (selection.All || selection.Previous) {
				return nil, fmt.Errorf("previous/full recording reads require one exact reference")
			}
			if len(refs) > 1 && a["tag"] == "" {
				resolve := func(ref string) (ThreadAddress, error) {
					return resolveOperationThread(c, map[string]string{"repo-scope": a["repo-scope"], "ref": ref})
				}
				return c.PeekSlots(call.Context, resolve, refs, lines, a["transcripts"] == "true"), nil
			}
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.PeekThreadSelection(call.Context, a["ref"], address, selection)
		case "list":
			return c.ThreadInventoryContext(call.Context)
		case "recover-plan":
			return c.RecoverPlan(call.Context)
		case "show":
			if err := requireOperationRepoScope(a); err != nil {
				return nil, err
			}
			matches, err := c.ResolveThreadReference(a["repo-scope"], a["ref"])
			// Every slot-resolution error is surfaced: it fails --show when no
			// thread matched, and is the slot report's error line when one did.
			// It never reduces the answer to "not a slot".
			slot, isSlot, slotErr := c.slotOfShowReference(call.Context, a["ref"], matches)
			failures, failureError := c.launchFailuresForShow(call.Context, a["repo-scope"], a["ref"], matches, slot)
			if err != nil && !((isSlot || len(failures) > 0 || failureError != "") && errors.Is(err, ErrThreadReferenceNotFound)) {
				if slotErr != nil {
					return nil, slotErr // the slot reference's own, more specific error
				}
				return nil, err
			}
			// Show reads the same classified inventory list does and then
			// narrows, rather than running its own evidence pass: a full-store
			// pass to display one thread pays a zellij snapshot per --show, and
			// projecting `matches` directly would skip the path
			// physicalization list does -- the same field printed two ways by
			// the two views this milestone unified.
			rows, err := c.ThreadInventoryContext(call.Context)
			if err != nil {
				return nil, err
			}
			wanted := make(map[ThreadAddress]bool, len(matches))
			for _, match := range matches {
				wanted[match.Address] = true
			}
			narrowed := make([]ThreadSummary, 0, len(matches))
			for _, row := range rows {
				if wanted[row.Address] {
					narrowed = append(narrowed, row)
				}
			}
			result := ShowResult{Threads: narrowed, LaunchFailures: failures, LaunchFailureError: failureError}
			if c.ScratchDiagnostic != nil {
				if isSlot && len(matches) == 0 {
					result.Scratch = append(result.Scratch, c.ScratchDiagnostic(call.Context, ThreadRecord{}))
				}
				for _, match := range matches {
					result.Scratch = append(result.Scratch, string(match.Address.Tag)+": "+c.ScratchDiagnostic(call.Context, match))
				}
			}
			switch {
			case slotErr != nil:
				result.Slot = &SlotReport{PlanError: "slot unresolved: " + slotErr.Error()}
			case isSlot:
				report := c.SlotReportFor(call.Context, slot, narrowed)
				result.Slot = &report
			}
			return result, nil
		case "archived":
			records, err := c.Threads.ArchivedThreads()
			if err != nil {
				return nil, err
			}
			return BuildArchivedInventory(records), nil
		case "alias":
			var alias *string
			if v, supplied := a["alias"]; supplied {
				alias = &v
			}
			if a["clear"] == "true" {
				if alias != nil {
					return nil, fmt.Errorf("alias: give a new alias or --clear, not both")
				}
				empty := ""
				alias = &empty
			}
			return c.RepositoryAlias(call.Context, a["ref"], alias)
		case "request-continuation":
			ordinal, err := strconv.ParseUint(a["launch-ordinal"], 10, 64)
			if err != nil || ordinal == 0 {
				return nil, fmt.Errorf("invalid continuation launch ordinal")
			}
			return c.RequestContinuation(call.Context, ThreadAddress{RepoScope: a["repo-scope"], Tag: ThreadTag(a["tag"])}, ContinuationSource{Agent: a["agent"], Session: a["session"], LaunchOrdinal: ordinal, ExpectedDigest: a["expected-digest"]}, a["path"])
		case "publish-description":
			if a["repo-scope"] == "" || a["tag"] == "" {
				return nil, fmt.Errorf("thread scope/tag are unavailable -- run this inside a couch-spawned session")
			}
			address := ThreadAddress{RepoScope: a["repo-scope"], Tag: ThreadTag(a["tag"])}
			summary := a["description"]
			return c.ApplyThreadMetadata(address, ThreadMetadataPatch{PublishedSummary: &summary})
		default:
			return nil, fmt.Errorf("%s is not a direct-store operation", call.Operation.Name)
		}
	}
}

func requireOperationRepoScope(args map[string]string) error {
	if args["repo-scope"] == "" {
		return fmt.Errorf("repository scope is unavailable")
	}
	return nil
}

// CouchLiveOwnerExecutor performs actor lifecycle effects owned by Couch. The
// terminal-local switch/attach effects are supplied by couchtty instead.
func CouchLiveOwnerExecutor(c *Couch) OperationExecutor {
	return func(call OperationCall) (any, error) {
		a := call.Args
		ctx := call.Context
		if ctx == nil {
			ctx = context.Background()
		}
		switch call.Operation.Name {
		case "prepare-switch-agent", "switch-agent":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			var argv *[]string
			if raw, present := a["argv"]; present {
				if len(raw) > 32768 {
					return nil, fmt.Errorf("startup parameters exceed transport limit")
				}
				parameters, err := DecodeArgumentArray([]byte(raw))
				if err != nil {
					return nil, err
				}
				argv = &parameters
			}
			if call.Operation.Name == "prepare-switch-agent" {
				if _, local := call.TypedPayload.(LocalForceSwitchPreview); local {
					return c.PrepareForcedAgentSwitch(ctx, address, a["agent"], argv)
				}
				return c.PrepareAgentSwitch(ctx, address, a["agent"], argv)
			}
			if value := a["interrupt-first"]; value != "" && value != "true" && value != "false" {
				return nil, fmt.Errorf("interrupt-first must be true or false")
			}
			known := a["known-config"] == "true"
			if argv == nil && !known {
				return nil, fmt.Errorf("accepted startup parameters are required")
			}
			var args []string
			if argv != nil {
				args = *argv
			}
			restart, force := acceptedAgentSwitchPayload(call.TypedPayload)
			return c.SwitchAgent(ctx, SwitchAgentRequest{Restart: restart, Force: force, InterruptFirst: a["interrupt-first"] == "true", Address: address, Agent: a["agent"], KnownConfig: known, Argv: args, AcceptedFingerprint: a["fingerprint"], Source: a["source"]})
		case "recovery-confirmation":
			return c.ConfirmRecovery(ctx, a["token"])
		case "reboot":
			target, err := resolveRecoveryOperationTarget(ctx, c, a)
			if err != nil && a["tag"] != "" {
				var address ThreadAddress
				address, err = resolveThreadForArchive(c, a)
				target = ResumeTarget{Address: address}
			}
			if err != nil {
				return nil, err
			}
			return c.Reboot(ctx, RebootTarget{Path: target.Path, Address: target.Address, Agent: a["agent"]})
		case "reap":
			if path := a["path"]; path != "" {
				return c.Reap(ctx, ReapTarget{Path: path})
			}
			address, err := resolveThreadForArchive(c, a)
			if err != nil {
				return nil, err
			}
			return c.Reap(ctx, ReapTarget{Address: address})
		case "recover":
			target := RecoverTarget{Path: a["path"]}
			if target.Path == "" {
				address, err := resolveThreadForArchive(c, a)
				if err != nil {
					return nil, err
				}
				target.Address = address
			}
			return c.Recover(ctx, target)
		case "continue-thread":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.Continue(ctx, address, a["request-id"])
		case "prepare-start":
			path := a["path"]
			if path == "" {
				path = "."
			}
			return c.PrepareStart(ctx, StartArgs{Cwd: path, Stack: a["agent"], Action: StartAction(a["action"])})
		case "start-hub":
			return c.StartInteractive(ctx, StartArgs{Cwd: a["path"]})
		case "start":
			// The SAME inputs the preview resolved from, so re-resolution is
			// comparable. Passing the RESOLVED agent where the operator gave
			// none would change AgentSource and therefore the fingerprint.
			rec, h, err := c.SpawnPrepared(ctx, StartArgs{
				Cwd: a["path"], Stack: a["agent"], Issue: a["issue"], Action: StartAction(a["action"]),
			}, StartResolutionFingerprint(a["fingerprint"]))
			if err != nil {
				return nil, err
			}
			return StartResult{Record: rec, Handle: h}, nil
		case "stop":
			recs, _, err := c.ResolveRef(a["ref"])
			if err != nil {
				return nil, err
			}
			switch {
			case len(recs) == 0:
				return nil, fmt.Errorf("%q has no running actor to stop", a["ref"])
			case len(recs) > 1:
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
		case OpRecoverPark:
			address, err := exactParkRecoveryAddress(a)
			if err != nil {
				return nil, err
			}
			return c.RecoverParkExpected(ctx, address, a["park-nonce"])
		case "park":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.Park(ctx, address, a["mode"])
		case "detach":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.Detach(ctx, address)
		case "relaunch":
			// resolveOperationThread: the ONE thread-addressing dialect. The
			// switcher sends {repo-scope, tag}, and reading only `ref` is how
			// Tab -> archive shipped broken (pair#181 M3).
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			if guard := a[requireSettledArg]; guard != "" {
				if c.LiveRestart == nil {
					return nil, &SlotOperationError{Code: LiveRestartUnavailable, Detail: "this Couch cannot observe live slots"}
				}
				if err := c.LiveRestart.ConfirmNotBusy(ctx, address, guard == settledForcedValue); err != nil {
					return nil, err
				}
			}
			return c.Relaunch(ctx, address)
		case OpReloadContext:
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.ReloadContext(ctx, address, a[requireSettledArg] == settledForcedValue)
		case OpPrepare:
			// Admission (prepareLiveRestart) was the whole job; hand back
			// where the CLI's refresh runs.
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return PrepareAdmission{Address: address, Path: a["path"]}, nil
		case "leave":
			// Mirrors park's mode argument rather than minting a second verb:
			// leaving is one operation whose disposition the pressed key picks.
			switch a["mode"] {
			case "", "detach":
				return c.Leave(ctx, LeaveDetach)
			case "park":
				return c.Leave(ctx, LeavePark)
			default:
				return nil, fmt.Errorf("leave: invalid mode %q (want detach or park)", a["mode"])
			}
		case "resume-candidates":
			target, err := resolveRecoveryOperationTarget(ctx, c, a)
			if err != nil {
				return nil, err
			}
			return c.ResumeCandidates(ctx, target)
		case "resume":
			target, err := resolveRecoveryOperationTarget(ctx, c, a)
			if err != nil {
				return nil, err
			}
			if key := a["candidate"]; key != "" {
				if a["warm-only"] != "" {
					return nil, fmt.Errorf("candidate selection cannot be automatic warm resume")
				}
				result, err := c.ResumeCandidate(ctx, target, key)
				return result, withRebootAdvice(err)
			}
			if a["warm-only"] == "true" {
				return c.ResumeTargetWith(ctx, target, ResumeOptions{WarmOnly: true})
			}
			return c.ResumeTarget(ctx, target)
		case "keys", "set-model", "interrupt":
			return nil, fmt.Errorf("%s requires the authenticated live supervisor and wrapper drive capability", call.Operation.Name)
		case "switch", "attach", "move-slot":
			return nil, fmt.Errorf("%s requires an active couch console", call.Operation.Name)
		default:
			return nil, fmt.Errorf("%s is not a live-owner operation", call.Operation.Name)
		}
	}
}

// resolveThreadForArchive addresses a thread without requiring it to be
// readable.
//
// Every other resolver proves the thread exists by decoding it, which is right
// for an operation that will act ON the record. Archive acts on the FILE: an
// unreadable record is the case it most needs to reach, and the store's own
// journal refuses an address that is not in the manifest, so existence is still
// checked -- just not by a decoder that the corrupt case fails by definition.
func resolveThreadForArchive(c *Couch, args map[string]string) (ThreadAddress, error) {
	if err := requireOperationRepoScope(args); err != nil {
		return ThreadAddress{}, err
	}
	if tag := args["tag"]; tag != "" {
		if args["ref"] != "" {
			return ThreadAddress{}, fmt.Errorf("thread ref and exact tag cannot both be supplied")
		}
		return ThreadAddress{RepoScope: args["repo-scope"], Tag: ThreadTag(tag)}, nil
	}
	return resolveOperationThread(c, args)
}

func resolveOperationThread(c *Couch, args map[string]string) (ThreadAddress, error) {
	if err := requireOperationRepoScope(args); err != nil {
		return ThreadAddress{}, err
	}
	if tag := args["tag"]; tag != "" {
		if args["ref"] != "" {
			return ThreadAddress{}, fmt.Errorf("thread ref and exact tag cannot both be supplied")
		}
		address := ThreadAddress{RepoScope: args["repo-scope"], Tag: ThreadTag(tag)}
		if _, err := c.Threads.GetThread(address); err != nil {
			return ThreadAddress{}, err
		}
		return address, nil
	}
	if args["ref"] == "" {
		return ThreadAddress{}, fmt.Errorf("thread reference is required")
	}
	matches, err := c.ResolveThreadReference(args["repo-scope"], args["ref"])
	if err != nil {
		return ThreadAddress{}, err
	}
	if len(matches) != 1 {
		return ThreadAddress{}, fmt.Errorf("thread reference %q did not resolve uniquely", args["ref"])
	}
	return matches[0].Address, nil
}

func resolveRecoveryOperationTarget(ctx context.Context, c *Couch, args map[string]string) (ResumeTarget, error) {
	if args["tag"] != "" {
		args = cloneStringMap(args)
		delete(args, "ref")
	}
	if path := args["path"]; path != "" {
		return ResumeTarget{Path: path}, nil
	}
	if args["tag"] == "" && c.Slots != nil {
		ref, ok, err := ParseWorkspaceReference(args["ref"])
		if err != nil {
			return ResumeTarget{}, err
		}
		if ok && ref.Number > 0 {
			path, _, err := c.WorkspaceReferencePath(ctx, args["ref"])
			return ResumeTarget{Path: path}, err
		}
	}
	address, err := resolveOperationThread(c, args)
	return ResumeTarget{Address: address}, err
}

// Recovery receives an already bound exact address. Its context-aware store
// read belongs inside the recovery gate, rather than an unbounded preflight.
func exactParkRecoveryAddress(args map[string]string) (ThreadAddress, error) {
	address := ThreadAddress{RepoScope: args["repo-scope"], Tag: ThreadTag(args["tag"])}
	if err := validateThreadAddress(address); err != nil {
		return ThreadAddress{}, err
	}
	return address, nil
}

// acceptedAgentSwitchPayload accepts only local typed authority. Legacy direct
// force consent remains valid, while wire maps and decoded empty values confer
// no restart or force permission.
func acceptedAgentSwitchPayload(payload any) (*EditedRestartIntent, *AgentSwitchForceConsent) {
	switch accepted := payload.(type) {
	case *AgentSwitchAcceptance:
		if accepted != nil {
			return cloneEditedRestartIntent(accepted.restart), accepted.force
		}
	case *AgentSwitchForceConsent:
		return nil, accepted
	}
	return nil, nil
}
