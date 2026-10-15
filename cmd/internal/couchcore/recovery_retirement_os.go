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
