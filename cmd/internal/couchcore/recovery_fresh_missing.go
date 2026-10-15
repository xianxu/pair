package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/orientation"
)

// startFreshRecoveryWithoutRecord carries the remaining workspace evidence into
// an explicit Fresh launch. Slot selection and version admission stay with the
// ordinary fresh transaction; missing screen evidence never grants write access.
func (c *Couch) startFreshRecoveryWithoutRecord(ctx context.Context, selection RecoverySelection) (RecoveryResumeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || selection.Kind != RecoveryFresh || selection.Target.Path == "" || selection.Target.Address != (ThreadAddress{}) {
		return RecoveryResumeResult{}, errors.New("missing-record Fresh recovery requires one slot path")
	}
	target, _, err := c.resolveSlotInput(ctx, selection.Target.Path)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	if target == nil || target.Kind != ThreadTargetSlot {
		return RecoveryResumeResult{}, errors.New("missing-record Fresh recovery requires an existing numbered slot")
	}
	family, err := c.slotFamily(ctx, target.Slot, false)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	cwd, err := ValidateFamilyPath(target.Slot.WorktreeRoot, family.RelativeStart)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	branch, issue := (OSRecoveryEvidenceReader{Git: c.Git}).recoveryBranchIssue(ctx, ThreadRecord{StartingPath: target.Slot.WorktreeRoot, WorkingPath: cwd})
	prompt, err := orientation.BuildRecoveryOrientation(orientation.RecoveryOrientationContext{SlotAddress: fmt.Sprintf("%s:%d", target.Slot.Repo, target.Slot.Number), Branch: branch, Issue: issue})
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	started, err := c.startFreshSlotWithPrompt(ctx, selection.Target.Path, "", false, nil, nil, prompt)
	return RecoveryResumeResult{Start: started, Notice: "started fresh; previous TTY capture unavailable; use branch and issue evidence"}, err
}
