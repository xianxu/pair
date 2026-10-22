package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// The wrapper connection token changes even when a re-exec keeps its PID/nonce.
func switchSourceToken(l couchmessage.SlotLiveness) string {
	raw, _ := json.Marshal(struct {
		Binding couchmessage.Binding
		Session couchmessage.SessionToken
	}{l.Binding, l.Session})
	return string(raw)
}

type switchServiceGuard struct {
	service atomic.Pointer[messageService]
}

// ConfirmAgentSwitch is shared by the human form and queued remote operations.
// An empty token selects the form's current source, never bypasses its checks.
func (g *switchServiceGuard) ConfirmAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence) error {
	return g.confirmAgentSwitchAdmission(ctx, address, source, evidence, false)
}

// Cooperative admission waives only the idle check, never source/Git proof.
// Input remains on the owned lifecycle phase, not a request worker.
func (g *switchServiceGuard) confirmAgentSwitchAdmission(ctx context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence, interruptFirst bool) error {
	return g.confirmAgentSwitchAdmissionObserved(ctx, address, source, evidence, interruptFirst, nil)
}

func (g *switchServiceGuard) confirmAgentSwitchAdmissionObserved(ctx context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence, interruptFirst bool, observed *retainedSwitchAdmission) error {
	s := g.service.Load()
	if s == nil {
		return errors.New("switch-agent: supervisor unavailable")
	}
	if evidence.Vacant() {
		if observed != nil {
			_, err := inspectVacantSwitchSourceObserved(ctx, s, address, source, evidence, false, observed)
			return err
		}
		return g.confirmVacantAgentSwitch(ctx, s, address, source, evidence)
	}
	_, err := inspectSwitchLiveSource(ctx, s, address, source, !interruptFirst)
	return err
}

func inspectSwitchLiveSource(ctx context.Context, s *messageService, address couchcore.ThreadAddress, source string, requireIdle bool) (couchmessage.SlotLiveness, error) {
	live, _, err := inspectSwitchLiveSourceWithOptions(ctx, s, address, source, switchSourceOptions{requireIdle: requireIdle})
	return live, err
}

type switchSourceOptions struct {
	requireIdle   bool
	allowUnclean  bool
	forceEvidence *couchcore.SwitchSourceEvidence
}

func inspectSwitchLiveSourceWithOptions(ctx context.Context, s *messageService, address couchcore.ThreadAddress, source string, options switchSourceOptions) (couchmessage.SlotLiveness, couchcore.SlotGitStatus, error) {
	fail := func(err error) (couchmessage.SlotLiveness, couchcore.SlotGitStatus, error) {
		return couchmessage.SlotLiveness{}, couchcore.SlotGitStatus{}, err
	}
	live, ok := s.LivenessForThread(address.RepoScope, string(address.Tag))
	if !ok || !s.isConnected(live.Binding) || source != "" && source != switchSourceToken(live) {
		return fail(errors.New("switch-agent: source changed or disconnected; retry switch"))
	}
	if options.requireIdle && (live.Settled == nil || !*live.Settled) {
		return fail(errors.New("switch-agent: source must be known idle; retry after its turn finishes"))
	}
	if _, err := s.authority.current(ctx, live.Binding); err != nil {
		return fail(err)
	}
	identity, err := s.connectedWorkspace(ctx, live.Binding)
	if err != nil {
		return fail(err)
	}
	if options.forceEvidence != nil {
		if err := verifyForcedSwitchLiveSource(ctx, s, live, address, *options.forceEvidence, identity); err != nil {
			return fail(err)
		}
	}
	git, err := inspectSwitchGit(ctx, s, identity.WorktreeRoot, options.allowUnclean)
	if err != nil {
		return fail(err)
	}
	latest, ok := s.LivenessForThread(address.RepoScope, string(address.Tag))
	if !ok || switchSourceToken(latest) != switchSourceToken(live) || options.requireIdle && (latest.Settled == nil || !*latest.Settled) || !s.isConnected(latest.Binding) {
		return fail(errors.New("switch-agent: source changed while checking; retry switch"))
	}
	if options.forceEvidence != nil {
		if err := verifyForcedSwitchLiveSource(ctx, s, latest, address, *options.forceEvidence, identity); err != nil {
			return fail(err)
		}
	}
	return latest, git, ctx.Err()
}

