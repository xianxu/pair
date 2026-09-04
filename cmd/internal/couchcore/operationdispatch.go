package couchcore

import (
	"context"
	"fmt"
	"strings"
)

// OperationCall is the transport-neutral request consumed by every operation
// client. TypedPayload is reserved for owner-local capabilities (currently a
// terminal-bearing StartResult); it is never inferred from argv.
type OperationCall struct {
	Name         string
	Args         map[string]string
	Implicit     bool
	TypedPayload any
	Context      context.Context
	Operation    Operation
}

type OperationExecutor func(OperationCall) (any, error)

// OperationExecutors supplies capability rather than hiding it in Operation.
// A store-only client intentionally leaves LiveOwner nil until #147 can route
// to the singleton process that owns the actors and terminal.
type OperationExecutors struct {
	DirectStore OperationExecutor
	LiveOwner   OperationExecutor
}

// OwnerRoutingRequiredError is the typed refusal for an owner-required call
// made outside the singleton Couch process.
type OwnerRoutingRequiredError struct{ Operation string }

func (e *OwnerRoutingRequiredError) Error() string {
	return fmt.Sprintf("%s requires the live couch owner; routing requires #147", e.Operation)
}

// DispatchOperation resolves one declaration, validates its schema, and calls
// exactly the executor named by that declaration.
func DispatchOperation(executors OperationExecutors, call OperationCall) (any, error) {
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
	if op.Execution == ExecuteDirectStore {
		return executors.DirectStore(call)
	}
	return executors.LiveOwner(call)
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
		case "list":
			return c.ThreadInventoryContext(call.Context)
		case "show":
			if err := requireOperationRepoScope(a); err != nil {
				return nil, err
			}
			matches, err := c.ResolveThreadReference(a["repo-scope"], a["ref"])
			if err != nil {
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
			return narrowed, nil
		case "archived":
			records, err := c.Threads.ArchivedThreads()
			if err != nil {
				return nil, err
			}
			return BuildArchivedInventory(records), nil
		case "archive":
			// resolveOperationThread, NOT ResolveThreadReference: the switcher
			// dispatches through threadEffect, which sends {repo-scope, tag}
			// and no ref. Reading only `ref` made every Tab -> archive fail
			// with "empty reference" -- the milestone's headline action,
			// unreachable from the only surface offering it. The two dialects
			// are the bug; this is the one that accepts both (ARCH-DRY).
			// resolveThreadForArchive, not resolveOperationThread: resolving by
			// tag calls GetThread, which DECODES -- so archiving an unreadable
			// record failed with the decode error at exactly the moment the
			// operator was trying to get rid of it. An archive target is
			// addressed, not read.
			address, err := resolveThreadForArchive(c, a)
			if err != nil {
				return nil, err
			}
			ctx := call.Context
			if ctx == nil {
				ctx = context.Background()
			}
			return c.ArchiveThread(ctx, address)
		case "name":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			name := a["name"]
			return c.ApplyThreadMetadata(address, ThreadMetadataPatch{Name: &name})
		case "describe":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			if d, supplied := a["description"]; supplied {
				return c.ApplyThreadMetadata(address, ThreadMetadataPatch{Description: &d})
			}
			record, err := c.Threads.GetThread(address)
			if err != nil {
				return nil, err
			}
			return record.Description, nil
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
		case "prepare-start":
			path := a["path"]
			if path == "" {
				path = "."
			}
			return c.PrepareStart(ctx, StartArgs{Cwd: path, Stack: a["agent"]})
		case "start":
			// The SAME inputs the preview resolved from, so re-resolution is
			// comparable. Passing the RESOLVED agent where the operator gave
			// none would change AgentSource and therefore the fingerprint.
			rec, h, err := c.SpawnPrepared(ctx, StartArgs{
				Cwd: a["path"], Stack: a["agent"], Issue: a["issue"],
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
		case "park":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			if c.PairLifecycle == nil {
				return nil, fmt.Errorf("Pair lifecycle controller is unavailable")
			}
			switch a["mode"] {
			case "", "normal":
				return c.PairLifecycle.Park(ctx, address)
			case "retry":
				return c.PairLifecycle.Retry(ctx, address)
			case "recover":
				return c.PairLifecycle.Recover(ctx, address)
			case "abandon":
				return c.PairLifecycle.Abandon(ctx, address)
			default:
				return nil, fmt.Errorf("park: invalid mode %q (want normal, retry, recover, or abandon)", a["mode"])
			}
		case "detach":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			return c.Detach(ctx, address)
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
		case "resume":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			record, handle, err := c.ResumeContext(ctx, address)
			if err != nil {
				return nil, err
			}
			return StartResult{Record: record, Handle: handle}, nil
		case "switch", "attach":
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
