package couchtty

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

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
	DisplayLabel  string
	LandingFrame  uint64
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
	state.Operations = cloneMenuOperations(state.Operations)
	for key, current := range state.Operations {
		if current.Origin.Attempt == job.Origin.Attempt {
			current.LandingFrame = state.Frames[0].Instance
			state.Operations[key] = current
			break
		}
	}
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
	if address != (couchcore.ThreadAddress{}) && state.Reattach.isLoading(address) {
		state.Notice = errorMenuNotice("busy: attaching")
		return state, nil
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
	label := ""
	if row, exists := menuThreadTarget(state, target.RowKey, address); exists {
		label = boundMenuOperationText(row.Label())
	}
	state.Operations[target] = MenuOperation{Origin: origin, Phase: MenuOperationRunning, StartedAt: state.ObservedAt, DisplayLabel: label, LandingFrame: state.Frames[0].Instance}
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
	if state.LandingAttempt == job.Origin.Attempt && (len(state.Frames) != 1 || state.Frames[0].Instance != job.LandingFrame) {
		state.LandingAttempt = 0
	}
	job.ResultAddress = event.Address
	job.Phase = MenuOperationSucceeded
	job.Detail = event.Operation + " complete"
	if !event.Success {
		job.Phase = MenuOperationFailed
		if event.Unconfirmed {
			job.Phase = MenuOperationUnconfirmed
		}
		job.Detail = boundMenuOperationText(event.Error)
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
	if event.Success && state.LandingAttempt == job.Origin.Attempt && len(state.Frames) == 1 && state.Frames[0].Instance == job.LandingFrame {
		state.Frames = append([]MenuFrame(nil), state.Frames...)
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
	job.Detail = boundMenuOperationText(detail)
	state.Operations[key] = job
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.OperationAttempt == attempt {
		state.Notice.Text = menuOperationProgressText(state, job.Origin.Operation, job.Origin.Address) + " " + job.Detail
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
	case MenuOperationFailed:
		return "failed: " + detail
	case MenuOperationUnconfirmed:
		return "unconfirmed: " + detail
	case MenuOperationNeedsConfirmation:
		return "confirmation needed"
	default:
		return detail
	}
}

// menuOperationWork counts actual projection/pruning visits for deterministic
// hot-path regressions. Production callers pass nil; it carries no menu state.
type menuOperationWork struct{ Rows, Jobs int }

// menuOperationRows indexes result addresses once rather than scanning all jobs
// separately for every rendered row. Work is one pass over rows plus jobs.
func menuOperationRows(state MenuState) map[couchcore.ThreadRowKey]MenuOperation {
	return menuOperationRowsMeasured(state, nil)
}

func menuOperationRowsMeasured(state MenuState, work *menuOperationWork) map[couchcore.ThreadRowKey]MenuOperation {
	rows := menuRows(state)
	byAddress := make(map[couchcore.ThreadAddress]couchcore.ThreadRowKey, len(rows))
	for _, row := range rows {
		if work != nil {
			work.Rows++
		}
		if row.Address != (couchcore.ThreadAddress{}) {
			byAddress[row.Address] = menuRowKey(row)
		}
	}
	out := make(map[couchcore.ThreadRowKey]MenuOperation, len(state.Operations))
	for target, job := range state.Operations {
		if work != nil {
			work.Jobs++
		}
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
	return pruneMenuOperationsMeasured(state, nil)
}

func pruneMenuOperationsMeasured(state MenuState, work *menuOperationWork) MenuState {
	if len(state.Operations) == 0 {
		return state
	}
	rows := menuRows(state)
	keys := make(map[couchcore.ThreadRowKey]bool, len(rows))
	addresses := make(map[couchcore.ThreadAddress]couchcore.ThreadRowKey, len(rows))
	for _, row := range rows {
		if work != nil {
			work.Rows++
		}
		key := menuRowKey(row)
		keys[key] = true
		if row.Address != (couchcore.ThreadAddress{}) {
			addresses[row.Address] = key
		}
	}
	retained := make(map[menuOperationTarget]MenuOperation, len(state.Operations))
	var missing MenuOperation
	var missingTarget menuOperationTarget
	missingFound := false
	for key, job := range state.Operations {
		if work != nil {
			work.Jobs++
		}
		if job.Phase == MenuOperationRunning {
			retained[key] = job
			continue
		}
		canonical := key.RowKey
		if !keys[canonical] {
			canonical = addresses[job.ResultAddress]
		}
		if canonical != (couchcore.ThreadRowKey{}) {
			target := menuOperationTarget{RowKey: canonical}
			previous, exists := retained[target]
			if !exists || previous.Phase != MenuOperationRunning && previous.Origin.Attempt < job.Origin.Attempt {
				retained[target] = job
			}
		} else if job.Detail != "" && (!missingFound || missing.Origin.Attempt < job.Origin.Attempt) {
			missing, missingTarget, missingFound = job, key, true
		}
	}
	if missingFound {
		detail := menuOperationTargetLabel(missingTarget, missing) + ": " + missing.Detail
		if missing.Phase == MenuOperationFailed || missing.Phase == MenuOperationUnconfirmed {
			state.Notice = errorMenuNotice(detail)
		} else {
			state.Notice = infoMenuNotice(detail)
		}
	}
	state.Operations = retained
	return state
}

func menuOperationTargetLabel(target menuOperationTarget, job MenuOperation) string {
	identity := target.StartPath
	if identity == "" {
		identity = target.RowKey.SlotPath
	}
	if identity == "" && job.Origin.Address != (couchcore.ThreadAddress{}) {
		identity = fmt.Sprintf("%s/%s", job.Origin.Address.RepoScope, job.Origin.Address.Tag)
	}
	if target.Couch {
		identity = "Couch"
	}
	if job.DisplayLabel != "" && job.DisplayLabel != identity {
		return job.DisplayLabel + " [" + identity + "]"
	}
	return identity
}

// selectedMenuOperationDetail keeps Ctrl+Y useful after navigating away from a
// global notice: the selected row owns its retained full diagnostic.
func selectedMenuOperationDetail(state MenuState) (string, bool) {
	if len(state.Frames) == 0 {
		return "", false
	}
	frame := state.CurrentFrame()
	key := frame.SelectedKey
	if menuFrameBindsThread(frame) {
		key = frame.RowKey
	}
	row, ok := menuThreadTarget(state, key, state.SelectedThreadAddress())
	if !ok {
		return "", false
	}
	job, ok := menuOperationForRow(state, row)
	if !ok || job.Phase != MenuOperationFailed && job.Phase != MenuOperationUnconfirmed {
		return "", false
	}
	return job.Detail, job.Detail != ""
}

func boundMenuOperationText(text string) string {
	text = strings.ToValidUTF8(text, "�")
	if len(text) <= menuTextLimit {
		return text
	}
	end := menuTextLimit - len("…")
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "…"
}
