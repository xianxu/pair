package couchtty

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func openRecoveryPreview(state MenuState, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	generation, ok := nextPreviewGeneration(&state)
	if !ok {
		return state, nil
	}
	target := couchcore.ResumeTarget{Address: row.Address}
	if row.Target.Kind == couchcore.ThreadTargetSlot {
		target = couchcore.ResumeTarget{Path: row.Target.Slot.WorktreeRoot}
	}
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameRecovery, Thread: row.Address, RowKey: menuRowKey(row), Generation: generation, PreviewPending: generation, RecoveryTarget: target})
	return state, []MenuEffect{{Preview: &PreviewRequest{Generation: generation, RecoveryTarget: &target}}}
}
func recoveryChoices(preview *couchcore.RecoveryPreview) []couchcore.RecoveryChoice {
	if preview == nil {
		return nil
	}
	var choices []couchcore.RecoveryChoice
	if preview.Proven != nil {
		choices = append(choices, *preview.Proven)
	}
	limit := 5
	if preview.Proven != nil {
		limit = 4
	}
	for i, candidate := range preview.Candidates {
		if i >= limit {
			break
		}
		if candidate.Key != "" {
			choices = append(choices, candidate)
		}
	}
	if preview.Fresh != nil {
		choices = append(choices, *preview.Fresh)
	}
	return choices
}
func reduceRecoveryPreview(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if event.Generation != frame.Generation || event.Generation != frame.PreviewPending {
		return state, nil
	}
	frame.PreviewPending = 0
	if event.Error != "" || event.RecoveryPreview == nil {
		state.Notice = errorMenuNotice(event.Error)
		frame.RecoveryPreview = &couchcore.RecoveryPreview{Address: frame.Thread, Incomplete: true, Reasons: []string{event.Error, "Conversation evidence unavailable; starting fresh may lose working context."}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}
		frame.SelectedItem = "0"
		return state, nil
	}
	preview := *event.RecoveryPreview
	preview.Candidates = append([]couchcore.RecoveryChoice(nil), preview.Candidates...)
	if preview.Proven != nil {
		proven := *preview.Proven
		preview.Proven = &proven
	}
	preview.Reasons = append([]string(nil), preview.Reasons...)
	if preview.Fresh != nil {
		fresh := *preview.Fresh
		preview.Fresh = &fresh
	}
	frame.RecoveryPreview = &preview
	frame.SelectedItem = "0"
	return state, nil
}

func reduceRecoveryKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if key.Kind == KeyEscape || key.Kind == KeyLeft {
		generation := frame.Generation
		state.Frames = state.Frames[:len(state.Frames)-1]
		return state, []MenuEffect{{CancelPreview: generation}}
	}
	choices := recoveryChoices(frame.RecoveryPreview)
	index, _ := strconv.Atoi(frame.SelectedItem)
	switch key.Kind {
	case KeyUp:
		if index > 0 {
			index--
		}
	case KeyDown:
		if index+1 < len(choices) {
			index++
		}
	case KeyEnter:
		if frame.PreviewPending != 0 || index < 0 || index >= len(choices) {
			return state, nil
		}
		choice := choices[index]
		if choice.Key == "" {
			if choice.NativeID == "" {
				if row, ok := menuThreadTarget(state, frame.RowKey, frame.Thread); ok {
					args := couchcore.ActorOperationArgs(row, "reboot")
					return dispatchMenuOperation(state, MenuEffect{RowKey: menuRowKey(row), Operation: "reboot", Args: args}, row.Address)
				}
				return state, nil
			}
		}
		args := map[string]string{"repo-scope": frame.Thread.RepoScope}
		if choice.Key != "" {
			args["candidate"] = choice.Key
		}
		if frame.RecoveryTarget.Path != "" {
			args["path"] = frame.RecoveryTarget.Path
		} else {
			args["repo-scope"] = frame.RecoveryTarget.Address.RepoScope
			args["tag"] = string(frame.RecoveryTarget.Address.Tag)
		}
		rowKey, address := frame.RowKey, frame.Thread
		next, effects := dispatchMenuOperation(state, MenuEffect{Operation: "resume", RowKey: rowKey, Args: args}, address)
		return next, effects
	}
	frame.SelectedItem = strconv.Itoa(index)
	return state, nil
}
func renderRecoveryMenu(frame MenuFrame, width, height int) []string {
	lines := []string{"resume candidates", "Choose explicitly; no automatic recovery."}
	if frame.PreviewPending != 0 {
		return append(lines, "Looking for matching conversations…")
	}
	selected, _ := strconv.Atoi(frame.SelectedItem)
	choices := recoveryChoices(frame.RecoveryPreview)
	for i, choice := range choices {
		marker := "  "
		if i == selected {
			marker = "▸ "
		}
		label := choice.Label
		nativeOnly := choice.SourceKind == sessioninventory.RecoverySourceNative
		if nativeOnly && !strings.Contains(strings.ToLower(label), "unconfirmed") {
			label += " (unconfirmed)"
		}
		if !nativeOnly && choice.Key != "" && (i == 0 && frame.RecoveryPreview.Proven == nil || i == 1 && frame.RecoveryPreview.Proven != nil) && len(frame.RecoveryPreview.Candidates) > 0 {
			label = "best · " + label
		}
		activity := "unknown"
		if !choice.LastActivity.IsZero() {
			activity = choice.LastActivity.Local().Format("2006-01-02 15:04 MST")
		}
		evidence := "  Original saved screen"
		if choice.NativeID == "" && choice.Key == "" {
			evidence = "  Fresh conversation"
		}
		if choice.NativeID != "" && choice.Key == "" {
			evidence = "  Established native binding"
		}
		if choice.NativeID != "" && choice.Key != "" {
			evidence = fmt.Sprintf("  matched %d/%d · tail %.0f%%", choice.Matched, choice.Total, choice.TailMatch)
		}
		activityLabel := "last activity"
		if nativeOnly {
			activityLabel = "native activity"
			evidence = "  Native transcript · physical cwd " + choice.CWD + " · launch association unknown"
		}
		lines = append(lines, marker+label, fmt.Sprintf("  %s · %s · %s %s", choice.Agent, choice.NativeID, activityLabel, activity), evidence, "  "+choice.Excerpt)
	}
	if frame.RecoveryPreview != nil {
		for _, evidence := range frame.RecoveryPreview.Candidates {
			if evidence.Key == "" {
				if evidence.SourceKind == sessioninventory.RecoverySourceNative {
					activity := "unknown"
					if !evidence.LastActivity.IsZero() {
						activity = evidence.LastActivity.Local().Format("2006-01-02 15:04 MST")
					}
					lines = append(lines, fmt.Sprintf("Recorded native evidence (unconfirmed): %s · %s", evidence.Agent, evidence.NativeID), "  physical cwd "+evidence.CWD+" · native activity "+activity, "  "+evidence.Excerpt)
				} else {
					lines = append(lines, fmt.Sprintf("Recorded evidence: %s · %s · tail %.0f%%", evidence.Agent, evidence.NativeID, evidence.TailMatch), "  "+evidence.Excerpt)
				}
			}
		}
		if screen := frame.RecoveryPreview.Screen; !screen.LastActivity.IsZero() {
			source := screen.ActivitySource
			if source == "" {
				source = "capture-file activity"
			}
			lines = append(lines, source+" "+screen.LastActivity.Local().Format("2006-01-02 15:04 MST"))
		}
		lines = append(lines, frame.RecoveryPreview.Reasons...)
		if frame.RecoveryPreview.Incomplete {
			lines = append(lines, "Scan incomplete.")
		}
		if frame.RecoveryPreview.Truncated {
			lines = append(lines, "More candidates omitted.")
		}
	}
	if len(choices) == 0 {
		lines = append(lines, "No recovery choices available.")
	}
	for i, line := range lines {
		lines[i] = clipMenuLine(rowtext.Sanitize(strings.TrimSpace(line)), width)
	}
	// Keep the selected evidence visible in short terminals.
	if len(lines) > height && height > 2 {
		start := 2 + selected*4
		if start+height-2 > len(lines) {
			start = len(lines) - (height - 2)
		}
		if start < 2 {
			start = 2
		}
		lines = append(lines[:2], lines[start:start+height-2]...)
	}
	return lines
}
