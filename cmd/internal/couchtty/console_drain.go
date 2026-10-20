package couchtty

import (
	"strconv"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// SetDrainBoard publishes an immutable snapshot to the Console event loop.
// The newest value replaces an unconsumed value; callers never wait for paint.
func (c *Console) SetDrainBoard(board couchcore.DrainBoard) {
	board = copyDrainBoard(board)
	for {
		select {
		case c.drainUpdates <- board:
			return
		default:
		}
		select {
		case <-c.drainUpdates:
		default:
		}
	}
}
func copyDrainBoard(b couchcore.DrainBoard) couchcore.DrainBoard {
	b.Rows = append([]couchcore.DrainRow(nil), b.Rows...)
	b.Queue.Owners = append([]string(nil), b.Queue.Owners...)
	b.Queue.Active = append([]couchcore.DrainActivity(nil), b.Queue.Active...)
	return b
}
func drainStatusText(b couchcore.DrainBoard) string {
	if b.ID == "" {
		return ""
	}
	state := drainStateText(b.State)
	if state == "" {
		return ""
	}
	if detail := rowtext.Sanitize(b.Detail); detail != "" {
		return state + ": " + detail
	}
	return state
}
func drainStateText(s couchcore.DrainVisibleState) string {
	switch s {
	case couchcore.DrainDraining, couchcore.DrainStuck, couchcore.DrainDrained, couchcore.DrainCancelled:
		return string(s)
	}
	return ""
}
func applyDrainStatus(m *StatusModel, b couchcore.DrainBoard) {
	m.Drain = drainStatusText(b)
	if m.Drain == "" {
		return
	}
	byWorkspace := make(map[string]couchcore.DrainVisibleState, len(b.Rows))
	for _, r := range b.Rows {
		byWorkspace[r.Workspace] = r.State
	}
	for i := range m.Actors {
		a := &m.Actors[i]
		workspace := a.Workspace
		if workspace == "" {
			workspace = a.Label
		}
		state, found := byWorkspace[workspace]
		if !found {
			state = byWorkspace[workspace+":0"]
		}
		a.Drain = state
	}
}
func drainSlotWorkspace(s couchcore.SlotIdentity) string {
	if s.Repo == "" {
		return ""
	}
	return s.Repo + ":" + strconv.Itoa(s.Number)
}
