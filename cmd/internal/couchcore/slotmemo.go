package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// setupInputsDigest hashes what setup reads: the host's HEAD, every
// construct/deps it would walk (the host's, then each present dependency's),
// and the weave that would run it, so an upgraded weave retries.
func setupInputsDigest(ctx context.Context, io ProvisionIO, layout SlotLayout) (string, error) {
	head, err := io.Run(ctx, ProvisionCommand{Dir: layout.Host(), Program: "git", Args: []string{"rev-parse", "HEAD"}})
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(head)
	if id, ok := io.(ProgramIdentifier); ok {
		h.Write([]byte("weave\x00" + id.ProgramIdentity("weave") + "\x00"))
	}
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

// SlotFailure records a slot's last blocking reconcile (pair#442): the step,
// the command and its budget, and the full cause with the command's stderr
// tail, so a failure the operator saw only on screen can be read afterward
// (couch --show). One per slot: the next failure replaces it and the next
// success removes it.
type SlotFailure struct {
	SchemaVersion int            `json:"schema_version"`
	At            time.Time      `json:"at"`
	Resource      SlotResourceID `json:"resource"`
	Step          ConvergeStep   `json:"step,omitempty"`
	Class         FailureClass   `json:"class"`
	Command       string         `json:"command,omitempty"`
	Timeout       string         `json:"timeout,omitempty"`
	TimedOut      bool           `json:"timed_out,omitempty"`
	Cause         string         `json:"cause"`
}

// boundCause keeps a cause's head (what failed) and tail (the end of its
// stderr) within limit bytes, eliding the middle.
func boundCause(cause string, limit int) string {
	if len(cause) <= limit {
		return cause
	}
	const marker = "\n…\n"
	head := limit / 4
	return strings.ToValidUTF8(cause[:head]+marker+cause[len(cause)-(limit-head-len(marker)):], "")
}

// fitSlotFailure shrinks the cause until the encoded record fits the
// provision store's record limit: a cause carries up to 64 KiB of stderr
// tail, and JSON escaping can grow it sixfold. It never drops the record.
func fitSlotFailure(r SlotFailure) SlotFailure {
	full := r.Cause
	for limit := provisionRecordLimit; limit > 0; limit /= 2 {
		r.Cause = boundCause(full, limit)
		if raw, err := json.Marshal(r); err == nil && len(raw) <= provisionRecordLimit {
			return r
		}
	}
	r.Cause = ""
	return r
}

func newSlotFailure(e *SlotReconcileError, at time.Time) SlotFailure {
	r := SlotFailure{SchemaVersion: 1, At: at.UTC(), Resource: e.Failure.Resource, Step: e.Failure.Step, Class: e.Failure.Class, Cause: e.Failure.Cause}
	if e.Err != nil {
		r.Cause = e.Err.Error()
	}
	var cmd *ProvisionCommandError
	if errors.As(e.Err, &cmd) {
		r.Command, r.TimedOut = cmd.Command, cmd.TimedOut
		if cmd.Timeout > 0 {
			r.Timeout = cmd.Timeout.String()
		}
	}
	return fitSlotFailure(r)
}

// recordSlotFailure keeps a blocking outcome under the creation lease, like
// every slot record. A record that cannot be written is not dropped silently:
// the caller's error says so.
func (p *WorkspaceProvisioner) recordSlotFailure(l SlotLayout, e *SlotReconcileError) error {
	lease, err := AcquireHostCreationLease(l.common)
	if err != nil {
		return err
	}
	defer lease.Close()
	return p.Store.Write(SlotFailurePath(l), newSlotFailure(e, time.Now()))
}

// clearSlotFailure removes the record once the slot converges.
func (p *WorkspaceProvisioner) clearSlotFailure(l SlotLayout) {
	if _, err := os.Lstat(SlotFailurePath(l)); err != nil {
		return
	}
	lease, err := AcquireHostCreationLease(l.common)
	if err != nil {
		return
	}
	defer lease.Close()
	_ = p.Store.Remove(SlotFailurePath(l))
}

// ReadSlotFailure reads a slot's last-failure record (nil when none).
func ReadSlotFailure(l SlotLayout) (*SlotFailure, error) {
	var r SlotFailure
	exists, err := (ProvisionStore{}).Read(SlotFailurePath(l), &r)
	if err != nil || !exists {
		return nil, err
	}
	return &r, nil
}
