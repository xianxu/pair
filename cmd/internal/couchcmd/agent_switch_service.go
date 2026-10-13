package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func switchTimeouts(getenv func(string) string) (time.Duration, time.Duration, error) {
	brief, settle := 3*time.Minute, 10*time.Minute
	for _, p := range []struct {
		name  string
		value *time.Duration
	}{{"COUCH_SWITCH_BRIEF_TIMEOUT", &brief}, {"COUCH_SWITCH_SETTLE_TIMEOUT", &settle}} {
		if raw := getenv(p.name); raw != "" {
			v, err := time.ParseDuration(raw)
			if err != nil || v <= 0 {
				return 0, 0, fmt.Errorf("%s must be a positive duration", p.name)
			}
			*p.value = v
		}
	}
	if brief >= settle || settle > 30*time.Minute {
		return 0, 0, errors.New("switch timeouts require brief < settle <= 30m")
	}
	return brief, settle, nil
}

// The connection token distinguishes wrapper re-exec, which retains the same
// PID, launch nonce and terminal session. Settled is an observation, not identity.
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

type switchTransaction struct{ ID, Source string }

func switchTransactionToken(a switchAdmission) string {
	raw, _ := json.Marshal(switchTransaction{a.RequestID, a.Source})
	return string(raw)
}

func (g *switchServiceGuard) ConfirmAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, source string) error {
	s := g.service.Load()
	if s == nil {
		return errors.New("switch-agent: supervisor unavailable")
	}
	var transaction switchTransaction
	if err := json.Unmarshal([]byte(source), &transaction); err != nil || transaction.ID == "" {
		return errors.New("switch-agent: invalid transaction authority")
	}
	co := s.switches.Load()
	if co == nil || !co.canExecute(transaction.ID, transaction.Source) {
		return errors.New("switch-agent: request no longer authorizes parking")
	}
	e := s.observeSwitch(ctx, switchAdmission{Source: transaction.Source, Address: address, Content: couchcore.SwitchContent{Kind: couchcore.SwitchContentFresh}})
	if !e.Alive || !e.SettledKnown || !e.Settled || !e.GitKnown || !e.Clean {
		return errors.New("switch-agent: source changed or is not known settled and clean; no park authorized")
	}
	return nil
}

func (s *messageService) observeSwitch(ctx context.Context, a switchAdmission) couchcore.AgentSwitchEvent {
	e := couchcore.AgentSwitchEvent{Kind: couchcore.AgentSwitchObserve, Source: a.Source}
	l, ok := s.LivenessForThread(a.Address.RepoScope, string(a.Address.Tag))
	if !ok || switchSourceToken(l) != a.Source || !s.isConnected(l.Binding) {
		return e
	}
	if _, err := s.authority.current(ctx, l.Binding); err != nil {
		return e
	}
	e.Alive = true
	if a.Content.Kind == "" {
		return e
	}
	e.SettledKnown = l.Settled != nil
	e.Settled = l.Settled != nil && *l.Settled
	if !e.Settled {
		return e
	}
	identity, err := s.connectedWorkspace(ctx, l.Binding)
	if err != nil {
		return e
	}
	git, err := s.authority.branch(ctx, identity.WorktreeRoot)
	e.GitKnown = err == nil
	e.Clean = err == nil && !git.Dirty
	// A slow Git read must not authorize the binding it began with after re-exec.
	latest, ok := s.LivenessForThread(a.Address.RepoScope, string(a.Address.Tag))
	if !ok || switchSourceToken(latest) != a.Source || latest.Settled == nil || !*latest.Settled {
		e.Alive = false
	}
	return e
}

func (s *messageService) switchCaller(ctx context.Context, r couchmessage.Request) (couchmessage.Binding, error) {
	b, err := s.broker.Caller(r.Scope, r.Tag, r.Session, r.Nonce)
	if err == nil {
		_, err = s.authority.current(ctx, b)
	}
	return b, err
}

