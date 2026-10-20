package wrapcmd

import (
	"strconv"
	"strings"
)

const codexWorkingSuffix = "esc to interrupt)"

// codexStatusLine reads the native status immediately above a qualified
// composer. Codex 0.163.0 leaves two blank rows between them; its short UI
// need not reach the physical bottom of the viewport.
func codexStatusLine(snapshot terminalSnapshot) string {
	if !snapshot.Complete {
		return ""
	}
	prompt, _, ok := codexComposerBounds(snapshot)
	if !ok {
		return ""
	}
	for y := prompt - 1; y >= 0 && y >= prompt-3; y-- {
		if line := terminalSnapshotRow(snapshot, y); strings.TrimSpace(line) != "" {
			// The native background summary can wrap the status. Preserve
			// full rows so a word split at the viewport edge rejoins intact.
			for above := y - 1; above >= 0 && above >= y-2 && snapshotRowPainted(snapshot, above); above-- {
				line = terminalSnapshotRow(snapshot, above) + line
			}
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// RecognizeCodexWorking is the shared current work proof for notification
// lifecycle and interrupt admission. Startup's MCP status has the same
// geometry and interrupt hint, so only measured turn-work wording qualifies.
func RecognizeCodexWorking(snapshot terminalSnapshot) bool {
	line := codexStatusLine(snapshot)
	if strings.HasPrefix(line, "• ") || strings.HasPrefix(line, "◦ ") {
		_, line, _ = strings.Cut(line, " ")
	} else {
		return false
	}
	for _, prefix := range []string{"Working (", "Waiting for background terminal ("} {
		if strings.HasPrefix(line, prefix) {
			middle, background, ok := strings.Cut(strings.TrimPrefix(line, prefix), codexWorkingSuffix)
			if !ok || !codexBackgroundSummary(background) {
				return false
			}
			return strings.ContainsAny(middle, "0123456789")
		}
	}
	return false
}

// A running shell appends this native summary to the same status row.
func codexBackgroundSummary(s string) bool {
	if s == "" {
		return true
	}
	fields := strings.Fields(s)
	if len(fields) != 13 || fields[2] != "background" {
		return false
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil || n <= 0 || strconv.Itoa(n) != fields[1] {
		return false
	}
	terminal := "terminal"
	if n != 1 {
		terminal = "terminals"
	}
	return s == " · "+fields[1]+" background "+terminal+" running · /ps to view · /stop to close"
}

func terminalSnapshotRow(snapshot terminalSnapshot, y int) string {
	var line strings.Builder
	line.Grow(snapshot.Width)
	for x := 0; x < snapshot.Width; x++ {
		cell := snapshot.CellAt(x, y)
		if cell == nil || cell.Content == "" {
			line.WriteByte(' ')
			continue
		}
		line.WriteString(cell.Content)
	}
	return line.String()
}

func (p *proxy) observeCodexWorking() {
	if p.agentBasename != "codex" || p.terminal == nil {
		return
	}
	snapshot := p.terminal.Snapshot()
	if !snapshot.Complete || !p.terminal.FrameComplete() {
		return // A partial redraw is not evidence that work stopped.
	}
	working := RecognizeCodexWorking(snapshot)
	if working == p.codexWorkingRendered {
		return
	}
	p.codexWorkingRendered = working
	kind := ObservationStopped
	if working {
		kind = ObservationWorking
	}
	p.processLifecycleObservation(TurnObservation{Kind: kind})
}
