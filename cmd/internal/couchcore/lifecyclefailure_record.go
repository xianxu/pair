package couchcore

import (
	"crypto/rand"
	"encoding/hex"
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

// BeginOperationAttempt binds an admitted action to its existing target facts.
// It performs no inventory, alias, process or filesystem resolution.
func (c *Couch) BeginOperationAttempt(op string, target LifecycleFailureTarget) (*OperationAttempt, error) {
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
