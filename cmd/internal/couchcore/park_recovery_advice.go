package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// parkRecoverySlotReference uses the record's actual storage origin. A local
// thread in a repository is not necessarily the repository's :0 slot row.
func (c *Couch) parkRecoverySlotReference(ctx context.Context, address ThreadAddress) string {
	if c == nil || c.Threads == nil {
		return ""
	}
	view := *c.Threads
	view.readOnly = true
	backend, err := view.storeForAddressContext(ctx, address)
	if err != nil || backend.slot == nil {
		return ""
	}
	if _, err := backend.PreviewThreadContext(ctx, address); err != nil {
		return ""
	}
	slot := backend.slot
	if slot.Number < 1 || !workspaceRepoName(slot.Repo) {
		return ""
	}
	return fmt.Sprintf("%s:%d", slot.Repo, slot.Number)
}

// ParkRecoveryAdvice names only an existing exact target; it never fabricates a
// primary-slot address from an arbitrary local conversation's path or tag.
func (c *Couch) ParkRecoveryAdvice(ctx context.Context, address ThreadAddress) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if ref := c.parkRecoverySlotReference(ctx, address); ref != "" {
		return SlotOperationCommand(OpRecoverPark, ref)
	}
	return "Tab → Recover park"
}

// ParkRecoveryNextAction observes advice, never launch authority. The subsequent
// explicit operation reobserves these facts through its ordinary admission path.
func (c *Couch) ParkRecoveryNextAction(ctx context.Context, record ThreadRecord) (string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil {
		return "", "", errors.New("park recovery advice requires a thread store")
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if record.Park != nil {
		return "", "", refuseResume(ResumeParking, "a park transaction is unfinished; "+c.ParkRecoveryAdvice(ctx, record.Address))
	}
	command := func(op, local string) string {
		if ref := c.parkRecoverySlotReference(ctx, record.Address); ref != "" {
			return SlotOperationCommand(op, ref)
		}
		return local
	}
	session, listed, err := c.listedTerminal(ctx, record)
	if err != nil {
		return "", "", err
	}
	if listed {
		return "continue-live", command("resume", "Enter"), nil
	}
	if session == "" {
		return "", "", refuseResume(ResumeUnknown, "exact terminal identity is unavailable; inspect before choosing an outcome")
	}
	if err := c.recoverySourceInactive(record); err != nil {
		return "", "", err
	}
	binding, bindingErr := c.resumeEvidence(ctx, record)
	// A resolver may return a typed diagnosis alongside a complete observation.
	// Real IO failures and incomplete observations are never confirmed absence.
	if bindingErr != nil && ResumeDiagnosticOf(bindingErr) == "" {
		return "", "", bindingErr
	}
	if binding.ObservationIncomplete || binding.Status == sessioninventory.BindingProvisional {
		return "", "", refuseResume(ResumeBindingProvisional, "native conversation observation is incomplete or provisional; inspect and retry")
	}
	switch binding.Status {
	case sessioninventory.BindingEstablished, sessioninventory.BindingUnbound, sessioninventory.BindingAmbiguous:
	default:
		return "", "", refuseResume(ResumeUnknown, "native conversation status is unknown; inspect and retry")
	}
	if bindingErr != nil && !ResumeRebootAdvice[ResumeDiagnosticOf(bindingErr)] {
		return "", "", bindingErr
	}
	_, eligibilityErr := DecideResume(ResumeEligibilityInput{Thread: record, Binding: binding, WorkingPathExists: c.workingPathExists(record)})
	if eligibilityErr == nil {
		return "resume-from-disk", command("resume", "Tab → Resume from disk"), nil
	}
	code := ResumeDiagnosticOf(eligibilityErr)
	if isBindingDiagnostic(code) || code == ResumeTombstoned {
		if ResumeRebootAdvice[code] {
			return "start-fresh", command(OpReboot, "Tab → Start fresh"), nil
		}
	}
	return "", "", eligibilityErr
}
