package couchtty

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
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
	Origin         MenuOperationOrigin
	Phase          MenuOperationPhase
	StartedAt      time.Time
	StageStartedAt time.Time
	DisplayLabel   string
	LandingFrame   uint64
	Detail         string
	ResultAddress  couchcore.ThreadAddress
	Recovery       *couchcore.RecoveryConfirmationOffer
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

// menuOperationIdentity derives aliases without changing a reservation or its
// immutable origin. Scope aliases belong only to running primary creation or
// replacement; ordinary actions otherwise retain exact-address ownership.
type menuOperationIdentity struct {
	RowKey       couchcore.ThreadRowKey
	Address      couchcore.ThreadAddress
	SlotPath     string
	PrimaryScope string
}

func menuOperationIdentityFor(target menuOperationTarget, job MenuOperation) menuOperationIdentity {
	identity := menuOperationIdentity{RowKey: target.RowKey, Address: job.ResultAddress}
	if target.RowKey.Kind == couchcore.ThreadTargetSlot {
		identity.SlotPath = target.RowKey.SlotPath
	}
	if job.Phase != MenuOperationRunning {
		return identity
	}
	if target.StartPath != "" {
		identity.SlotPath = target.StartPath
		if scope, err := launcher.ResolveRepoScope(target.StartPath); err == nil {
			identity.PrimaryScope = scope.Key
		}
	} else if target.RowKey.Kind == couchcore.ThreadTargetOrdinary && menuOperationReplacesAddress(job.Origin) {
		identity.PrimaryScope = target.RowKey.Address.RepoScope
	}
	return identity
}

func menuOperationIdentitiesOverlap(a, b menuOperationIdentity) bool {
	return a.RowKey != (couchcore.ThreadRowKey{}) && a.RowKey == b.RowKey ||
		a.Address != (couchcore.ThreadAddress{}) && (a.Address == b.Address || a.Address == b.RowKey.Address) ||
		b.Address != (couchcore.ThreadAddress{}) && b.Address == a.RowKey.Address ||
		a.SlotPath != "" && a.SlotPath == b.SlotPath ||
		a.PrimaryScope != "" && (a.PrimaryScope == b.PrimaryScope || b.RowKey.Kind == couchcore.ThreadTargetOrdinary && a.PrimaryScope == b.RowKey.Address.RepoScope) ||
		b.PrimaryScope != "" && a.RowKey.Kind == couchcore.ThreadTargetOrdinary && b.PrimaryScope == a.RowKey.Address.RepoScope
}

func preferMenuOperation(candidate, previous MenuOperation) bool {
	if previous.Phase == 0 {
		return true
	}
	if (candidate.Phase == MenuOperationRunning) != (previous.Phase == MenuOperationRunning) {
		return candidate.Phase == MenuOperationRunning
	}
	return candidate.Origin.Attempt > previous.Origin.Attempt
}

