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
			return c.ThreadInventory()
		case "show":
			if err := requireOperationRepoScope(a); err != nil {
				return nil, err
			}
			matches, err := c.ResolveThreadReference(a["repo-scope"], a["ref"])
			if err != nil {
				return nil, err
			}
			return BuildThreadInventory(matches), nil
		case "name":
			if err := requireOperationRepoScope(a); err != nil {
				return nil, err
			}
			matches, err := c.ResolveThreadReference(a["repo-scope"], a["ref"])
			if err != nil {
				return nil, err
			}
			name := a["name"]
			return c.ApplyThreadMetadata(matches[0].Address, ThreadMetadataPatch{Name: &name})
		case "describe":
			if err := requireOperationRepoScope(a); err != nil {
				return nil, err
			}
			matches, err := c.ResolveThreadReference(a["repo-scope"], a["ref"])
			if err != nil {
				return nil, err
			}
			if d, supplied := a["description"]; supplied {
				return c.ApplyThreadMetadata(matches[0].Address, ThreadMetadataPatch{Description: &d})
			}
			return matches[0].Description, nil
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
		switch call.Operation.Name {
		case "start":
			path := a["path"]
			if path == "" {
				path = "."
			}
			rec, h, err := c.Spawn(StartArgs{Cwd: path, Stack: a["agent"]})
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
				return c.PairLifecycle.Park(context.Background(), address)
			case "retry":
				return c.PairLifecycle.Retry(context.Background(), address)
			case "recover":
				return c.PairLifecycle.Recover(context.Background(), address)
			case "abandon":
				return c.PairLifecycle.Abandon(context.Background(), address)
			default:
				return nil, fmt.Errorf("park: invalid mode %q (want normal, retry, recover, or abandon)", a["mode"])
			}
		case "leave":
			return c.Leave(context.Background())
		case "resume":
			address, err := resolveOperationThread(c, a)
			if err != nil {
				return nil, err
			}
			record, handle, err := c.Resume(address)
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