func (s *messageService) handleAgentSwitch(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	refusal := func(err error) couchmessage.Response {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if !r.Confirmed {
		return couchmessage.Response{Code: "confirmation-required", Error: "switch-agent requires --confirm"}
	}
	co := s.switches.Load()
	if co == nil {
		return refusal(errors.New("this Couch runs no agent switches"))
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return refusal(err)
	}
	if r.SwitchRequest != "" {
		live, ok := s.LivenessForThread(caller.Scope, caller.Tag)
		if !ok || live.Binding != caller {
			return refusal(errors.New("callback wrapper is no longer current"))
		}
		return s.activateSwitchResponse(co, co.callback(ctx, caller, r, switchSourceToken(live)))
	}
	if response, held := co.existing(caller, r); held {
		return s.activateSwitchResponse(co, response)
	}
	a, err := s.admitSwitchSource(ctx, caller, r)
	if err != nil {
		return refusal(err)
	}
	prepared, err := s.prepareSwitch(ctx, a.Address, r.Agent, nil)
	if err != nil {
		return refusal(err)
	}
	argv := prepared.Profile.Argv
	if argv == nil {
		argv = []string{}
	}
	if err = launcher.ValidateFreshAgentArgs(r.Agent, argv); err != nil {
		return refusal(err)
	}
	prepared, err = s.prepareSwitch(ctx, a.Address, r.Agent, &argv)
	if err != nil {
		return refusal(err)
	}
	a.Prepared = prepared
	return s.activateSwitchResponse(co, co.admit(ctx, caller, r, a))
}

func (s *messageService) activateSwitchResponse(co *switchCoordinator, r couchmessage.Response) couchmessage.Response {
	if r.Operation != nil && r.Code == "accepted" {
		id := r.Operation.ID
		r.AfterWrite = func(err error) {
			if err == nil {
				co.activate(id)
			} else {
				co.cancel(id, err.Error())
			}
		}
	}
	return r
}

func (s *messageService) admitSwitchSource(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request) (switchAdmission, error) {
	var source couchmessage.Binding
	self := r.Target == "self"
	if self {
		source = caller
	} else {
		var bindings []couchmessage.Binding
		if connected := s.connected.Load(); connected != nil {
			for b, live := range *connected {
				if live {
					bindings = append(bindings, b)
				}
			}
		}
		b, _, err := s.tailBinding(ctx, couchmessage.Request{Target: r.Target}, bindings)
		if err != nil {
			return switchAdmission{}, err
		}
		source = b
	}
	l, ok := s.LivenessForThread(source.Scope, source.Tag)
	if !ok || l.Binding != source {
		return switchAdmission{}, errors.New("switch-agent: source wrapper unavailable")
	}
	if _, err := s.authority.current(ctx, source); err != nil {
		return switchAdmission{}, err
	}
	identity, err := s.connectedWorkspace(ctx, source)
	if err != nil {
		return switchAdmission{}, err
	}
	a := switchAdmission{Source: switchSourceToken(l), Target: source.Slot, Address: couchcore.ThreadAddress{RepoScope: source.Scope, Tag: couchcore.ThreadTag(source.Tag)}, Requester: caller}
	a.Fallback = couchcore.SwitchContent{Kind: couchcore.SwitchContentFallback, Body: fmt.Sprintf("Previous agent: %s. Previous terminal session: %s.\nDiscover the preserved native transcript and full terminal history with:\ncouch --peek %s --previous --transcripts --all\nNo authored work summary was received. Read the durable issue and plan before continuing.\n", source.Agent, source.Session, source.Slot)}
	if !self {
		if l.Settled == nil || !*l.Settled {
			return switchAdmission{}, errors.New("switch-agent: source must be known settled; wait for its turn to finish")
		}
		git, err := s.authority.branch(ctx, identity.WorktreeRoot)
		if err != nil {
			return switchAdmission{}, fmt.Errorf("switch-agent: Git state unknown: %w", err)
		}
		if git.Dirty {
			return switchAdmission{}, errors.New("switch-agent: source is dirty; ask it to commit first or use its self handoff")
		}
		if r.Body == "" && !git.Detached && git.Branch == *identity.RestingBranch {
			a.Content = couchcore.SwitchContent{Kind: couchcore.SwitchContentFresh}
		}
		if r.Body != "" && caller != source {
			origin, err := s.connectedWorkspace(ctx, caller)
			if err != nil || !s.authority.isHub(origin) {
				return switchAdmission{}, errors.New("switch-agent: only the source or supervisor hub may author this slot's handoff")
			}
		}
	}
	if r.Body != "" {
		a.Content = couchcore.SwitchContent{Kind: couchcore.SwitchContentAuthored, Body: r.Body}
	}
	return a, nil
}

func (s *messageService) requestSwitchBrief(ctx context.Context, a switchAdmission, id string) error {
	// Revalidate before sending: an old request must not ask a replacement to
	// write a handoff for a source that has already disappeared.
	l, ok := s.LivenessForThread(a.Address.RepoScope, string(a.Address.Tag))
	if !ok || switchSourceToken(l) != a.Source {
		return errors.New("switch-agent: source replaced before brief request")
	}
	if _, err := s.authority.current(ctx, l.Binding); err != nil {
		return err
	}
	body := fmt.Sprintf("Agent switch requested for %s to %s (request %s). Commit your work, write a short handoff to a local file, then run:\ncouch --switch-agent self --agent %s --handoff-file PATH --request %s --confirm\nPreserve the exact request ID. Include the durable issue/plan and next action; point the next agent to couch --peek %s --previous --transcripts --all. End your turn after the acknowledgment; Couch waits for it to settle before switching.", a.Target, a.Agent, id, couchcore.ShellQuote(a.Agent), couchcore.ShellQuote(id), a.Target)
	receipt, err := s.broker.Send(ctx, a.Requester, "switch-"+id, couchmessage.Route{Target: a.Target}, body)
	if err != nil {
		return err
	}
	if receipt.Status != couchmessage.Queued && receipt.Status != couchmessage.Submitted {
		return fmt.Errorf("switch brief request was not admitted: %s", receipt.Status)
	}
	return nil
}

func (s *messageService) installSwitchCoordinator(console *couchtty.Console, c *couchcore.Couch, guard *switchServiceGuard, brief, settle time.Duration) {
	co := newSwitchCoordinator(s.slotOps, brief, settle)
	co.observe = s.observeSwitch
	co.requestBrief = s.requestSwitchBrief
	co.notice = func(text string) { console.Notify(couchtty.Notice{Kind: "switch-agent", Body: text, Control: true}) }
	co.enqueue = func(a switchAdmission, content couchcore.SwitchContent, started func(), finished func(any, error)) error {
		key, err := s.slotOps.queueKey(s.lifetime, a.Target)
		if err != nil {
			return err
		}
		transaction := switchTransactionToken(a)
		if err = guard.ConfirmAgentSwitch(s.lifetime, a.Address, transaction); err != nil {
			return err
		}
		argv, err := json.Marshal(a.Prepared.Profile.Argv)
		if err != nil {
			return err
		}
		return console.EnqueueRemoteOperation(key, "switch-agent", func(ctx context.Context) (couchcore.OperationCall, error) {
			if err := guard.ConfirmAgentSwitch(ctx, a.Address, transaction); err != nil {
				return couchcore.OperationCall{}, err
			}
			return couchcore.OperationCall{Name: "switch-agent", Args: map[string]string{"repo-scope": a.Address.RepoScope, "tag": string(a.Address.Tag), "agent": a.Agent, "argv": string(argv), "fingerprint": a.Prepared.Fingerprint, "content-kind": string(content.Kind), "body": content.Body, "source": transaction}}, nil
		}, started, finished)
	}
	s.prepareSwitch = c.PrepareAgentSwitch
	console.SetAgentSwitchAdmission(s.admitSwitchForm)
	console.SetOrientationCompletion(co.delivery)
	guard.service.Store(s)
	s.switches.Store(co)
	s.workers.Add(1)
	go func() { defer s.workers.Done(); co.run(s.lifetime) }()
}

// The form is an operator action, but uses exactly the same source checks,
// pending reservation and waiting loop as the socket. Accepted argv stays fixed.
func (s *messageService) admitSwitchForm(ctx context.Context, request couchcore.SwitchAgentRequest) (string, error) {
	co := s.switches.Load()
	if co == nil {
		return "", errors.New("switch coordinator unavailable")
	}
	live, ok := s.LivenessForThread(request.Address.RepoScope, string(request.Address.Tag))
	if !ok {
		return "", errors.New("switch-agent: resume the source first so it can author its handoff")
	}
	caller := live.Binding
	if hub, ok := s.hubBinding(); ok {
		caller = hub
	}
	id, err := newRequestID()
	if err != nil {
		return "", err
	}
	r := couchmessage.Request{Op: "switch-agent", ID: id, Target: live.Binding.Slot, Agent: request.Agent, Confirmed: true}
	a, err := s.admitSwitchSource(ctx, caller, r)
	if err != nil {
		return "", err
	}
	prepared, err := s.prepareSwitch(ctx, request.Address, request.Agent, &request.Argv)
	if err != nil {
		return "", err
	}
	if prepared.Fingerprint != request.AcceptedFingerprint {
		return "", couchcore.ErrSwitchResolutionChanged
	}
	a.Prepared = prepared
	response := co.admit(ctx, caller, r, a)
	if response.Operation == nil || response.Code != "accepted" {
		return "", fmt.Errorf("switch-agent: %s", response.Error)
	}
	co.activate(id)
	return id, nil
}
