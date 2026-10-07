package couchtty

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

// MenuControl is one operator-entered switcher surface. README checks consume
// this inventory so a new key cannot ship undocumented.
type MenuControl struct {
	Keys   string
	Action string
}

var menuControls = []MenuControl{
	{Keys: "typeahead", Action: "filter"},
	{Keys: "Space", Action: "toggle normal/focus view when filter is empty"},
	{Keys: "↑↓", Action: "select"},
	{Keys: "Enter", Action: "switch/resume"},
	{Keys: "Tab", Action: "actions"},
	{Keys: "Left", Action: "back"},
	{Keys: "Right", Action: "forward"},
	{Keys: "Ctrl-Space", Action: "start"},
	{Keys: "Ctrl-Backspace", Action: "previous"},
	{Keys: "Ctrl-Return", Action: "jump to the newest page, from an actor"},
	{Keys: "Alt+d", Action: "detach this thread · all + leave couch here"},
	{Keys: "Alt+x", Action: "park this thread · all + leave couch here"},
	{Keys: "click", Action: "an actor's chip or row switches to it · empty space does nothing"},
	{Keys: "Alt+n", Action: "relaunch: new Pair binary, same conversation (Ctrl+Alt+n aliases it)"},
	{Keys: "Escape", Action: "clear/back"},
	{Keys: "Tab → reboot", Action: "archive this conversation and start a fresh agent"},
}

// MenuControls returns the shared, immutable-by-copy key inventory.
func MenuControls() []MenuControl {
	return append([]MenuControl(nil), menuControls...)
}

const (
	menuFilterLimit = 1024
	menuNameLimit   = 1024
	menuTextLimit   = 4096
)

// MenuFrameKind is a fail-safe frame vocabulary; zero is never interactive.
type MenuFrameKind uint8

const (
	MenuFrameUnknown MenuFrameKind = iota
	MenuFrameRoot
	MenuFrameActions
	MenuFrameConfirmation
	MenuFrameText
	MenuFrameStart
	MenuFrameSwitchAgent
)

// MenuRootView is the root list mode, retained for the owning console lifetime.
type MenuRootView uint8

const (
	MenuViewNormal MenuRootView = iota
	MenuViewFocus
)

// MenuFrame owns the navigation state for exactly one menu level.
type MenuFrame struct {
	SwitchPrepared *couchcore.PreparedAgentSwitch
	SwitchStage    int
	SwitchEdited   bool
	// Rune distance from the end; zero keeps newly prefilled parameters at end.
	SwitchCursorFromEnd int

	Instance        uint64
	Kind            MenuFrameKind
	View            MenuRootView
	Filter          string
	SelectedKey     couchcore.ThreadRowKey
	RowKey          couchcore.ThreadRowKey
	SelectedAddress couchcore.ThreadAddress
	SelectedItem    string
	Thread          couchcore.ThreadAddress
	Action          string
	// Mode qualifies Action for a confirmation whose verb is not enough on its
	// own: a `leave` frame has to remember whether the operator asked to detach
	// every thread or to park them, because the two differ by every agent.
	Mode            string
	Input           string
	FormField       MenuFormField
	Path            string
	Agent           string
	AgentSticky     bool
	Generation      uint64
	PreviewPending  uint64
	PreviewAccepted uint64
	// PreviewResolution is the accepted preview ITSELF, not a hand-copied
	// shadow of its fields. The frame used to mirror path/agent/sources into
	// separate members and rebuild the commit arguments from them, which is
	// how a call site could silently commit something the operator never
	// previewed (pair#170 M4).
	PreviewResolution    couchcore.StartResolution
	SubmitGeneration     uint64
	CompletionRequest    CompletionIdentity
	CompletionPath       string
	CompletionPending    bool
	CompletionCandidates []string
	CompletionSelected   int
	CompletionTruncated  bool
}

type MenuFormField uint8

const (
	MenuFieldUnknown MenuFormField = iota
	MenuFieldPath
	MenuFieldAgent
)

type MenuNoticeLevel uint8

const (
	MenuNoticeUnknown MenuNoticeLevel = iota
	MenuNoticeInfo
	MenuNoticeProgress
	MenuNoticeError
)

type MenuNotice struct {
	Level MenuNoticeLevel
	Text  string
	Owner MenuProgressOwner
}

type MenuProgressOwner struct {
	PreviewGeneration uint64
	OperationAttempt  uint64
	Completion        CompletionIdentity
}

func infoMenuNotice(text string) MenuNotice { return MenuNotice{Level: MenuNoticeInfo, Text: text} }
func progressMenuNotice(text string) MenuNotice {
	return MenuNotice{Level: MenuNoticeProgress, Text: text}
}
func errorMenuNotice(text string) MenuNotice { return MenuNotice{Level: MenuNoticeError, Text: text} }

// setBookkeepingNotice writes a frame-validity message WITHOUT erasing an
// operation's own result.
//
// A refresh discarding a frame is bookkeeping, and "thread action is no longer
// applicable" tells the operator nothing they can act on. An operation's failure
// notice does: `park-ok-resume-failed`'s entire value is the sentence saying
// Enter resumes the thread. The two collided precisely on relaunch, because its
// park makes the thread non-live and so invalidates the very frame whose
// operation just failed -- so the next refresh replaced the recovery
// instructions with bookkeeping, on the one outcome that most needs them.
func setBookkeepingNotice(state *MenuState, text string) {
	if state.Notice.Level == MenuNoticeError && state.Notice.Owner.OperationAttempt != 0 {
		return
	}
	state.Notice = errorMenuNotice(text)
}

// MenuState is immutable-by-copy reducer state. Frames retain identities and
// text; the inventory remains one separately owned slice.
type MenuState struct {
	Orientation       map[couchcore.ThreadAddress]orientation.Request
	Inventory         []couchcore.ActionableThreadSummary
	InventoryReady    bool
	RefreshPending    bool
	ProjectionPending bool
	// ProjectionAfterGeneration is the newest refresh already admitted when a
	// mutation committed. Only a later snapshot can represent that mutation.
	ProjectionAfterGeneration uint64
	Frames                    []MenuFrame
	ActiveAddress             couchcore.ThreadAddress
	Agents                    []string
	RootAgent                 string
	Attention                 map[couchcore.ThreadAddress][]AttentionMessage
	InFlight                  MenuOperationOrigin
	// Reattach is couch's own background reattach pass (pair#206). It lives in
	// MenuState so the one transition authority owns every interleaving with
	// the operator's own operations.
	Reattach           ReattachPass
	PreviewSequence    uint64
	CompletionSequence uint64
	OperationSequence  uint64
	FrameSequence      uint64
	SpinnerPhase       uint8
	Notice             MenuNotice
	// SlotGit is the last slot git observation per checkout path (pair#317),
	// rebuilt over each refresh's probe set. Display evidence only.
	SlotGit map[string]couchcore.SlotGitStatus
	// Activity is each live thread's last observed activity (pair#247): its
	// operator's input or its agent's work. A thread with no entry has not been
	// probed yet and draws unfaded. Display evidence only.
	Activity map[couchcore.ThreadAddress]time.Time
	// Palette is what the host terminal reported about its colours, which idle
	// fading blends toward (pair#247). One copy, read by both renderers.
	Palette Palette
}

// MenuOperationOrigin captures the exact frame that emitted asynchronous
// work, so completion does not depend on whichever frame is visible later.
type MenuOperationOrigin struct {
	RowKey      couchcore.ThreadRowKey
	PanelOrigin bool
	// ContinuationID correlates durable automatic work without routing it
	// through the reattachment pass. PreserveFocus controls child adoption.
	ContinuationID string
	PreserveFocus  bool
	// Manual suppresses the attention capture, so the landing is classified
	// arrivalOrdinary even on a paging actor. A click is always a manual switch;
	// Enter on a paging actor is not.
	Manual bool
	// Background marks an attempt couch dispatched on its own, not the
	// operator: the reattach pass (pair#206). It never occupies InFlight, and
	// its completion never takes focus.
	Background bool

	Operation        string
	Attempt          uint64
	Address          couchcore.ThreadAddress
	FrameInstance    uint64
	FrameKind        MenuFrameKind
	Depth            int
	AttentionCapture AttentionCapture
}

type MenuEventKind uint8

const (
	MenuEventUnknown MenuEventKind = iota
	MenuEventKey
	MenuEventRefreshStarted
	MenuEventInventory
	MenuEventOperationResult
	MenuEventPreviewResult
	MenuEventCompletionResult
	MenuEventParkHotkey
	MenuEventTick
	// MenuEventMouseSwitch is a click on an actor. It dispatches the SAME
	// declared `switch` operation Enter dispatches; the only difference is that
	// it is always MANUAL, so ctrl+backspace undoes it even when the clicked
	// actor was paging. Enter on a paging actor is a notification hop and
	// therefore non-pinning, which is the one input where the two legitimately
	// differ (pair#172).
	MenuEventMouseSwitch

	// MenuEventNotice reports a console-side refusal on the menu's own surface.
	// The status row is behind the panel while the switcher owns the screen, so
	// a refusal sent there would read to the operator as the key doing nothing.
	MenuEventNotice
	// MenuEventReattachArm arms the background reattach pass with the startup
	// root it must never reattach (pair#206).
	MenuEventReattachArm
	// MenuEventSlotGit lands one slot git refresh pass (pair#317).
	MenuEventSlotGit
	// MenuEventActivity lands one idle-fading activity pass (pair#247).
	MenuEventActivity
	// MenuEventPalette records a colour the host terminal reported (pair#247).
	MenuEventPalette
)

