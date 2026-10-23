package couchcore

import (
	"context"
	"errors"
	"sync/atomic"
)

// AgentSwitchForceGuard proves the same owned source while waiving idle and
// clean-tree restrictions only for an explicitly confirmed local operation.
type AgentSwitchForceGuard interface {
	PrepareForcedAgentSwitch(context.Context, ThreadAddress, SwitchSourceEvidence) (string, *int, error)
	ConfirmForcedAgentSwitch(context.Context, ThreadAddress, string, SwitchSourceEvidence) error
}

// LocalForceSwitchPreview is an owner-local request, never an operation arg.
type LocalForceSwitchPreview struct{}

// AgentSwitchForceOffer carries no serialized authority. Its source and exact
// accepted profile are frozen before the operator's one risk confirmation.
type AgentSwitchForceOffer struct {
	owner        *Couch
	fingerprint  string
	source       string
	changedFiles *int
	consumed     atomic.Bool
}

// ChangedFiles returns display evidence, nil when the observation is unknown.
func (o *AgentSwitchForceOffer) ChangedFiles() *int {
	if o == nil || o.changedFiles == nil {
		return nil
	}
	n := *o.changedFiles
	return &n
}

// AgentSwitchForceConsent is local authority; JSON cannot recreate its offer.
type AgentSwitchForceConsent struct{ offer *AgentSwitchForceOffer }

// Confirm is called only by the operator form's explicit risk confirmation.
func (o *AgentSwitchForceOffer) Confirm() *AgentSwitchForceConsent {
	return &AgentSwitchForceConsent{offer: o}
}

func (c *Couch) PrepareForcedAgentSwitch(ctx context.Context, a ThreadAddress, agent string, argv *[]string) (PreparedAgentSwitch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	p, err := c.PrepareAgentSwitch(ctx, a, agent, argv)
	if err != nil {
		return p, err
	}
	guard, ok := c.SwitchGuard.(AgentSwitchForceGuard)
	if !ok {
		return p, errors.New("switch-agent: local force source proof unavailable")
	}
	source, count, err := guard.PrepareForcedAgentSwitch(ctx, a, p.SourceEvidence())
	if err != nil {
		return p, err
	}
	// Copy display evidence; adapters cannot mutate a confirmed offer afterward.
	if count != nil {
		n := *count
		count = &n
	}
	p.Force = &AgentSwitchForceOffer{owner: c, fingerprint: p.Fingerprint, source: source, changedFiles: count}
	return p, nil
}

func (c *Couch) consumeAgentSwitchForce(consent *AgentSwitchForceConsent, p PreparedAgentSwitch) (string, error) {
	if consent == nil || consent.offer == nil {
		return "", errors.New("switch-agent: local force consent unavailable")
	}
	o := consent.offer
	if o.owner != c || o.fingerprint != p.Fingerprint || !o.consumed.CompareAndSwap(false, true) {
		return "", errors.New("switch-agent: local force consent stale or consumed; review again")
	}
	return o.source, nil
}
