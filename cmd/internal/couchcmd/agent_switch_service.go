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
func (g *switchServiceGuard) ConfirmAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, source string) error {
	s := g.service.Load()
	if s == nil {
		return errors.New("switch-agent: supervisor unavailable")
	}
	live, ok := s.LivenessForThread(address.RepoScope, string(address.Tag))
	if !ok || !s.isConnected(live.Binding) || source != "" && source != switchSourceToken(live) {
		return errors.New("switch-agent: source changed or disconnected; retry switch")
	}
	if live.Settled == nil || !*live.Settled {
		return errors.New("switch-agent: source must be known idle; retry after its turn finishes")
	}
	if _, err := s.authority.current(ctx, live.Binding); err != nil {
		return err
	}
	identity, err := s.connectedWorkspace(ctx, live.Binding)
	if err != nil {
		return err
	}
	git, err := s.authority.branch(ctx, identity.WorktreeRoot)
	if err != nil {
		return fmt.Errorf("switch-agent: Git state unknown: %w", err)
	}
	if git.Dirty {
		return errors.New("switch-agent: source is dirty; ask it to commit first")
	}
	latest, ok := s.LivenessForThread(address.RepoScope, string(address.Tag))
	if !ok || switchSourceToken(latest) != switchSourceToken(live) || latest.Settled == nil || !*latest.Settled || !s.isConnected(latest.Binding) {
		return errors.New("switch-agent: source changed while checking; retry switch")
	}
	return ctx.Err()
}

type agentSwitchService struct {
	guard   *switchServiceGuard
	prepare func(context.Context, couchcore.ThreadAddress, string) (couchcore.PreparedAgentSwitch, error)
	enqueue func(string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error
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
	if err != nil {
		return refuse(err)
	}
	if source.Scope == caller.Scope && source.Tag == caller.Tag {
		return refuse(errors.New("switch-agent: a caller cannot switch its own slot"))
	}
	if err := s.authorizeHubBinding(ctx, caller, source); err != nil {
		return refuse(err)
	}
	address := couchcore.ThreadAddress{RepoScope: source.Scope, Tag: couchcore.ThreadTag(source.Tag)}
	live, ok := s.LivenessForThread(source.Scope, source.Tag)
	if !ok || live.Binding != source {
		return refuse(errors.New("switch-agent: source wrapper unavailable"))
	}
	token := switchSourceToken(live)
	if err = service.guard.ConfirmAgentSwitch(ctx, address, token); err != nil {
		return refuse(err)
	}
	prepared, err := service.prepare(ctx, address, r.Agent)
	if err != nil {
		return refuse(err)
	}
	guard := func(ctx context.Context) error {
		if s.authority.hubs != nil {
			if _, err := s.switchCaller(ctx, r); err != nil {
				return err
			}
		}
		if err := s.authorizeHubBinding(ctx, caller, source); err != nil {
			return err
		}
		return service.guard.ConfirmAgentSwitch(ctx, address, token)
	}
	return s.slotOps.admit(ctx, caller, r, func(key string, started func(), _ couchcore.OperationProgress, finished func(any, error)) error {
		return service.enqueue(key, func(ctx context.Context) (couchcore.OperationCall, error) {
			call := couchcore.OperationCall{Name: "switch-agent", Args: map[string]string{
				"repo-scope": address.RepoScope, "tag": string(address.Tag), "agent": r.Agent,
				"known-config": "true", "fingerprint": prepared.Fingerprint, "source": token,
			}}
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
	if old.Op != r.Op || old.Target != r.Target || old.Agent != r.Agent {
		return couchmessage.Response{Code: "id-conflict", Error: "request ID already names another operation"}, true
	}
	return couchmessage.Response{Code: "accepted", Operation: &old}, true
}

func (s *messageService) installAgentSwitchService(console *couchtty.Console, c *couchcore.Couch, guard *switchServiceGuard) {
	guard.service.Store(s)
	lifecycleQueue := newLifecycleRemoteQueue(console, c)
	s.switches.Store(&agentSwitchService{guard: guard, prepare: c.PrepareKnownAgentSwitch,
		enqueue: func(key string, prepare func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error)) error {
			return lifecycleQueue.Enqueue(key, "switch-agent", prepare, started, finished)
		},
	})
}
