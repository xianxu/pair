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

// BeginOperationAttempt copies available evidence without resolving or binding a target.
func (c *Couch) BeginOperationAttempt(op string, target LifecycleFailureTarget) (*OperationAttempt, error) {
	return c.lifecycleFailureRecorder().Begin(op, target)
}

// OperationAttemptForCall is the production dispatcher factory. Prepared owner
// calls may carry canonical paths or exact thread identity; unresolved calls
// retain their dispatched reference without acquiring lifecycle authority.
func (c *Couch) OperationAttemptForCall(call OperationCall) (*OperationAttempt, error) {
	op, known := operationByName(call.Name)
	if !known || !OperationRecordsLifecycleFailure(op) {
		return nil, nil
	}
	return c.BeginOperationAttempt(call.Name, lifecycleFailureTargetForCall(call))
}

// lifecycleFailureTargetForCall copies evidence and never resolves a reference.
func lifecycleFailureTargetForCall(call OperationCall) LifecycleFailureTarget {
	target := LifecycleFailureTarget{
		Address:      completeFailureAddress(ThreadAddress{RepoScope: call.Args["repo-scope"], Tag: ThreadTag(call.Args["tag"])}, ThreadAddress{}),
		StartingPath: call.Args["path"], ParkNonce: call.Args["park-nonce"], Target: call.Args["ref"],
	}
	if call.FailureTarget != nil {
		target = *call.FailureTarget
	}
	target.Address = completeFailureAddress(target.Address, ThreadAddress{})
	if target.StartingPath == "" && filepath.IsAbs(call.Args["ref"]) {
		target.StartingPath = call.Args["ref"]
	}
	if !filepath.IsAbs(target.StartingPath) || filepath.Clean(target.StartingPath) != target.StartingPath {
		if target.Target == "" {
			target.Target = target.StartingPath
		}
		target.StartingPath = "" // Resolution belongs to the operation, not diagnostics.
	}
	if target.Target == "" && target.StartingPath == "" && target.Address.Tag == "" {
		target.Target = "unresolved " + call.Name
	}
	return target
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
