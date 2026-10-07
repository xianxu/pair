package couchtty

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// openRecover starts the switcher's recover (#399 Task 9b). Whether it needs a
// confirmation is the recovery report's answer (ConfirmByPlan), and the report
// runs sdlc, so Enter opens the confirmation frame in a waiting state and asks
// for the preview off the UI thread, as switch-agent does. The preview's answer
// then dispatches at once, keeps the frame for a confirmation, or closes it with
// the hold.
func openRecover(state MenuState, thread couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	generation, ok := nextPreviewGeneration(&state)
	if !ok {
		return state, nil
	}
	if !appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameConfirmation, RowKey: menuRowKey(thread), Thread: thread.Address, Action: "recover",
		SelectedItem: "cancel", Generation: generation, PreviewPending: generation,
	}) {
		return state, nil
	}
	request := PreviewRequest{Generation: generation, RecoverArgs: couchcore.ActorOperationArgs(thread, "recover")}
	return state, []MenuEffect{{Preview: &request}}
}

// reduceRecoverPreview lands a prepare-recover answer on the waiting frame.
func reduceRecoverPreview(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if event.Generation == 0 || event.Generation != frame.Generation || event.Generation != frame.PreviewPending {
		return state, nil
	}
	frame.PreviewPending = 0
	abandon := func(notice string) (MenuState, []MenuEffect) {
		state.Frames = state.Frames[:len(state.Frames)-1]
		state.Notice = errorMenuNotice(notice)
		return state, nil
	}
	p := event.RecoverPreview
	switch {
	case event.Error != "":
		return abandon("recover: " + event.Error)
	case p == nil || (p.Hold == "" && len(p.Steps) == 0):
		return abandon("invalid recover preview")
	case p.Hold != "":
		return abandon("recover holds: " + p.Hold)
	}
	thread, ok := menuThreadTarget(state, frame.RowKey, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	if !p.Confirm {
		state.Frames = state.Frames[:len(state.Frames)-1]
		return dispatchRecover(state, thread, *p, false)
	}
	frame.RecoverPreview = p
	return state, nil
}

// dispatchRecover sends recover with the steps the operator was shown, so the
// owner refuses if the row's plan changed since ("the row changed; review
// again"), the stale-preview rule switch-agent applies to its source. A slot row
// is keyed by row: a reboot step replaces its record.
func dispatchRecover(state MenuState, row couchcore.ActionableThreadSummary, preview couchcore.RecoverPreview, confirmed bool) (MenuState, []MenuEffect) {
	args := couchcore.ActorOperationArgs(row, "recover")
	args["steps"] = strings.Join(preview.Steps, ",")
	if confirmed {
		args["confirmed"] = "true"
	}
	next, effects := dispatchMenuOperation(state, MenuEffect{Operation: "recover", Args: args}, row.Address)
	if len(effects) > 0 && row.Target.Kind == couchcore.ThreadTargetSlot {
		next.InFlight.RowKey = menuRowKey(row)
	}
	return next, effects
}
