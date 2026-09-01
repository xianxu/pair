package wrapcmd

import (
	"strings"
)

const codexWorkingPrefix = "• Working ("
const codexWorkingSuffix = "esc to interrupt)"

// RecognizeCodexWorking recognizes Codex's live rendered status bar. It is
// deliberately confined to the bottom status region and exact line framing so
// quoted output and prose containing the same words cannot become activity.
func RecognizeCodexWorking(snapshot terminalSnapshot) bool {
	if snapshot.Width <= 0 || snapshot.Height < 3 {
		return false
	}
	firstRow := snapshot.Height - 8
	if firstRow < 0 {
		firstRow = 0
	}
	lastRow := snapshot.Height - 3
	for y := firstRow; y <= lastRow; y++ {
		line := strings.TrimSpace(terminalSnapshotRow(snapshot, y))
		if !strings.HasPrefix(line, codexWorkingPrefix) || !strings.HasSuffix(line, codexWorkingSuffix) {
			continue
		}
		middle := strings.TrimSuffix(strings.TrimPrefix(line, codexWorkingPrefix), codexWorkingSuffix)
		if !strings.ContainsAny(middle, "0123456789…") {
			continue
		}
		return true
	}
	return false
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
	working := RecognizeCodexWorking(p.terminal.Snapshot())
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
