package couchtty

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func openSwitchAgent(state MenuState, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	agents := launcher.AgentInventory()
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameSwitchAgent, Action: "switch-agent", Thread: address, Agent: agents[0], SelectedItem: "cancel"})
	return state, nil
}

func reduceSwitchAgentKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if key.Kind == KeyEscape && frame.SwitchStage == 2 {
		frame.SwitchStage = 1
		frame.SelectedItem = "force-switch"
		frame.SwitchPrepared.Force = nil
		return state, nil
	}
	if key.Kind == KeyEscape {
		state.Frames = state.Frames[:len(state.Frames)-1]
		return state, nil
	}
	if menuFrameRunningOrigin(state, *frame).Operation != "" {
		return state, nil
	}
	switch frame.SwitchStage {
	case 0:
		switch key.Kind {
		case KeyUp:
			selectStartAgent(frame, launcher.AgentInventory(), -1)
		case KeyDown:
			selectStartAgent(frame, launcher.AgentInventory(), 1)
		case KeyEnter:
			frame.SwitchStage = 1
			frame.SelectedItem = "parameters"
			return requestSwitchAgentPreview(state, false)
		}
	case 1:
		switch key.Kind {
		case KeyTab, KeyDown:
			moveSwitchFocus(frame, 1)
		case KeyUp:
			moveSwitchFocus(frame, -1)
		case KeyLeft, KeyRight:
			if frame.SelectedItem == "parameters" {
				length := utf8.RuneCountInString(frame.Input)
				if key.Kind == KeyLeft && frame.SwitchCursorFromEnd < length {
					frame.SwitchCursorFromEnd++
				} else if key.Kind == KeyRight && frame.SwitchCursorFromEnd > 0 {
					frame.SwitchCursorFromEnd--
				}
			} else if key.Kind == KeyRight {
				moveSwitchFocus(frame, 1)
			} else {
				moveSwitchFocus(frame, -1)
			}
		case KeyRune:
			if frame.SelectedItem == "parameters" && key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && key.Rune >= 32 && key.Rune != 127 && len(frame.Input)+utf8.RuneLen(key.Rune) <= 4096 {
				text, cursor := switchParameterCursor(*frame)
				frame.Input = string(text[:cursor]) + string(key.Rune) + string(text[cursor:])
				frame.SwitchEdited = true
				invalidateSwitchEdit(&state, frame)
			}
		case KeyBackspace:
			if frame.SelectedItem == "parameters" {
				text, cursor := switchParameterCursor(*frame)
				if cursor > 0 {
					frame.Input = string(text[:cursor-1]) + string(text[cursor:])
					frame.SwitchEdited = true
					invalidateSwitchEdit(&state, frame)
				}
			}
		case KeyEnter:
			switch frame.SelectedItem {
			case "parameters":
				frame.SelectedItem = "switch"
			case "cancel":
				state.Frames = state.Frames[:len(state.Frames)-1]
			case "switch", "interrupt-switch", "force-switch":
				if frame.PreviewPending != 0 {
					return state, nil
				}
				if frame.SwitchPrepared == nil {
					return requestSwitchAgentPreview(state, false)
				}
				return requestSwitchAgentPreview(state, true)
			}
		}
	case 2:
		switch key.Kind {
		case KeyLeft, KeyRight, KeyTab, KeyUp, KeyDown:
			if frame.SelectedItem == "cancel" {
				frame.SelectedItem = "confirm-force"
			} else {
				frame.SelectedItem = "cancel"
			}
		case KeyEnter:
			if frame.SelectedItem == "cancel" {
				frame.SwitchStage = 1
				frame.SelectedItem = "force-switch"
				frame.SwitchPrepared.Force = nil
				return state, nil
			}
			if frame.SelectedItem == "confirm-force" && frame.SwitchPrepared != nil && frame.SwitchPrepared.Force != nil {
				effect := acceptedSwitchEffect(*frame.SwitchPrepared)
				effect.TypedPayload = frame.SwitchPrepared.Force.Confirm()
				return dispatchMenuOperation(state, effect, frame.Thread)
			}
		}
	}

	return state, nil
}

func switchParameterCursor(frame MenuFrame) ([]rune, int) {
	text := []rune(frame.Input)
	return text, max(0, len(text)-frame.SwitchCursorFromEnd)
}