type MenuEvent struct {
	RowKey       couchcore.ThreadRowKey
	Kind         MenuEventKind
	Key          PanelKey
	Address      couchcore.ThreadAddress
	Inventory    []couchcore.ActionableThreadSummary
	InventorySet bool
	Operation    string
	// Mode is the operation's disposition where it has one: which of park or
	// detach a whole-couch `leave` applies to every live thread.
	Mode       string
	Attempt    uint64
	Success    bool
	Error      string
	Generation uint64
	// ProjectionAfterGeneration records the newest inventory generation that
	// predates a committed operation mutation.
	ProjectionAfterGeneration uint64
	SwitchPrepared            *couchcore.PreparedAgentSwitch
	Prepared                  *couchcore.PreparedStart
	Completion                *CompletionResult
	// Background marks the completion of a reattach-pass attempt (pair#206),
	// which is routed to the pass and never to the operator's in-flight slot.
	Background bool
	// Diagnostic is ResumeDiagnosticOf(err) for an operation result, so the pass
	// tells a skip from a failure by code rather than by matching error text.
	Diagnostic couchcore.ResumeDiagnosticCode
	// SlotGit and SlotGitFailed are one slot git pass: the paths it observed and
	// the paths whose probe failed. Together they are the pass's probe set.
	SlotGit       map[string]couchcore.SlotGitStatus
	SlotGitFailed map[string]bool
	// Activity and ActivityFailed are one activity pass (pair#247), shaped
	// like SlotGit: the threads observed and the threads whose probe failed.
	Activity       map[couchcore.ThreadAddress]time.Time
	ActivityFailed map[couchcore.ThreadAddress]bool
	// Palette is the host terminal's colours as known after one reply.
	Palette Palette
}

// MenuEffect is an operation request for the thin Console shell.
type MenuEffect struct {
	CopyOrientation *orientation.Request
	Operation       string
	Attempt         uint64
	Args            map[string]string
	Preview         *PreviewRequest
	Completion      *CompletionRequest
	// Background marks a reattach-pass effect (pair#206): enqueued without the
	// operator's in-flight slot, and attached without taking focus.
	Background bool
}

func NewMenuState(inventory []couchcore.ActionableThreadSummary, active couchcore.ThreadAddress) MenuState {
	owned := orderedMenuInventory(inventory)
	root := MenuFrame{Instance: 1, Kind: MenuFrameRoot}
	if len(owned) > 0 {
		selectMenuRow(&root, owned[0])
	}
	return MenuState{Inventory: owned, InventoryReady: inventory != nil, Frames: []MenuFrame{root}, ActiveAddress: active, FrameSequence: 1}
}

func (s MenuState) CurrentFrame() MenuFrame {
	if len(s.Frames) == 0 {
		return MenuFrame{}
	}
	return s.Frames[len(s.Frames)-1]
}

// VisibleMenuThreads applies the same exact-over-fuzzy field rule as operation
// resolution, but only to the already-present in-memory inventory.
func VisibleMenuThreads(state MenuState) []couchcore.ActionableThreadSummary {
	frame := state.CurrentFrame()
	if frame.Kind != MenuFrameRoot {
		return nil
	}
	return visibleMenuRows(state, frame)
}

func visibleRootThreads(inventory []couchcore.ActionableThreadSummary, frame MenuFrame) []couchcore.ActionableThreadSummary {
	if frame.Filter == "" {
		return append([]couchcore.ActionableThreadSummary(nil), inventory...)
	}
	ref, workspaceRef, refErr := couchcore.ParseWorkspaceReference(frame.Filter)
	var exact, fuzzy []couchcore.ActionableThreadSummary
	for _, row := range inventory {
		slotRow := row.Target.Kind == couchcore.ThreadTargetSlot
		if workspaceRef {
			slot := row.Target.Slot
			// A filter lists every candidate, so a repository prefix or alias
			// prefix narrows without the uniqueness a resolved reference needs.
			if slotRow && refErr == nil && ref.Number == slot.Number && (ref.Repo == "" || strings.HasPrefix(slot.Repo, ref.Repo) || (row.RepositoryAlias != "" && strings.HasPrefix(row.RepositoryAlias, ref.Repo))) {
				exact = append(exact, row)
			}
			continue
		}
		match, err := couchcore.ClassifyThreadReferenceFields(couchcore.ThreadReferenceFields{
			Address: row.Address, Label: row.Label(), WorkingPath: row.WorkingPath, Summary: menuFocusSummary(row),
		}, frame.Filter)
		if err != nil {
			continue
		}
		// Slot rows also accept partial tag text; exact tags still win across all rows.
		if slotRow && match == couchcore.ThreadReferenceNone && row.Address.Tag != "" && strings.Contains(strings.ToLower(string(row.Address.Tag)), strings.ToLower(frame.Filter)) {
			match = couchcore.ThreadReferenceFuzzy
		}
		switch match {
		case couchcore.ThreadReferenceExact:
			exact = append(exact, row)
		case couchcore.ThreadReferenceFuzzy:
			fuzzy = append(fuzzy, row)
		}
	}
	if len(exact) > 0 || workspaceRef {
		return exact
	}
	return fuzzy
}

// menuDescriptionMatches uses the same displayed text in either root view.
func menuDescriptionMatches(row couchcore.ActionableThreadSummary, query string) bool {
	query = strings.TrimSpace(query)
	return query != "" && strings.Contains(strings.ToLower(menuFocusSummary(row)), strings.ToLower(query))
}

// clearsPreviousNotice reports whether an event retires the message on screen.
//
// Only what the OPERATOR does retires it. A message answers the last thing they
// did and stands until they do the next thing; a background refresh is not the
// next thing. This condition used to name only the three result kinds, so a
// routine MenuEventInventory wiped the notice before anything else in the
// transition ran -- which erased the recovery instructions from a failed
// relaunch (`it is parked, Enter resumes it`) roughly one refresh later, and
// made the guard inside reconcileMenuFrames unreachable on the production path
// because the notice it was protecting had already been zeroed. A test that
// drives reconcileMenuFrames directly cannot see any of this; it only shows
// through ReduceMenu.
func clearsPreviousNotice(kind MenuEventKind) bool {
	switch kind {
	case MenuEventOperationResult, MenuEventPreviewResult, MenuEventCompletionResult,
		MenuEventInventory, MenuEventRefreshStarted, MenuEventTick:
		return false
	}
	return true
}

