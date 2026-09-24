package couchtty

import (
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The background reattach pass (pair#206).
//
// After `couch` attaches the thread for its directory, every other thread whose
// agent is still running behind a client-less zellij session is reattached, one
// at a time, behind the operator's back. It is pure state inside MenuState, so
// the menu's single transition authority owns every interleaving with the
// operator's own operations -- the transitions table in the #206 plan is its
// specification, and these functions are its cells.
//
// The operator's UX (2026-09-11): every pending thread appears at once as a
// placeholder -- greyed in the status bar and the switcher, with a spinner on
// the one currently starting -- and none can be clicked or selected until it
// attaches. So nothing here ever claims the operator's in-flight slot.

// ReattachPhase is the pass's lifecycle.
type ReattachPhase uint8

const (
	// ReattachIdle has never been armed.
	ReattachIdle ReattachPhase = iota
	// ReattachArmed waits for the first successful inventory to seed from.
	ReattachArmed
	// ReattachRunning has a queue or an attempt in flight.
	ReattachRunning
	// ReattachDone has drained its queue. Distinct from Idle so a second arm
	// cannot re-seed and reattach threads the operator detached meanwhile.
	ReattachDone
)

// ReattachPass is the pass's whole state. It holds ADDRESSES, never rows, so a
// stale inventory row can never be acted on through it.
type ReattachPass struct {
	Phase ReattachPhase
	// Root is the startup thread, excluded explicitly: the Spec requires the
	// exclusion to be the pass's own rule, not an inheritance from the
	// inventory classifying a hosted thread as live.
	Root couchcore.ThreadAddress
	// Queue is the threads not yet attempted, in attempt order.
	Queue []couchcore.ThreadAddress
	// Loading is the thread with an attempt in flight; zero when none is.
	Loading couchcore.ThreadAddress
	// LoadingAttempt is that attempt's identity, drawn from the same counter as
	// the operator's operations so the two can never be confused.
	LoadingAttempt uint64
	// Attached maps a thread that just attached to the refresh generation
	// current when it landed. The first inventory admitted after that
	// generation drops it: expiring on "the inventory shows it live" instead
	// would strand the row forever if the pane exited before the next refresh.
	Attached map[couchcore.ThreadAddress]uint64
	// Failed maps a thread whose attempt failed to what its row says after
	// "reattach failed: ": the refusal's code, or the first line of a failure
	// that has none (decision 11).
	Failed map[couchcore.ThreadAddress]string
}

// reattachFailedCode is what a failed row says when the failure carried neither
// a refusal code nor any text. Without it the row would read "reattach failed: "
// with nothing after.
const reattachFailedCode = "reattach-failed"

func cloneReattachPass(pass ReattachPass) ReattachPass {
	next := pass
	next.Queue = append([]couchcore.ThreadAddress(nil), pass.Queue...)
	if pass.Attached != nil {
		next.Attached = make(map[couchcore.ThreadAddress]uint64, len(pass.Attached))
		for address, generation := range pass.Attached {
			next.Attached[address] = generation
		}
	}
	if pass.Failed != nil {
		next.Failed = make(map[couchcore.ThreadAddress]string, len(pass.Failed))
		for address, code := range pass.Failed {
			next.Failed[address] = code
		}
	}
	return next
}

// PassState is what the pass means for one row.
type PassState uint8

const (
	PassNone PassState = iota
	PassQueued
	PassLoading
	PassFailed
	PassAttached
)

// PassView is the pass's verdict on one row. The #206 plan's "pass view" table
// is its contract; this is the one function that computes it.
type PassView struct {
	State PassState
	// Diagnostic is set for PassFailed.
	Diagnostic string
}

// Pending reports a row the pass has not finished with: it is drawn greyed and
// is neither selectable nor clickable, because it is not ready.
func (v PassView) Pending() bool { return v.State == PassQueued || v.State == PassLoading }

// passViewOf is the only place that knows what the pass means for a row. The
// states are disjoint by construction -- an address leaves the queue as it
// becomes Loading, and Loading resolves to exactly one of Attached, Failed or
// nothing -- so the order below only decides ties that cannot occur.
func passViewOf(pass ReattachPass, address couchcore.ThreadAddress) (PassView, bool) {
	if address == (couchcore.ThreadAddress{}) {
		return PassView{}, false
	}
	if pass.Loading == address {
		return PassView{State: PassLoading}, true
	}
	if slices.Contains(pass.Queue, address) {
		return PassView{State: PassQueued}, true
	}
	if code, failed := pass.Failed[address]; failed {
		return PassView{State: PassFailed, Diagnostic: code}, true
	}
	if _, attached := pass.Attached[address]; attached {
		return PassView{State: PassAttached}, true
	}
	return PassView{}, false
}

// reattachCandidate is the seed rule: a thread whose agent is still running, or
// one whose proof could not be ASKED. Unknown is "we could not ask", not "not
// detached", so it is included; warm-only re-proves every thread at attempt
// time and refuses one that is not really warm, which the pass then skips.
func reattachCandidate(row couchcore.ActionableThreadSummary) bool {
	if row.Detached() {
		return true
	}
	return row.State == couchcore.ThreadUnusable && row.Reason == couchcore.ReasonUnknown
}

// armReattach is cell 1: Idle -> Armed, remembering the root to exclude. Arming
// is once per console; any other phase ignores it.
func armReattach(state MenuState, root couchcore.ThreadAddress) MenuState {
	if state.Reattach.Phase != ReattachIdle {
		return state
	}
	state.Reattach.Phase = ReattachArmed
	state.Reattach.Root = root
	return state
}

// seedReattach is cell 3: the first successful inventory after arming fills the
// queue, once. The queue is never extended afterwards (cell 4), so a thread the
// operator detaches during the pass is not reattached behind their back.
func seedReattach(state MenuState, inventory []couchcore.ActionableThreadSummary) MenuState {
	if state.Reattach.Phase != ReattachArmed {
		return state
	}
	var candidates []couchcore.ActionableThreadSummary
	for _, row := range inventory {
		if row.Address == state.Reattach.Root || !reattachCandidate(row) {
			continue
		}
		candidates = append(candidates, row)
	}
	// Most recently active first -- the thread the operator most likely wants
	// next -- with ties broken by address so the order is deterministic.
	slices.SortStableFunc(candidates, func(a, b couchcore.ActionableThreadSummary) int {
		if c := b.LastActiveAt.Compare(a.LastActiveAt); c != 0 {
			return c
		}
		return strings.Compare(a.Address.RepoScope+"\x00"+string(a.Address.Tag), b.Address.RepoScope+"\x00"+string(b.Address.Tag))
	})
	state.Reattach.Queue = make([]couchcore.ThreadAddress, 0, len(candidates))
	for _, row := range candidates {
		state.Reattach.Queue = append(state.Reattach.Queue, row.Address)
	}
	state.Reattach.Phase = ReattachRunning
	return state
}

// advanceReattach starts the next attempt when the pass may.
//
// It emits nothing while an attempt is already in flight (one at a time), and
// nothing while the OPERATOR has an operation in flight (cell 10). The second is
// what makes quitting safe: the queue worker starts the next request the moment
// it pushes the previous result, so without the hold a leave dispatched
// mid-pass would be followed by one more reattach that is then cancelled.
func advanceReattach(state MenuState) (MenuState, []MenuEffect) {
	pass := state.Reattach
	if pass.Phase != ReattachRunning || pass.Loading != (couchcore.ThreadAddress{}) || state.InFlight.Operation != "" {
		return state, nil
	}
	// The counter guard is unreachable in practice, since it takes 2^64
	// operations. It is there because the increment below would wrap to 0,
	// which is the "no attempt" identity finishReattach refuses, and the pass
	// would then hang on an attempt it can never finish. Ending the pass is
	// the safe answer.
	if len(pass.Queue) == 0 || state.OperationSequence == ^uint64(0) {
		state.Reattach.Phase = ReattachDone
		return state, nil
	}
	next := pass.Queue[0]
	state.Reattach.Queue = append([]couchcore.ThreadAddress(nil), pass.Queue[1:]...)
	state.OperationSequence++
	state.Reattach.Loading = next
	state.Reattach.LoadingAttempt = state.OperationSequence
	effect := threadEffect("resume", next)
	effect.Attempt = state.OperationSequence
	effect.Background = true
	// The pass may only REATTACH. warm-only refuses a thread that stopped being
	// warm -- parked meanwhile, or its session gone -- before any effect, so
	// the pass can never start an agent.
	effect.Args["warm-only"] = "true"
	return state, []MenuEffect{effect}
}

// finishReattach applies one completed pass attempt: cells 5-8.
func finishReattach(state MenuState, event MenuEvent) MenuState {
	pass := state.Reattach
	if !event.Background || pass.LoadingAttempt == 0 || event.Attempt != pass.LoadingAttempt || event.Address != pass.Loading {
		return state // cell 8: not the attempt in flight
	}
	address := pass.Loading
	state.Reattach.Loading = couchcore.ThreadAddress{}
	state.Reattach.LoadingAttempt = 0
	switch {
	case event.Success:
		// Cell 5. The pass, not the lagging inventory, is the authority for the
		// row until a refresh newer than this one lands.
		if state.Reattach.Attached == nil {
			state.Reattach.Attached = map[couchcore.ThreadAddress]uint64{}
		}
		state.Reattach.Attached[address] = event.ProjectionAfterGeneration
	case event.Diagnostic == couchcore.ResumeNotDetached || event.Diagnostic == couchcore.ResumeSessionGone:
		// Cell 6: the thread stopped being warm before its turn. Skipped
		// silently -- it was never going to be reattached, so it did not fail.
	default:
		// Cell 7. A refusal carries a code. A failure that is not one (a spawn
		// error, a registration timeout) carries only its text, so its row
		// shows that text's first line (decision 11).
		code := string(event.Diagnostic)
		if code == "" {
			code = firstErrorLine(event.Error)
		}
		if code == "" {
			code = reattachFailedCode
		}
		if state.Reattach.Failed == nil {
			state.Reattach.Failed = map[couchcore.ThreadAddress]string{}
		}
		state.Reattach.Failed[address] = code
	}
	return state
}

// firstErrorLine is the part of an error a one-line switcher row can show. It
// is stored as the error wrote it; the renderer sanitizes it and fits it to the
// row (passSuffix).
func firstErrorLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}

// expireAttached is cell 11: an inventory newer than the generation a thread
// attached at is authoritative for it again, whatever it says.
func expireAttached(state MenuState, generation uint64) MenuState {
	for address, landed := range state.Reattach.Attached {
		if generation > landed {
			delete(state.Reattach.Attached, address)
		}
	}
	return state
}

// clearReattachFailure is cell 9: the operator resuming a failed thread by hand
// clears its mark. A failed row is no longer pending, so it is selectable.
func clearReattachFailure(state MenuState, address couchcore.ThreadAddress) MenuState {
	delete(state.Reattach.Failed, address)
	return state
}

// ReattachPlaceholder is one pending thread for the status bar.
type ReattachPlaceholder struct {
	Address couchcore.ThreadAddress
	// Loading marks the one thread currently starting, which carries the
	// spinner; the rest are queued.
	Loading bool
}

// pendingPlaceholders are the status bar's placeholder chips: the thread
// starting now, then the queue, in attempt order. That order is what keeps a
// chip in place when it resolves: attached chips are drawn in attach order and
// placeholders after them, so the thread that just attached takes the column
// its placeholder held.
func pendingPlaceholders(pass ReattachPass) []ReattachPlaceholder {
	var out []ReattachPlaceholder
	if pass.Loading != (couchcore.ThreadAddress{}) {
		out = append(out, ReattachPlaceholder{Address: pass.Loading, Loading: true})
	}
	for _, address := range pass.Queue {
		out = append(out, ReattachPlaceholder{Address: address})
	}
	return out
}

// The view applied where rows are LOOKED UP (pair#206 plan, "The pass view").
//
// While the pass owns a row the inventory is behind it, and every reader that
// consulted the inventory first would get a stale answer -- three plan-review
// rounds each found another such reader. So the view is applied here, once, and
// TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups fails any menu code
// that reads the inventory some other way.

// overlayPassRow applies the pass's view to one inventory row. The only state it
// OVERRIDES is Attached: a thread the pass just reattached is live in this
// console even while the inventory still says detached, so every reader --
// Enter's verb, the leave count, rendering -- treats it as live. Pending and
// failed rows keep their inventory state; readers ask passViewOf for those.
func overlayPassRow(pass ReattachPass, row couchcore.ActionableThreadSummary) couchcore.ActionableThreadSummary {
	if view, owned := passViewOf(pass, row.Address); owned && view.State == PassAttached {
		row.State = couchcore.ThreadLive
		row.Reason = ""
	}
	return row
}

// menuRows is every inventory row with the pass's view applied.
func menuRows(state MenuState) []couchcore.ActionableThreadSummary {
	rows := make([]couchcore.ActionableThreadSummary, len(state.Inventory))
	for i, row := range state.Inventory {
		rows[i] = overlayPassRow(state.Reattach, row)
	}
	return rows
}

// menuThread is one row with the pass's view applied.
func menuThread(state MenuState, address couchcore.ThreadAddress) (couchcore.ActionableThreadSummary, bool) {
	row, ok := findMenuThread(state.Inventory, address)
	if !ok {
		return row, false
	}
	return overlayPassRow(state.Reattach, row), true
}

// visibleMenuRows is the root list after the filter, with the pass's view.
func visibleMenuRows(state MenuState, frame MenuFrame) []couchcore.ActionableThreadSummary {
	return visibleRootThreads(menuRows(state), frame)
}

// menuRowSelectable: a row the pass has not finished with is not ready, so the
// cursor, auto-select and a click all skip it.
func menuRowSelectable(state MenuState, address couchcore.ThreadAddress) bool {
	view, owned := passViewOf(state.Reattach, address)
	return !owned || !view.Pending()
}

// replaceMenuInventory is the one place a new inventory is installed. It
// returns the prior rows, which frame reconciliation compares against.
func replaceMenuInventory(state MenuState, inventory []couchcore.ActionableThreadSummary) (MenuState, []couchcore.ActionableThreadSummary) {
	previous := append([]couchcore.ActionableThreadSummary(nil), state.Inventory...)
	state.Inventory = orderedMenuInventory(inventory)
	return state, previous
}

// orderedMenuInventory derives order when a snapshot enters the reducer. Viewed
// lookups can then overlay pass state without sorting again on each keystroke.
func orderedMenuInventory(inventory []couchcore.ActionableThreadSummary) []couchcore.ActionableThreadSummary {
	entries := PresentThreads(inventory, nil)
	rows := make([]couchcore.ActionableThreadSummary, len(entries))
	for i, entry := range entries {
		rows[i] = entry.Row
	}
	return rows
}
