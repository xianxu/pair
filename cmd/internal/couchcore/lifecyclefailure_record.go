package couchcore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"
)

func (c *Couch) lifecycleFailureRecorder() *LifecycleFailureRecorder {
	c.failureRecorderMu.Lock()
	defer c.failureRecorderMu.Unlock()
	if c.failureRecorder == nil {
		now := time.Now
		if c.Clock != nil {
			now = c.Clock.Now
		}
		c.failureRecorder = NewLifecycleFailureRecorder(c.LaunchFailures, now, func() (string, error) {
			entropy := c.Entropy
			if entropy == nil {
				entropy = rand.Reader
			}
			var raw [16]byte
			if _, err := io.ReadFull(entropy, raw[:]); err != nil {
				return "", err
			}
			return "operation-" + hex.EncodeToString(raw[:]), nil
		})
	}
	return c.failureRecorder
}

// BeginOperationAttempt binds a tag-only admitted action once to its exact
// record path. Cached queue targets already carry that physical path and do
// no additional IO here. Callers invoke this only after receipt admission.
func (c *Couch) BeginOperationAttempt(op string, target LifecycleFailureTarget) (*OperationAttempt, error) {
	if target.StartingPath == "" && target.Address.Tag != "" {
		if c.Threads == nil {
			return nil, fmt.Errorf("%s: exact thread binding unavailable; refresh and retry", op)
		}
		record, err := c.Threads.GetThread(target.Address)
		if err != nil {
			return nil, fmt.Errorf("%s: exact thread binding unavailable; refresh and retry: %w", op, err)
		}
		if record.Address != target.Address || !filepath.IsAbs(record.StartingPath) || filepath.Clean(record.StartingPath) != record.StartingPath {
			return nil, fmt.Errorf("%s: canonical thread slot unavailable; refresh and retry", op)
		}
		target.StartingPath = record.StartingPath
	}
	return c.lifecycleFailureRecorder().Begin(op, target)
}

// OperationAttemptForCall is the production dispatcher factory. Prepared owner
// calls already carry canonical paths or exact thread identity; unresolved calls
// do not acquire a fabricated diagnostic target.
func (c *Couch) OperationAttemptForCall(call OperationCall) (*OperationAttempt, error) {
	op, known := operationByName(call.Name)
	if !known || !OperationRecordsLifecycleFailure(op) {
		return nil, nil
	}
	target := LifecycleFailureTarget{
		Address:      ThreadAddress{RepoScope: call.Args["repo-scope"], Tag: ThreadTag(call.Args["tag"])},
		StartingPath: call.Args["path"], ParkNonce: call.Args["park-nonce"],
	}
	if call.FailureTarget != nil {
		target = *call.FailureTarget
	}
	if target.StartingPath == "" && filepath.IsAbs(call.Args["ref"]) {
		target.StartingPath = call.Args["ref"]
	}
	if !filepath.IsAbs(target.StartingPath) || filepath.Clean(target.StartingPath) != target.StartingPath {
		target.StartingPath = "" // Resolution belongs to the operation, not diagnostics.
	}
	if target.StartingPath == "" && target.Address.Tag == "" {
		return nil, nil
	}
	return c.BeginOperationAttempt(call.Name, target)
}

// LifecycleFailureTargetForRow freezes a selected row before queued work.
func LifecycleFailureTargetForRow(row ActionableThreadSummary) LifecycleFailureTarget {
	target := LifecycleFailureTarget{Address: row.Address, StartingPath: row.StartingPath, ParkNonce: row.ParkNonce}
	if row.Target.Kind == ThreadTargetSlot {
		target.StartingPath = row.Target.Slot.WorktreeRoot
		target.Slot = row.Target.Slot.Repo + ":" + strconv.Itoa(row.Target.Slot.Number)
	}
	return target
}
