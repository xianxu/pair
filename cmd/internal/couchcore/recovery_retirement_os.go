package couchcore

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

type recoveryRetirementEvidence struct {
	processes     []launcher.RecoveryProcess
	digest        string
	check, retire func(context.Context) error
}

type recoveryRetirementPreparation func(context.Context, string, string, string, string) (recoveryRetirementEvidence, error)

// PrepareConfirmedTerminal snapshots exact retirement authority and exposes
// only copied process summaries plus closures over the opaque launcher plan.
func (r OSRecoveryEvidenceReader) PrepareConfirmedTerminal(ctx context.Context, record ThreadRecord) (RecoveryRetirement, error) {
	return r.prepareConfirmedTerminal(ctx, record, func(ctx context.Context, root, scope, tag, session string) (recoveryRetirementEvidence, error) {
		plan, err := launcher.PrepareConfirmedRecoveryTerminal(ctx, root, scope, tag, session)
		if err != nil {
			return recoveryRetirementEvidence{}, err
		}
		return recoveryRetirementEvidence{processes: plan.Processes(), digest: plan.Digest(), check: func(ctx context.Context) error { return launcher.RecheckRecoveryTerminal(ctx, plan) }, retire: func(ctx context.Context) error { return launcher.RetireRecoveryTerminal(ctx, plan) }}, nil
	})
}

func (r OSRecoveryEvidenceReader) prepareConfirmedTerminal(ctx context.Context, record ThreadRecord, prepare recoveryRetirementPreparation) (RecoveryRetirement, error) {
	root, name, err := r.recoveryTerminalAddress(ctx, record)
	if err != nil {
		return RecoveryRetirement{}, err
	}
	evidence, err := prepare(ctx, root, record.Address.RepoScope, string(record.Address.Tag), name)
	if err != nil {
		return RecoveryRetirement{}, err
	}
	if evidence.digest == "" || evidence.check == nil || evidence.retire == nil {
		return RecoveryRetirement{}, errors.New("confirmed retirement evidence unavailable")
	}
	stops := make([]RecoveryStopProcess, 0, len(evidence.processes))
	for _, p := range evidence.processes {
		stops = append(stops, RecoveryStopProcess{PID: p.PID, Identity: p.Identity, Role: p.Role})
	}
	return RecoveryRetirement{Stops: stops, Digest: evidence.digest, Check: evidence.check, Retire: evidence.retire}, nil
}

// Both terminal modes resolve the same physical owner and exact recorded name.
// A guessed tag-derived name must never become terminal retirement authority.
func (r OSRecoveryEvidenceReader) recoveryTerminalAddress(ctx context.Context, record ThreadRecord) (root, name string, err error) {
	if ctx == nil {
		return "", "", errors.New("recovery terminal requires context")
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	root, err = filepath.EvalSymlinks(r.DataDir)
	if err != nil {
		return "", "", err
	}
	if _, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)}); err != nil {
		return "", "", err
	}
	name = recordSessionName(record)
	if name == "" {
		name, err = (ScopedThreadArtifactCollisionChecker{GlobalDataDir: root}).PairSessionName(ctx, record.Address)
		if err != nil {
			return "", "", err
		}
	}
	if name == "" {
		return "", "", errors.New("exact recovery terminal name unavailable")
	}
	return root, name, nil
}

// recoveryParkHelper selects only the normal, still-connected lifecycle path.
// Damaged or unproved sources retain the exact confirmed terminal path instead.
// The returned helper must also appear in the person's stop summary.
func (c *Couch) recoveryParkHelper(ctx context.Context, record ThreadRecord) *ThreadIncarnation {
	if c.PairLifecycle == nil || c.PairLifecycle.validate() != nil || c.Proc == nil || record.Park != nil || record.Continuation != nil || startClaimed(record) {
		return nil
	}
	incarnation, err := soleParkableIncarnation(record)
	if err != nil || incarnation.State != IncarnationLive || observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) != Live {
		return nil
	}
	binding, err := observeRecordSession(ctx, c.PairLifecycle.Sessions, record)
	if err != nil || !binding.Present || binding.Name == "" {
		return nil
	}
	return &incarnation
}

func (c *Couch) parkPreparedRecovery(ctx context.Context, record ThreadRecord, helper *ThreadIncarnation) (ThreadRecord, error) {
	if helper == nil {
		return record, nil
	}
	switch observeExactProcess(c.Proc, ProcessIdentity{PID: helper.PID, Identity: helper.Identity}) {
	case Dead:
		return record, nil
	case Unknown:
		return record, errors.New("confirmed recovery client helper could not be rechecked; refresh the recovery choice")
	}
	current, err := soleParkableIncarnation(record)
	if err != nil || current.PID != helper.PID || current.Identity != helper.Identity || current.State != helper.State {
		return record, errors.New("recovery client helper changed; refresh the recovery choice")
	}
	parked, err := c.PairLifecycle.ParkExpected(ctx, record.Address, record.Revision)
	if err != nil {
		// A published request or partial completion cannot be replaced with a
		// second retirement strategy. Preserve its existing transaction to retry.
		return record, c.withParkRecoveryAdvice(ctx, parked.Thread, err)
	}
	return parked.Thread, nil
}