func requestSwitchAgentPreview(state MenuState, final bool) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	request := PreviewRequest{SwitchAddress: frame.Thread, Agent: frame.Agent}
	if final {
		argv, err := launcher.ParseLaunchParameters(frame.Input)
		if err != nil {
			state.Notice = errorMenuNotice(err.Error())
			return state, nil
		}
		raw, _ := json.Marshal(argv)
		request.SwitchArgv = string(raw)
		request.SwitchForce = frame.SelectedItem == "force-switch"
	}
	generation, ok := nextPreviewGeneration(&state)
	if !ok {
		return state, nil
	}
	frame = &state.Frames[len(state.Frames)-1]
	frame.Generation = generation
	frame.PreviewPending = generation
	if final {
		frame.SubmitGeneration = generation
		frame.SwitchSubmitAction = frame.SelectedItem
	} else {
		frame.SubmitGeneration = 0
		frame.SwitchSubmitAction = ""
	}
	request.Generation = generation
	return state, []MenuEffect{{Preview: &request}}
}

func reduceSwitchAgentPreview(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if event.Generation == 0 || event.Generation != frame.Generation || event.Generation != frame.PreviewPending {
		return state, nil
	}
	frame.PreviewPending = 0
	if event.Error != "" {
		state.Notice = errorMenuNotice(event.Error)
		return state, nil
	}
	p := event.SwitchPrepared
	if p == nil || p.Address != frame.Thread || p.Profile.Agent != frame.Agent || p.Fingerprint == "" {
		state.Notice = errorMenuNotice("invalid switch preview")
		return state, nil
	}
	previous := frame.SwitchPrepared
	frame.SwitchPrepared = p
	if frame.SubmitGeneration == event.Generation {
		frame.SubmitGeneration = 0
		if previous != nil && (previous.SourceRevision != p.SourceRevision || previous.SourceAgent != p.SourceAgent || previous.WorkingPath != p.WorkingPath) {
			state.Notice = errorMenuNotice("Source changed; review the updated switch details and select Switch again.")
			return state, nil
		}
		if frame.SwitchSubmitAction == "force-switch" {
			frame.SwitchSubmitAction = ""
			if p.Force == nil {
				state.Notice = errorMenuNotice("Force source proof unavailable; review again.")
				return state, nil
			}
			frame.SwitchStage = 2
			frame.SelectedItem = "cancel"
			return state, nil
		}
		effect := acceptedSwitchEffect(*p)
		if frame.SwitchSubmitAction == "interrupt-switch" {
			effect.Args["interrupt-first"] = "true"
		}
		frame.SwitchSubmitAction = ""
		return dispatchMenuOperation(state, effect, frame.Thread)
	}
	if !frame.SwitchEdited {
		frame.Input = launcher.FormatLaunchParameters(p.Profile.Argv)
		frame.SwitchCursorFromEnd = 0
	}

	return state, nil
}

func renderSwitchAgentMenu(frame MenuFrame, width, height int) []string {
	var lines []string
	switch frame.SwitchStage {
	case 0:
		return renderItemMenuFrame("switch coding agent", launcher.AgentInventory(), frame.Agent, "", width, height)
	case 1:
		source := "loading source"
		if frame.SwitchPrepared != nil {
			source = frame.SwitchPrepared.SourceAgent
		}
		mark := "  "
		if frame.SelectedItem == "parameters" {
			mark = "▸ "
		}
		lines = []string{"switch coding agent", fmt.Sprintf("%s → %s · fresh conversation", source, frame.Agent)}
		if p := frame.SwitchPrepared; p != nil {
			switch {
			case p.Conflict:
				lines = append(lines, "Configuration conflict", "Explicitly accept parameters to replace it.")
			case p.Borrowed != nil:
				lines = append(lines, "Borrowed — not saved for this repository", "From repository: "+p.Borrowed.RepoIdentity)
			case p.Configured:
				lines = append(lines, "Saved for this repository", "Accept edits to save them after startup.")
			default:
				lines = append(lines, "No saved configuration for this repository", "Explicit empty parameters are allowed.")
			}
		} else {
			lines = append(lines, "Resolving startup parameters", "Explicit empty parameters are allowed.")
		}
		lines = append(lines, mark+"parameters  "+frame.Input)
		button := func(item, label string) string {
			if frame.SelectedItem == item {
				return "▸ " + label
			}
			return "  " + label
		}
		lines = append(lines, button("switch", "Switch")+" · "+button("interrupt-switch", "Interrupt then switch"), button("force-switch", "Force switch")+" · "+button("cancel", "Cancel"))
		if frame.PreviewPending != 0 {
			lines = append(lines, "resolving startup parameters…")
		}
		lines = append(lines, "←→: text cursor · Tab/↑↓: focus · Enter: select · Escape: cancel")
	case 2:
		lines = []string{"force switch coding agent"}
		lines = append(lines, wrapNote("Force switch? The current turn's unsaved progress may be lost.", width)...)
		lines = append(lines, wrapNote(forceSwitchFileNotice(frame.SwitchPrepared.Force.ChangedFiles()), width)...)
		confirm, cancel := "  Force switch", "  Cancel"
		if frame.SelectedItem == "confirm-force" {
			confirm = "▸ Force switch"
		} else {
			cancel = "▸ Cancel"
		}
		lines = append(lines, confirm+" · "+cancel, "←→/Tab: focus · Enter: select · Escape: cancel")
	}

	for i, line := range lines {
		lines[i] = clipMenuLine(line, width)
	}
	return lines
}

