package couchtty

import (
	"strconv"
	"strings"
)

// couchScopedAction names the confirmations that act on Couch as a whole
// rather than on one thread: they bind no row, so a frame resolver must not
// discard them for lacking one.
func couchScopedAction(action string) bool {
	return action == "leave" || action == "wake"
}

// wakeTargets is the asleep set's hub rows (pair#830), in launch order with
// the main hub last: each resume lands on its slot, so the last dispatched is
// where the operator ends up. Workers are never in it; each hub wakes its own.
// The hub match is the reattach pass's (seedReattach): HubOwnership over the
// presented path.
func wakeTargets(state MenuState) []ThreadPresentation {
	var targets, main []ThreadPresentation
	for _, entry := range PresentThreads(menuRows(state), nil, state.Hub, state.HubOwnership) {
		if entry.Row.AsleepAt == nil || !state.HubOwnership.IsHub(entry.Path) {
			continue
		}
		if entry.Path == state.HubOwnership.Main() {
			main = append(main, entry)
			continue
		}
		targets = append(targets, entry)
	}
	return append(targets, main...)
}

func wakeItem(targets []ThreadPresentation) string {
	labels := make([]string, 0, len(targets))
	for _, t := range targets {
		labels = append(labels, t.Label)
	}
	noun := " asleep hubs: "
	if len(targets) == 1 {
		noun = " asleep hub: "
	}
	return "wake " + strconv.Itoa(len(targets)) + noun + strings.Join(labels, ", ")
}

// openWakeConfirmation asks before waking: waking starts agents, as Alt+x
// stops them. With nothing asleep there is nothing to confirm.
func openWakeConfirmation(state MenuState) (MenuState, []MenuEffect) {
	if len(wakeTargets(state)) == 0 {
		state.Notice = infoMenuNotice("no asleep hubs to wake")
		return state, nil
	}
	state.Frames = state.Frames[:1]
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Action: "wake", SelectedItem: "cancel"})
	return state, nil
}

// dispatchWake resumes every asleep hub through the one row resume the
// switcher's Enter uses, so admission, recovery and outcome reporting are the
// same. The set is re-read now, not taken from when the frame opened. A hub
// the queue refuses (busy) is named; waking again retries only what is still
// asleep.
func dispatchWake(state MenuState) (MenuState, []MenuEffect) {
	state.Frames = state.Frames[:1]
	var effects []MenuEffect
	var started, refused []string
	for _, target := range wakeTargets(state) {
		next, more := dispatchMenuRow(state, "resume", target.Row)
		if len(more) == 0 {
			refused = append(refused, target.Label+" ("+next.Notice.Text+")")
			continue
		}
		state, effects = next, append(effects, more...)
		started = append(started, target.Label)
	}
	switch {
	case len(started) == 0 && len(refused) == 0:
		state.Notice = infoMenuNotice("no asleep hubs to wake")
	case len(refused) == 0:
		state.Notice = infoMenuNotice("waking " + strings.Join(started, ", "))
	default:
		text := "not woken: " + strings.Join(refused, "; ")
		if len(started) > 0 {
			text = "waking " + strings.Join(started, ", ") + "; " + text
		}
		state.Notice = errorMenuNotice(text)
	}
	return state, effects
}
