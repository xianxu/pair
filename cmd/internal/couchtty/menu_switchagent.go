package couchtty

import (
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"strings"
	"unicode/utf8"
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
			return requestSwitchAgentPreview(state, false)
		}
	case 1:
		switch key.Kind {
		case KeyRune:
			if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && key.Rune >= 32 && key.Rune != 127 && len(frame.Input)+utf8.RuneLen(key.Rune) <= 4096 {
				frame.Input += string(key.Rune)
				frame.SwitchEdited = true
				frame.SwitchPrepared = nil
				invalidateSwitchEdit(&state, frame)
			}
		case KeyBackspace:
			frame.Input = removeLastRune(frame.Input)
			frame.SwitchEdited = true
			frame.SwitchPrepared = nil
			invalidateSwitchEdit(&state, frame)
		case KeyEnter:
			if frame.PreviewPending != 0 {
				return state, nil
			}
			return requestSwitchAgentPreview(state, true)
		}
	case 2:
		switch key.Kind {
		case KeyTab, KeyUp, KeyDown, KeyLeft, KeyRight:
			if frame.SelectedItem == "cancel" {
				frame.SelectedItem = "switch"
			} else {
				frame.SelectedItem = "cancel"
			}
		case KeyEnter:
			if frame.SelectedItem != "switch" {
				state.Frames = state.Frames[:len(state.Frames)-1]
				return state, nil
			}
			if frame.SwitchPrepared == nil {
				return state, nil
			}
			p := frame.SwitchPrepared
			effect := threadEffect("switch-agent", frame.Thread)
			raw, _ := json.Marshal(p.Profile.Argv)
			effect.Args["agent"] = p.Profile.Agent
			effect.Args["argv"] = string(raw)
			effect.Args["fingerprint"] = p.Fingerprint
			return dispatchMenuOperation(state, effect, frame.Thread)
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
	frame.SwitchPrepared = nil
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
	if frame.SubmitGeneration == event.Generation {
		frame.SwitchPrepared = p
		frame.SwitchStage = 2
		frame.SelectedItem = "cancel"
	} else if !frame.SwitchEdited {
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
		lines = []string{"switch coding agent", "agent: " + frame.Agent, "Edit startup parameters (empty is allowed)", "▸ parameters  " + frame.Input, "Enter: review switch · Escape: cancel"}
		if frame.PreviewPending != 0 {
			lines = append(lines, "loading startup parameters…")
		}
	case 2:
		p := frame.SwitchPrepared
		if p == nil {
			return []string{"switch coding agent", "preview unavailable"}
		}
		lines = []string{"switch coding agent", fmt.Sprintf("%s → %s · same thread", p.SourceAgent, p.Profile.Agent), "Fresh conversation; read source context and summarize orientation.", "parameters: " + launcher.FormatLaunchParameters(p.Profile.Argv)}
		for _, item := range []string{"switch", "cancel"} {
			mark := "  "
			if item == frame.SelectedItem {
				mark = "▸ "
			}
			lines = append(lines, mark+strings.ToUpper(item[:1])+item[1:])
		}
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