// ReduceMenu is the single total transition authority for menu input and
// asynchronous completions. The initial slice ownership is cloned before any
// transition so callers can retain prior states safely.
func ReduceMenu(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	next := cloneMenuState(state)
	if event.Kind == MenuEventTick {
		if menuProgressMatches(next.Notice, event) {
			next.SpinnerPhase = (next.SpinnerPhase + 1) % 4
		}
		return next, nil
	}
	if event.Kind == MenuEventSlotGit {
		next.SlotGit = mergeObservations(next.SlotGit, event.SlotGit, event.SlotGitFailed)
		return next, nil
	}
	if event.Kind == MenuEventActivity {
		next.Activity = mergeObservations(next.Activity, event.Activity, event.ActivityFailed)
		return next, nil
	}
	if event.Kind == MenuEventPalette {
		next.Palette = event.Palette
		return next, nil
	}
	if event.Kind == MenuEventOperationResult && event.Background {
		return advanceReattach(finishReattach(next, event))
	}
	if event.Kind == MenuEventOperationResult && !menuOperationMatches(next.InFlight, event) {
		return next, nil
	}
	if event.Kind == MenuEventCompletionResult && !completionResultMatches(next, event.Completion) {
		return next, nil
	}
	if next.Notice.Level != MenuNoticeProgress && clearsPreviousNotice(event.Kind) {
		next.Notice = MenuNotice{}
	}
	if event.Kind == MenuEventNotice {
		next.Notice = errorMenuNotice(event.Error)
		return next, nil
	}
	if event.Kind == MenuEventParkHotkey {
		return reduceParkHotkey(next, event)
	}
	if event.Kind == MenuEventMouseSwitch {
		thread, ok := menuThreadTarget(next, event.RowKey, event.Address)
		// A pending row is not ready: a click on it lands nowhere (pair#206).
		if !ok || enterOperationFor(thread) == "" || !menuRowSelectable(next, event.Address) {
			return next, nil
		}
		next.Frames = next.Frames[:1]
		selectMenuRow(&next.Frames[0], thread)
		state, effects := dispatchMenuRow(next, enterOperationFor(thread), thread)
		// Only mark a dispatch that HAPPENED. dispatchThreadOperation refuses
		// when another operation is in flight and returns the state unchanged;
		// marking that would leave Manual set on someone else's operation.
		if len(effects) > 0 {
			state.InFlight.Manual = true
		}
		return state, effects
	}
	if event.Kind == MenuEventReattachArm {
		return armReattach(next, event.Address), nil
	}
	if event.Kind == MenuEventRefreshStarted {
		next.RefreshPending = true
		if !next.InventoryReady && next.Notice.Level != MenuNoticeProgress {
			next.Notice = infoMenuNotice("thread inventory unavailable")
		}
		return next, nil
	}
	if event.Kind == MenuEventInventory {
		next.RefreshPending = false
		if event.Error != "" {
			next.Notice = errorMenuNotice("thread inventory unavailable: " + event.Error)
			// An armed pass stays armed until an inventory it can seed from
			// (cell 2); a running one carries on, since warm-only re-proves each
			// thread at attempt time (cell 4).
			return advanceReattach(next)
		}
		if !next.ProjectionPending || event.Generation > next.ProjectionAfterGeneration {
			next.ProjectionPending = false
			next.ProjectionAfterGeneration = 0
		}
		var previous []couchcore.ActionableThreadSummary
		next, previous = replaceMenuInventory(next, event.Inventory)
		next.InventoryReady = true
		// The pass seeds from the first inventory after arming, and an inventory
		// newer than an attach is authoritative for that thread again (pair#206
		// cells 3 and 11). Both BEFORE reconciling, so the selection already
		// skips rows that just became pending.
		next = seedReattach(next, event.Inventory)
		next = expireAttached(next, event.Generation)
		next = reconcileMenuFrames(next, previous)
		return advanceReattach(next)
	}
	if event.Kind == MenuEventOperationResult {
		if event.InventorySet {
			next.ProjectionPending = false
			next.ProjectionAfterGeneration = 0
			var previous []couchcore.ActionableThreadSummary
			next, previous = replaceMenuInventory(next, event.Inventory)
			next = reconcileMenuFrames(next, previous)
		}
		next = reduceOperationResult(next, event)
		// A successful leave ends the console, so it ends the pass: advancing
		// here would enqueue one more reattach only for Stop to cancel it, which
		// is the exact outcome cell 10 exists to prevent (pair#206). Otherwise
		// the operator's slot has just cleared, which is when a pass held behind
		// it may resume.
		if event.Operation == "leave" && event.Success {
			next.Reattach.Phase = ReattachDone
			return next, nil
		}
		return advanceReattach(next)
	}
	if event.Kind == MenuEventPreviewResult {
		return reducePreviewResult(next, event)
	}
	if event.Kind == MenuEventCompletionResult {
		return reduceCompletionResult(next, *event.Completion), nil
	}
	if len(next.Frames) == 0 || event.Kind != MenuEventKey {
		return next, nil
	}
	if event.Key.Kind == KeyCtrlSpace {
		return openStartForm(next)
	}
	switch next.CurrentFrame().Kind {
	case MenuFrameRoot:
		return reduceRootKey(next, event.Key)
	case MenuFrameActions:
		return reduceActionKey(next, event.Key)
	case MenuFrameConfirmation:
		return reduceConfirmationKey(next, event.Key)
	case MenuFrameText:
		return reduceTextKey(next, event.Key)
	case MenuFrameSwitchAgent:
		return reduceSwitchAgentKey(next, event.Key)
	case MenuFrameStart:
		return reduceStartKey(next, event.Key)
	default:
		return next, nil
	}
}

func menuProgressMatches(notice MenuNotice, event MenuEvent) bool {
	if notice.Level != MenuNoticeProgress {
		return false
	}
	if notice.Owner.OperationAttempt != 0 {
		return event.Attempt == notice.Owner.OperationAttempt && event.Generation == 0
	}
	if notice.Owner.PreviewGeneration != 0 {
		return event.Generation == notice.Owner.PreviewGeneration && event.Attempt == 0
	}
	return false
}

func reduceRootKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyTab)
	frame := &state.Frames[len(state.Frames)-1]
	switch key.Kind {
	case KeyRune:
		if key.Rune == ' ' && frame.Filter == "" {
			if frame.View == MenuViewFocus {
				frame.View = MenuViewNormal
			} else {
				frame.View = MenuViewFocus
			}
			reconcileRootSelection(&state, frame.SelectedAddress)
			return state, nil
		}
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 {
			candidate := frame.Filter + string(key.Rune)
			if len(candidate) <= menuFilterLimit {
				frame.Filter = candidate
			}
		}
		reconcileRootSelection(&state, frame.SelectedAddress)
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileRootSelection(&state, frame.SelectedAddress)
	case KeyUp:
		moveRootSelection(&state, -1)
	case KeyDown:
		moveRootSelection(&state, 1)
	case KeyEnter:
		thread, ok := selectedMenuThread(state)
		if !ok {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		// A row that cannot be acted on says why. Silence is what the operator
		// reports as a bug, and dispatching a switch the console will refuse
		// ("not attached to this console") reports the wrong layer -- the
		// console's problem, not the thread's.
		operation := enterOperationFor(thread)
		if operation == "" {
			state.Notice = errorMenuNotice(enterRefusalNotice(thread))
			return state, nil
		}
		return dispatchMenuRow(state, operation, thread)
	case KeyTab:
		thread, ok := selectedMenuThread(state)
		if !ok {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		items := menuActionsFor(state, thread)
		if len(items) == 0 {
			// Busy, unknown and continuation-pending rows offer nothing; an
			// empty action list would be a screen with no way forward.
			state.Notice = errorMenuNotice(thread.Label() + ": " + unusableThreadNotice(thread) + " · no actions")
			return state, nil
		}
		appendMenuFrame(&state, MenuFrame{
			Kind: MenuFrameActions, RowKey: menuRowKey(thread), Thread: thread.Address, SelectedItem: items[0],
		})
	case KeyEscape:
		if frame.Filter != "" {
			selected := frame.SelectedAddress
			frame.Filter = ""
			reconcileRootSelection(&state, selected)
			return state, nil
		}
		active, ok := menuThread(state, state.ActiveAddress)
		if !ok || !active.Live() {
			state.Notice = errorMenuNotice("no live thread can receive focus")
			return state, nil
		}
		return dispatchThreadOperation(state, "switch", active.Address)
	}
	return state, nil
}

// menuFrameBindsThread reports whether a frame's validity depends on a durable
// thread still being live and visible.
//
// Almost every frame does: an actions list, a park confirmation and an alias
// prompt are all *about* one thread, and must vanish when it does. The `leave`
// confirmation is the exception -- it is about couch itself. It used to ride
// the root actor's address, so every thread-bound check passed by accident;
// with the root actor gone (#170) it carries no address, and an all-detached
// couch (the normal resting state once leave detaches rather than parks) has no
// live thread to borrow one from.
//
// One predicate rather than a `leave` exception at each of the five sites that
// resolve a frame's thread: the sites ask about frame SCOPE, which is a real
// property, instead of each re-deriving what leave means.
func menuFrameBindsThread(frame MenuFrame) bool {
	if frame.Kind == MenuFrameConfirmation && frame.Action == "leave" {
		return false
	}
	return frame.Kind == MenuFrameSwitchAgent || frame.Kind == MenuFrameActions || frame.Kind == MenuFrameConfirmation || frame.Kind == MenuFrameText
}

// reduceParkHotkey handles the Alt+x/Alt+d ownership-boundary chords.
//
// It returns effects because detach dispatches IMMEDIATELY -- it needs no
// confirmation, so there is no later Enter to carry the effect. Park and leave
// still return none: they only open a confirmation frame, and the effect comes
// when that frame is confirmed.
func reduceParkHotkey(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	switch event.Operation {
	case "park", "detach", "leave", "relaunch":
	default:
		state.Notice = errorMenuNotice("action is unavailable")
		return state, nil
	}
	if event.Operation == "leave" {
		// The whole-couch scope. The key already chose the disposition, so the
		// only thing left to decide is confirmation -- and that rides the
		// disposition, not the scope: park stops agents at either scope and is
		// confirmed at either, detach stops nothing and is confirmed at
		// neither.
		state.Frames = state.Frames[:1]
		switch couchcore.LeaveDisposition(event.Mode) {
		case couchcore.LeaveDetach:
			return dispatchMenuOperation(state, leaveEffect(couchcore.LeaveDetach), couchcore.ThreadAddress{})
		case couchcore.LeavePark:
			appendMenuFrame(&state, MenuFrame{
				Kind: MenuFrameConfirmation, Action: "leave", Mode: event.Mode, SelectedItem: "cancel",
			})
			return state, nil
		default:
			state.Notice = errorMenuNotice("action is unavailable")
			return state, nil
		}
	}
	if event.Operation == "detach" {
		thread, ok := menuThread(state, event.Address)
		if !ok || !thread.Live() {
			state.Notice = errorMenuNotice("active thread is no longer actionable")
			return state, nil
		}
		state.Frames = state.Frames[:1]
		state.Frames[0].SelectedAddress = event.Address
		return dispatchThreadOperation(state, "detach", event.Address)
	}
	if event.Operation == "park" || event.Operation == "relaunch" {
		thread, ok := menuThread(state, event.Address)
		if !ok || !thread.Live() {
			state.Notice = errorMenuNotice("only a running thread can be " + pastParticiple(event.Operation))
			return state, nil
		}
		// The chord asks the same table the switcher's Tab list does; its
		// confirmation is re-checked against it on Enter, so opening one the
		// row does not offer would only end in "no longer applicable". A
		// live row lacks park or relaunch only while a request is unfinished.
		if !containsMenuItem(menuActionItems(thread), event.Operation) {
			state.Notice = errorMenuNotice(thread.Label() + ": " + event.Operation + " is not offered while its continuation is unfinished")
			return state, nil
		}
	}
	state.Frames = state.Frames[:1]
	state.Frames[0].SelectedAddress = event.Address
	if !appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameConfirmation, Action: event.Operation, Thread: event.Address, SelectedItem: "cancel",
	}) {
		return state, nil
	}
	return state, nil
}

