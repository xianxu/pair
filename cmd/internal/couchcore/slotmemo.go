package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// SetupAttempt remembers a setup that failed with a hand-off cause (R5), keyed
// by a digest of its inputs, so later opens do not re-run a compile that
// cannot succeed until an input changes. A retryable failure is never
// remembered. It lives beside the marker and dies with the registration.
type SetupAttempt struct {
	SchemaVersion int       `json:"schema_version"`
	Digest        string    `json:"digest"`
	Failure       string    `json:"failure"`
	At            time.Time `json:"at"`
}

// setupInputsDigest hashes what setup reads: the host's HEAD and every
// construct/deps it would walk (the host's, then each present dependency's).
func setupInputsDigest(ctx context.Context, io ProvisionIO, layout SlotLayout) (string, error) {
	head, err := io.Run(ctx, ProvisionCommand{Dir: layout.Host(), Program: "git", Args: []string{"rev-parse", "HEAD"}})
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(head)
	read := func(dir string) {
		raw, err := os.ReadFile(filepath.Join(dir, "construct", "deps"))
		h.Write([]byte(dir + "\x00"))
		if err == nil {
			h.Write(raw)
		}
		h.Write([]byte{0})
	}
	read(layout.Host())
	if declared, err := DeclaredDepsOf(layout.Env(), layout.Host()); err == nil {
		for _, d := range declared.Deps {
			if d.Present && !d.Outside {
				read(d.Path)
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// rememberFailedSetup records a hand-off setup failure for its inputs.
func rememberFailedSetup(ctx context.Context, p *WorkspaceProvisioner, layout SlotLayout, admin string, failure ReconcileFailure) error {
	digest, err := setupInputsDigest(ctx, p.IO, layout)
	if err != nil {
		return err
	}
	return p.Store.Write(SetupAttemptPath(admin), SetupAttempt{SchemaVersion: 1, Digest: digest, Failure: failure.Cause, At: time.Now().UTC()})
}

// knownFailedSetup reports the remembered failure when its inputs are
// unchanged.
func knownFailedSetup(ctx context.Context, io ProvisionIO, layout SlotLayout, admin string) (string, bool) {
	var attempt SetupAttempt
	exists, err := (ProvisionStore{}).Read(SetupAttemptPath(admin), &attempt)
	if err != nil || !exists || attempt.SchemaVersion != 1 {
		return "", false
	}
	digest, err := setupInputsDigest(ctx, io, layout)
	if err != nil || digest != attempt.Digest {
		return "", false
	}
	return attempt.Failure, true
}
