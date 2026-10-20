package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

// RecoveryCapturePreserver copies a validated original recording through the
// existing capture retention owner. It must reject any changed input identity;
// it returns the archived identities, never a manufactured park receipt.
type RecoveryCapturePreserver func(context.Context, ThreadRecord, RecoveryObservation) (RecoveryObservation, error)

// RecoveryTerminalPreparation captures exact, positively attributed server
// generations. The returned action may retire only that unchanged set.
type RecoveryTerminalPreparation func(context.Context, ThreadRecord) (func(context.Context) error, error)

func selectRecoveryChoice(p RecoveryPreview, key string) (RecoveryChoice, error) {
	if p.Incomplete || p.Snapshot == nil {
		return RecoveryChoice{}, errors.New("recovery evidence incomplete; refresh resume candidates")
	}
	_, err := decodeRecoveryCandidateKey(key)
	if err != nil {
		return RecoveryChoice{}, err
	}
	for _, choice := range append(append([]RecoveryChoice(nil), p.Candidates...), func() []RecoveryChoice {
		if p.Fresh != nil {
			return []RecoveryChoice{*p.Fresh}
		}
		return nil
	}()...) {
		if choice.Key != key {
			continue
		}
		expected := RecoveryCandidateSelector{Version: 1, Address: p.Address, Agent: choice.Agent, NativeID: choice.NativeID, Digest: p.Snapshot.Digest}
		if p.Snapshot.Source != nil {
			expected.Version, expected.Source, expected.ExpiresAt = 2, p.Snapshot.Source, p.Snapshot.ExpiresAt
		}
		if _, err := ValidateRecoveryCandidateKey(key, expected); err != nil {
			return RecoveryChoice{}, err
		}
		return choice, nil
	}
	return RecoveryChoice{}, errors.New("recovery choice changed; refresh resume candidates")
}

// ResumeCandidate never guesses a choice. The two Couch observations bracket
// the durable start claim; Pair validates its persisted intent once more before
// composing native resume argv. Ordinary resume keeps its binding contract.
func (c *Couch) ResumeCandidate(ctx context.Context, t ResumeTarget, key string) (RecoveryResumeResult, error) {
	return c.RecoverSelection(ctx, RecoverySelection{Target: t, Kind: RecoveryInferred, CandidateKey: key})
}

func (c *Couch) recoverySlotAddress(ctx context.Context, record ThreadRecord) (string, error) {
	backend, err := c.Threads.storeForAddress(record.Address)
	if err != nil {
		return "", err
	}
	if backend.slot != nil {
		return fmt.Sprintf("%s:%d", backend.slot.Repo, backend.slot.Number), nil
	}
	root, err := c.Git.RunContext(ctx, record.StartingPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(root) || !workspaceRepoName(filepath.Base(root)) {
		return "", errors.New("recovery cannot identify the primary workspace")
	}
	return filepath.Base(root) + ":0", nil
}

func (c *Couch) prepareRecoveryRetirement(ctx context.Context, record ThreadRecord) (func(context.Context, ThreadRecord) (ThreadRecord, error), error) {
	if err := c.recoverySourceInactive(record); err != nil {
		return nil, err
	}
	if err := c.confirmRecoveryAbsent(ctx, record); err == nil {
		return nil, nil
	} else if c.RecoveryTerminal == nil {
		return nil, err
	}
	retire, err := c.RecoveryTerminal(ctx, record)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, current ThreadRecord) (ThreadRecord, error) {
		if err := retire(ctx); err != nil {
			return current, err
		}
		return current, nil
	}, nil
}

func (c *Couch) confirmRecoveryAbsent(ctx context.Context, record ThreadRecord) error {
	for _, inc := range record.Incarnations {
		if inc.Start != nil && inc.PID == 0 && inc.Identity == "" {
			continue
		}
		if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
			return errors.New("recovery source still runs or its ownership is uncertain; settle and park it before choosing a conversation")
		}
	}
	observation, err := c.observeRecoverySession(ctx, record, RecoveryEvidence{})
	if err != nil {
		return err
	}
	if observation.Presence != PresenceAbsent {
		return errors.New("recovery terminal still exists; retire its verified owner before relaunch")
	}
	return nil
}

// recoverySourceInactive prevents implicit retirement of an attributed source.
// The confirmed terminal preparer is required before attributed live owners
// may be retired; this fallback alone never grants stop authority.
func (c *Couch) recoverySourceInactive(record ThreadRecord) error {
	for _, owner := range c.ObserveRecordedProcesses([]ThreadRecord{record}) {
		if owner.Liveness != Dead {
			return errors.New("recovery source is running or its ownership is unknown; requires exact stop confirmation or Start fresh with a new terminal identity")
		}
	}
	return nil
}