// pastParticiple is the operator-facing verb form, with a FALLBACK.
//
// It was an inline two-entry map indexed by the operation, so a third operation
// joining the guard above -- two lines away -- would have produced "only a
// running thread can be " and nothing else. A lookup that can silently yield an
// empty word is worse than a clumsy one.
func pastParticiple(operation string) string {
	switch operation {
	case "park":
		return "parked"
	case "relaunch":
		return "relaunched"
	}
	return operation + "ed"
}

// enterOperationFor is what landing on a row DOES: a live row switches, and a
// row that offers resume resumes -- parked, detached, or unusable where resume
// has a route (a slot, a recoverable survivor, a retained request). Every
// other row does nothing on Enter and says why. It reads the action table
// rather than restating it, so Enter can never land on an action the row does
// not offer.
//
// One authority, because a click must take Enter's rule rather than a restatement
// of it -- the restatement had already diverged on its first day (pair#172).
func enterOperationFor(thread couchcore.ActionableThreadSummary) string {
	if thread.Live() {
		return "switch"
	}
	if containsMenuItem(menuActionItems(thread), "resume") {
		return "resume"
	}
	return ""
}

// enterRefusalNotice is what Enter says on a row it will not act on: why, and
// -- when the row offers one -- the way forward (menuRowAdviceOf).
func enterRefusalNotice(thread couchcore.ActionableThreadSummary) string {
	notice := thread.Label() + ": " + unusableThreadNotice(thread)
	if next := menuRowAdviceOf(menuRowFactsOf(thread)).Enter.Text; next != "" {
		notice += " · " + next
	}
	return notice
}

func reduceActionKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyEnter)
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := menuThreadTarget(state, frame.RowKey, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	items := menuActionsFor(state, thread)
	switch key.Kind {
	case KeyRune:
		candidate := frame.Filter + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuFilterLimit {
			frame.Filter = candidate
		}
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyUp:
		moveItemSelection(frame, filterMenuItems(items, frame.Filter), -1)
	case KeyDown:
		moveItemSelection(frame, filterMenuItems(items, frame.Filter), 1)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if !containsMenuItem(items, frame.SelectedItem) {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		switch frame.SelectedItem {
		case "copy-orientation":
			request := state.Orientation[thread.Address]
			return state, []MenuEffect{{CopyOrientation: &request}}
		case "add-slot":
			path := menuAddSlotPath(thread)
			state, _ = openStartForm(state)
			if state.CurrentFrame().Kind != MenuFrameStart {
				return state, nil
			}
			form := &state.Frames[len(state.Frames)-1]
			form.Path, form.FormField = path, MenuFieldAgent
			return requestStartPreview(state)
		case "switch-agent":
			return openSwitchAgent(state, thread.Address)
		case "alias":
			// The genuine special case: it collects text before it can run,
			// which no declaration expresses.
			appendMenuFrame(&state, MenuFrame{
				Kind: MenuFrameText, RowKey: menuRowKey(thread), Thread: thread.Address, Action: frame.SelectedItem,
			})
		default:
			// Everything else asks the DECLARATION whether it confirms, instead
			// of being listed here. This switch used to name park and archive
			// (confirm) and detach and resume (dispatch); relaunch joined the
			// action list and neither arm, so Enter on it fell through the whole
			// switch and did nothing at all. Detach still destroys nothing and
			// still runs without a confirmation -- that asymmetry with park is
			// why both actions exist -- but it is now DECLARED, not remembered.
			confirms, declared := couchcore.OperationConfirms(frame.SelectedItem)
			if !declared {
				state.Notice = errorMenuNotice(frame.SelectedItem + " is not a declared operation")
				return state, nil
			}
			if confirms {
				appendMenuFrame(&state, MenuFrame{
					Kind: MenuFrameConfirmation, RowKey: menuRowKey(thread), Thread: thread.Address, Action: frame.SelectedItem, SelectedItem: "cancel",
				})
				return state, nil
			}
			return dispatchMenuRow(state, frame.SelectedItem, thread)
		}
	}
	return state, nil
}

func reduceConfirmationKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyEnter)
	frame := &state.Frames[len(state.Frames)-1]
	binds := menuFrameBindsThread(*frame)
	var thread couchcore.ActionableThreadSummary
	if binds {
		found, ok := menuThreadTarget(state, frame.RowKey, frame.Thread)
		if !ok {
			return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
		}
		thread = found
	}
	items := confirmationMenuItems(state, *frame)
	visible := filterMenuItems(items, frame.Filter)
	switch key.Kind {
	case KeyRune:
		candidate := frame.Filter + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuFilterLimit {
			frame.Filter = candidate
		}
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyUp:
		moveItemSelection(frame, visible, -1)
	case KeyDown:
		moveItemSelection(frame, visible, 1)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if !containsMenuItem(visible, frame.SelectedItem) {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		if frame.SelectedItem == "cancel" {
			state.Frames = state.Frames[:len(state.Frames)-1]
			return state, nil
		}
		confirms, _ := couchcore.OperationConfirms(frame.Action)
		if frame.SelectedItem != frame.Action || !confirms ||
			(binds && !menuFrameOperationInFlight(state, *frame) && !containsMenuItem(menuActionItems(thread), frame.Action)) {
			return discardThreadFrames(state, frame.Thread, "thread action is no longer applicable"), nil
		}
		if frame.Action == "leave" {
			return dispatchMenuOperation(state, leaveEffect(couchcore.LeaveDisposition(frame.Mode)), couchcore.ThreadAddress{})
		}
		return dispatchMenuRow(state, frame.Action, thread)
	}
	return state, nil
}

func reduceTextKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	if key.Kind == KeyLeft {
		key.Kind = KeyEscape
	}
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := menuThreadTarget(state, frame.RowKey, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	limit := menuTextLimit
	if frame.Action == "alias" {
		limit = menuNameLimit
	}
	switch key.Kind {
	case KeyRune:
		candidate := frame.Input + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= limit {
			frame.Input = candidate
		}
	case KeyBackspace:
		frame.Input = removeLastRune(frame.Input)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if frame.Action == "alias" {
			// The alias belongs to the repository, not the thread: address it by
			// primary root. An empty entry clears it.
			args := map[string]string{"ref": menuRepositoryRoot(thread)}
			if frame.Input == "" {
				args["clear"] = "true"
			} else {
				args["alias"] = frame.Input
			}
			return dispatchMenuOperation(state, MenuEffect{Operation: "alias", Args: args}, thread.Address)
		}
	}
	return state, nil
}

func openStartForm(state MenuState) (MenuState, []MenuEffect) {
	current := state.CurrentFrame().Kind
	if current == MenuFrameStart || current == MenuFrameText {
		return state, nil
	}
	if current != MenuFrameRoot && current != MenuFrameActions && current != MenuFrameConfirmation {
		return state, nil
	}
	agent := ""
	for _, candidate := range state.Agents {
		if candidate == state.RootAgent {
			agent = candidate
			break
		}
	}
	if agent == "" && len(state.Agents) > 0 {
		agent = state.Agents[0]
	}
	generation, ok := nextPreviewGeneration(&state)
	if !ok {
		return state, nil
	}
	if !appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameStart, FormField: MenuFieldPath, Agent: agent, Generation: generation,
	}) {
		return state, nil
	}
	return state, nil
}

func reduceStartKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	switch key.Kind {
	case KeyRune:
		if frame.FormField != MenuFieldPath || key.Rune == utf8.RuneError || !utf8.ValidRune(key.Rune) {
			return state, nil
		}
		candidate := frame.Path + string(key.Rune)
		if utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuTextLimit {
			invalidateStartCompletion(&state, frame)
			frame.Path = candidate
			invalidateStartPreview(&state, frame)
		}
	case KeyBackspace:
		if frame.FormField == MenuFieldPath {
			before := frame.Path
			frame.Path = removeLastRune(frame.Path)
			if frame.Path != before {
				invalidateStartCompletion(&state, frame)
				invalidateStartPreview(&state, frame)
			}
		}
	case KeyTab:
		if frame.FormField == MenuFieldPath {
			if len(frame.CompletionCandidates) > 0 {
				frame.CompletionSelected = (frame.CompletionSelected + 1) % len(frame.CompletionCandidates)
				return state, nil
			}
			return requestPathCompletion(state)
		}
	case KeyUp:
		if len(frame.CompletionCandidates) > 0 {
			frame.CompletionSelected = (frame.CompletionSelected - 1 + len(frame.CompletionCandidates)) % len(frame.CompletionCandidates)
		} else if frame.FormField == MenuFieldAgent {
			frame.FormField = MenuFieldPath
			invalidateStartCompletion(&state, frame)
		}
	case KeyDown:
		if len(frame.CompletionCandidates) > 0 {
			frame.CompletionSelected = (frame.CompletionSelected + 1) % len(frame.CompletionCandidates)
		} else if frame.FormField == MenuFieldPath {
			frame.FormField = MenuFieldAgent
			invalidateStartCompletion(&state, frame)
			return requestStartPreview(state)
		}
	case KeyLeft:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, -1) && invalidateStartPreview(&state, frame) {
				return requestStartPreview(state)
			}
		}
	case KeyRight:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, 1) && invalidateStartPreview(&state, frame) {
				return requestStartPreview(state)
			}
		}
	case KeyEnter:
		if len(frame.CompletionCandidates) > 0 {
			path := frame.CompletionCandidates[frame.CompletionSelected]
			invalidateStartCompletion(&state, frame)
			frame.Path = path
			invalidateStartPreview(&state, frame)
			return state, nil
		}
		if frame.PreviewAccepted == frame.Generation && frame.PreviewResolution.Fingerprint != "" {
			return dispatchMenuOperation(state, startMenuEffect(*frame), couchcore.ThreadAddress{})
		}
		frame.SubmitGeneration = frame.Generation
		state.SpinnerPhase = 0
		state.Notice = MenuNotice{
			Level: MenuNoticeProgress,
			Text:  "resolving",
			Owner: MenuProgressOwner{PreviewGeneration: frame.Generation},
		}
		if frame.PreviewPending == frame.Generation {
			return state, nil
		}
		return requestStartPreview(state)
	case KeyEscape:
		if len(frame.CompletionCandidates) > 0 {
			invalidateStartCompletion(&state, frame)
			return state, nil
		}
		if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.PreviewGeneration == frame.Generation {
			state.Notice = MenuNotice{}
		}
		state.Frames = state.Frames[:len(state.Frames)-1]
	}
	return state, nil
}

func selectStartAgent(frame *MenuFrame, agents []string, delta int) bool {
	if len(agents) == 0 {
		return false
	}
	index := 0
	for i, agent := range agents {
		if agent == frame.Agent {
			index = i
			break
		}
	}
	index = (index + delta + len(agents)) % len(agents)
	if frame.Agent != agents[index] || !frame.AgentSticky {
		frame.Agent = agents[index]
		frame.AgentSticky = true
		return true
	}
	return false
}

func invalidateStartPreview(state *MenuState, frame *MenuFrame) bool {
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.PreviewGeneration == frame.Generation {
		state.Notice = MenuNotice{}
	}
	generation, ok := nextPreviewGeneration(state)
	clearStartPreview(frame)
	if ok {
		frame.Generation = generation
	} else {
		frame.Generation = 0
	}
	return ok
}

func requestPathCompletion(state MenuState) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if frame.CompletionPending && frame.CompletionPath == frame.Path {
		return state, nil
	}
	query := SplitCompletionPath(frame.Path)
	if query.Immediate != "" {
		invalidateStartCompletion(&state, frame)
		frame.Path = query.Immediate
		invalidateStartPreview(&state, frame)
		return state, nil
	}
	if state.CompletionSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("path completion identity exhausted")
		return state, nil
	}
	state.CompletionSequence++
	identity := CompletionIdentity{FrameInstance: frame.Instance, Generation: state.CompletionSequence}
	invalidateStartCompletion(&state, frame)
	frame.CompletionRequest = identity
	frame.CompletionPath = frame.Path
	frame.CompletionPending = true
	request := CompletionRequest{Identity: identity, Path: frame.Path}
	return state, []MenuEffect{{Completion: &request}}
}

func invalidateStartCompletion(state *MenuState, frame *MenuFrame) {
	identity := frame.CompletionRequest
	frame.CompletionRequest = CompletionIdentity{}
	frame.CompletionPath = ""
	frame.CompletionPending = false
	frame.CompletionCandidates = nil
	frame.CompletionSelected = 0
	frame.CompletionTruncated = false
	if state.Notice.Owner.Completion == identity && identity != (CompletionIdentity{}) {
		state.Notice = MenuNotice{}
	}
}

func completionResultMatches(state MenuState, result *CompletionResult) bool {
	if result == nil || result.Identity.FrameInstance == 0 || result.Identity.Generation == 0 || state.CurrentFrame().Kind != MenuFrameStart {
		return false
	}
	frame := state.CurrentFrame()
	return frame.Instance == result.Identity.FrameInstance && frame.CompletionRequest == result.Identity
}

func reduceCompletionResult(state MenuState, result CompletionResult) MenuState {
	frame := &state.Frames[len(state.Frames)-1]
	frame.CompletionPending = false
	frame.CompletionCandidates = nil
	frame.CompletionSelected = 0
	frame.CompletionTruncated = false
	owner := MenuProgressOwner{Completion: result.Identity}
	if result.Error != "" {
		state.Notice = MenuNotice{Level: MenuNoticeError, Text: result.Error, Owner: owner}
		return state
	}
	paths := append([]string(nil), result.Matches.Paths...)
	if len(paths) == 0 {
		state.Notice = MenuNotice{Level: MenuNoticeInfo, Text: "no matching directories", Owner: owner}
		return state
	}
	if len(paths) == 1 {
		frame.Path = paths[0]
		invalidateStartCompletion(&state, frame)
		invalidateStartPreview(&state, frame)
		return state
	}
	frame.CompletionCandidates = paths
	frame.CompletionTruncated = result.Matches.Truncated
	return state
}

func nextPreviewGeneration(state *MenuState) (uint64, bool) {
	if state.PreviewSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("start preview identity exhausted")
		return 0, false
	}
	state.PreviewSequence++
	return state.PreviewSequence, true
}

func clearStartPreview(frame *MenuFrame) {
	frame.PreviewPending = 0
	frame.PreviewAccepted = 0
	frame.PreviewResolution = couchcore.StartResolution{}
	frame.SubmitGeneration = 0
}

func requestStartPreview(state MenuState) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if frame.Generation == 0 {
		return state, nil
	}
	if frame.PreviewPending == frame.Generation ||
		(frame.PreviewAccepted == frame.Generation && frame.PreviewResolution.Fingerprint != "") {
		return state, nil
	}
	path := frame.Path
	if path == "" {
		path = "."
	}
	request := PreviewRequest{Generation: frame.Generation, Path: path, Action: couchcore.StartCreate}
	if frame.AgentSticky {
		request.Agent = frame.Agent
	}
	frame.PreviewPending = frame.Generation
	return state, []MenuEffect{{Preview: &request}}
}

func reducePreviewResult(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	if state.CurrentFrame().Kind == MenuFrameSwitchAgent {
		return reduceSwitchAgentPreview(state, event)
	}
	if state.CurrentFrame().Kind != MenuFrameStart {
		return state, nil
	}
	frame := &state.Frames[len(state.Frames)-1]
	if event.Generation != frame.Generation || event.Generation != frame.PreviewPending {
		return state, nil
	}
	frame.PreviewPending = 0
	if event.Error != "" || event.Prepared == nil || event.Prepared.Resolution.Fingerprint == "" {
		frame.SubmitGeneration = 0
		state.Notice = errorMenuNotice(event.Error)
		if state.Notice.Text == "" {
			state.Notice = errorMenuNotice("start preview failed")
		}
		return state, nil
	}
	frame.PreviewAccepted = event.Generation
	frame.PreviewResolution = event.Prepared.Resolution
	frame.Agent = event.Prepared.Resolution.Profile.Agent
	if frame.SubmitGeneration != event.Generation {
		return state, nil
	}
	frame.SubmitGeneration = 0
	return dispatchMenuOperation(state, startMenuEffect(*frame), couchcore.ThreadAddress{})
}

// startMenuEffect submits exactly what the preview accepted.
//
// It used to pass an opaque grant token the owner had to look up. It now passes
// the resolution's own inputs plus its fingerprint, so the owner re-derives and
// compares rather than trusting a snapshot -- the same refusal, without the
// capability table (pair#170 M4).
func startMenuEffect(frame MenuFrame) MenuEffect {
	return MenuEffect{Operation: "start", Args: frame.PreviewResolution.CommitArgs()}
}

