package couchtty

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// A target remains owned through execution and adoption. A prepared start can
// precede its inventory row; leave owns the whole console instead of a slot.
type menuOperationTarget struct {
	RowKey    couchcore.ThreadRowKey
	StartPath string
	Couch     bool
}

type MenuOperationPhase uint8

const (
	MenuOperationRunning MenuOperationPhase = iota + 1
	MenuOperationSucceeded
	MenuOperationFailed
	MenuOperationUnconfirmed
	MenuOperationNeedsConfirmation
)

type MenuOperation struct {
	Origin        MenuOperationOrigin
	Phase         MenuOperationPhase
	StartedAt     time.Time
	Detail        string
	ResultAddress couchcore.ThreadAddress
	Recovery      *couchcore.RecoveryConfirmationOffer
}

func cloneMenuOperations(jobs map[menuOperationTarget]MenuOperation) map[menuOperationTarget]MenuOperation {
	if jobs == nil {
		return nil
	}
	out := make(map[menuOperationTarget]MenuOperation, len(jobs))
	for key, job := range jobs {
		out[key] = job
	}
	return out
}

func menuOperationForAttempt(state MenuState, attempt uint64) (menuOperationTarget, MenuOperation, bool) {
	if attempt == 0 {
		return menuOperationTarget{}, MenuOperation{}, false
	}
	for key, job := range state.Operations {
		if job.Phase == MenuOperationRunning && job.Origin.Attempt == attempt {
			return key, job, true
		}
	}
	return menuOperationTarget{}, MenuOperation{}, false
}
func menuOriginForAttempt(state MenuState, attempt uint64) MenuOperationOrigin {
	_, job, _ := menuOperationForAttempt(state, attempt)
	return job.Origin
}
func menuHasRunningOperation(state MenuState) bool {
	for _, job := range state.Operations {
		if job.Phase == MenuOperationRunning {
			return true
		}
	}
	return false
}
func menuOperationForRow(state MenuState, row couchcore.ActionableThreadSummary) (MenuOperation, bool) {
	if job, ok := state.Operations[menuOperationTarget{RowKey: menuRowKey(row)}]; ok {
		return job, true
	}
	for _, job := range state.Operations {
		if row.Address != (couchcore.ThreadAddress{}) && job.ResultAddress == row.Address {
			return job, true
		}
	}
	return MenuOperation{}, false
}
func menuRunningForAddress(state MenuState, address couchcore.ThreadAddress) (MenuOperation, bool) {
	if row, ok := menuThread(state, address); ok {
		if job, exists := menuOperationForRow(state, row); exists && job.Phase == MenuOperationRunning {
			return job, true
		}
	}
	for _, job := range state.Operations {
		if job.Phase == MenuOperationRunning && job.Origin.Address == address {
			return job, true
		}
	}
	return MenuOperation{}, false
}
func replaceMenuOperationOrigin(state MenuState, origin MenuOperationOrigin) MenuState {
	key, job, ok := menuOperationForAttempt(state, origin.Attempt)
	if !ok {
		return state
	}
	state.Operations = cloneMenuOperations(state.Operations)
	job.Origin = origin
	state.Operations[key] = job
	return state
}
func menuOperationTargetFor(state MenuState, effect MenuEffect, address couchcore.ThreadAddress) (menuOperationTarget, bool) {
	if effect.Operation == "leave" {
		return menuOperationTarget{Couch: true}, true
	}
	if effect.RowKey != (couchcore.ThreadRowKey{}) {
		return menuOperationTarget{RowKey: effect.RowKey}, true
	}
	if row, ok := menuThread(state, address); ok && address != (couchcore.ThreadAddress{}) {
		return menuOperationTarget{RowKey: menuRowKey(row)}, true
	}
	if address.RepoScope != "" && address.Tag != "" {
		return menuOperationTarget{RowKey: couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: address}}, true
	}
	if effect.Operation == "start" {
		resolution := state.CurrentFrame().PreviewResolution
		if key, err := resolution.Target.RowKey(); err == nil {
			return menuOperationTarget{RowKey: key}, true
		}
		if path := string(resolution.Worktree); filepath.IsAbs(path) {
			return menuOperationTarget{StartPath: filepath.Clean(path)}, true
		}
	}
	return menuOperationTarget{}, false
}

