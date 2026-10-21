package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// startFreshRecoveryWithoutRecord carries the remaining workspace evidence into
// an explicit Fresh launch. Version admission and record publication stay with
// the ordinary Fresh transaction; workspace validation never converges setup.
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
	orient := &threadrecord.RecoveryOrientation{Branch: branch, Issue: issue, Unavailable: "Previous TTY capture unavailable; use branch and issue evidence."}
	started, err := c.startFreshSlotWithRecovery(ctx, selection.Target.Path, selection.Agent, false, nil, nil, orient, freshSlotRecordlessRecovery)
	return RecoveryResumeResult{Mode: Fresh, Start: started, Notice: "started fresh; previous TTY capture unavailable; use branch and issue evidence; any unproved owners were left untouched under their original identities; workspace setup was not refreshed"}, err
}

// recoveryExistingWorkspace proves the retained checkout without converging
// setup or repairing a host underneath a foreign or unknown agent.
type recoveryExistingWorkspace struct {
	c    *Couch
	slot SlotIdentity
}

func (r recoveryExistingWorkspace) Ensure(ctx context.Context, request ProvisionRequest) (ProvisionResult, error) {
	if err := ctx.Err(); err != nil {
		return ProvisionResult{}, err
	}
	if request.Path != r.slot.PrimaryRoot || request.Slot != r.slot.Number {
		return ProvisionResult{}, errors.New("recovery workspace changed")
	}
	if r.c == nil || r.c.Slots == nil {
		return ProvisionResult{}, errors.New("recovery workspace catalog unavailable")
	}
	repository, err := r.c.Slots.Discover(ctx, r.slot.PrimaryRoot)
	if err != nil {
		return ProvisionResult{}, err
	}
	for _, candidate := range repository.Slots {
		if candidate.Identity.WorktreeRoot != r.slot.WorktreeRoot || candidate.Identity.Number != r.slot.Number {
			continue
		}
		if !candidate.Verified || candidate.Err != nil || candidate.Identity != r.slot {
			return ProvisionResult{}, errors.New("recovery requires an unchanged verified checkout; repair cannot run beside unproved owners")
		}
		physical, err := filepath.EvalSymlinks(candidate.Identity.WorktreeRoot)
		if err != nil {
			return ProvisionResult{}, err
		}
		if physical != r.slot.WorktreeRoot {
			return ProvisionResult{}, errors.New("recovery checkout identity changed")
		}
		return ProvisionResult{SchemaVersion: 1, Address: fmt.Sprintf("%s:%d", r.slot.Repo, r.slot.Number), Path: r.slot.WorktreeRoot, Disposition: "reused"}, ctx.Err()
	}
	return ProvisionResult{}, errors.New("recovery slot disappeared")
}
func (c *Couch) selectRecordlessRecoverySlot(ctx context.Context, path string) (*ThreadStore, SlotIdentity, WorkspaceReadiness, error) {
	target, _, err := c.resolveSlotInput(ctx, path)
	if err != nil {
		return nil, SlotIdentity{}, nil, err
	}
	if target == nil || target.Kind != ThreadTargetSlot {
		return nil, SlotIdentity{}, nil, errors.New("recordless recovery requires an existing numbered slot")
	}
	slot := target.Slot
	readiness := recoveryExistingWorkspace{c: c, slot: slot}
	if _, err := readiness.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number}); err != nil {
		return nil, slot, nil, err
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return nil, slot, nil, err
	}
	if err := c.Threads.EnrollSlotRepository(ctx, repository); err != nil {
		return nil, slot, nil, err
	}
	local := newSlotThreadStore(c.Namespace, slot)
	local.coordinator = c.Threads.coordinator
	return local, slot, readiness, nil
}