func menuOperationForRow(state MenuState, row couchcore.ActionableThreadSummary) (MenuOperation, bool) {
	identity := menuOperationIdentityFor(menuOperationTarget{RowKey: menuRowKey(row)}, MenuOperation{ResultAddress: row.Address})
	var found MenuOperation
	for target, job := range state.Operations {
		if menuOperationIdentitiesOverlap(menuOperationIdentityFor(target, job), identity) && preferMenuOperation(job, found) {
			found = job
		}
	}
	return found, found.Phase != 0
}
func menuRunningForAddress(state MenuState, address couchcore.ThreadAddress) (MenuOperation, bool) {
	if address == (couchcore.ThreadAddress{}) {
		return MenuOperation{}, false
	}
	row, ok := menuThread(state, address)
	if !ok {
		row = couchcore.ActionableThreadSummary{Address: address}
	}
	job, exists := menuOperationForRow(state, row)
	return job, exists && job.Phase == MenuOperationRunning
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
	candidate := menuOperationIdentityFor(target, MenuOperation{Origin: MenuOperationOrigin{Operation: effect.Operation, Address: address, RowKey: target.RowKey}, Phase: MenuOperationRunning})
	for key, job := range state.Operations {
		if job.Phase == MenuOperationRunning && (key.Couch || target.Couch || menuOperationIdentitiesOverlap(menuOperationIdentityFor(key, job), candidate)) {
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
func updateOperationProgress(state MenuState, attempt uint64, detail string, now time.Time) MenuState {
	key, job, ok := menuOperationForAttempt(state, attempt)
	if !ok {
		return state
	}
	state.Operations = cloneMenuOperations(state.Operations)
	detail = boundMenuOperationText(detail)
	if detail != job.Detail {
		job.StageStartedAt = now
	}
	job.Detail = detail
	state.Operations[key] = job
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.OperationAttempt == attempt {
		state.Notice.Text = menuOperationProgressText(state, job.Origin.Operation, job.Origin.Address) + " " + job.Detail
	}
	return state
}

// The renderer supplies its clock; no timer or IO belongs to this projection.
func menuOperationElapsed(job MenuOperation, now time.Time) time.Duration {
	started := job.StartedAt
	if !job.StageStartedAt.IsZero() {
		started = job.StageStartedAt
	}
	if started.IsZero() || !now.After(started) {
		return 0
	}
	return now.Sub(started).Truncate(time.Second)
}

func menuOperationRowText(job MenuOperation, now time.Time) string {
	detail := strings.TrimSpace(rowtext.Sanitize(job.Detail))
	switch job.Phase {
	case MenuOperationRunning:
		elapsed := menuOperationElapsed(job, now)
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

// menuOperationIndex is a temporary projection, discarded after each view.
// Its fixed number of map operations per job/row keeps aliases linear even
// when several visible ordinary rows share an owned primary scope.
type menuOperationIndex struct {
	byKey     map[couchcore.ThreadRowKey]MenuOperation
	byAddress map[couchcore.ThreadAddress]MenuOperation
	bySlot    map[string]MenuOperation
	byScope   map[string]MenuOperation
}

func (index menuOperationIndex) add(target menuOperationTarget, job MenuOperation) {
	identity := menuOperationIdentityFor(target, job)
	if identity.RowKey != (couchcore.ThreadRowKey{}) && preferMenuOperation(job, index.byKey[identity.RowKey]) {
		index.byKey[identity.RowKey] = job
	}
	if identity.Address != (couchcore.ThreadAddress{}) && preferMenuOperation(job, index.byAddress[identity.Address]) {
		index.byAddress[identity.Address] = job
	}
	if identity.SlotPath != "" && preferMenuOperation(job, index.bySlot[identity.SlotPath]) {
		index.bySlot[identity.SlotPath] = job
	}
	if identity.PrimaryScope != "" && preferMenuOperation(job, index.byScope[identity.PrimaryScope]) {
		index.byScope[identity.PrimaryScope] = job
	}
}

func (index menuOperationIndex) forRow(row couchcore.ActionableThreadSummary) MenuOperation {
	key := menuRowKey(row)
	job := index.byKey[key]
	if candidate := index.byAddress[row.Address]; candidate.Phase != 0 && preferMenuOperation(candidate, job) {
		job = candidate
	}
	if key.Kind == couchcore.ThreadTargetSlot {
		if candidate := index.bySlot[key.SlotPath]; candidate.Phase != 0 && preferMenuOperation(candidate, job) {
			job = candidate
		}
	} else if key.Kind == couchcore.ThreadTargetOrdinary {
		if candidate := index.byScope[key.Address.RepoScope]; candidate.Phase != 0 && preferMenuOperation(candidate, job) {
			job = candidate
		}
	}
	return job
}

func menuOperationRowsMeasured(state MenuState, work *menuOperationWork) map[couchcore.ThreadRowKey]MenuOperation {
	rows := menuRows(state)
	index := menuOperationIndex{byKey: make(map[couchcore.ThreadRowKey]MenuOperation), byAddress: make(map[couchcore.ThreadAddress]MenuOperation), bySlot: make(map[string]MenuOperation), byScope: make(map[string]MenuOperation)}
	for target, job := range state.Operations {
		if work != nil {
			work.Jobs++
		}
		index.add(target, job)
	}
	out := make(map[couchcore.ThreadRowKey]MenuOperation, len(rows))
	for _, row := range rows {
		if work != nil {
			work.Rows++
		}
		if job := index.forRow(row); job.Phase != 0 {
			out[menuRowKey(row)] = job
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