func selectBusyMenuRow(state MenuState, row couchcore.ActionableThreadSummary) (MenuState, bool) {
	job, ok := menuOperationForRow(state, row)
	if !ok {
		return state, false
	}
	if job.Phase == MenuOperationNeedsConfirmation && job.Recovery != nil {
		appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Action: "recovery-confirmation", Thread: job.Origin.Address, RowKey: job.Origin.RowKey, SelectedItem: "cancel", RecoveryConfirmation: job.Recovery})
		return state, true
	}
	if job.Phase != MenuOperationRunning {
		return state, false
	}
	state.LandingAttempt = job.Origin.Attempt
	state.Frames = append([]MenuFrame(nil), state.Frames[:1]...)
	selectMenuRow(&state.Frames[0], row)
	state.Notice = MenuNotice{Level: MenuNoticeProgress, Text: menuOperationProgressText(state, job.Origin.Operation, job.Origin.Address), Owner: MenuProgressOwner{OperationAttempt: job.Origin.Attempt}}
	return state, true
}

func dispatchMenuOperation(state MenuState, effect MenuEffect, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	if effect.Operation == "" {
		return state, nil
	}
	target, ok := menuOperationTargetFor(state, effect, address)
	if !ok {
		state.Notice = errorMenuNotice("no exact operation target is available")
		return state, nil
	}
	for key, job := range state.Operations {
		if job.Phase == MenuOperationRunning && (key == target || key.Couch || target.Couch) {
			state.Notice = errorMenuNotice("busy: " + job.Origin.Operation)
			return state, nil
		}
	}
	if state.OperationSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("operation attempt identity exhausted")
		return state, nil
	}
	state.OperationSequence++
	effect.Attempt = state.OperationSequence
	if effect.Operation == "resume" {
		state = clearReattachFailure(state, address)
	}
	origin := MenuOperationOrigin{RowKey: target.RowKey, Operation: effect.Operation, Attempt: effect.Attempt, Address: address, FrameInstance: state.CurrentFrame().Instance, FrameKind: state.CurrentFrame().Kind, Depth: len(state.Frames)}
	state.Operations = cloneMenuOperations(state.Operations)
	if state.Operations == nil {
		state.Operations = make(map[menuOperationTarget]MenuOperation)
	}
	state.Operations[target] = MenuOperation{Origin: origin, Phase: MenuOperationRunning, StartedAt: state.ObservedAt}
	if menuOperationLands(effect.Operation) {
		state.LandingAttempt = origin.Attempt
	}
	// An admitted action closes its own form. Later results never resurrect it.
	if len(state.Frames) > 1 {
		state.Frames = append([]MenuFrame(nil), state.Frames[:1]...)
	}
	state.SpinnerPhase = 0
	state.Notice = MenuNotice{Level: MenuNoticeProgress, Text: menuOperationProgressText(state, effect.Operation, address), Owner: MenuProgressOwner{OperationAttempt: effect.Attempt}}
	return state, []MenuEffect{effect}
}
func menuOperationLands(operation string) bool {
	switch operation {
	case "switch", "start", "resume", "reboot", "recover", "recovery-confirmation", "relaunch", "reload-context", "switch-agent":
		return true
	}
	return false
}
func reduceOperationResult(state MenuState, event MenuEvent) MenuState {
	key, job, ok := menuOperationForAttempt(state, event.Attempt)
	if !ok || !menuOperationMatches(job.Origin, event) {
		return state
	}
	job.ResultAddress = event.Address
	job.Phase = MenuOperationSucceeded
	job.Detail = event.Operation + " complete"
	if !event.Success {
		job.Phase = MenuOperationFailed
		if event.Unconfirmed {
			job.Phase = MenuOperationUnconfirmed
		}
		job.Detail = event.Error
		if job.Detail == "" {
			job.Detail = event.Operation + " failed"
		}
		state.Notice = errorMenuNotice(job.Detail)
		state.Notice.Owner = MenuProgressOwner{OperationAttempt: job.Origin.Attempt}
	} else if state.Notice.Owner.OperationAttempt == job.Origin.Attempt {
		state.Notice = MenuNotice{}
	}
	if event.RecoveryConfirmation != nil {
		offer := *event.RecoveryConfirmation
		offer.Stops = append([]couchcore.RecoveryStopProcess(nil), offer.Stops...)
		job.Phase = MenuOperationNeedsConfirmation
		job.Detail = "confirmation needed"
		job.Recovery = &offer
	}
	state.Operations = cloneMenuOperations(state.Operations)
	state.Operations[key] = job
	if event.Success && !event.InventorySet && operationNeedsProjectionRefresh(event.Operation) {
		state.ProjectionPending = true
		state.ProjectionAfterGeneration = event.ProjectionAfterGeneration
	}
	// Only a still-current landing may change the selected row. Newer frames,
	// filtering and operator navigation are independent of this completion.
	if event.Success && state.LandingAttempt == job.Origin.Attempt && len(state.Frames) == 1 {
		state.Frames[0].SelectedAddress = event.Address
		if job.Origin.RowKey.Kind == couchcore.ThreadTargetSlot {
			state.Frames[0].SelectedKey = job.Origin.RowKey
		}
		if event.Operation != "start" {
			reconcileRootSelection(&state, event.Address)
		}
	}
	return state
}
func updateOperationProgress(state MenuState, attempt uint64, detail string) MenuState {
	key, job, ok := menuOperationForAttempt(state, attempt)
	if !ok {
		return state
	}
	state.Operations = cloneMenuOperations(state.Operations)
	job.Detail = detail
	state.Operations[key] = job
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.OperationAttempt == attempt {
		state.Notice.Text = menuOperationProgressText(state, job.Origin.Operation, job.Origin.Address) + " " + detail
	}
	return state
}
func menuOperationRowText(job MenuOperation, now time.Time) string {
	detail := strings.TrimSpace(rowtext.Sanitize(job.Detail))
	switch job.Phase {
	case MenuOperationRunning:
		elapsed := time.Duration(0)
		if !job.StartedAt.IsZero() && now.After(job.StartedAt) {
			elapsed = now.Sub(job.StartedAt).Truncate(time.Second)
		}
		text := job.Origin.Operation + "… " + elapsed.String()
		if detail != "" {
			text += " · " + detail
		}
		return text
	case MenuOperationFailed, MenuOperationUnconfirmed:
		return "failed: " + detail
	case MenuOperationNeedsConfirmation:
		return "confirmation needed"
	default:
		return detail
	}
}