func inspectSwitchGit(ctx context.Context, s *messageService, path string, allowUnclean bool) (couchcore.SlotGitStatus, error) {
	git, err := s.authority.branch(ctx, path)
	if ctx.Err() != nil {
		return couchcore.SlotGitStatus{}, ctx.Err()
	}
	if err != nil {
		if allowUnclean {
			return couchcore.SlotGitStatus{}, nil
		}
		return couchcore.SlotGitStatus{}, fmt.Errorf("switch-agent: Git state unknown: %w", err)
	}
	if git.Dirty && !allowUnclean {
		return couchcore.SlotGitStatus{}, errors.New("switch-agent: source is dirty; ask it to commit first")
	}
	return git, nil
}

// The core's exact vacancy classification replaces only the live-wrapper gate.
// Workspace/Git proof and retained selection must still survive both checks.
func (g *switchServiceGuard) confirmVacantAgentSwitch(ctx context.Context, s *messageService, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence) error {
	_, err := inspectVacantSwitchSource(ctx, s, address, source, evidence, false)
	return err
}

func inspectVacantSwitchSource(ctx context.Context, s *messageService, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence, force bool) (couchcore.SlotGitStatus, error) {
	return inspectVacantSwitchSourceObserved(ctx, s, address, source, evidence, force, nil)
}

func inspectVacantSwitchSourceObserved(ctx context.Context, s *messageService, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence, force bool, observed *retainedSwitchAdmission) (couchcore.SlotGitStatus, error) {
	fail := func(err error) (couchcore.SlotGitStatus, error) { return couchcore.SlotGitStatus{}, err }
	if source != "" {
		return fail(errors.New("switch-agent: live source changed; review the parked source again"))
	}
	if s.authority.record == nil || s.authority.workspace == nil || s.authority.branch == nil {
		return fail(errors.New("switch-agent: vacant source verification unavailable"))
	}
	if observed != nil && (force || observed.address != address || observed.workspace.WorktreeRoot != evidence.WorkingPath() || observed.workspace.RepoIdentity != evidence.RepositoryIdentity()) {
		return fail(errors.New("switch-agent: admission workspace does not match the prepared source"))
	}
	current := func() error {
		if observed != nil {
			if err := observed.recheck(ctx); err != nil {
				return err
			}
		}
		if live, ok := s.LivenessForThread(address.RepoScope, string(address.Tag)); ok && s.isConnected(live.Binding) {
			return errors.New("switch-agent: vacant source reconnected; review it again")
		}
		record, err := s.authority.record(ctx, address)
		if err != nil {
			return err
		}
		if !evidence.MatchesRecord(record) {
			return errors.New("switch-agent: vacant source changed; review it again")
		}
		return ctx.Err()
	}
	if err := current(); err != nil {
		return fail(err)
	}
	var identity couchcore.WorkspaceIdentity
	var err error
	if observed != nil {
		identity = observed.workspace
	} else {
		identity, err = s.authority.workspace(ctx, evidence.WorkingPath())
	}
	if err != nil {
		return fail(err)
	}
	if force {
		if err := verifyForcedSwitchWorkspace(ctx, s, evidence, identity); err != nil {
			return fail(err)
		}
	}
	git, err := inspectSwitchGit(ctx, s, identity.WorktreeRoot, force)
	if err != nil {
		return fail(err)
	}
	if force {
		if err := verifyForcedSwitchWorkspace(ctx, s, evidence, identity); err != nil {
			return fail(err)
		}
	}
	return git, current()
}

