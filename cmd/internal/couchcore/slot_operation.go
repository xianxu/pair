package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Slot-operation refusal codes (pair#367 M2): why a resume or reboot request
// for one slot was not run. They reach the caller as a receipt's Code.
const (
	SlotOpUnknownSlot = "unknown-slot" // the address names no slot Couch can resolve
	SlotOpNoThread    = "no-thread"    // no Couch row stands for the slot
	SlotOpAmbiguous   = "ambiguous"    // several rows stand for the slot
	SlotOpNotOffered  = "not-offered"  // ActorActions does not offer the operation on the row
)

// SlotOperationError is a typed refusal: a code from the list above and the
// fact behind it.
type SlotOperationError struct {
	Code   string
	Detail string
}

func (e *SlotOperationError) Error() string { return e.Code + ": " + e.Detail }

// slotOperations are the actor operations a slot target may request: the ONE
// list the protocol, the socket and admission all read (IsSlotOperation), so a
// new verb cannot be half-added.
var slotOperations = []string{"resume", "reboot", "reap", "recover", OpRelaunch, OpReloadContext}

// IsSlotOperation reports whether op is a remote slot operation.
func IsSlotOperation(op string) bool { return slices.Contains(slotOperations, op) }

// SlotOperationTakesForceUnknown: both live verbs accept --force-unknown.
func SlotOperationTakesForceUnknown(op string) bool { return op == OpRelaunch || op == OpReloadContext }

// SlotOperationTakesSameBinary: only relaunch has a freshness rule to override,
// so --same-binary anywhere else is refused rather than silently ignored.
func SlotOperationTakesSameBinary(op string) bool { return op == OpRelaunch }

// SelectSlotRow picks the one actionable row that stands for a slot: a :N
// slot row by its host checkout, or, for :0, the row IsPrimaryRow accepts (a
// thread in a subdirectory of the primary is never the slot's row). Pure.
func SelectSlotRow(rows []ActionableThreadSummary, number int, path string) (ActionableThreadSummary, error) {
	path = filepath.Clean(path)
	scope, scopeErr := launcher.ResolveRepoScope(path)
	var found []ActionableThreadSummary
	for _, row := range rows {
		switch {
		case number > 0:
			if row.Target.Kind == ThreadTargetSlot && filepath.Clean(row.Target.Slot.WorktreeRoot) == path {
				found = append(found, row)
			}
		case scopeErr == nil && IsPrimaryRow(row, path, scope.Key):
			found = append(found, row)
		}
	}
	switch len(found) {
	case 0:
		return ActionableThreadSummary{}, &SlotOperationError{Code: SlotOpNoThread, Detail: "no Couch thread stands for " + path}
	case 1:
		return found[0], nil
	}
	return ActionableThreadSummary{}, &SlotOperationError{Code: SlotOpAmbiguous, Detail: fmt.Sprintf("%d Couch threads stand for %s", len(found), path)}
}

// ActorOperationArgs is the one row → arguments mapping for resume and
// reboot, read by the switcher and the socket alike (ARCH-DRY). A slot row is
// addressed by its host checkout (its record may be missing or replaced under
// the row), with the checkout's own repository scope, which resume's
// declaration requires. An ordinary row is addressed by its exact tag, and a
// detached one resumes warm-only: it may be reattached, never cold-started,
// even if the record changes before the queued operation runs.
func ActorOperationArgs(row ActionableThreadSummary, op string) map[string]string {
	if row.Target.Kind == ThreadTargetSlot {
		scope := row.Address.RepoScope
		if resolved, err := launcher.ResolveRepoScope(row.Target.Slot.WorktreeRoot); err == nil {
			scope = resolved.Key
		}
		return map[string]string{"path": row.Target.Slot.WorktreeRoot, "repo-scope": scope}
	}
	args := map[string]string{"repo-scope": row.Address.RepoScope, "tag": string(row.Address.Tag)}
	if op == "resume" && row.Detached() {
		args["warm-only"] = "true"
	}
	return args
}

// SlotOperationCommand is the CLI text that runs op on a slot through the
// running Couch. --confirm is added exactly when the operation's declaration
// requires a confirmation (OperationConfirms).
func SlotOperationCommand(op, address string) string {
	command := "couch --" + op + " " + address
	if confirms, _ := OperationConfirms(op); confirms {
		command += " --confirm"
	}
	return command
}

// SendToCommand is the CLI text that delivers message to the slot's own
// agent, the message quoted for a POSIX shell.
func SendToCommand(address, message string) string {
	return "couch --send-to " + address + " --message " + ShellQuote(message)
}

// ShellQuote quotes s as one POSIX shell word.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// PrepareSlotOperation resolves a slot address and admits op on it against
// the inventory as it stands now: the address must name a slot, exactly one
// row must stand for it, and ActorActions must offer op on that row, the same
// table the switcher offers from. It returns the call to dispatch.
//
// relaunch and reload-context (pair#421) act on LIVE rows, which ActorActions
// never offers anything for; DecideLiveRestart replaces the offer check for
// them, over facts the injected LiveRestartProbe gathers now. The returned
// note (freshness, an override used) belongs on the caller's receipt.
func (c *Couch) PrepareSlotOperation(ctx context.Context, op, target string, opts LiveRestartOptions) (OperationCall, string, error) {
	if !slices.Contains(slotOperations, op) {
		return OperationCall{}, "", fmt.Errorf("%q is not a slot operation", op)
	}
	call, note, err := c.prepareSlotOperation(ctx, op, target, opts)
	return call, note, err
}

func (c *Couch) prepareSlotOperation(ctx context.Context, op, target string, opts LiveRestartOptions) (OperationCall, string, error) {
	ref, recognized, err := ParseWorkspaceReference(target)
	if err != nil || !recognized || ref.Repo == "" {
		return OperationCall{}, "", &SlotOperationError{Code: SlotOpUnknownSlot, Detail: target + " is not an exact repo:N address"}
	}
	path, recognized, err := c.WorkspaceReferencePath(ctx, target)
	if err != nil || !recognized || path == "" {
		detail := target + " names no slot"
		if err != nil {
			detail += ": " + err.Error()
		}
		return OperationCall{}, "", &SlotOperationError{Code: SlotOpUnknownSlot, Detail: detail}
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return OperationCall{}, "", err
	}
	row, err := SelectSlotRow(rows, ref.Number, path)
	if err != nil {
		return OperationCall{}, "", err
	}
	if op == OpRelaunch || op == OpReloadContext {
		return c.prepareLiveRestart(ctx, op, target, row, path, opts)
	}
	offered := ActorActions(ActorRowFactsOf(row))
	if !slices.Contains(offered, op) && !(op == "recover" && RecoverOffered(offered)) {
		state := string(row.State)
		if row.Reason != "" {
			state += " (" + string(row.Reason) + ")"
		}
		return OperationCall{}, "", &SlotOperationError{Code: SlotOpNotOffered, Detail: target + " is " + state}
	}
	return OperationCall{Name: op, Args: ActorOperationArgs(row, op), Implicit: true, Context: ctx}, "", nil
}