// menuOperationRows indexes result addresses once rather than scanning all jobs
// separately for every rendered row. Work is one pass over rows plus jobs.
func menuOperationRows(state MenuState) map[couchcore.ThreadRowKey]MenuOperation {
	byAddress := make(map[couchcore.ThreadAddress]couchcore.ThreadRowKey, len(menuRows(state)))
	for _, row := range menuRows(state) {
		if row.Address != (couchcore.ThreadAddress{}) {
			byAddress[row.Address] = menuRowKey(row)
		}
	}
	out := make(map[couchcore.ThreadRowKey]MenuOperation, len(state.Operations))
	for target, job := range state.Operations {
		key := target.RowKey
		if resolved, ok := byAddress[job.ResultAddress]; ok {
			key = resolved
		}
		if key != (couchcore.ThreadRowKey{}) {
			if previous, ok := out[key]; !ok || previous.Origin.Attempt < job.Origin.Attempt {
				out[key] = job
			}
		}
	}
	return out
}

func pruneMenuOperations(state MenuState) MenuState {
	if len(state.Operations) == 0 {
		return state
	}
	keys := make(map[couchcore.ThreadRowKey]bool, len(menuRows(state)))
	addresses := make(map[couchcore.ThreadAddress]bool, len(menuRows(state)))
	for _, row := range menuRows(state) {
		keys[menuRowKey(row)] = true
		if row.Address != (couchcore.ThreadAddress{}) {
			addresses[row.Address] = true
		}
	}
	retained := make(map[menuOperationTarget]MenuOperation, len(state.Operations))
	for key, job := range state.Operations {
		if job.Phase == MenuOperationRunning || keys[key.RowKey] || addresses[job.ResultAddress] {
			retained[key] = job
		} else if job.Detail != "" {
			state.Notice = infoMenuNotice(fmt.Sprintf("%s/%s: %s", job.Origin.Address.RepoScope, job.Origin.Address.Tag, job.Detail))
		}
	}
	state.Operations = retained
	return state
}