type agentSwitchService struct {
	resolveTarget func(context.Context, string) (couchcore.ThreadAddress, couchcore.WorkspaceIdentity, error)
	recheckTarget func(context.Context, string, couchcore.ThreadAddress, couchcore.WorkspaceIdentity) error
	guard         *switchServiceGuard
	prepare       func(context.Context, couchcore.ThreadAddress, string, *[]string) (couchcore.PreparedAgentSwitch, error)
	enqueue       func(string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error
}

// This observation belongs only to one admission. Queued and effect guards
// retain the full resolver; they never receive this value as workspace proof.
type retainedSwitchAdmission struct {
	address   couchcore.ThreadAddress
	workspace couchcore.WorkspaceIdentity
	recheck   func(context.Context) error
}

func (s *messageService) authorizeRetainedSwitchWorkspace(ctx context.Context, caller couchmessage.Binding, identity couchcore.WorkspaceIdentity) error {
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return err
	}
	if origin.WorktreeRoot == identity.WorktreeRoot {
		return errors.New("switch-agent: a caller cannot switch its own slot")
	}
	slot, ok := hubWorkspaceSlot(identity)
	if !ok {
		return errors.New("switch-agent: target slot identity unavailable")
	}
	if s.authority.hubs != nil {
		return authorizeHubMutation(origin, slot, s.authority.hubs.snapshot())
	}
	return ctx.Err()
}

func (s *messageService) switchCaller(ctx context.Context, r couchmessage.Request) (couchmessage.Binding, error) {
	caller, err := s.broker.Caller(r.Scope, r.Tag, r.Session, r.Nonce)
	if err == nil {
		_, err = s.authority.current(ctx, caller)
	}
	return caller, err
}

func (s *messageService) handleAgentSwitch(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	refuse := func(err error) couchmessage.Response {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	service := s.switches.Load()
	if service == nil || s.slotOps == nil {
		return refuse(errors.New("switch-agent unavailable"))
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return refuse(err)
	}
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return refuse(err)
	}
	if !s.authority.isHub(origin) {
		return refuse(errors.New("switch-agent: only the supervisor hub may switch another slot; use the operator form"))
	}
	// Resolve a repeat before probing the replaced target. Receipt intent includes
	// the destination agent, so reusing an ID cannot silently change its meaning.
	if response, held := s.slotOps.existingSwitch(caller, r); held {
		return response
	}
	var bindings []couchmessage.Binding
	if connected := s.connected.Load(); connected != nil {
		for b, live := range *connected {
			if live {
				bindings = append(bindings, b)
			}
		}
	}
	source, _, err := s.tailBinding(ctx, couchmessage.Request{Target: r.Target}, bindings)
	var address couchcore.ThreadAddress
	token := ""
	var authorize func(context.Context) error
	var observed *retainedSwitchAdmission
	switch {
	case err == nil:
		address = couchcore.ThreadAddress{RepoScope: source.Scope, Tag: couchcore.ThreadTag(source.Tag)}
		live, ok := s.LivenessForThread(source.Scope, source.Tag)
		if !ok || live.Binding != source {
			return refuse(errors.New("switch-agent: source wrapper unavailable"))
		}
		token = switchSourceToken(live)
		authorize = func(ctx context.Context) error { return s.authorizeHubBinding(ctx, caller, source) }
	case errors.Is(err, couchmessage.ErrUnavailable) && service.resolveTarget != nil:
		var identity couchcore.WorkspaceIdentity
		address, identity, err = service.resolveTarget(ctx, r.Target)
		if err != nil {
			return refuse(err)
		}
		if service.recheckTarget != nil {
			observed = &retainedSwitchAdmission{address: address, workspace: identity,
				recheck: func(ctx context.Context) error { return service.recheckTarget(ctx, r.Target, address, identity) }}
		}
		authorize = func(ctx context.Context) error {
			currentAddress, current, err := service.resolveTarget(ctx, r.Target)
			if err != nil {
				return err
			}
			if currentAddress != address || current.WorktreeRoot != identity.WorktreeRoot || current.RepoIdentity != identity.RepoIdentity || current.PrimaryRoot != identity.PrimaryRoot {
				return errors.New("switch-agent: retained slot selection changed; review it again")
			}
			return s.authorizeRetainedSwitchWorkspace(ctx, caller, current)
		}
	default:
		return refuse(err)
	}
	if address.RepoScope == caller.Scope && string(address.Tag) == caller.Tag {
		return refuse(errors.New("switch-agent: a caller cannot switch its own slot"))
	}
	authorizeAdmission := authorize
	if observed != nil {
		authorizeAdmission = func(ctx context.Context) error {
			if err := observed.recheck(ctx); err != nil {
				return err
			}
			return s.authorizeRetainedSwitchWorkspace(ctx, caller, observed.workspace)
		}
	}
	if err := authorizeAdmission(ctx); err != nil {
		return refuse(err)
	}
	var explicit *[]string
	if argv, present := r.AgentArgs.Argv(); present {
		explicit = &argv
	}
	prepared, err := service.prepare(ctx, address, r.Agent, explicit)
	if err != nil {
		return refuse(err)
	}
	if err = service.guard.confirmAgentSwitchAdmissionObserved(ctx, address, token, prepared.SourceEvidence(), r.InterruptFirst, observed); err != nil {
		return refuse(err)
	}
	guard := func(ctx context.Context) error {
		if s.authority.hubs != nil {
			if _, err := s.switchCaller(ctx, r); err != nil {
				return err
			}
		}
		if err := authorize(ctx); err != nil {
			return err
		}
		return service.guard.confirmAgentSwitchAdmission(ctx, address, token, prepared.SourceEvidence(), r.InterruptFirst)
	}
	return s.slotOps.admit(ctx, caller, r, func(key string, started func(), finished func(any, error)) error {
		return service.enqueue(key, func(ctx context.Context) (couchcore.OperationCall, error) {
			call := couchcore.OperationCall{Name: "switch-agent", Args: map[string]string{
				"repo-scope": address.RepoScope, "tag": string(address.Tag), "agent": r.Agent,
				"fingerprint": prepared.Fingerprint, "source": token,
			}}
			if r.InterruptFirst {
				call.Args["interrupt-first"] = "true"
			}
			if explicit != nil {
				raw, _ := json.Marshal(*explicit)
				call.Args["argv"] = string(raw)
			} else {
				call.Args["known-config"] = "true"
			}
			call.FailureTarget = &couchcore.LifecycleFailureTarget{Address: address, Target: r.Target}
			if err := guard(ctx); err != nil {
				return call, err
			}
			if s.authority.hubs != nil {
				call.TypedPayload = hubMutationPayload{original: call.TypedPayload, guard: guard}
			}
			return call, nil
		}, started, finished)
	})
}

