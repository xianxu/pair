package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// SetBlockedThreads hands the Console the threads whose agent is behind an
// input-holding picker (pair#465). It never blocks: the newest set replaces
// one not yet taken, and Console.Run applies it and repaints, so the message
// service's loop never waits on the Console.
func (c *Console) SetBlockedThreads(blocked map[couchcore.ThreadAddress]bool) {
	for {
		select {
		case c.blockedUpdates <- blocked:
			return
		default:
		}
		select {
		case <-c.blockedUpdates:
		default:
		}
	}
}