func menuActionsFor(state MenuState, thread couchcore.ActionableThreadSummary) []string {
	items := menuActionItems(thread)
	if thread.Address.RepoScope != "" && thread.Address.Tag != "" {
		items = append(items, "resume-candidates")
	}
	return items
}

func invalidateSwitchEdit(state *MenuState, frame *MenuFrame) {
	generation, ok := nextPreviewGeneration(state)
	if ok {
		frame.Generation = generation
		frame.PreviewPending = 0
		frame.SubmitGeneration = 0
		frame.SwitchSubmitAction = ""
	}
}

func moveSwitchFocus(frame *MenuFrame, delta int) {
	items := []string{"parameters", "switch", "interrupt-switch", "force-switch", "cancel"}
	index := 0
	for i, item := range items {
		if frame.SelectedItem == item {
			index = i
			break
		}
	}
	frame.SelectedItem = items[(index+delta+len(items))%len(items)]
}

// menuRepositoryRoot shares the switcher's scope-matched root derivation,
// never its human label or fallback working-path display.
func menuRepositoryRoot(thread couchcore.ActionableThreadSummary) string {
	thread = presentationRow(thread)
	if thread.Target.Kind == couchcore.ThreadTargetSlot {
		return thread.Target.Slot.PrimaryRoot
	}
	return presentationRoot(thread.StartingPath, thread.Address.RepoScope)
}

// menuAddSlotPath projects the selected family's starting directory into the
// primary checkout. The create form carries that explicit choice through preview.
func menuAddSlotPath(thread couchcore.ActionableThreadSummary) string {
	thread = presentationRow(thread)
	primary := menuRepositoryRoot(thread)
	if primary == "" {
		return ""
	}
	root := primary
	if thread.Target.Kind == couchcore.ThreadTargetSlot {
		root = thread.Target.Slot.WorktreeRoot
	}
	relative, err := couchcore.RelativeFamilyPath(root, thread.StartingPath)
	if err != nil {
		return ""
	}
	path, err := couchcore.ProjectFamilyPath(primary, relative)
	if err != nil {
		return ""
	}
	return path
}

// menuAliasOffered puts the repository alias on a live primary (:0) row only:
// the alias names the repository, and :0 is the row that stands for it.
func menuAliasOffered(thread couchcore.ActionableThreadSummary) bool {
	return thread.Live() && thread.Target.Kind != couchcore.ThreadTargetSlot && menuRepositoryRoot(thread) != ""
}

func acceptedSwitchEffect(p couchcore.PreparedAgentSwitch) MenuEffect {
	effect := threadEffect("switch-agent", p.Address)
	raw, _ := json.Marshal(p.Profile.Argv)
	effect.Args["agent"] = p.Profile.Agent
	effect.Args["argv"] = string(raw)
	effect.Args["fingerprint"] = p.Fingerprint
	return effect
}

func forceSwitchFileNotice(count *int) string {
	if count == nil {
		return "Uncommitted file count is unknown; on-disk files stay as they are for the new agent."
	}
	return fmt.Sprintf("%d uncommitted files stay as they are for the new agent", *count)
}