// unusableThreadNotice is what Enter says about a row it will not act on. It
// separates the repairable cases from the finished ones, because "your agent is
// still running, couch just lost the pointer" and "this is over" call for very
// different reactions.
func unusableThreadNotice(thread couchcore.ActionableThreadSummary) string {
	if notice := menuRowNotice(menuRowFactsOf(thread)); notice != "" {
		return notice
	}
	if thread.Recovery != nil && thread.Recovery.Diagnosis != "" {
		return thread.Recovery.Diagnosis
	}
	if thread.Orphan != nil {
		return launcher.OrphanDiagnostic(thread.Orphan.Session, thread.Orphan.PID)
	}
	switch thread.Reason {
	case couchcore.ReasonBindingLost:
		return "its native conversation binding is unavailable; cold resume requires a verified binding"
	case couchcore.ReasonSessionGone:
		return "the session is gone"
	case couchcore.ReasonNeverStarted:
		return "it never started"
	case couchcore.ReasonInvalid:
		return "its record is not valid"
	case couchcore.ReasonUnreadable:
		return "couch could not read its record -- it may have been written by a newer couch"
	case couchcore.ReasonPathMissing:
		return "its working path is unavailable"
	case couchcore.ReasonProfileMissing:
		return "it has no saved launch to resume from"
	case couchcore.ReasonAgentUnsupported:
		return "its saved agent is not supported by this build"
	case couchcore.ReasonUnknown:
		return "couch could not check its state this refresh"
	case couchcore.ReasonOrphanedServer:
		return "its zellij server is running but lost its socket; reap it to resume"
	}
	return string(thread.Reason)
}

// confirmationMenuItems names what the operator is about to accept.
//
// The item is the only place that naming can live: the frame title never
// reaches the screen, since RenderMenuView overwrites line 0 with the
// breadcrumb. So the destructive whole-couch form spells out its cost here --
// that many agents stop, any of them possibly mid-turn -- exactly as the
// per-thread park names the thread it kills.
func confirmationMenuItems(state MenuState, frame MenuFrame) []string {
	if frame.Action == "leave" {
		live := 0
		for _, thread := range menuRows(state) {
			if thread.Live() {
				live++
			}
		}
		if couchcore.LeaveDisposition(frame.Mode) != couchcore.LeavePark || live == 0 {
			return []string{"cancel", "leave couch"}
		}
		if live == 1 {
			return []string{"cancel", "leave couch, parking 1 live thread"}
		}
		return []string{"cancel", "leave couch, parking " + strconv.Itoa(live) + " live threads"}
	}
	thread, _ := menuThreadTarget(state, frame.RowKey, frame.Thread)
	// The item's FIRST WORD is its id (menuItemID), and Enter dispatches only
	// when that id equals frame.Action. So the action name is prepended
	// STRUCTURALLY rather than written out per case: relaunch shipped with
	// park's hand-written label, which made one confirmation both misdescribe
	// the operation ("park brain" under a "relaunch" breadcrumb) and refuse to
	// run it, because "park" != "relaunch" at the dispatch guard. A default
	// that spells another action's name is not a default, it is a lie.
	item := frame.Action + " " + thread.Label()
	switch frame.Action {
	case "reap":
		item += reapConfirmationCost(thread)
	case "reboot":
		// Say what rebooting COSTS, because the frame title never reaches the
		// screen and "reboot" alone does not say the conversation goes.
		if cost := menuRowAdviceOf(menuRowFactsOf(thread)).RebootCost.Text; cost != "" {
			// Nothing can start where there is no directory: reboot files the
			// record and stops, and the operator needs the next step here.
			item += cost
			break
		}
		item += " — archives this conversation, starts a fresh agent"
		if thread.Detached() {
			// A detached row's agent is RUNNING behind a session couch does
			// not host, and reboot stops that session before archiving
			// (archive's own quiesce). This confirmation is the last thing
			// between the operator and that (operator decision, pair#363).
			//
			// "may survive" is a MEASUREMENT, not hedging: `zellij
			// delete-session --force` reaps a pane by SIGHUP, and a pane
			// process that inherited SIG_IGN outlives it (measured 2026-09-17;
			// #274, still open, owns making the stop a promise).
			agent := thread.Agent
			if agent == "" {
				agent = "agent"
			}
			item += ", stops its session; its running " + agent + " may survive"
		}
	case "relaunch":
		// Same reason, different confusion: the one thing an operator needs to
		// know here is what park would have destroyed and this does not, and
		// nothing else on a two-item screen says the conversation survives.
		item += " — new Pair, same conversation"
	}
	return []string{"cancel", item}
}

func filterMenuItems(items []string, query string) []string {
	if query == "" {
		return append([]string(nil), items...)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(menuItemLabel(item)), strings.ToLower(query)) {
			out = append(out, item)
		}
	}
	return out
}

func menuItemLabel(item string) string {
	if item == "add-slot" {
		return "add slot"
	}
	if item == "switch-agent" {
		return "switch coding agent"
	}
	if item == "copy-orientation" {
		return "Copy orientation prompt"
	}
	return item
}

func menuItemID(item string) string {
	if before, _, found := strings.Cut(item, " "); found {
		return before
	}
	return item
}

func reconcileItemSelection(frame *MenuFrame, items []string) {
	if containsMenuItem(items, frame.SelectedItem) {
		return
	}
	frame.SelectedItem = ""
	if len(items) > 0 {
		frame.SelectedItem = menuItemID(items[0])
	}
}

