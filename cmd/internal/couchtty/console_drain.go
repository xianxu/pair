package couchtty

import (
	"strconv"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
	for i := range b.Rows {
		r := &b.Rows[i]
		if r.IdleMillis != nil {
			value := *r.IdleMillis
			r.IdleMillis = &value
		}
		r.Settled = copyDrainFact(r.Settled)
		r.Picker = copyDrainFact(r.Picker)
		r.Background = copyDrainFact(r.Background)
	}
	return b
}
func copyDrainFact(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func drainStatusText(b couchcore.DrainBoard) string {
	if len(b.Rows) == 0 {
		return ""
	}
	return "drain " + strconv.Itoa(len(b.Rows))
}
func applyDrainStatus(m *StatusModel, b couchcore.DrainBoard) {
	m.Drain = drainStatusText(b)
}