func (s *slotOperations) existingSwitch(caller couchmessage.Binding, r couchmessage.Request) (couchmessage.Response, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	old, ok := s.receipts[receiptKey{caller.Scope, caller.Tag, r.ID}]
	if !ok {
		return couchmessage.Response{}, false
	}
	if old.Op != r.Op || old.Target != r.Target || old.Agent != r.Agent || old.IntentDigest != operationIntentDigest(r) {
		return couchmessage.Response{Code: "id-conflict", Error: "request ID already names another operation"}, true
	}
	return couchmessage.Response{Code: "accepted", Operation: &old}, true
}

func (s *messageService) installAgentSwitchService(console *couchtty.Console, c *couchcore.Couch, guard *switchServiceGuard) {
	guard.service.Store(s)
	lifecycleQueue := newLifecycleRemoteQueue(console, c)
	s.switches.Store(&agentSwitchService{resolveTarget: c.ResolveAgentSwitchTarget, recheckTarget: c.RecheckAgentSwitchTarget, guard: guard, prepare: func(ctx context.Context, address couchcore.ThreadAddress, agent string, argv *[]string) (couchcore.PreparedAgentSwitch, error) {
		if argv == nil {
			return c.PrepareKnownAgentSwitch(ctx, address, agent)
		}
		return c.PrepareAgentSwitch(ctx, address, agent, argv)
	},
		enqueue: func(key string, prepare func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error)) error {
			return lifecycleQueue.Enqueue(key, "switch-agent", prepare, started, finished)
		},
	})
}