func moveItemSelection(frame *MenuFrame, items []string, delta int) {
	if len(items) == 0 {
		frame.SelectedItem = ""
		return
	}
	index := 0
	for i, item := range items {
		if menuItemID(item) == frame.SelectedItem {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(items) {
		index = len(items) - 1
	}
	frame.SelectedItem = menuItemID(items[index])
}

func containsMenuItem(items []string, want string) bool {
	for _, item := range items {
		if menuItemID(item) == want {
			return true
		}
	}
	return false
}

func discardThreadFrames(state MenuState, address couchcore.ThreadAddress, notice string) MenuState {
	state.Frames = state.Frames[:1]
	state.Frames[0].SelectedAddress = address
	reconcileRootSelection(&state, address)
	state.Notice = errorMenuNotice(notice)
	return state
}

// reconcileRootSelection and moveRootSelection consider only SELECTABLE rows.
// A row the reattach pass has not finished with is drawn, greyed, but it is
// not ready, so the cursor and auto-select both skip it (pair#206). If every
// visible row is pending there is no selection, and Enter reports so.
func reconcileRootSelection(state *MenuState, preferred couchcore.ThreadAddress) {
	selectable := selectableRootRows(*state)
	frame := &state.Frames[0]
	preferredKey := frame.SelectedKey
	if preferredKey.Kind == couchcore.ThreadTargetSlot && preferred == frame.SelectedAddress {
		for _, row := range selectable {
			if menuRowKey(row) == preferredKey {
				selectMenuRow(frame, row)
				return
			}
		}
	}
	if preferred != (couchcore.ThreadAddress{}) {
		for _, row := range selectable {
			if row.Address == preferred {
				selectMenuRow(frame, row)
				return
			}
		}
	}
	frame.SelectedAddress = couchcore.ThreadAddress{}
	frame.SelectedKey = couchcore.ThreadRowKey{}
	if len(selectable) > 0 {
		selectMenuRow(frame, selectable[0])
	}
}

func moveRootSelection(state *MenuState, delta int) {
	selectable := selectableRootRows(*state)
	if len(selectable) == 0 {
		state.Frames[0].SelectedAddress = couchcore.ThreadAddress{}
		return
	}
	current := 0
	for i, thread := range selectable {
		if (state.Frames[0].SelectedKey.Kind == couchcore.ThreadTargetSlot && menuRowKey(thread) == state.Frames[0].SelectedKey) || (state.Frames[0].SelectedKey.Kind != couchcore.ThreadTargetSlot && thread.Address == state.Frames[0].SelectedAddress) {
			current = i
			break
		}
	}
	current += delta
	if current < 0 {
		current = 0
	}
	if current >= len(selectable) {
		current = len(selectable) - 1
	}
	selectMenuRow(&state.Frames[0], selectable[current])
}

func selectableRootRows(state MenuState) []couchcore.ActionableThreadSummary {
	visible := visibleMenuRows(state, state.Frames[0])
	selectable := make([]couchcore.ActionableThreadSummary, 0, len(visible))
	for _, row := range visible {
		if menuRowSelectable(state, row.Address) {
			selectable = append(selectable, row)
		}
	}
	return selectable
}

func reconcileMenuFrames(state MenuState, previous ...[]couchcore.ActionableThreadSummary) MenuState {
	priorInventory := state.Inventory
	if len(previous) > 0 {
		priorInventory = previous[0]
	}
	if len(state.Frames) == 0 || state.Frames[0].Kind != MenuFrameRoot {
		state.Frames = nil
		appendMenuFrame(&state, MenuFrame{Kind: MenuFrameRoot})
		return state
	}
	original := append([]MenuFrame(nil), state.Frames...)
	root := original[0]
	state.Frames = []MenuFrame{root}
	reconcileRootSelection(&state, root.SelectedAddress)

	invalidThreadFrame := false
	var bound couchcore.ThreadAddress
	for _, frame := range original[1:] {
		if frame.Kind == MenuFrameStart || !menuFrameBindsThread(frame) {
			// Not about a thread, so no inventory change can invalidate it.
			// Without this a leave confirmation is dropped ASYNCHRONOUSLY by
			// the next refresh -- a keystroke-only test would never see it.
			if frame.Kind == MenuFrameConfirmation {
				reconcileItemSelection(&frame, filterMenuItems(confirmationMenuItems(state, frame), frame.Filter))
			}
			state.Frames = append(state.Frames, frame)
			continue
		}
		if invalidThreadFrame {
			continue
		}
		thread, ok := menuThreadTarget(state, frame.RowKey, frame.Thread)
		if !ok && menuOperationReplacesAddress(state.InFlight) && menuFrameTargetsInFlight(state, frame) {
			// A reboot retires its own row: the :0 record leaves under its
			// old tag and returns under a new one. The frames that launched it
			// wait for its result, which restores them -- the operation is the
			// authority on its own frames, not a row it is itself replacing.
			if frame.Kind == MenuFrameActions {
				bound = frame.Thread
			}
			state.Frames = append(state.Frames, frame)
			continue
		}
		if !ok {
			invalidThreadFrame = true
			setBookkeepingNotice(&state, hiddenThreadNotice(priorInventory, frame.Thread))
			continue
		}
		frame.Thread = thread.Address
		switch frame.Kind {
		case MenuFrameActions:
			if bound != (couchcore.ThreadAddress{}) {
				invalidThreadFrame = true
				setBookkeepingNotice(&state, "thread action is no longer applicable")
				continue
			}
			reconcileItemSelection(&frame, filterMenuItems(menuActionsFor(state, thread), frame.Filter))
			bound = frame.Thread
		case MenuFrameConfirmation:
			// ONE rule for every action: a confirmation survives while its row
			// still offers its action. It used to be liveness plus two
			// special cases (archive and fresh-slot were the actions FOR rows
			// that are not live), and each new action had to remember which.
			//
			// The exception is a frame whose OWN operation is still in flight,
			// found by an operator watching a relaunch succeed. Relaunch parks
			// before it resumes, and reboot replaces its row outright, so the
			// thread is briefly not what the frame was opened on BY ITS OWN
			// DOING; a refresh landing in that window judged the confirmation
			// stale and reported "thread action is no longer applicable" over
			// an operation that went on to work. Target AND operation: an
			// exemption wider than its rationale is not scoped to the window
			// it explains.
			confirms, _ := couchcore.OperationConfirms(frame.Action)
			if (bound != (couchcore.ThreadAddress{}) && bound != frame.Thread) || !confirms ||
				(!menuFrameOperationInFlight(state, frame) && !containsMenuItem(menuActionItems(thread), frame.Action)) {
				invalidThreadFrame = true
				setBookkeepingNotice(&state, "thread action is no longer applicable")
				continue
			}
			reconcileItemSelection(&frame, filterMenuItems(confirmationMenuItems(state, frame), frame.Filter))
		case MenuFrameSwitchAgent:
			if bound != frame.Thread || (!containsMenuItem(menuActionItems(thread), "switch-agent") && state.InFlight.Operation != "switch-agent") {
				invalidThreadFrame = true
				continue
			}
		case MenuFrameText:
			textAllowed := menuFrameOperationInFlight(state, frame) || containsMenuItem(menuActionItems(thread), frame.Action)
			if bound != frame.Thread || !textAllowed {
				invalidThreadFrame = true
				setBookkeepingNotice(&state, "thread input is no longer applicable")
				continue
			}
		default:
			invalidThreadFrame = true
			setBookkeepingNotice(&state, "menu frame is no longer valid")
			continue
		}
		state.Frames = append(state.Frames, frame)
	}
	return state
}

func hiddenThreadNotice(previous []couchcore.ActionableThreadSummary, address couchcore.ThreadAddress) string {
	label := string(address.Tag)
	if thread, found := findMenuThread(previous, address); found {
		label = thread.Label()
	}
	return "thread " + label + " (" + address.RepoScope + "/" + string(address.Tag) + ") is no longer actionable"
}

func reduceOperationResult(state MenuState, event MenuEvent) MenuState {
	origin := state.InFlight
	if !menuOperationMatches(origin, event) {
		return state
	}
	state.InFlight = MenuOperationOrigin{}
	if origin.Address != (couchcore.ThreadAddress{}) && !menuOperationReplacesAddress(origin) {
		if _, stillActionable := menuThread(state, origin.Address); !stillActionable {
			return state
		}
	}
	originFrame, originVisible := menuOperationOriginFrame(state, origin)
	if !event.Success {
		if event.Operation == "switch-agent" && originVisible && origin.FrameKind == MenuFrameSwitchAgent {
			frame := &state.Frames[origin.Depth-1]
			frame.SwitchStage = 1
			frame.SelectedItem = "parameters"
			frame.PreviewPending = 0
		}
		// park and leave CLOSE their confirmation on failure: both are terminal
		// dispositions, and a failed one leaves nothing to retry from that
		// screen. relaunch and reboot deliberately keep theirs -- relaunch's
		// commonest refusal ("its agent has not completed a turn yet") is
		// transient and self-healing, and a refused reboot (a session that
		// survived its quiesce, a profile that did not resolve) is retried from
		// the same row, so the operator wants to stay put and press Enter again
		// rather than re-navigate.
		if (event.Operation == "park" || event.Operation == "leave") && origin.FrameKind == MenuFrameConfirmation && originVisible {
			state = restoreMenuPrefixPreservingStart(state, origin.Depth-1, origin)
		}
		state.Notice = errorMenuNotice(event.Error)
		if state.Notice.Text == "" {
			state.Notice = errorMenuNotice(event.Operation + " failed")
		}
		state.Notice.Owner = MenuProgressOwner{OperationAttempt: origin.Attempt}
		return state
	}
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.OperationAttempt == origin.Attempt {
		state.Notice = MenuNotice{}
	}
	if !event.InventorySet && operationNeedsProjectionRefresh(event.Operation) {
		state.ProjectionPending = true
		state.ProjectionAfterGeneration = event.ProjectionAfterGeneration
	}

	switch event.Operation {
	case "switch":
	case "alias":
		if origin.FrameKind == MenuFrameText && originVisible && originFrame.Thread == origin.Address && originFrame.Action == event.Operation {
			state.Frames = state.Frames[:origin.Depth-1]
		}
	case "start", "reboot":
		if originVisible {
			if origin.RowKey.Kind == couchcore.ThreadTargetSlot {
				state.Frames[0].SelectedKey = origin.RowKey
			}
			state = restoreMenuPrefixPreservingStart(state, 1, origin)
			state.Frames[0].SelectedAddress = event.Address
		}
	case "park", "detach", "resume", "leave", "relaunch", "switch-agent", "retry-continuation", "dismiss-continuation":
		state = restoreMenuPrefixPreservingStart(state, 1, origin)
		state.Frames[0].SelectedAddress = event.Address
		if origin.RowKey.Kind == couchcore.ThreadTargetSlot {
			state.Frames[0].SelectedKey = origin.RowKey
		}
		reconcileRootSelection(&state, event.Address)
	}
	return state
}

// endsItsOwnChild names the operations whose child exit is EXPECTED, so the two
// sites that need the answer cannot disagree.
//
// They existed as two hand-written lists -- the expectedExits bridge and
// consumeExpectedParkExitLocked, both in console.go -- because the
// exit/completion race resolves in either order and each half needed the same
// fact. Named, not pointed at: this comment said "the switch below" and then
// the function moved here, leaving the reference aimed at whatever happened to
// follow it. A relocation carries its comment's positional claims with it, and
// the fix is to stop making positional claims. A third operation had to appear in both or the
// operator gets a spurious child-exited notice for work they asked for; deriving
// it is what stops the next one being added to one list only (ARCH-DRY).
func endsItsOwnChild(operation string) bool {
	switch operation {
	case "park", "detach", "relaunch", "switch-agent", "retry-continuation", "continue-thread":
		return true
	}
	return false
}

// operationNeedsProjectionRefresh is the exhaustive projection policy for all
// switcher operations. Mutations whose result does not carry an actionable
// inventory must visibly defer to the next provider refresh. Switch only moves
// terminal focus; leave terminates the console and has no next frame to update.
func operationNeedsProjectionRefresh(operation string) bool {
	switch operation {
	case "start", "park", "detach", "resume", "reboot", "reap", "alias", "relaunch", "switch-agent", "retry-continuation", "dismiss-continuation", "continue-thread":
		return true
	case "switch", "leave":
		return false
	default:
		// Fail safe: a new successful operation may have changed the inventory.
		return true
	}
}

func restoreMenuPrefixPreservingStart(state MenuState, keep int, origin MenuOperationOrigin) MenuState {
	overlays := make([]MenuFrame, 0, 1)
	for _, frame := range state.Frames {
		if frame.Kind == MenuFrameStart && frame.Instance != origin.FrameInstance {
			overlays = append(overlays, frame)
		}
	}
	if keep < 0 {
		keep = 0
	}
	if keep > len(state.Frames) {
		keep = len(state.Frames)
	}
	frames := append([]MenuFrame(nil), state.Frames[:keep]...)
	for _, overlay := range overlays {
		alreadyPresent := false
		for _, frame := range frames {
			if frame.Instance == overlay.Instance {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			frames = append(frames, overlay)
		}
	}
	state.Frames = frames
	return state
}

func menuOperationOriginFrame(state MenuState, origin MenuOperationOrigin) (MenuFrame, bool) {
	if origin.Depth < 1 || origin.Depth > len(state.Frames) {
		return MenuFrame{}, false
	}
	frame := state.Frames[origin.Depth-1]
	if frame.Instance != origin.FrameInstance || frame.Kind != origin.FrameKind {
		return MenuFrame{}, false
	}
	return frame, true
}

func menuOperationMatches(origin MenuOperationOrigin, event MenuEvent) bool {
	if origin.Operation == "" || origin.Attempt == 0 || origin.Attempt != event.Attempt || origin.Operation != event.Operation {
		return false
	}
	if menuOperationReplacesAddress(origin) {
		return true
	}
	if origin.Operation == "start" && origin.Address == (couchcore.ThreadAddress{}) {
		return !event.Success || event.Address != (couchcore.ThreadAddress{})
	}
	return origin.Address == event.Address
}

func dispatchThreadOperation(state MenuState, operation string, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	effect := threadEffect(operation, address)
	if thread, ok := menuThread(state, address); ok && actorOperation(operation) {
		// Resume and reboot take the row's arguments from the one mapping the
		// socket's admission also reads (warm-only on a detached row).
		effect.Args = couchcore.ActorOperationArgs(thread, operation)
	}
	return dispatchMenuOperation(state, effect, address)
}

func dispatchMenuOperation(state MenuState, effect MenuEffect, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	if effect.Operation == "" || state.InFlight.Operation != "" {
		return state, nil
	}
	if state.OperationSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("operation attempt identity exhausted")
		return state, nil
	}
	state.OperationSequence++
	effect.Attempt = state.OperationSequence
	if effect.Operation == "retry-continuation" || effect.Operation == "dismiss-continuation" {
		// threadEffect already addresses the row by its exact tag. Adding `ref`
		// as well is refused by resolveOperationThread, which is how the switcher's
		// retry never reached its thread (#280).
		if thread, ok := menuThread(state, address); ok && thread.Continuation != nil {
			effect.Args["request-id"] = thread.Continuation.RequestID
		}
	}
	if effect.Operation == "resume" {
		// The operator resuming a failed thread by hand clears its mark
		// (pair#206 cell 9).
		state = clearReattachFailure(state, address)
	}
	state.InFlight = MenuOperationOrigin{
		Operation:     effect.Operation,
		Attempt:       effect.Attempt,
		Address:       address,
		FrameInstance: state.CurrentFrame().Instance,
		FrameKind:     state.CurrentFrame().Kind,
		Depth:         len(state.Frames),
	}
	state.SpinnerPhase = 0
	state.Notice = MenuNotice{
		Level: MenuNoticeProgress,
		Text:  menuOperationProgressText(state, effect.Operation, address),
		Owner: MenuProgressOwner{OperationAttempt: effect.Attempt},
	}
	return state, []MenuEffect{effect}
}

func menuOperationProgressText(state MenuState, operation string, address couchcore.ThreadAddress) string {
	label := string(address.Tag)
	if thread, ok := menuThread(state, address); ok {
		label = thread.Label()
	}
	switch operation {
	case "start":
		return "starting thread"
	case "resume":
		return "resuming " + label
	case "park":
		return "parking " + label
	case "detach":
		return "detaching " + label
	case "leave":
		return "leaving couch"
	case "relaunch":
		return "restarting " + label + "'s pair…"
	case "retry-continuation":
		return "retrying continuation for " + label
	case "dismiss-continuation":
		return "dismissing continuation for " + label
	case "reboot":
		return "rebooting " + label
	case "reap":
		return "reaping " + label + "'s orphaned server…"
	default:
		return operation
	}
}

func appendMenuFrame(state *MenuState, frame MenuFrame) bool {
	if state.FrameSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("menu frame identity exhausted")
		return false
	}
	state.FrameSequence++
	frame.Instance = state.FrameSequence
	state.Frames = append(state.Frames, frame)
	return true
}

// SelectedThreadAddress is the thread the operator is pointing at, from ANY
// depth in the hierarchy.
//
// Only the ROOT frame carries SelectedAddress; a frame the operator has drilled
// into (actions, confirmation, text) carries Thread instead. Reading
// CurrentFrame().SelectedAddress therefore answers correctly only at the root
// and returns the zero address everywhere else -- so Alt+n in the switcher
// refused with "no thread selected" whenever the operator had opened a row's
// actions first, which is the ordinary way to be looking at a thread.
//
// One method rather than each chord handler picking a frame: "which thread is
// the operator pointing at" has one answer, and a second copy of it would be a
// second chance to pick the wrong frame.
func (s MenuState) SelectedThreadAddress() couchcore.ThreadAddress {
	if len(s.Frames) == 0 {
		return couchcore.ThreadAddress{}
	}
	if frame := s.CurrentFrame(); menuFrameBindsThread(frame) && frame.Thread != (couchcore.ThreadAddress{}) {
		return frame.Thread
	}
	return s.Frames[0].SelectedAddress
}

func selectedMenuThread(state MenuState) (couchcore.ActionableThreadSummary, bool) {
	return menuThreadTarget(state, state.CurrentFrame().SelectedKey, state.CurrentFrame().SelectedAddress)
}

func findMenuThread(inventory []couchcore.ActionableThreadSummary, address couchcore.ThreadAddress) (couchcore.ActionableThreadSummary, bool) {
	for _, thread := range inventory {
		if thread.Address == address {
			return thread, true
		}
	}
	return couchcore.ActionableThreadSummary{}, false
}

func threadEffect(operation string, address couchcore.ThreadAddress) MenuEffect {
	return MenuEffect{Operation: operation, Args: map[string]string{
		"repo-scope": address.RepoScope,
		"tag":        string(address.Tag),
	}}
}

// leaveEffect is the whole-couch operation. It addresses no thread because it
// addresses all of them: the disposition is the entire argument.
func leaveEffect(disposition couchcore.LeaveDisposition) MenuEffect {
	return MenuEffect{Operation: "leave", Args: map[string]string{"mode": string(disposition)}}
}

func hierarchyNavigationKey(key PanelKey, forward PanelKeyKind) PanelKey {
	switch key.Kind {
	case KeyLeft:
		key.Kind = KeyEscape
	case KeyRight:
		key.Kind = forward
	}
	return key
}

// mergeObservations rebuilds one background pass's observations over its probe
// set: a fresh observation replaces, a failed probe keeps the last value (a
// stale glyph or fade beats an error in chrome, and a stale thread must not
// flash bright), and a key the pass no longer probes is dropped. Slot git
// (pair#317) and idle activity (pair#247) share it.
func mergeObservations[K comparable, V any](prev, observed map[K]V, failed map[K]bool) map[K]V {
	next := make(map[K]V, len(observed)+len(failed))
	for key, value := range observed {
		next[key] = value
	}
	for key := range failed {
		if value, ok := prev[key]; ok {
			next[key] = value
		}
	}
	return next
}

func cloneMenuState(state MenuState) MenuState {
	next := state
	if state.Orientation != nil {
		next.Orientation = make(map[couchcore.ThreadAddress]orientation.Request, len(state.Orientation))
		for k, v := range state.Orientation {
			next.Orientation[k] = v
		}
	}
	next.Inventory = append([]couchcore.ActionableThreadSummary(nil), state.Inventory...)
	next.Frames = append([]MenuFrame(nil), state.Frames...)
	for i := range next.Frames {
		next.Frames[i].CompletionCandidates = append([]string(nil), state.Frames[i].CompletionCandidates...)
	}
	next.Agents = append([]string(nil), state.Agents...)
	next.Reattach = cloneReattachPass(state.Reattach)
	if state.SlotGit != nil {
		next.SlotGit = make(map[string]couchcore.SlotGitStatus, len(state.SlotGit))
		for path, status := range state.SlotGit {
			next.SlotGit[path] = status
		}
	}
	if state.Activity != nil {
		next.Activity = make(map[couchcore.ThreadAddress]time.Time, len(state.Activity))
		for address, at := range state.Activity {
			next.Activity[address] = at
		}
	}
	if state.Attention != nil {
		next.Attention = make(map[couchcore.ThreadAddress][]AttentionMessage, len(state.Attention))
		for address, messages := range state.Attention {
			next.Attention[address] = append([]AttentionMessage(nil), messages...)
		}
	}
	return next
}

// reapConfirmationCost says what reaping costs, because the operator is about to
// end a process tree whose agent may still be writing (#399). It names the
// server so the confirmation is about an exact process, not a label.
func reapConfirmationCost(thread couchcore.ActionableThreadSummary) string {
	if o := thread.Orphan; o != nil {
		return fmt.Sprintf(" — ends orphaned server PID %d and everything under it (pair wrap/term/title, nvim, the agent)", o.PID)
	}
	return " — ends its orphaned server and everything under it (pair wrap/term/title, nvim, the agent)"
}
