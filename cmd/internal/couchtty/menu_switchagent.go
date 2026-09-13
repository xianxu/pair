package couchtty

import (
	"encoding/json"
	"fmt"
	"strings"
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
	if key.Kind == KeyEscape {
		state.Frames = state.Frames[:len(state.Frames)-1]
		return state, nil
	}
	if state.InFlight.Operation != "" {
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
		case KeyTab, KeyDown, KeyRight:
			moveSwitchFocus(frame, 1)
		case KeyUp, KeyLeft:
			moveSwitchFocus(frame, -1)
		case KeyRune:
			if frame.SelectedItem == "parameters" && frame.SwitchPrepared != nil && key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && key.Rune >= 32 && key.Rune != 127 && len(frame.Input)+utf8.RuneLen(key.Rune) <= 4096 {
				frame.Input += string(key.Rune)
				frame.SwitchEdited = true
				invalidateSwitchEdit(&state, frame)
			}
		case KeyBackspace:
			if frame.SelectedItem == "parameters" && frame.SwitchPrepared != nil {
				frame.Input = removeLastRune(frame.Input)
				frame.SwitchEdited = true
				invalidateSwitchEdit(&state, frame)
			}
		case KeyEnter:
			switch frame.SelectedItem {
			case "parameters":
				frame.SelectedItem = "switch"
			case "cancel":
				state.Frames = state.Frames[:len(state.Frames)-1]
			case "switch":
				if frame.PreviewPending != 0 {
					return state, nil
				}
				if frame.SwitchPrepared == nil {
					return requestSwitchAgentPreview(state, false)
				}
				return requestSwitchAgentPreview(state, true)
			}
		}
	}

	return state, nil
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
	} else {
		frame.SubmitGeneration = 0
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
		effect := threadEffect("switch-agent", frame.Thread)
		raw, _ := json.Marshal(p.Profile.Argv)
		effect.Args["agent"] = p.Profile.Agent
		effect.Args["argv"] = string(raw)
		effect.Args["fingerprint"] = p.Fingerprint
		return dispatchMenuOperation(state, effect, frame.Thread)
	}
	if !frame.SwitchEdited {
		frame.Input = launcher.FormatLaunchParameters(p.Profile.Argv)
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
		lines = []string{"switch coding agent", fmt.Sprintf("%s → %s · same thread", source, frame.Agent), "Fresh conversation; read source context and summarize orientation.", "Edit startup parameters (empty is allowed)", mark + "parameters  " + frame.Input}
		for _, item := range []string{"switch", "cancel"} {
			mark = "  "
			if item == frame.SelectedItem {
				mark = "▸ "
			}
			lines = append(lines, mark+strings.ToUpper(item[:1])+item[1:])
		}
		if frame.PreviewPending != 0 {
			lines = append(lines, "resolving startup parameters…")
		}
		lines = append(lines, "Tab/↑↓: focus · Enter: select · Escape: cancel")
	}

	for i, line := range lines {
		lines[i] = clipMenuLine(line, width)
	}
	return lines
}

func menuActionsFor(state MenuState, thread couchcore.ActionableThreadSummary) []string {
	items := menuActionItems(thread)
	if _, ok := state.Orientation[thread.Address]; ok {
		items = append(items, "copy-orientation")
	}
	return items
}

func invalidateSwitchEdit(state *MenuState, frame *MenuFrame) {
	generation, ok := nextPreviewGeneration(state)
	if ok {
		frame.Generation = generation
		frame.PreviewPending = 0
		frame.SubmitGeneration = 0
	}
}

func moveSwitchFocus(frame *MenuFrame, delta int) {
	items := []string{"parameters", "switch", "cancel"}
	index := 0
	for i, item := range items {
		if frame.SelectedItem == item {
			index = i
			break
		}
	}
	frame.SelectedItem = items[(index+delta+len(items))%len(items)]
}
